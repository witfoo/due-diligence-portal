<script lang="ts">
	// Dashboard / Landing page
	// Redirects to /login if not authenticated, /documents if authenticated.
</script>

<main>
	<div class="hero">
		<h1>Due Diligence Portal</h1>
		<p>Securely share documents with investors.</p>
		<a href="/login" class="btn-primary">Sign In</a>
	</div>
</main>

<style>
	main {
		display: flex;
		justify-content: center;
		align-items: center;
		min-height: 100vh;
	}

	.hero {
		text-align: center;
		max-width: 480px;
		padding: 2rem;
	}

	h1 {
		font-size: 2.5rem;
		font-weight: 300;
		margin-bottom: 1rem;
	}

	p {
		color: var(--dd-text-secondary);
		font-size: 1.125rem;
		margin-bottom: 2rem;
	}

	.btn-primary {
		display: inline-block;
		padding: 0.875rem 2rem;
		background: var(--dd-primary);
		color: #fff;
		border: none;
		font-size: 1rem;
		cursor: pointer;
		text-decoration: none;
	}

	.btn-primary:hover {
		background: #0353e9;
		text-decoration: none;
	}
</style>
