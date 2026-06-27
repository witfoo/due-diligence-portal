# Adversarial Code Audit — Due Diligence Portal

**Date:** 2026-06-27
**Scope:** Entire codebase (Go API, SvelteKit UI, SQLite schema/migrations, middleware, deployment) — ~16 kLOC.
**Method:** Multi-agent adversarial review. 11 finder dimensions + 2 completeness critics fanned out across the
codebase; **every** finding was then re-checked by an independent skeptic that read the actual code and tried to
refute it. 109 candidate findings examined → **102 confirmed**, 7 refuted. Findings below are deduplicated and
merged into distinct issues.

> The reported symptom — **"document uploads gives a 404"** — is real and root-caused. It is the first finding
> below (C1). It is one instance of a broader class: **five SvelteKit route directories exist but are completely
> empty (no `+page.svelte`)**, so several core user journeys dead-end on a blank not-found screen even though the
> backend endpoints work.

---

## Executive summary

| Severity | Count | Theme |
| --- | --- | --- |
| 🔴 Critical | 6 | Upload/detail/Q&A pages missing; **document access control (IDOR) is wide open**; downloads 401; Q&A leaks across investors; forgeable JWT secret |
| 🟠 High | 14 | NDA flow non-functional; watermark never applied; invite/onboarding broken; non-transactional writes; FTS 500s; spoofable IPs; DoS; localStorage tokens; no client auth guard |
| 🟡 Medium | 21 | CSS injection, MIME trust, analytics always-zero/forgeable, audit-trail gaps, self-lockout, brute-force, swallowed errors |
| 🔵 Low | 26 | Validation gaps, contract drift, FTS rowid fragility, weak filename/CSS sanitizers, misleading "embedded UI" comments |
| ⚪ Info | 6 | Hardening notes; build/test baseline is green |

**The headline is not the upload 404 — it is broken document authorization.** Any authenticated user (including a
low-privilege *investor*) can read and download **every** document in the data room by hitting the by-ID endpoints
directly (`GET /documents/:id`, `/download`, `/versions/:version`, `POST /documents/search`). The per-investor
access-grant model is enforced in exactly one place (the list view) and bypassed everywhere else. For a
confidential M&A data room this is the most serious defect in the system. See **C4**.

**Cross-cutting themes**
1. **Half-wired features.** The backend implements features the UI never reaches: upload, document detail, Q&A
   threads, NDA signing, invite acceptance, watermarking, view analytics, branding asset upload, email
   notifications. CLAUDE.md claims "All phases complete"; the data plane and control plane were built but never
   connected end to end.
2. **Authorization enforced inconsistently.** List filters by access grant; Get/Download/Search/analytics do not.
   NDA signing never gates anything. No client-side route guard exists.
3. **No write atomicity.** Document create/version writes run as two independent autocommit statements; a failure
   between them orphans rows or corrupts the "current version" pointer.
4. **Trust of client-supplied data.** MIME types, `X-Forwarded-For`, NDA signer names, view-event durations, and
   branding colors are all trusted without validation.

A short remediation roadmap is at the end.

---

## 🔴 Critical

### C1. Document upload is unreachable — `/documents/upload` route dir is empty (the reported 404)
**Category:** missing-feature · **Where:** `ui/src/routes/documents/upload/` (empty, no `+page.svelte`); link at [documents/+page.svelte:68](ui/src/routes/documents/+page.svelte#L68)

The "Upload Document" button links to `/documents/upload`, but that route directory contains **no `+page.svelte`**
(only `.`/`..`). The app is built with `@sveltejs/adapter-static` and `fallback: 'index.html'` (SPA mode), so the
Go server returns `index.html` with HTTP 200, then SvelteKit's client router finds no matching route and renders
its built-in "Not found" page. The backend `POST /api/v1/documents` ([document_handler.go:121](internal/handler/document_handler.go#L121))
is fully implemented and works — there is simply **no UI to reach it**.

**Impact:** The core write path of the data room is completely unreachable through the product. Admins/uploaders
cannot add a single document.

**Fix:** Create `ui/src/routes/documents/upload/+page.svelte` with a file picker + name/description/category form
that POSTs `multipart/form-data` (`file`, `name`, `description`, `category_id`, `tags`) to `/api/v1/documents` via
the typed client, then `goto('/documents')` on success.

---

### C2. Four more empty route dirs dead-end core navigation (document detail, Q&A thread, NDA sign, unauthorized)
**Category:** missing-feature · **Where:** `ui/src/routes/documents/[id]/`, `ui/src/routes/qa/[threadId]/`, `ui/src/routes/nda-sign/[token]/`, `ui/src/routes/unauthorized/` (all empty)

Beyond upload, four further route directories exist with no `+page.svelte`:

- **`documents/[id]`** — every document name links to `/documents/{id}` ([documents/+page.svelte:110](ui/src/routes/documents/+page.svelte#L110)). There is no detail/version-history/access-grant view; clicking a document name dead-ends.
- **`qa/[threadId]`** — every thread card links to `/qa/{thread.id}` ([qa/+page.svelte:89](ui/src/routes/qa/+page.svelte#L89)). Users can *create* threads but can never open one to read messages or reply — the entire `qa_messages` flow is unreachable (Q&A is "write-then-lost").
- **`nda-sign/[token]`** — the NDA signing page (see H1 for the deeper backend story).
- **`unauthorized`** — the intended RBAC-denial target. Note: nothing actually redirects here today (no reference anywhere), so it is unwired scaffolding rather than a reachable-but-broken path.

**Impact:** Document detail and Q&A threads — primary navigation on the two core resources — dead-end on a blank,
unbranded not-found screen (there is also no `+error.svelte`, see M1). Directly contradicts "All phases complete."

**Fix:** Implement `+page.svelte` for each: `documents/[id]` → `GET /documents/:id` (doc + versions, emit a
view-event); `qa/[threadId]` → `GET /qa/:id` (render messages, post replies, change status); `unauthorized` → a
static denial page. (`nda-sign` covered in H1.)

---

### C3. Document download is broken for everyone — a plain `<a href>` can't send the JWT, so every download 401s
**Category:** contract-mismatch · **Where:** [documents/+page.svelte:120](ui/src/routes/documents/+page.svelte#L120); [jwt_auth.go:25-28](internal/middleware/jwt_auth.go#L25-L28); [document_handler.go:46](internal/handler/document_handler.go#L46)

The Download action is a full-navigation hyperlink: `<a href="/api/v1/documents/{doc.id}/download">`. The route is
on the JWT-protected group and the auth middleware reads the token **only** from the `Authorization` header — there
is no cookie or query-param fallback. A browser navigating to an `<a href>` does **not** attach that header, so the
request reaches `JWTAuth` with no token and returns 401. The token lives in `localStorage` and is only attached by
the fetch-based client, which this anchor bypasses entirely. Worse, because the 401 comes from a top-level
navigation (not the client's `fetch`), the client's 401 handler never fires — the user lands on a raw JSON error
page `{"success":false,"error":"missing authorization header"}`.

**Impact:** Document download — the other core feature of a data room — is completely broken for all users. The
same transport flaw applies to any future `<a href>` to `/versions/:version` and `/branding/assets/:key`.

**Fix:** Download via the typed client: `fetch` with the `Authorization` header, read the response as a `Blob`, and
trigger a programmatic `URL.createObjectURL` download. Alternatively add a short-lived signed download-token (query
param) or an HttpOnly cookie the middleware also accepts for GET download routes.

---

### C4. Broken document access control (IDOR): any investor can read & download every document
**Category:** rbac / broken access control · **Where:** [document_handler.go](internal/handler/document_handler.go) — `Get` (99-118), `Download` (383-409), `DownloadVersion` (349-381), `Search` (416-437)

This is the most serious defect in the system. The per-investor **access-grant** model is enforced in exactly one
place — the `List` handler ([document_handler.go:71-87](internal/handler/document_handler.go#L71-L87) loops
`permRepo.HasAccess` for investors). **Every other read path skips it:**

- **`GET /documents/:id`** (Get, line 99) — returns document metadata + full version history by ID with no access/role check.
- **`GET /documents/:id/download`** (Download, line 383) — streams the raw current-version file blob by ID with no check. **Full file exfiltration.**
- **`GET /documents/:id/versions/:version`** (DownloadVersion, line 349) — streams any historical version blob with no check (additionally exposes superseded/redacted prior versions).
- **`POST /documents/search`** (Search, line 416) — runs an unfiltered FTS query and returns every matching document (name, description, tags, category, size, uploader) to any authenticated caller.

All four are registered on the auth-only group with **no `RequireRole`** and **no `HasAccess` call**.

**Impact:** An investor (e.g. a competing bidder) can enumerate document IDs — or just run a broad search — and
download every confidential document (financials, contracts, IP), including documents explicitly never shared with
them and prior versions that may contain redacted data. Complete bypass of the data room's central security control.

**Fix:** Centralize one `canAccessDocument(ctx, userID, role, docID)` helper that, for non-admin/non-company roles,
checks **both** `ResourceDocument` and `ResourceCategory` grants (resolving the document's category — see H2), and
call it from `Get`, `Download`, `DownloadVersion`, **and** `Search` (filter results) before returning data. Prefer
pushing the permission filter into SQL over N per-doc calls. Return 403/404 when not granted.

---

### C5. Q&A threads are global — investors can read every other bidder's questions, answers, and internal notes
**Category:** rbac · **Where:** [qa_handler.go:39-56,92-113](internal/handler/qa_handler.go#L39-L113); [qa_repository.go:79-105](internal/repository/qa_repository.go#L79-L105)

`ListThreads` queries `qa_threads` with **no `asked_by`/user filter** (repo: `WHERE status = ? ... LIMIT ? OFFSET ?`),
and `GetThread` fetches any thread by ID with **no ownership check**. The schema has an `asked_by` column but it is
never used to scope results. Any authenticated investor sees all threads and can open any by ID. Compounding this,
`ListMessages` ([qa_repository.go:166-186](internal/repository/qa_repository.go#L166-L186)) returns **all** messages
including `is_internal = 1`, and `GetThread` returns them verbatim with no role filter — even though `PostMessage`
carefully restricts *setting* `is_internal` to admin/company roles, establishing those messages are meant to be
hidden from investors.

**Impact:** In a multi-bidder data room, one investor can read all other bidders' due-diligence questions and the
company's confidential answers — revealing competitors' interest areas and deal strategy — plus company-side
internal-only commentary deliberately marked private. Severe confidentiality breach. (Currently reachable via the
API directly; the empty `qa/[threadId]` page from C2 only hides it from the rendered UI, not from the live endpoint.)

**Fix:** Scope `ListThreads`/`GetThread` to the requesting user for the investor role (`asked_by = userID`), and add
a role-aware `ListMessages` that appends `AND is_internal = 0` for investor callers. Admin/company roles may see all.

---

### C6. Hardcoded fallback JWT signing secret allows full authentication bypass
**Category:** security · **Where:** [cmd/main.go:97](cmd/main.go#L97); [auth_service.go:40-44,257-273](internal/service/auth_service.go#L40-L44)

`DD_JWT_SECRET` falls back to the literal string `"dev-secret-change-in-production-32chars"` when unset — a value
committed to source control. Tokens are HS256 (symmetric), so anyone who knows this string can mint a valid JWT for
any `user_id` with `role=admin`. **Nothing** in the binary checks that the secret was overridden, has entropy, or
isn't the known default; the server boots and signs/validates normally. `docs/ENVIRONMENT.md` says it "Must be set
in production" but nothing enforces it. (`start.sh` does generate a random secret via `openssl rand -hex 32`, so
deployments using that script are protected — but `./portal` run directly, or any custom Docker/env invocation,
silently uses the public default.)

**Impact:** Any deployment that forgets `DD_JWT_SECRET` is a complete, silent authentication & RBAC bypass — forge
an admin token and own every endpoint, including all document downloads and permission management.

**Fix:** At startup, `log.Fatal` if `DD_JWT_SECRET` is empty, equals the known default, or is shorter than 32 bytes,
unless an explicit `DD_DEV_MODE=true` is set. Never embed a production-plausible default literal.

---

## 🟠 High

### H1. The entire NDA workflow is non-functional — no signing UI, no caller, no access gating
**Category:** missing-feature · **Where:** [nda_handler.go:24-32,129-216](internal/handler/nda_handler.go#L24-L216); `ui/src/routes/nda-sign/[token]/` (empty); [document_handler.go:37-48](internal/handler/document_handler.go#L37-L48) (no NDA gate)

NDA is half-built across three layers: (1) the `nda-sign/[token]` route is empty, so there is no signing page;
(2) **no UI anywhere calls** `GET /nda/status` or `POST /nda/sign/:templateId` (grep finds only admin *template*
management) — so investors can never sign; (3) there is **no NDA-enforcement middleware** — document
list/get/download/search and Q&A never consult `HasSigned`, so an investor who never signed has identical access to
one who did. There is also a **contract mismatch**: the empty UI route is token-based (`/nda-sign/{token}`) but the
backend is authenticated and template-id-based (`/nda/sign/:templateId`) with no token concept — even building the
page requires reconciling the two.

**Impact:** A stated legal/compliance gating control for the data room is entirely inert. Admins can create NDA
templates, but no one can sign them through the product, and signing would not restrict access anyway.

**Fix:** Decide the contract (authenticated template-id gate is simplest): on login/layout call `GET /nda/status`;
if an investor is unsigned, redirect to a signing page that POSTs `/nda/sign/:templateId`. Add middleware on the
document/Q&A groups that returns 403 (pointing at the sign flow) when an active template exists and `HasSigned` is
false. Align or remove the `nda-sign/[token]` route shape.

---

### H2. The investor access filter ignores category-level grants — and is the sole enforcement point
**Category:** rbac · **Where:** [document_handler.go:71-87](internal/handler/document_handler.go#L71-L87); [permission_repository.go:127-141](internal/repository/permission_repository.go#L127-L141)

The `List` investor filter only checks `ResourceDocument` grants (line 77) and **never** `ResourceCategory` grants,
even though the permission model and `PermissionHandler` explicitly support category grants. An investor granted an
entire category therefore sees **zero** documents in the list. Meanwhile this single filter is the *only* place
access is enforced, so the C4 IDOR endpoints sidestep it entirely. (Note: download-vs-view is not the bug — the
access hierarchy makes `download` include `view`; the bug is the hardcoded resource type.)

**Impact:** The access model is internally inconsistent: legitimate category-scoped investors see nothing, while
direct-ID endpoints leak everything.

**Fix:** The same `canAccessDocument` helper from C4 must resolve the document's category and check **both**
`ResourceDocument` and `ResourceCategory` grants, used by `List` filtering and every by-ID endpoint.

---

### H3. RefreshToken accepts an access token as a refresh token, and refresh tokens are never rotated or revoked
**Category:** security · **Where:** [auth_service.go:137-154,257-273](internal/service/auth_service.go#L137-L154)

Access and refresh tokens are minted by the same `generateToken()` with **identical claims**, differing only in
`ExpiresAt`. There is no `token_type`/scope claim, so: a short-lived *access* token is fully accepted at the refresh
endpoint, and a 7-day *refresh* token is accepted anywhere an access token is (it passes `JWTAuth` identically).
There is also no rotation/revocation/`jti`, so logout or compromise cannot invalidate a refresh token before expiry.

**Impact:** Token confusion. A leaked refresh token works as a bearer access token against every protected API for a
week, eliminating the benefit of the 15-minute access-token TTL; nothing can revoke it.

**Fix:** Add a `TokenType` claim (`access`/`refresh`); require `refresh` in `RefreshToken` and `access` in `JWTAuth`.
Persist refresh `jti` to support rotation/revocation and a real logout (see L18).

---

### H4. Document writes are non-transactional — failures orphan rows or corrupt the "current version" pointer
**Category:** data-integrity · **Where:** [document_handler.go:184-202](internal/handler/document_handler.go#L184-L202) (Upload), [document_handler.go:331-341](internal/handler/document_handler.go#L331-L341) (UploadVersion)

No repository code ever opens a transaction; all writes run in SQLite autocommit. Two cases:

- **Upload** does `Create` (INSERT `documents`) then a separate `CreateVersion` (INSERT `document_versions`). If the
  second fails (disk full, BLOB too large, context cancel/client disconnect), the `documents` row is already
  committed with `current_version=1` but **no version row**. The doc appears in `List` but `Download` calls
  `GetVersion` → `ErrVersionNotFound` → 404 forever, and it can't self-heal (Upload always mints a fresh ID).
- **UploadVersion** does `CreateVersion` then a separate `Update` of `current_version`/`mime_type`/`file_size`. If
  the `Update` fails, the new version is committed but `current_version` still points at the old one, so `Download`
  serves the **stale** file and metadata diverges from the actual blob. A retry recomputes the same version number
  and hits `UNIQUE(document_id, version_number)` → permanent 500 (the document gets stuck).

**Impact:** Silent data corruption / orphan rows on any mid-write failure. Not attacker-triggerable, but a real
integrity hazard for the system of record.

**Fix:** Wrap each pair in a single `sql.Tx` (add `CreateWithVersion` and a transactional `AddVersion`). With
`SetMaxOpenConns(1)` already serializing writes, the transaction is cheap and closes both windows (and the
concurrent-version race, see L7).

---

### H5. FTS5 search passes raw user input into `MATCH` — common queries 500 and leak schema
**Category:** bug · **Where:** [document_repository.go:217,230](internal/repository/document_repository.go#L217-L230); [document_handler.go:417-427](internal/handler/document_handler.go#L417-L427)

`Search` injects the raw query string directly into `documents_fts MATCH ?`. The `?` bind prevents *SQL* injection
but does **not** neutralize the FTS5 query grammar. Empirically (modernc.org/sqlite v1.48.1): an unbalanced quote →
`unterminated string`; `(`, trailing `OR`, `NEAR(` → `fts5: syntax error`; `*` → `unknown special query`; `col:val`
→ `no such column: col`. Each is a SQL error that the handler maps to **HTTP 500**, and `col:` is a schema-enumeration
oracle via differential error messages.

**Impact:** Any legitimate search containing a quote, parenthesis, asterisk, colon, or the tokens `OR`/`AND`/`NEAR`
returns 500 instead of results — search is broken for many real document names — and the error path leaks FTS column
structure and is an availability/enumeration vector.

**Fix:** Don't pass raw input to `MATCH`. Wrap it as a single quoted FTS5 string literal (escape embedded `"` as
`""`, surround in quotes; optionally append `*` to the last token for prefix search), or strip FTS operator chars.
In the handler, map a malformed-query error to 400 (or empty results), never surfacing raw FTS errors.

---

### H6. Archived documents (and ungranted documents) leak through full-text search
**Category:** data-integrity · **Where:** [document_repository.go:214-232](internal/repository/document_repository.go#L214-L232)

`List` filters every query with `is_archived = 0`, but `Search` does **not** — both the count and result queries
return archived rows. Archiving is the app's only deletion path (`Archive` sets `is_archived=1`), and the FTS delete
trigger fires only on a real SQL `DELETE` (which never happens), so archived rows stay fully indexed. (As noted in
C4, `Search` *also* omits the per-investor access-grant filter, so it leaks non-archived ungranted documents too.)

**Impact:** A withdrawn/archived confidential document (e.g. a merger plan, or one whose access was revoked) remains
discoverable by name/description/tags via `POST /documents/search`, even though the list view hides it.

**Fix:** Add `AND d.is_archived = 0` to the `Search` result query and make the count consistent by counting through
the same join; additionally apply the investor access-grant filter (C4).

---

### H7. Initial admin password is logged to stdout in plaintext (persisted to Docker logs)
**Category:** security · **Where:** [auth_service.go:221-228](internal/service/auth_service.go#L221-L228)

When `DD_ADMIN_PASSWORD` is unset (the documented default; compose passes `${DD_ADMIN_PASSWORD:-}` = empty),
`EnsureAdminExists` generates a random password and writes it via `fmt.Printf("[INFO] Generated admin password: %s")`.
The compose files configure json-file logging with rotation (10m × 3), so this cleartext credential is persisted to
the Docker logging backend; `start.sh` even greps container logs for "admin password" to surface it.

**Impact:** The initial admin credential is written to log files / aggregation in cleartext (CWE-532). Anyone with
read access to container logs, log shippers, SIEM, or backups gains full admin access, and it persists across log
rotation well beyond first boot.

**Fix:** Don't log the password. Write it once to a `0600` file on the data volume, or require `DD_ADMIN_PASSWORD`
to be set and fail fast if empty in production; force a password reset on first admin login.

---

### H8. Watermarking is config-only and is never applied to any download
**Category:** missing-feature · **Where:** [document_handler.go:380,408](internal/handler/document_handler.go#L380-L408); [admin/watermark/+page.svelte:55](ui/src/routes/admin/watermark/+page.svelte#L55)

The watermark feature has a full config CRUD path (handler, repo, `watermark_config` table, admin UI) but the
watermark is **never applied** when documents are served. Both `Download` and `DownloadVersion` return the raw stored
bytes via `c.Blob(...)` with no watermarking step; `NewDocumentHandler` isn't even given the watermark repo as a
dependency. The admin UI literally says "Configure dynamic watermarks applied to downloaded documents" — which is false.

**Impact:** A leak-attribution/security feature advertised in the UI does nothing. Admins believe downloaded
documents are watermarked (e.g. with investor identity); they are served unaltered — a false sense of protection.

**Fix:** Inject the watermark repo into `DocumentHandler` and render the configured text/position/opacity onto
PDF/image bytes before `c.Blob` in both download paths — or, at minimum, fix the misleading admin copy and mark the
feature unimplemented.

---

### H9. Invite acceptance is unreachable — email and UI point at a `/register` page that does not exist
**Category:** missing-feature · **Where:** [email_service.go:50](internal/service/email_service.go#L50); `ui/src/routes/` (no `register/` dir); [admin/users/+page.svelte:32](ui/src/routes/admin/users/+page.svelte#L32)

The invite flow is half-wired: `CreateInvite` mints a token, the email links to `%s/register?token=%s`, and
`POST /api/v1/auth/register` consumes a token to set name+password — but there is **no `ui/src/routes/register/`**,
so the invite link dead-ends. The admin Users page compounds it by just surfacing the raw token as on-screen text
("Invite sent. Token: …") with no page to use it in.

**Impact:** Invited investors/company members can never complete registration through the product — the entire
invite-based onboarding path is non-functional end to end despite a working backend endpoint. (SMTP is also disabled
by default, so no email is even sent in the default config.)

**Fix:** Create `ui/src/routes/register/+page.svelte` that reads `?token=`, collects name+password, POSTs
`/auth/register`, and logs the user in with the returned tokens. Keep the email URL in sync with the route. See also
L21 (inviting `admin` 500s) and L16 (token shown in plaintext).

---

### H10. No client-side auth/role guard — protected and admin pages render for anyone
**Category:** rbac · **Where:** [+layout.svelte](ui/src/routes/+layout.svelte) (no redirect); no `hooks.client.ts`/`+layout.ts`; [admin/+layout.svelte](ui/src/routes/admin/+layout.svelte) (no guard)

There is no route guard anywhere: no `hooks.client.ts`, no `+layout.ts`/`+layout.server.ts`, and the root layout
only conditionally renders the nav — it never redirects unauthenticated users away from `/documents`, `/qa`,
`/analytics`, or `/admin/*`. Access control is delegated entirely to the API returning 401. A logged-in *investor*
who navigates to `/admin/users` renders the full admin UI; only the data fetch fails. `NavHeader` merely *hides* the
Admin link for non-admins — nothing blocks direct navigation.

**Impact:** Unauthenticated/under-privileged users see protected layouts and admin form controls (flash of admin UI,
then empty tables). Security depends entirely on every backend endpoint enforcing RBAC correctly — and several do
not (C4, H6, the analytics endpoint below). Information disclosure of UI structure plus confusing UX.

**Fix:** Add a `+layout.ts`/`hooks.client.ts` guard (adapter-static rules out `+layout.server.ts`) that awaits
`authStore.restore()`, checks `isAuthenticated` and the required role, and redirects to `/login` or `/unauthorized`
before rendering. Gate `/admin/*` on `isAdmin` specifically.

---

### H11. JWT access token stored in `localStorage` (XSS → account takeover); refresh token dropped
**Category:** security · **Where:** [authStore.svelte.ts:21,29,35](ui/src/lib/stores/authStore.svelte.ts#L21); [client.ts:46,51,56](ui/src/lib/api/client.ts#L46); CSP at [security_headers.go:19-20](internal/middleware/security_headers.go#L19-L20)

The access token is persisted in `localStorage` (`dd_auth_token`), readable by any JS on the origin. The CSP allows
`'unsafe-inline'` and admin-controlled branding/custom CSS is injected into the page (see M4/M5/L4), so any XSS or
malicious branding/SVG payload can exfiltrate the bearer token. The login response's `refresh_token` is silently
discarded, so there is no rotation either.

**Impact:** Any XSS on the origin yields full account takeover by reading `localStorage` and replaying the token;
tokens persist until expiry and survive tab close.

**Fix:** Prefer an `HttpOnly; Secure; SameSite` cookie set by the backend so JS can't read the session token. If
`localStorage` stays, minimize TTL, implement refresh rotation, and tighten CSP (drop `'unsafe-inline'`).

---

### H12. Spoofable client IP (`c.RealIP()` trusts `X-Forwarded-For` from anyone) — rate limiter and audit/NDA IPs are forgeable
**Category:** security · **Where:** [rate_limit.go:56](internal/middleware/rate_limit.go#L56); [audit_log.go:61](internal/middleware/audit_log.go#L61); [nda_handler.go:206](internal/handler/nda_handler.go#L206); no `IPExtractor` in [cmd/main.go](cmd/main.go)

The rate limiter keys on `c.RealIP()` and the audit log + NDA signature record store it as the IP. Echo v4 defaults
to trusting `X-Forwarded-For`/`X-Real-IP` from **any** source unless `e.IPExtractor` is set — and it is configured
nowhere. So any client can set `X-Forwarded-For: <random>` per request.

**Impact:** Rate limiting is trivially bypassed (rotate the XFF header → a fresh per-IP bucket every request →
the 200/min/IP login throttle never applies). Audit logs and legally-significant NDA-signing IPs are poisonable with
arbitrary forged values, undermining the forensic value of the audit trail.

**Fix:** Set a trust-aware extractor before middleware: `echo.ExtractIPDirect()` when not proxied, or
`echo.ExtractIPFromXFFHeader(echo.TrustLoopback(true), echo.TrustPrivateNet(true))` behind a known proxy, with
trusted CIDRs from env.

---

### H13. No request body size limit — pre-auth JSON endpoints allow memory-exhaustion DoS
**Category:** security · **Where:** [cmd/main.go:69-94](cmd/main.go#L69-L94) (no `BodyLimit`); upload soft-cap at [document_handler.go:127](internal/handler/document_handler.go#L127)

There is no `echomw.BodyLimit()` in the global chain. Upload handlers enforce a per-file cap via `file.Size`, but
that value comes from the client-declared multipart header (soft), and **every** JSON endpoint — including the
pre-auth `POST /auth/login` and `/auth/register` — calls `c.Bind()` with no size cap and no read deadline.

**Impact:** An unauthenticated attacker can POST a multi-gigabyte JSON body (or a slow-read body) to `/auth/login`
and force proportional memory allocation → cheap OOM/DoS. Rate limiting doesn't help: one oversized request does the
damage. The 100 MB upload is also read fully into memory via `io.ReadAll`.

**Fix:** Add `e.Use(echomw.BodyLimit("1M"))` early, with a larger group-scoped limit (e.g. 110 MB) on the upload
route, and rely on `http.MaxBytesReader` semantics rather than the client-declared `file.Size`. Add server read
timeouts.

---

### H14. Anyone can poison view analytics — `POST /analytics/view-event` has no access check and trusts client-supplied metrics
**Category:** data-integrity · **Where:** [analytics_handler.go:29,73-103](internal/handler/analytics_handler.go#L73-L103); [analytics_repository.go:29-41](internal/repository/analytics_repository.go#L29-L41)

The endpoint is registered with **no `RequireRole`** and **no document access check**. It takes a client-supplied
`document_id` plus arbitrary `duration_ms` and `page_count` and writes a `view_events` row attributed to the caller —
for any document, including ones they were never granted. These rows feed `GetEngagementSummary`,
`GetDocumentAnalytics` (avg duration), and `GetUserAnalytics` that admins rely on for diligence decisions.

**Impact:** Admin-facing engagement analytics are fully attacker-controllable — inflate/deflate apparent interest,
fabricate views for inaccessible documents, set any duration. (Ironically the legitimate UI never calls this
endpoint at all — see M2 — so today the *only* caller would be a malicious one.)

**Fix:** Require at least view access to `document_id` (reuse the C4 helper) before recording; clamp/validate
`duration_ms`/`page_count` to sane non-negative bounds; derive `page_count` server-side where possible.

---

<!-- The remaining findings (Medium/Low/Info) follow. Each is independently confirmed against the code. -->

## 🟡 Medium

#### M1. No +error.svelte anywhere: empty/unmatched routes render SvelteKit's default unstyled error page
`missing-feature` · `missing-ui-pages`  
**Where:** ui/src/routes/ (no +error.svelte at any level); svelte.config.js:11 (fallback:'index.html'); cmd/main.go:389-410 (spaFileHandler SPA fallback)  
**Issue:** find ui/src -name '+error.svelte' returns nothing. Because svelte.config.js sets fallback:'index.html' (svelte.config.js:11) and the Go server serves index.html for unknown paths (cmd/main.go:377-401 spaFileHandler), every broken link above (upload, [id], [threadId], nda-sign/[token], unauthorized) does NOT hard-404 at the server — it loads the SPA, which then fails to match a route and shows SvelteKit's built-in bare error page with NO NavHeader, NO branding, and a generic '404 Not Found'.  
**Impact:** All the missing-page navigations degrade to a jarring, unbranded, unactionable error screen with no way back into the app (no nav). Magnifies every dead-link finding above.  
**Fix:** Add ui/src/routes/+error.svelte that renders the NavHeader, a friendly message, and a link back to /documents, so unmatched routes fail gracefully.

#### M2. POST /analytics/view-event is never called by the UI, so all view/viewer analytics are permanently zero
`missing-feature` · `api-contract`  
**Where:** internal/handler/analytics_handler.go:29 (route) and :72-103 (handler); internal/repository/analytics_repository.go:110-159 (GetEngagementSummary reads view_events); no caller anywhere under ui/src  
**Issue:** The backend exposes POST /analytics/view-event (analytics_handler.go:29, handler at :72) to record `view_events`, which feed GetEngagementSummary's TotalViews/UniqueViewers/RecentViewCount and per-document/per-user analytics. A repo-wide grep of ui/src for `view-event` returns nothing — no Svelte page or store ever records a view. The analytics dashboard (ui/src/routes/analytics/+page.svelte) reads `total_views`, `unique_viewers`, `recent_view_count` from GET /analytics/dashboard, but nothing ever writes view events.  
**Impact:** The Analytics dashboard will always display 0 Total Views, 0 Unique Viewers, and 0 Recent Views regardless of real activity, making the analytics feature non-functional and misleading. Per-document and per-user analytics endpoints likewise return empty data.  
**Fix:** Call `api.post('/analytics/view-event', { document_id })` from the document-detail view (and/or when a download/preview occurs). Since the documents/[id] detail page does not exist yet, wire view-event recording into that page when it is built, plus optionally on download.

#### M3. refresh_token is issued and POST /auth/refresh exists, but the UI discards the token and never refreshes; sessions hard-expire to forced re-login
`contract-mismatch` · `api-contract`  
**Where:** ui/src/routes/login/+page.svelte:16-31 (reads access_token only, discards refresh_token); ui/src/lib/stores/authStore.svelte.ts:17-39 (setAuth/loadFromStorage persist only dd_auth_token); ui/src/lib/api/client.ts:86-92 (401 -> clear + redirect /login, no refresh); backend: internal/handler/auth_handler.go:27,61-65,117-135; token TTL: internal/service/auth_service.go:20-21  
**Issue:** Login returns `{ user, access_token, refresh_token }` (auth_handler.go:61-65). The login page types and reads `access_token` only, storing it via authStore.setAuth and ignoring `refresh_token` entirely (login/+page.svelte:24-31). authStore.setAuth/loadFromStorage persist only `dd_auth_token` (authStore.svelte.ts:17-39); the refresh token is never stored. The backend POST /auth/refresh endpoint (auth_handler.go:27/:117, expecting `{ refresh_token }`) has no UI caller (grep `auth/refresh` in ui/src: no matches). When the access token expires, client.ts:86-92 catches the 401 and immediately redirects to /login, with no refresh attempt.  
**Impact:** The refresh-token half of the auth contract is dead on the client. Users are abruptly bounced to the login screen mid-session whenever the access token expires (potentially losing unsaved form input in NDA/branding/category create flows), even though a working refresh endpoint exists.  
**Fix:** Persist refresh_token at login, and in client.ts on a 401 attempt POST /auth/refresh with the stored refresh token before clearing auth and redirecting; on success replace the access token and retry the original request once.

#### M4. Branding color values are stored and rendered into a <style> block with zero sanitization, enabling stored CSS injection
`security` · `injection-input`  
**Where:** internal/handler/branding_handler.go:52-71 (UpdateConfig sanitizes only CustomCSS at :59-62); internal/repository/branding_repository.go:72-118 (UpsertConfig persists color/font fields verbatim, no validation); ui/src/lib/theme/branding-engine.ts:42-50 (builds `${cssVar}: ${value};` and font_family into a <style> textContent); internal/middleware/security_headers.go:20-21 (style-src 'unsafe-inline', img-src data: blob:)  
**Issue:** UpdateConfig sanitizes only config.CustomCSS (branding_handler.go:59-62). The 16 color fields (PrimaryColor, BackgroundColor, etc.) and FontFamily are persisted verbatim — branding_repository.go has no hex/format validation. On the client, applyBrandingCSS builds `${cssVar}: ${value};` (branding-engine.ts:46) and FontFamily (line:50) directly from these stored values and injects them via `style.textContent` into a <style> element in document.head. Because the values are never validated as colors, an admin-supplied (or any value that reaches this config) like `red; } body{ background:#000 } *{x:` breaks out of the declaration and injects arbitrary CSS rules into the whole document. CSP `style-src 'self' 'unsafe-inline'` (security_headers.go:20) explicitly permits this inline style, and img-src allows `data:` and `blob:`, so injected CSS can exfiltrate via background-image style attribute selectors / CSS-based data leakage.  
**Impact:** Stored CSS injection affecting every page for every user. Allows UI defacement, clickjacking-style overlays, and CSS-exfiltration of on-page content (e.g. attribute selectors + background URLs). Combined with the unsanitized custom_css path it is a persistent injection surface.  
**Fix:** Validate every color field server-side against a strict pattern (e.g. `^#(?:[0-9a-fA-F]{3}|[0-9a-fA-F]{6})$` or an allowlist of `rgb()/rgba()` forms) in UpdateConfig before UpsertConfig, rejecting on mismatch. Validate FontFamily against a conservative charset. On the client, also CSS.escape / validate values before interpolating into textContent. Do not rely solely on the admin role being trusted.

#### M5. sanitize.CSS does not prevent declaration/block breakout (no escaping of } { ; or <>), giving weak protection for custom_css
`security` · `injection-input`  
**Where:** pkg/sanitize/sanitize.go:49-81 (cssBlockedPatterns + CSS); consumed at internal/handler/branding_handler.go:59-61; injected at ui/src/lib/theme/branding-engine.ts:57 via style.textContent line 63  
**Issue:** sanitize.CSS only regex-replaces a blocklist of substrings (@import, url(, expression(, behavior:, javascript:, data:, etc.). It performs no structural escaping: characters `{ } ; < >` pass through untouched, and the function returns the input verbatim if no blocked pattern matches. The output is later concatenated as a trusted-looking block into the global <style> by branding-engine.ts:57 (`css += ... config.custom_css`). Blocklist CSS sanitization is inherently bypassable (e.g. comment/whitespace tricks like `url/**/(` are not matched by `url\s*\(`, and `\65 xpression` CSS escapes evade `expression\(`). More importantly, since custom_css is injected as a raw CSS block, an attacker can still author arbitrary selectors/rules (the blocklist only targets a few keywords), so CSS-exfiltration and overlay attacks remain possible even on sanitized input.  
**Impact:** custom_css remains an injection surface: blocklist bypasses (comment insertion, CSS unicode escapes) can smuggle url()/expression-equivalents, and arbitrary selector rules enable CSS data-exfiltration/UI redress despite the sanitizer. Defense is incomplete rather than absent.  
**Fix:** Prefer an allowlist CSS approach (parse declarations, allow only known-safe properties/values) over substring blocklisting; at minimum, make the regexes resistant to CSS comment/escape evasion (strip /* */ comments and decode CSS escapes before matching) and consider disallowing custom_css entirely in favor of the structured color fields. Note `modified` boolean is discarded at branding_handler.go:60 — surface it to the admin.

#### M6. Upload has no MIME-type whitelist; client-supplied Content-Type is trusted for storage and later set as download Content-Type
`security` · `injection-input`  
**Where:** internal/handler/document_handler.go:156-159 (Upload), :306-308 + :337 (CreateVersion, not cited), :377-380 (DownloadVersion), :405-408 (Download)  
**Issue:** On upload, mimeType is taken from the client-supplied multipart `Content-Type` header verbatim (document_handler.go:156) and only falls back to http.DetectContentType when empty (line 158) — there is no allowlist and no reconciliation against the sniffed type. That attacker-controlled MIME is stored and, on download, written straight into the response `Content-Type` header (document_handler.go:377 and 405) and passed to c.Blob. Downloads do set `Content-Disposition: attachment` (line 378/406) and global nosniff is on, which mitigates inline rendering, but storing an arbitrary client MIME (e.g. text/html) is still risky for any future inline/preview path and means the server vouches for an unverified type.  
**Impact:** Attacker controls the stored and served Content-Type for documents. Currently constrained by attachment+nosniff, but any code path that serves without attachment (preview, thumbnails, NDA/report rendering) would inherit a content-type-confusion/XSS risk. It also undermines integrity expectations of stored metadata.  
**Fix:** Always derive MIME from http.DetectContentType on the actual bytes (or validate the client value against an allowlist and reject mismatches). Keep `Content-Disposition: attachment` for all binary downloads and never serve user documents inline.

#### M7. Duplicate top-level migrations/ directory is dead code and a correctness trap
`quality` · `data-migrations`  
**Where:** internal/repository/migrate.go:12 (//go:embed migrations/*.sql) and :18/:33; dead dir = /home/fish/GitHub/due-diligence-portal/migrations/ ; live dir = /home/fish/GitHub/due-diligence-portal/internal/repository/migrations/  
**Issue:** migrate.go embeds with `//go:embed migrations/*.sql`, which is resolved relative to the migrate.go package directory, so ONLY internal/repository/migrations/*.sql is ever executed. The root-level migrations/ directory is never read by the application. Both directories currently contain byte-identical files (verified via diff -q: 001/002/003 all identical), so they are in sync today — but nothing keeps them in sync.  
**Impact:** A developer editing the obvious top-level migrations/ (the one referenced in CLAUDE.md docs and DB section) will see zero effect at runtime and ship a schema that diverges from what actually loads. Latent footgun for every future schema change.  
**Fix:** Delete the root-level migrations/ directory (or replace it with a symlink / a README pointing at internal/repository/migrations/). Update CLAUDE.md which references `migrations/` as if it were the live path.

#### M8. CORS allows all origins (*) together with the Authorization header on every route
`security` · `config-secrets-deploy`  
**Where:** cmd/main.go:89-94  
**Issue:** Global CORS is configured with AllowOrigins: []string{"*"} and AllowHeaders including echo.HeaderAuthorization, applied via e.Use to the entire server including all /api/v1 endpoints. The comment says 'CORS for development' but this is the only CORS config and ships in the production docker image. It is not gated by any environment/dev flag.  
**Impact:** Any website on the internet can issue authenticated cross-origin requests to the portal API. While wildcard origin disallows credentialed cookie requests, this app uses bearer JWT in the Authorization header (not cookies), so a malicious page that has somehow obtained or can prompt for a token can freely call the API cross-origin, and the wildcard removes any origin-based defense-in-depth. For an internal due-diligence portal handling confidential documents, exposing a permissive CORS surface is inappropriate.  
**Fix:** Make allowed origins configurable via env (e.g. DD_CORS_ORIGINS) and default to the portal's own origin / deny cross-origin. Only enable wildcard under an explicit dev flag. Never combine AllowOrigins:* with reflected credentials/Authorization in production.

#### M9. Login endpoint shares the global 200 req/min IP rate limit — insufficient brute-force protection
`security` · `config-secrets-deploy`  
**Where:** internal/middleware/rate_limit.go:39-65 (skip list 44-53); cmd/main.go:85-87; internal/handler/auth_handler.go:25,37-66  
**Issue:** The only rate limiting is a single global limiter of DD_RATE_LIMIT (default 200) requests per minute per IP (cmd/main.go:85-87). The middleware skips static assets and /health,/ready but applies the same generic budget to POST /api/v1/auth/login. There is no stricter per-account or per-login throttle and no lockout. 200 password guesses/minute = 12,000/hour per source IP against the login endpoint.  
**Impact:** Permits online password brute-force / credential-stuffing at 200 attempts/min/IP against the admin and user accounts (which use bcrypt-hashed passwords; the initial admin password may be a known-format hex if generated, or operator-chosen). Distributed attackers multiply this by IP. Combined with the weak-secret finding, account takeover risk is elevated.  
**Fix:** Add a dedicated, much lower rate limit (e.g. 5-10/min) keyed on login endpoint + IP and/or username, and implement temporary account lockout / exponential backoff on repeated failures.

#### M10. Branding asset upload/serve endpoints are not wired into the admin branding UI
`missing-feature` · `feature-completeness`  
**Where:** ui/src/routes/admin/branding/+page.svelte (UI gap); endpoints in internal/handler/branding_handler.go:37-39  
**Issue:** branding_handler.go implements GET/POST/DELETE /branding/assets/:key (lines 37-39) for logo/favicon BLOBs stored in branding_assets, validated against domain.ValidAssetKeys. But the admin branding UI only calls GET/PUT/DELETE /branding (config), never the asset endpoints: grep over admin/branding/+page.svelte shows only api.get('/branding'), api.put('/branding'), api.delete('/branding') — no /branding/assets, no file input, no upload. So admins cannot upload a logo or favicon through the product.  
**Impact:** The white-label branding feature is partially unusable: color/text config works, but the visual assets (logo, favicon) that define the white-label cannot be set via the UI. The asset endpoints are dead weight from the client perspective.  
**Fix:** Add a file-upload section to admin/branding/+page.svelte that POSTs multipart to /branding/assets/{logo|favicon|...} for each valid key, and renders/clears them. Wire the branding engine to consume the served asset URLs.

#### M11. Errors are swallowed silently in nearly every data load (empty catch sets blank state)
`quality` · `frontend-quality`  
**Where:** ui/src/routes/documents/+page.svelte:17-19,27-29,41-43; qa/+page.svelte:17-19; admin/users/+page.svelte:17-19; admin/categories/+page.svelte:18-20; admin/branding/+page.svelte:15-17; admin/watermark/+page.svelte:22-24; admin/audit/+page.svelte:15-17; analytics/+page.svelte:22-24  
**Issue:** Almost all load functions use `catch { list = []; }` or `catch { config = null; }` with no error surfaced to the user and no logging. A network failure, 500, or 403 is indistinguishable from a genuinely empty result. For documents/qa/categories/users/audit the user sees 'No documents found' / 'No users found' even when the request actually errored. The only distinction in some pages is a separate '!config' branch (branding/watermark/analytics) that says 'Failed to load', but it is still triggered by both empty and error states. No error state, no retry affordance.  
**Impact:** Failures masquerade as empty data. Users and admins cannot tell a real error from no records, leading to wrong conclusions (e.g., 'there are no audit entries' when the audit fetch 403'd). Debugging is hard with no console logging either.  
**Fix:** Capture the error into an `error` $state, distinguish error from empty, render an error banner with a Retry button, and console.error the caught value. Only qa/createThread (qa/+page.svelte:30) and createThread bother logging.

#### M12. restore() is fire-and-forget in $effect; user object can be null while protected pages render
`bug` · `frontend-quality`  
**Where:** ui/src/routes/+layout.svelte:17-22 (restore() unawaited in $effect) and ui/src/lib/stores/authStore.svelte.ts:12,45-82 (isAuthenticated derived + async restore); also ui/src/lib/components/NavHeader.svelte:6-7,18-19 (role-gated links)  
**Issue:** The layout's $effect calls authStore.restore() without awaiting it. restore() is async (GET /auth/me). Pages render immediately with authStore.user still null. NavHeader and any role-derived UI (NavHeader.svelte:6-7 showAnalytics/showAdmin) evaluate before the user is hydrated, so on a fresh load with a valid stored token the Admin/Analytics nav links flicker absent until /auth/me resolves, then appear. isAuthenticated (authStore:12) requires both token AND user, so it is briefly false on every reload even for a valid session. Because $effect runs only in the browser after mount, there is also no loading gate to prevent the flash.  
**Impact:** Reactivity/UX bug: role-gated navigation and any isAuthenticated-dependent UI flicker on every page load; a guard built on isAuthenticated (if added naively) would wrongly bounce valid users to /login during the restore window.  
**Fix:** Await restore() in a +layout.ts load (runs before render) or render a loading state while authStore.loading is true and user is unresolved; only compute role-gated UI after restore completes.

#### M13. Admin Users page can invite but cannot edit, disable, or change roles (POST without matching update affordance)
`missing-feature` · `frontend-quality`  
**Where:** ui/src/routes/admin/users/+page.svelte:80-106 (table), 23-40 (only mutating action sendInvite)  
**Issue:** The Users page lists users with a Status column showing Active/Disabled (line 97-99) and a Role column, but provides no UI to disable/enable a user, change a role, or edit/delete. The only mutating action is sendInvite (POST /users/invite). There is no row action column at all. If the backend exposes user update/disable endpoints (the schema implies is_active management), there is no affordance to reach them; if it doesn't, the displayed Status is read-only with no way to act on a compromised or departed user.  
**Impact:** Admins cannot deactivate or re-role users from the portal, a core access-management requirement for a due-diligence data room. A departing investor cannot be cut off via the UI.  
**Fix:** Add per-row actions (Disable/Enable, Change Role, Resend/Revoke invite) wired to the corresponding PATCH/PUT/DELETE user endpoints; if those endpoints are missing, build them.

#### M14. Audit IDs are fully deterministic from the clock (no entropy) — concurrent writes collide on PRIMARY KEY and are silently dropped
`data-integrity` · `middleware-crosscutting`  
**Where:** internal/middleware/audit_log.go:69-80 (generateAuditID), called at audit_log.go:41; error swallowed at audit_log.go:64-66; schema at internal/repository/migrations/001_initial_schema.sql:127  
**Issue:** generateAuditID() builds all 16 bytes from time.Now().UnixNano(): bytes 0-7 are the big-endian nanosecond timestamp and bytes 8-15 are byte(now + int64(i)) — also derived from the same 'now', with NO randomness. Two audit writes that observe the same UnixNano value produce byte-identical IDs. audit_log.id is `TEXT PRIMARY KEY` (001_initial_schema.sql:127). On collision the INSERT fails the PK constraint; Log() returns the error, but LogFromContext (audit_log.go:64-66) only does fmt.Printf to stdout and discards it — the caller never sees the failure.  
**Impact:** Under concurrent mutating requests (multiple admins/users acting at once), audit entries can be silently lost, leaving gaps in the compliance/forensic trail — directly harmful for a due-diligence audit log. Ironically the schema already provides a strong default `lower(hex(randomblob(16)))` (001_initial_schema.sql:127) which this weak generator overrides.  
**Fix:** Generate the ID with crypto/rand (e.g. 16 random bytes hex-encoded), or simply omit id from the INSERT column list and let the schema's randomblob(16) DEFAULT populate it. Additionally, surface Log() errors (at least increment a metric / log at ERROR with request id) rather than swallowing them.

#### M15. Q&A message posting is not audited — gap in the audit trail for a mutating, content-creating action
`data-integrity` · `middleware-crosscutting`  
**Where:** internal/handler/qa_handler.go:121-167 (PostMessage); success path / missing audit at line 166 where response.Created returns with no preceding h.audit.LogFromContext call  
**Issue:** Every mutating endpoint calls h.audit.LogFromContext(...) — thread create (qa_handler.go:87), status change (197), document upload/update/archive/version/download, permission grant/update/revoke, etc. But PostMessage (POST /qa/:id/messages, registered qa_handler.go:28) creates a QAMessage and returns at line 162-166 with NO audit call. Notably it can also create internal-only messages (isInternal at line 141-147) which are precisely the messages whose authorship most needs an audit record.  
**Impact:** Q&A is a core collaboration surface in the portal; message posts — including internal notes visible only to admins/company members — leave no audit entry, so 'who said what when' cannot be reconstructed from the audit log. This is a compliance gap for a due-diligence data room.  
**Fix:** Add h.audit.LogFromContext(c, domain.AuditQAMessagePosted, "qa_thread", threadID, "", fmt.Sprintf("internal=%t", isInternal)) after successful CreateMessage at qa_handler.go:164 (defining the action constant in domain if absent).

#### M16. Three of four email notification methods are dead code; investors/admins never get Q&A, document, or NDA-signed emails
`missing-feature` · `critic-gaps`  
**Where:** internal/service/email_service.go:58 (SendQANotification), :71 (SendDocumentNotification), :83 (SendNDASignedNotification)  
**Issue:** SendQANotification, SendDocumentNotification, and SendNDASignedNotification are fully implemented and tested but have ZERO callers anywhere in the codebase (verified: grep for each across all non-test .go files returns nothing). Only SendInvite is wired (user_handler.go:172). So posting a Q&A reply, uploading a document, and signing an NDA send no email even when DD_SMTP_ENABLED=true. The qa_handler.PostMessage, document_handler.Upload, and nda_handler.Sign paths never reference emailSvc — QAHandler/DocumentHandler don't even hold an EmailService reference.  
**Impact:** The entire email notification subsystem (3 of 4 templates) is non-functional. Investors are never notified of answers to their questions; admins are never notified of NDA signatures or new documents. A core 'portal' collaboration feature silently does nothing.  
**Fix:** Inject *service.EmailService into QAHandler, DocumentHandler, and NDAHandler and call the respective Send* methods (best-effort, non-fatal like the invite path) after the mutating write succeeds; or delete the three methods if intentionally unused. Add a handler-level test asserting the call happens.

#### M17. Re-granting access to a user who already has a grant on the same resource returns opaque 500; admins cannot change access level via Grant
`bug` · `critic-gaps`  
**Where:** internal/handler/permission_handler.go:101-103 (Grant maps all repo errors to response.InternalError -> 500); internal/repository/permission_repository.go:39-48 (plain INSERT, no ON CONFLICT); UNIQUE constraint at internal/repository/migrations/001_initial_schema.sql:116  
**Issue:** access_grants has UNIQUE(user_id, resource_type, resource_id) (001_initial_schema.sql:access_grants), and permissionRepository.Grant uses a plain INSERT (no UPSERT/ON CONFLICT). When an admin grants access to a user who already holds ANY grant on that resource — the normal 'upgrade view to download' operation — the INSERT violates the unique constraint, the repo returns an error, and the handler maps every repo error to response.InternalError (500). There is no Update-by-resource path exposed; PUT /permissions/:id needs the grant ID, which the admin UI does not necessarily have.  
**Impact:** Admins cannot upgrade or change a user's access level (e.g. view->download) without first finding and Revoking the existing grant by ID. The common path returns an unexplained 500. Access management is effectively broken for any user who already has a grant.  
**Fix:** Use INSERT ... ON CONFLICT(user_id, resource_type, resource_id) DO UPDATE SET access_level=excluded.access_level, expires_at=excluded.expires_at, granted_by=excluded.granted_by in Grant; or detect the unique violation in the handler and return 409 Conflict with a clear message.

#### M18. Q&A thread creation and message posting accept arbitrary document_id/category_id with no access check; bogus IDs surface as 500
`rbac` · `critic-gaps`  
**Where:** internal/handler/qa_handler.go:59-90 (CreateThread); also PostMessage qa_handler.go:121-167; schema internal/repository/migrations/001_initial_schema.sql:152-153; wiring cmd/main.go:160-161; FK pragma internal/repository/sqlite.go:24,46  
**Issue:** CreateThread stores client-supplied DocumentID and CategoryID (qa_threads has nullable FKs to documents/categories) with no check that the asking investor has any grant on that document/category. Beyond the already-noted 'threads are global' read issue, this lets an investor enumerate or reference documents/categories they cannot access by attaching them to a thread, and have the (admin/company-visible) thread reveal those resource IDs. A non-existent document_id/category_id hits the FK and the repo error maps to response.InternalError (500) rather than a 400, since CreateThread has no existence/ownership validation before insert.  
**Impact:** Investors can bind questions to resources they have no access to (info leak of valid resource IDs to staff and in thread listings), and malformed references return opaque 500s instead of validation errors.  
**Fix:** Validate that document_id/category_id exist and that the caller has at least view access before creating the thread; on missing resource return 404/400, not 500.

#### M19. NDA Sign trusts a fully client-supplied signer_name unrelated to the authenticated identity; the legal signature record can be forged
`data-integrity` · `critic-gaps`  
**Where:** internal/handler/nda_handler.go:162-216 (specifically lines 170-172 validation, 199-207 signature construction, 203 SignerName, 205 SignerCompany)  
**Issue:** Sign records sig.SignerName from req.SignerName (arbitrary client string) while SignerEmail/UserID come from the authenticated token. There is no requirement that signer_name match the user's actual name, no minimal validation beyond non-empty, and signer_company is also free text. The NDA signature is the legal artifact admins review (ListSignatures, SendNDASignedNotification), yet its primary human-readable field is attacker-controlled and can name a third party.  
**Impact:** An NDA signature record can attribute the signature to any name/company the signer types, undermining the evidentiary value of the NDA-signing feature. Combined with the already-noted fact that signing does not gate access, the feature is both unenforced and forgeable.  
**Fix:** Either bind signer_name to the authenticated user's stored name (server-side) or, if a typed legal name is required, validate/store it alongside the immutable authenticated user_id+email and display both, making clear the typed name is self-asserted; add length/charset validation.

#### M20. PUT /users/:id allows an admin to deactivate or change the role of their own account (self-lockout / accidental privilege loss)
`rbac` · `critic-security2`  
**Where:** internal/handler/user_handler.go:77-119 (Update) vs 121-140 (Deactivate); self-guard at 126-128  
**Issue:** Deactivate deliberately blocks self-deactivation (`if middleware.GetUserID(c) == id { return BadRequest('cannot deactivate your own account') }`), but the Update handler honors `is_active: false` and arbitrary `role` changes for ANY id including the caller's own, with no equivalent self-guard. The protection in Deactivate is therefore trivially bypassed through the Update endpoint.  
**Impact:** The sole/last admin can lock themselves out by PUTting is_active=false on their own id (or demote themselves to investor), leaving the portal with no admin and no in-app recovery path. This also defeats the intent of the explicit self-deactivation guard, making it security theater.  
**Fix:** Apply the same self-protection in Update: reject is_active=false and role downgrades when id == caller's own user ID, and/or block deactivating/demoting the last remaining active admin (count active admins before allowing the change).

#### M21. Audit-log details/value fields are built from unsanitized user input, enabling audit-record forgery and log injection
`data-integrity` · `critic-security2`  
**Where:** internal/middleware/audit_log.go:38-44 (Log INSERT); call sites internal/handler/nda_handler.go:213, internal/handler/permission_handler.go:105-106, internal/handler/user_handler.go:167  
**Issue:** AuditLogger.Log inserts resource_name and details verbatim into audit_log. Numerous call sites concatenate raw, attacker-controlled values into the details string with no sanitization — e.g. user_handler.go:175 `"role="+req.Role`, permission_handler.go:114 `"user="+req.UserID+" level="+req.AccessLevel`, qa ChangeStatus `"status="+req.Status`, NDA `"signer="+req.SignerName`. Several of these values are validated before audit (role/status), but UserID, ResourceID, SignerName, document names and similar are not and may contain newlines, control characters, or forged `key=value` pairs.  
**Impact:** An attacker who controls any of these fields (e.g. SignerName, a crafted document name, an invite email) can inject newlines/fake key=value pairs into the audit details, forging or obscuring entries in the tamper-evidence trail that admins read. Because the audit log is the security-of-record, polluting it weakens incident response and non-repudiation.  
**Fix:** Run all free-text audit fields (resource_name, details, and any concatenated user input) through pkg/sanitize.LogValue (the CWE-117 sanitizer that already exists in this codebase) before persisting, and prefer structured/JSON details over ad-hoc string concatenation.

## 🔵 Low

**L1. Audit page filter `?action=document.viewed` can never match because view events are never recorded** (`contract-mismatch`)  
ui/src/routes/admin/audit/+page.svelte:38 (filter option); backend chain: internal/handler/audit_handler.go:50 (?action= -> AuditFilter.Action), internal/handler/analytics_handler.go:100 (sole production writer of domain.AuditDocumentViewed), internal/domain/audit.go:15 (constant)  
The audit filter dropdown offers `document.viewed` (audit/+page.svelte:38) and the backend audit List honors `?action=` (audit_handler.go:50). The `document.viewed` audit action is written only inside RecordViewEvent (analytics_handler.go:100), which—per the view-event finding—has no UI caller. So while the action constant exists (domain/audit.go:15 AuditDocumentViewed) the filter option will always yield an empty result set in normal operation.  
_Fix:_ Wire up POST /analytics/view-event (see the view-event finding); no audit-page change is required once views are actually recorded.

**L2. Category list pages render the response as a flat array but the backend returns a nested tree** (`contract-mismatch`)  
ui/src/routes/admin/categories/+page.svelte:16,87 and ui/src/routes/documents/+page.svelte:25,84 (consumers); backend internal/handler/category_handler.go:35,40 + internal/repository/category_repository.go:81-87,171-187 (buildTree)  
GET /categories returns a nested tree via `catRepo.ListAsTree` (category_handler.go:34-40), where child categories appear under each parent's `children` field (domain/document.go:52 `Children []*Category json:"children"`; api.ts Category.children?). Both consumers treat the payload as a flat list and only iterate the top level: admin/categories/+page.svelte:87 `{#each categories as cat}` (no recursion) and documents/+page.svelte:84 `{#each categories as cat}` for the filter dropdown. Any category with a non-null parent_id is nested under its parent and will not be shown.  
_Fix:_ Either flatten the tree client-side (recurse `children` into a flat list, optionally with indentation) before rendering, or add a query flag / separate flat endpoint and have these pages request the flat form.

**L3. JWT validation does not assert issuer/expiry-required and permits empty-secret signing** (`security`)  
internal/service/auth_service.go:157-173 (ValidateToken); minter at :257-273  
ValidateToken checks only that the signing method is HMAC and that token.Valid is true. It does not require/verify the Issuer claim ("dd-portal" is set but never asserted) and does not enforce that an expiry is present (jwt/v5 will validate exp if present, but a token minted with no exp would not be rejected by an explicit policy). Combined with the shared secret, the validator is permissive. There is also no leeway/clock-skew or audience binding.  
_Fix:_ Use jwt.WithValidMethods([]string{"HS256"}), jwt.WithIssuer("dd-portal"), and jwt.WithExpirationRequired() parser options to enforce algorithm, issuer, and mandatory expiry explicitly.

**L4. SVG/CSS sanitizer for uploaded branding assets exists and is tested but is never invoked; raw SVG stored and served, enabling stored XSS** (`security`)  
internal/handler/branding_handler.go:99-154 (UploadAsset), :85-97 (GetAsset); pkg/sanitize/sanitize.go:108-130 (unused IsSVGElementBlocked/IsSVGAttrBlocked); gate: cmd/main.go:173 + internal/middleware/jwt_auth.go:25-28  
UploadAsset accepts files for keys including `logo`, `favicon`, `login_background` (domain/branding.go:32-40). It reads the bytes, calls http.DetectContentType (branding_handler.go:128) only to set MimeType, and stores the raw bytes — there is NO MIME allowlist and NO SVG sanitization. The SVG sanitizer helpers IsSVGElementBlocked/IsSVGAttrBlocked exist and are unit-tested (sanitize_test.go:67-83) but a codebase grep shows they are never called by any handler. GetAsset (branding_handler.go:96) serves the stored bytes back with `c.Blob(http.StatusOK, asset.MimeType, asset.FileData)` using the stored/detected MIME. http.DetectContentType returns `image/svg+xml` for SVG content, so a malicious SVG containing `<script>` or `onload=` is served as image/svg+xml. While `X-Content-Type-Options: nosniff` is set globally (security_headers.go:11), an attacker can navigate directly to /api/v1/branding/assets/logo; a document served as image/svg+xml with embedded script executes as same-origin script in the browser (SVG is an active content type). CSP script-src is `'self' 'unsafe-inline'` (security_headers.go:19), so the inline SVG script runs.  
_Fix:_ In UploadAsset, enforce a strict MIME/type allowlist per key (e.g. PNG/JPEG/ICO for favicon/logo; if SVG is allowed, run it through a real SVG sanitizer that uses the existing IsSVGElementBlocked/IsSVGAttrBlocked during XML traversal and reject/strip blocked nodes). On GetAsset, force a safe Content-Type for SVG (serve as `text/plain` or `application/octet-stream` with `Content-Disposition: attachment`, or render SVG only inside an <img>/CSP-sandboxed context). Reject uploads whose detected type is not in the allowlist.

**L5. sanitize.FileName strips '..' non-recursively, leaving traversal sequences reconstructable** (`security`)  
pkg/sanitize/sanitize.go:133-143 (the ".." removal is line 134)  
FileName does a single-pass `strings.ReplaceAll(name, "..", "")` then removes `/` and `\`. Single-pass replacement of `..` is reconstructable: input `....//` -> after removing `..` once becomes `..` (the outer pair removed, inner pair remains is wrong order — concretely `"...."` -> ReplaceAll removes both non-overlapping `..` giving `""`, but `".../."`-style and `"..../"` inputs can leave residual `..`). More robustly, sequences like `..../` collapse to `../` style artifacts because ReplaceAll is not applied until stable. Currently FileName output is only used for Content-Disposition filename and audit log values (document_handler.go:155/204, :379/407), not for filesystem path construction (documents are stored as DB BLOBs), so real-world impact is low — but the function is a shared 'safe filename' primitive that other call sites may trust for path building.  
_Fix:_ Use filepath.Base + an allowlist regex (e.g. keep only `[A-Za-z0-9._-]`, collapse repeated dots) and loop the `..` removal until stable, or reject names containing path separators outright. Add test cases for `....//`, `..%2f`, and unicode separators.

**L6. Branding asset key path param is allowlisted on write/get-via-handler but DeleteAsset/GetAsset reach repo without ValidAssetKeys check** (`security`)  
internal/handler/branding_handler.go:86-88 (GetAsset), :158-160 (DeleteAsset) vs :103 (UploadAsset)  
UploadAsset validates `domain.ValidAssetKeys[key]` (branding_handler.go:103) before touching the repo, but GetAsset (line 86-88) and DeleteAsset (line 158-160) pass the raw `:key` path param straight to brandingRepo.GetAsset/DeleteAsset with no allowlist check. The repo presumably parametrizes the key in SQL (no injection), so this is not SQLi, but it means arbitrary key strings are accepted for read/delete. Combined with the absence of key normalization, this is an inconsistent-validation smell: only the write path is gated. If the repo ever does a LIKE/prefix or the key is used to build a cache path, unbounded keys become a problem; today it mainly allows probing for non-allowlisted keys and inconsistent error surfaces.  
_Fix:_ Apply the same `if !domain.ValidAssetKeys[key] { return BadRequest }` guard at the top of GetAsset and DeleteAsset for consistency and to bound the input space.

**L7. Concurrent UploadVersion calls compute version number from a stale read (lost-update / duplicate version race)** (`data-integrity`)  
internal/handler/document_handler.go:274-346 (read at 274, compute at 317, CreateVersion at 331); UNIQUE constraint at internal/repository/migrations/001_initial_schema.sql:98  
newVersionNum is derived as doc.CurrentVersion+1 (line 317) from a GetByID read taken at line 274, outside any transaction or lock. Two concurrent POST /documents/:id/versions requests both read CurrentVersion=N, both compute N+1, and both attempt CreateVersion with version_number=N+1. There is no UNIQUE(document_id, version_number) enforcement guaranteed at the handler level (relies on schema; even if the schema has it, one request 500s confusingly; if it does not, two rows share the same version_number). The SetMaxOpenConns(1) serializes the SQL statements but does NOT serialize the read-modify-write across the two HTTP requests.  
_Fix:_ Compute the next version number inside a transaction via SELECT MAX(version_number)+1 ... (or COALESCE(MAX,0)+1) FROM document_versions WHERE document_id=? with the insert in the same tx, and add/verify a UNIQUE(document_id, version_number) constraint so the DB is the source of truth.

**L8. time.Parse errors are systematically discarded across repositories, silently producing zero-value timestamps** (`bug`)  
internal/repository/document_repository.go:47,159,179,205,269-270,291-292; internal/repository/qa_repository.go:42-43,74-75 (also 131-132,162,182)  
Every timestamp round-trip ignores the parse error, e.g. `doc.CreatedAt, _ = time.Parse(time.RFC3339, now)` (line 47), `v.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)` (line 159, 179, 205), and scanDocument/scanDocumentRow at lines 269-270 and 291-292. The same `, _ =` pattern repeats in qa_repository.go (CreateThread lines 42-43, GetThread 74-75). If a stored timestamp is ever malformed (manual edit, schema drift, legacy row, non-RFC3339 default), the field becomes the Go zero time (0001-01-01) with no error surfaced, and downstream sorting/analytics/HasMore math silently misbehaves.  
_Fix:_ At minimum log/wrap parse failures, or centralize timestamp parsing in a helper that returns an error and propagate it. Storing timestamps as INTEGER unix epoch (or using a single parse helper that defaults predictably and logs via sanitize) would eliminate the class of bug.

**L9. FTS external-content table relies on stable implicit rowid; VACUUM or rowid reuse silently breaks search-to-document join** (`data-integrity`)  
internal/repository/migrations/002_fts_indexes.sql:4-9 (content=documents, content_rowid=rowid) and internal/repository/document_repository.go:227 (JOIN documents d ON d.rowid = fts.rowid); documents PK at internal/repository/migrations/001_initial_schema.sql:66 (id TEXT PRIMARY KEY)  
documents_fts is declared `content=documents, content_rowid=rowid`. `documents.id` is a TEXT PRIMARY KEY, so the table has the default implicit integer rowid, and the FTS index keys on that rowid. The Search join is `JOIN documents d ON d.rowid = fts.rowid`. Implicit rowids are not stable across `VACUUM` (rowids can be renumbered when there is no INTEGER PRIMARY KEY), and the FTS index is not automatically rebuilt by VACUUM. After a VACUUM the fts rowids and documents rowids can disagree, returning the wrong document (or none) for a search hit.  
_Fix:_ Either (a) add `WITHOUT ROWID` awareness by giving documents an explicit stable surrogate and keying FTS on it, or (b) after any VACUUM run `INSERT INTO documents_fts(documents_fts) VALUES('rebuild');`, and document that VACUUM requires an FTS rebuild. At minimum add a comment in 002 noting the rowid-stability dependency.

**L10. No FTS 'rebuild' for pre-existing rows; if 002 is applied to a DB that already has documents, they are unsearchable** (`data-integrity`)  
internal/repository/migrations/002_fts_indexes.sql:4-29 and internal/repository/migrate.go:17-46 (search path: internal/repository/document_repository.go:217,226-230)  
Migrations are run unconditionally on every boot and are guarded only by IF NOT EXISTS. The FTS virtual table and its triggers are created empty; population happens exclusively via the AFTER INSERT trigger going forward. There is no `INSERT INTO documents_fts(documents_fts) VALUES('rebuild');`. On a brand-new DB this is fine (documents are inserted after triggers exist). But if 002 is ever introduced into an existing deployment that already has documents (the realistic upgrade path), all rows that predate the migration are absent from documents_fts and will never appear in search.  
_Fix:_ Append `INSERT INTO documents_fts(documents_fts) VALUES('rebuild');` to 002 (it is cheap and idempotent for external-content FTS5 tables, repopulating from the content table on each boot), or gate it behind a one-time check.

**L11. UI is NOT embedded in the binary despite code comments claiming it is; production relies on filesystem assets** (`contract-mismatch`)  
cmd/main.go:332-357 (esp. 344, 349-356) and 412-413; comments also at 183, 333-334  
Comments at cmd/main.go:183, 333-334 and 349 state 'the UI is embedded in the binary' and 'try embedded FS (will be available in production builds)'. In reality setupStaticFiles only ever serves from the filesystem via http.Dir(uiBuildPath); the 'embedded' fallback path just returns a static placeholder HTML page. Line 413 `var _ fs.FS` is a no-op blank declaration with comment 'needed for future embed.FS usage' — there is no //go:embed of the UI anywhere (the only go:embed is for migrations at migrate.go:12).  
_Fix:_ Either actually embed the UI with //go:embed all:ui/build and serve from the embedded FS as the primary path, or remove the misleading 'embedded in the binary' comments and the dead `var _ fs.FS` line and document that DD_UI_PATH must point at built assets.

**L12. Self-signed cert mode is the default and only ever issues a localhost certificate** (`config`)  
cmd/main.go:50 (DD_TLS_MODE default "self-signed"); cmd/main.go:294-306 ensureSelfSignedCert template, Subject.CommonName "localhost" at line 298 and DNSNames []string{"localhost"} at line 305; docker/docker-compose.yml (ports "443:8443", DD_TLS_MODE: self-signed default)  
DD_TLS_MODE defaults to 'self-signed' (cmd/main.go:50) and ensureSelfSignedCert hardcodes Subject CommonName 'localhost' and DNSNames []string{"localhost"} (lines 299, 305). There is no SAN for the deployment hostname or IP, and the cert is regenerated only if absent. docker-compose.yml maps it to 443 as the default deployment.  
_Fix:_ Allow a DD_TLS_HOSTNAME/DD_TLS_SAN env to populate CommonName/DNSNames/IPAddresses for the self-signed cert, and document that custom mode should be used for any non-localhost deployment.

**L13. DD_MAX_UPLOAD_SIZE has two independent defaults (handler 100MB vs nothing server-wide) and no upper sanity bound** (`config`)  
internal/handler/document_handler.go:21,33-34 (const + env read); guard at lines 127 and 287; pkg/envconfig/envconfig.go:34-41 (no validation)  
defaultMaxUploadSize is defined as a const in the handler (100MB) and read via envconfig.GetEnvInt64("DD_MAX_UPLOAD_SIZE", defaultMaxUploadSize). There is no validation that the env value is positive or bounded; a negative or huge value is accepted verbatim. A value of 0 or negative would make file.Size > h.maxUploadSize always false, disabling the only upload guard, and a very large value with no BodyLimit (see separate finding) amplifies the memory-DoS risk.  
_Fix:_ Validate the parsed value at startup (reject <=0, cap at a sane maximum), log the effective limit, and enforce it via BodyLimit middleware in addition to the per-handler check.

**L14. notification_preferences table has no repository, handler, route, or UI — completely dangling** (`missing-feature`)  
internal/repository/migrations/001_initial_schema.sql:280-286 (table); internal/service/email_service.go:58,71,83 (send helpers ignore prefs)  
The notification_preferences table is created by migration 001 and listed in CLAUDE.md as a core table, but there is no domain model, repository, handler, route, or UI for it. grep `NotificationPref|notification_pref` across internal/domain and internal/repository returns only the schema SQL file. The only 'notification' references in Go are SMTP email helpers (SendQANotification, SendDocumentNotification) which send unconditionally and never consult any preference.  
_Fix:_ Either implement the full path (domain model + repository + handler + /users/:id/notifications routes + UI, and have EmailService consult preferences before sending), or remove the table and CLAUDE.md claim if out of scope.

**L15. Audit log export endpoint does not exist despite being part of the audit feature set** (`missing-feature`)  
internal/handler/audit_handler.go:26-31  
The audit handler registers only GET /audit, GET /audit/document/:id, GET /audit/user/:id (lines 28-30). There is no export/CSV endpoint — grep `export|csv|CSV|Export` in audit_handler.go returns nothing. For a due-diligence audit trail, exportable evidence is a typical expectation and is absent both server- and client-side (admin/audit UI lists entries only).  
_Fix:_ Add GET /audit/export (CSV/JSON, honoring the same AuditFilter) and an Export button in admin/audit. Low priority relative to the broken NDA/invite/watermark paths.

**L16. Invite role select omits 'admin'; invite token displayed in plaintext in the UI** (`quality`)  
ui/src/routes/admin/users/+page.svelte:67-70 (role select) and :32 (token in inviteMessage)  
The invite role <select> offers only 'investor' and 'company_member' (lines 68-69) — there is no way to invite another admin through the UI, so the admin role can only ever be seeded out-of-band. Separately, on success the raw invite token is rendered into the page: inviteMessage = `Invite sent. Token: ${res.data.token}` (line 32). The token is the secret that grants account creation; surfacing it in the DOM (and the 'Invite sent' wording implies an email was also sent) means the secret is exposed in plaintext on screen and in any screen recording.  
_Fix:_ Add 'Administrator' to the role options if admins should be invitable (gated to current admins). Do not render the raw token; rely on the emailed invite link, or show a copy-once masked value with a clear warning.

**L17. Forms lack validation, accessible labeling, and submit semantics (only login uses a real <form>)** (`quality`)  
ui/src/routes/admin/users/+page.svelte:64-72; admin/categories/+page.svelte:63-71; admin/nda/+page.svelte:78-83; qa/+page.svelte:61-70; admin/branding/+page.svelte:89-94; admin/watermark/+page.svelte:90-110  
Most create/edit panels are loose <div>s with bare <input>/<select>/<button onclick> rather than <form onsubmit>, so Enter does not submit, there is no native required validation, and screen readers get no form grouping. Inputs in admin/users, admin/categories, admin/nda, and the qa new-thread input have placeholders but no associated <label for> (placeholder-as-label is an a11y failure). Validation is only trim-truthiness in JS (e.g. categories addCategory:29 `if (!newName.trim()) return;` silently no-ops with no message). The branding color text inputs (branding/+page.svelte:93) accept any free text with no hex validation, feeding the raw-injection issue above. Watermark opacity/font_size/color inputs (watermark/+page.svelte) likewise have no bounds enforcement beyond HTML min/max which is bypassable.  
_Fix:_ Wrap each create panel in <form onsubmit>, add <label for>/aria-label to every control, surface validation errors instead of silent returns, and constrain color/number inputs with validated patterns.

**L18. Logout does not call the API and uses a hard window.location redirect, leaving a valid token server-side** (`security`)  
ui/src/lib/components/NavHeader.svelte:27-30 and ui/src/lib/stores/authStore.svelte.ts:25-31  
handleLogout calls authStore.clearAuth() (removes localStorage token) then `window.location.href = '/login'`. It never calls a backend logout/revocation endpoint, so the JWT remains valid until natural expiry; anyone who captured it (see localStorage XSS finding) can keep using it after the user 'logs out'. Using window.location.href instead of SvelteKit goto() forces a full reload (minor) but more importantly there is no server-side session invalidation.  
_Fix:_ Call a backend POST /auth/logout that revokes/blacklists the token (or rotates the refresh token) before clearing client state; if stateless JWTs are kept, document that logout is best-effort and shorten token TTL.

**L19. Global Cache-Control: no-store applied to all responses including immutable hashed static assets** (`quality`)  
internal/middleware/security_headers.go:16 (set), registered globally at cmd/main.go:71, wraps SPA static handler at cmd/main.go:345  
SecurityHeaders() sets `Cache-Control: no-store` unconditionally (security_headers.go:16) and is registered globally (cmd/main.go:71), so it runs on every route including the SPA static handler (cmd/main.go:345/389) that serves content-hashed /_app/* bundles, fonts, and images. no-store forbids the browser from caching these immutable, fingerprinted assets.  
_Fix:_ Scope no-store to API/sensitive responses only (e.g. apply it in a group middleware on /api/v1, or skip it when the path starts with /_app/ or /fonts/ and instead set Cache-Control: public, max-age=31536000, immutable for fingerprinted assets).

**L20. a11y warning: form label not associated with a control in admin/branding** (`quality`)  
/home/fish/GitHub/due-diligence-portal/ui/src/routes/admin/branding/+page.svelte:90  
`npx svelte-check` reports one warning: `A11y: A form label must be associated with a control` at admin/branding/+page.svelte line 90. The label is rendered without a `for`/`id` association or wrapping its control, so screen-reader users cannot reliably associate the label text with its input.  
_Fix:_ Associate the label with its control: add `for="<id>"` to the `<label>` and a matching `id` on the input, or nest the input inside the `<label>`.

**L21. Inviting an admin passes ValidateRole but violates the invite_tokens CHECK constraint, returning a generic 500** (`contract-mismatch`)  
internal/repository/migrations/001_initial_schema.sql:33 (invite_tokens role CHECK) vs internal/domain/user.go:16-20 (ValidRoles includes admin) and internal/service/auth_service.go:181 (CreateInvite calls ValidateRole); error mishandled at internal/handler/user_handler.go:157-164 (switch default -> InternalError)  
CreateInvite validates role with domain.ValidateRole, and ValidRoles includes 'admin' (domain/user.go:17). But the invite_tokens table CHECK restricts role IN ('company_member','investor') only. So CreateInvite(email,'admin',...) passes validation, then CreateInviteToken's INSERT fails the CHECK constraint; the handler's switch only maps ErrInvalidEmail/ErrInvalidRole and falls through to response.InternalError (500). The user_handler invite path never special-cases 'admin'.  
_Fix:_ Add an explicit allowlist for invitable roles (company_member, investor) in CreateInvite that returns ErrInvalidRole for 'admin', so the handler returns a clean 400 'invalid role'. Keep ValidRoles for login/general use but introduce ValidInviteRoles.

**L22. Register marks invite token used non-atomically after user creation (TOCTOU + ignored failure)** (`data-integrity`)  
internal/service/auth_service.go:92-135 (mark at 130-132, check at 98-100); repo internal/repository/user_repository.go:175-183 (MarkInviteTokenUsed) and :37-50 (Create)  
Register checks invite.UsedAt, then Creates the user, then separately calls MarkInviteTokenUsed — and the mark failure is only logged ('[WARN] Failed to mark invite token used'), not surfaced. The UsedAt check and the mark are not transactional, so two concurrent Register calls with the same token both pass the UsedAt==nil check before either marks it used. The email UNIQUE constraint blocks a literal duplicate account, but the second attempt fails inside Create with a generic wrapped error -> 500, and if MarkInviteTokenUsed silently fails the token stays replayable until expiry.  
_Fix:_ Perform the user creation and the invite-token consumption in a single transaction, and make MarkInviteTokenUsed a conditional UPDATE (... WHERE token=? AND used_at IS NULL) whose RowsAffected==0 aborts the registration as ErrTokenUsed.

**L23. Permission Grant does not validate access_level allowlist; invalid values return 500 instead of 400** (`quality`)  
internal/handler/permission_handler.go:71-77 (Grant) and :124 (Update)  
Grant validates resource_type against an allowlist but NOT access_level. Valid levels are view/download/upload/manage (domain/permission.go and the access_grants CHECK). A typo or arbitrary value (e.g. 'read') passes the handler, then violates the DB CHECK constraint and the repo error maps to response.InternalError (500). Update has the same gap (permission_handler.go:124 only checks non-empty).  
_Fix:_ Validate req.AccessLevel against {view,download,upload,manage} in both Grant and Update and return BadRequest with a clear message; share a domain.ValidAccessLevels map.

**L24. Branding UpdateConfig silently mutates CustomCSS and discards the 'modified' signal; no rejection or feedback on blocked content** (`quality`)  
internal/handler/branding_handler.go:58-62 (sanitize.CSS at pkg/sanitize/sanitize.go:67-81)  
UpdateConfig calls sanitize.CSS(config.CustomCSS) but discards the second return value (the 'modified' bool) with `_`. When the sanitizer strips @import/url()/expression()/javascript: etc., it replaces them with '/* blocked */' and silently stores the mutated CSS, returning 200 with no indication that the admin's input was altered. Combined with the already-noted weakness that sanitize.CSS does not block }{ breakout, an admin gets neither protection nor feedback.  
_Fix:_ Capture the modified flag; if true, either reject with 400 listing the blocked patterns or return 200 with a message/meta indicating the CSS was sanitized so the admin can review.

**L25. Analytics DocumentAnalytics/UserAnalytics return empty zero-value objects for nonexistent IDs instead of 404** (`quality`)  
internal/handler/analytics_handler.go:48-70 (DocumentAnalytics + UserAnalytics) and internal/repository/analytics_repository.go:62-63, 96-97  
GetDocumentAnalytics/GetUserAnalytics return a zero-value struct (just the echoed ID) on sql.ErrNoRows, and the handlers return 200 with that empty object. There is no check that the document or user actually exists — a caller requesting analytics for a random/nonexistent id gets a 200 with all-zero stats, indistinguishable from a real resource that simply has no views. This also lets a company_member confirm/deny existence indirectly only by absence of a name field.  
_Fix:_ Validate existence of the document/user first (or detect the LEFT JOIN name being empty AND view_count 0) and return 404 for unknown IDs; otherwise document that zero-value means 'no recorded activity'.

**L26. Login is vulnerable to user-enumeration / timing oracle: bcrypt compare is skipped for unknown or disabled accounts** (`security`)  
internal/service/auth_service.go:56-67 (Login), mapped in internal/handler/auth_handler.go:49-56  
Login returns ErrInvalidCredentials immediately when GetByEmail fails (no bcrypt work performed) and returns the distinct ErrAccountDisabled before any password check when the account is inactive. The bcrypt.CompareHashAndPassword (cost 12, ~hundreds of ms) only runs for existing+active accounts. This yields both a response-content oracle (disabled vs invalid maps to 403 vs 401 in auth_handler.go:51-56) and a timing oracle distinguishing existing from non-existing accounts.  
_Fix:_ Always perform a bcrypt comparison against a constant dummy hash when the user is missing or disabled to equalize timing, and return an identical generic 401 for invalid/disabled/unknown (move the disabled check after the password comparison, or fold it into the generic response).

## ⚪ Info / hardening notes

**I1. RowsAffected error is discarded in Update and Archive, masking driver failures and misclassifying as not-found** — internal/repository/document_repository.go:127 (Update) and :141 (Archive)  
Both Update (line 127 `rows, _ := result.RowsAffected()`) and Archive (line 141 `rows, _ := result.RowsAffected()`) ignore the error returned by RowsAffected. If the driver returned an error here, rows would be 0 and the code returns domain.ErrDocumentNotFound, which the handler maps to HTTP 404 — misrepresenting a backend failure as 'document not found'. The same ignored-error pattern is consistent with the codebase but is a real correctness smell on a write path. _Fix:_ Capture and check the error: `rows, err := result.RowsAffected(); if err != nil { return fmt.Errorf("rows affected: %w", err) }` before testing rows == 0.

**I2. List pagination Page computation would divide by zero if limit ever reaches 0 (currently guarded, but fragile)** — internal/handler/document_handler.go:53-94 (division at :92, guard at :56)  
Page is computed as `(offset / limit) + 1` (line 92) and HasMore as `offset+limit < total` (line 94). limit defaults to 50 and is only overwritten when the parsed value satisfies `n > 0 && n <= 100` (line 56), so today limit can never be 0 and there is no panic. However the divide-by-zero safety is entirely incidental to the input-validation guard; any future change that allows limit=0 (or a different caller of docRepo.List passing 0) produces an integer divide-by-zero panic in the handler. Search hardcodes PageSize=50 so it is unaffected. _Fix:_ Defensively clamp limit to a minimum of 1 right before the Meta computation (e.g. `if limit < 1 { limit = 1 }`), decoupling the safety of the division from the parsing branch.

**I3. document_versions.uploaded_by / documents.uploaded_by FK to users(id) with no ON DELETE; coupled with soft-delete it is consistent but undocumented** — internal/repository/migrations/001_initial_schema.sql:70,96,128,155-156,186,193-194,240,250-251; internal/repository/user_repository.go:114-126  
Many tables reference users(id) with no ON DELETE action (documents.uploaded_by:70, document_versions.uploaded_by:96, audit_log.user_id:128, qa_threads.asked_by/assigned_to:155-156, nda_templates.created_by:186, nda_signatures.template_id/user_id:193-194, branding_assets.uploaded_by:240, view_events.user_id/document_id:250-251). The app only soft-deletes users (Deactivate sets is_active=0, user_repository.go:114-126) and there is no hard DELETE FROM users path, so these FKs never fire — the design is internally consistent. Flagging because it is a load-bearing invariant that is nowhere documented: if a hard user delete is ever added, it will FK-fail across ~8 tables. _Fix:_ Document the soft-delete-only invariant in CLAUDE.md / schema comments, or choose explicit ON DELETE policies (e.g. audit_log.user_id ON DELETE SET NULL since it is already nullable) so the schema encodes the intent.

**I4. showNav driven by token presence only, so nav renders during the unauthenticated/invalid-token window** — ui/src/routes/+layout.svelte:13-14  
showNav = hasToken && !isPublicRoute, where hasToken = authStore.token !== null. The token is set from localStorage before /auth/me validates it (authStore.loadFromStorage). So if a stale/expired token sits in localStorage, the full nav header renders until a request 401s and client.ts redirects. The nav is gated on token existence, not on a verified session (isAuthenticated, which also requires user). _Fix:_ Gate showNav on authStore.isAuthenticated (verified user) rather than raw token presence, after awaiting restore().

**I5. Build/test health baseline: Go build, vet, tests, UI build, svelte-check, vitest, and eslint all pass** — repo root + ui/ (Go module github.com/witfoo/due-diligence-portal)  
Recording what is healthy so the defects above are isolated. Verified in this review: `go build ./...` exit 0 (no output), `go vet ./...` exit 0, and `go test ./...` passes every package (handler, middleware, repository, service, envconfig, sanitize); the repo root is not a Go package so it is excluded from `./...`. UI: `npm run build` succeeds (adapter-static), `npx svelte-check` reports 0 errors (1 a11y warning — see L20), `npm test` (vitest) passes, and `npm run lint` exits clean. Note: a finder agent transiently created a stray empty `scratch_fts_test.go` in its own sandbox and flagged it as a build break; that file does **not** exist in the repo (verified) and was correctly refuted — there is no such defect. _Fix:_ Keep CI running all of the above; add a guard so empty/placeholder `*_test.go` files and unimplemented-but-linked routes fail the pipeline rather than shipping silently green.

**I6. Response envelope omitempty drops Meta.Total and pagination fields when zero, breaking client-side pagination math** — pkg/response/response.go:25-28  
Meta uses `json:"total,omitempty"`, `page,omitempty`, `page_size,omitempty`, and `has_more,omitempty`. When a list is genuinely empty (total=0) or on the first page (page semantics), these fields are omitted entirely from the JSON. A client computing 'has more pages' as data.length < meta.total, or rendering 'Page X', receives undefined for meta.total/meta.has_more when total is 0, and cannot distinguish 'no results' from 'field absent'. has_more=false (the common non-last-page-negative case) is also always omitted, so a client checking `if (meta.has_more)` works but one checking `meta.has_more === false` never sees it. _Fix:_ Remove omitempty from Total, Page, PageSize, and HasMore in Meta so the envelope always reports the full pagination state; these are meaningful at zero/false.

---

## Refuted (investigated, not real)

For transparency, 7 candidate findings were checked and **dismissed** by the verification pass:

| Candidate | Why dismissed |
| --- | --- |
| "Empty `scratch_fts_test.go` breaks `go test ./...`" | File does not exist in the repo — a finder agent created it transiently in its own sandbox. Build/vet/test are green (see I5). |
| "`Permission Grant` doesn't validate the target user exists / access_level" | Partially true but mis-scoped; the real, confirmed issues are tracked as M17 (re-grant 500) and L23 (access_level allowlist). |
| "Upload size check bypassable via spoofed Content-Length" | The `file.Size`-based check and `io.ReadAll` behavior are real concerns but captured accurately under H13; the spoof framing as stated was incorrect. |
| "`DELETE categories` has no ON DELETE handling → opaque FK error" | Not reproducible as described against the actual schema/handler. |
| "`Recover()` ordered before `RequestID` leaks IDs / stack traces" | Middleware order does not produce the claimed effect; panic stacks are not leaked to clients. |
| "Inviting `admin` truncates bcrypt at 72 bytes" | Conflated two things; the real invite-role defect is L21 (CHECK-constraint 500). |
| "Admin nav lands on `/admin/users` with no role enforcement" | The nav link is hidden for non-admins; the real gap (no *hard* guard on direct navigation) is captured as H10. |

---

## Suggested remediation order

**1 — Stop the bleeding (security, do first):**
- C6 / fail-fast on default `DD_JWT_SECRET`; C4 document authorization on Get/Download/DownloadVersion/Search;
  C5 scope Q&A to the user + hide internal messages; H12 set a trust-aware `IPExtractor`; H7 stop logging the admin
  password; H13 add `BodyLimit`; H14 gate/validate view-events.

**2 — Make the product usable (the reported bug + core journeys):**
- C1 upload page; C2 document-detail + Q&A-thread pages (and M1 `+error.svelte`); C3 fetch-based download;
  H9 `/register` page; H1 NDA signing page + gate.

**3 — Integrity & correctness:**
- H4 wrap document writes in transactions; H5 FTS query escaping + 400s; H6 archived/ungranted search filter;
  H3 token-type claim; M14 audit IDs via crypto/rand; M17 grant upsert.

**4 — Feature completeness & polish:**
- H8 apply watermark (or remove the claim); H2 category-grant filter; M2 emit view-events; M16 wire email
  notifications; M10 branding asset upload; M13 user management actions; then the Low/Info hardening items.

**Process:** add CI gates so an **unimplemented-but-linked route** and **placeholder `*_test.go`** files fail the
build rather than shipping "green" (this class of defect is why the data plane and UI drifted apart).
