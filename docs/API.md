# API Reference

Base URL: `/api/v1`

All responses use the standard envelope:

```json
{
  "success": true,
  "message": "Description",
  "data": {},
  "meta": { "count": 10, "total": 50, "page": 1, "page_size": 50 },
  "timestamp": "2026-04-04T10:30:00Z"
}
```

## Authentication

| Method | Path | Auth | Description |
| --- | --- | --- | --- |
| POST | `/auth/login` | No | Login with email + password. Returns JWT tokens. |
| POST | `/auth/register` | No | Register via invite token. |
| POST | `/auth/refresh` | No | Refresh access token. |
| GET | `/auth/me` | Yes | Current user profile. |

## Documents

| Method | Path | Auth | Role | Description |
| --- | --- | --- | --- | --- |
| GET | `/documents` | Yes | All | List documents (filter: `?category_id=`) |
| GET | `/documents/:id` | Yes | All | Document detail + version history |
| POST | `/documents` | Yes | Admin, Company | Upload document (multipart) |
| PUT | `/documents/:id` | Yes | Admin, Company | Update metadata |
| DELETE | `/documents/:id` | Yes | Admin, Company | Archive (soft delete) |
| POST | `/documents/:id/versions` | Yes | Admin, Company | Upload new version |
| GET | `/documents/:id/versions/:v` | Yes | All | Download specific version |
| GET | `/documents/:id/download` | Yes | All | Download current version |
| POST | `/documents/search` | Yes | All | Full-text search (`{"query":"..."}`) |

## Categories

| Method | Path | Auth | Role | Description |
| --- | --- | --- | --- | --- |
| GET | `/categories` | Yes | All | List categories (tree structure) |
| POST | `/categories` | Yes | Admin | Create category |
| PUT | `/categories/:id` | Yes | Admin | Update category |
| DELETE | `/categories/:id` | Yes | Admin | Delete (must be empty) |

## Permissions

| Method | Path | Auth | Role | Description |
| --- | --- | --- | --- | --- |
| GET | `/permissions/document/:id` | Yes | Admin | List grants for document |
| GET | `/permissions/category/:id` | Yes | Admin | List grants for category |
| POST | `/permissions` | Yes | Admin | Grant access |
| PUT | `/permissions/:id` | Yes | Admin | Update access level |
| DELETE | `/permissions/:id` | Yes | Admin | Revoke access |

## Q&A

| Method | Path | Auth | Role | Description |
| --- | --- | --- | --- | --- |
| GET | `/qa` | Yes | All | List threads (filter: `?status=`) |
| POST | `/qa` | Yes | All | Create question |
| GET | `/qa/:id` | Yes | All | Thread with messages |
| POST | `/qa/:id/messages` | Yes | All | Post message |
| PATCH | `/qa/:id/status` | Yes | Admin, Company | Change status |

## NDA

| Method | Path | Auth | Role | Description |
| --- | --- | --- | --- | --- |
| GET | `/nda/templates` | Yes | Admin | List templates |
| POST | `/nda/templates` | Yes | Admin | Create template |
| PUT | `/nda/templates/:id` | Yes | Admin | Update template |
| GET | `/nda/status` | Yes | All | Check signing status |
| POST | `/nda/sign/:templateId` | Yes | All | Sign NDA |
| GET | `/nda/signatures` | Yes | Admin | List all signatures |

## Audit

| Method | Path | Auth | Role | Description |
| --- | --- | --- | --- | --- |
| GET | `/audit` | Yes | Admin | List audit log (filter: `?action=&user_id=`) |
| GET | `/audit/document/:id` | Yes | Admin | Audit trail for document |
| GET | `/audit/user/:id` | Yes | Admin | Activity for user |

## Analytics

| Method | Path | Auth | Role | Description |
| --- | --- | --- | --- | --- |
| GET | `/analytics/dashboard` | Yes | Admin, Company | Engagement summary |
| GET | `/analytics/documents/:id` | Yes | Admin, Company | Per-document analytics |
| GET | `/analytics/users/:id` | Yes | Admin, Company | Per-user engagement |
| POST | `/analytics/view-event` | Yes | All | Record view event |

## Branding

| Method | Path | Auth | Role | Description |
| --- | --- | --- | --- | --- |
| GET | `/branding` | Yes | All | Get branding config |
| PUT | `/branding` | Yes | Admin | Update branding (CSS sanitized) |
| DELETE | `/branding` | Yes | Admin | Reset to defaults |
| GET | `/branding/assets/:key` | Yes | All | Get asset binary |
| POST | `/branding/assets/:key` | Yes | Admin | Upload asset (max 2MB) |
| DELETE | `/branding/assets/:key` | Yes | Admin | Delete asset |

Valid asset keys: `logo`, `logo_dark`, `favicon`, `login_background`, `email_header`, `report_header`, `report_footer`

## Watermark

| Method | Path | Auth | Role | Description |
| --- | --- | --- | --- | --- |
| GET | `/watermark` | Yes | Admin | Get watermark config |
| PUT | `/watermark` | Yes | Admin | Update watermark config |
| DELETE | `/watermark` | Yes | Admin | Reset to defaults |

## Users

| Method | Path | Auth | Role | Description |
| --- | --- | --- | --- | --- |
| GET | `/users` | Yes | Admin | List users |
| GET | `/users/:id` | Yes | Admin | Get user |
| PUT | `/users/:id` | Yes | Admin | Update user |
| DELETE | `/users/:id` | Yes | Admin | Deactivate user |
| POST | `/users/invite` | Yes | Admin | Create invite |

## Health (no auth, no `/api/v1` prefix)

| Method | Path | Description |
| --- | --- | --- |
| GET | `/health` | Basic health check |
| GET | `/ready` | Readiness (SQLite ping) |
| GET | `/version` | Build version info |
