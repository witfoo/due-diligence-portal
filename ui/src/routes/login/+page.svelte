<script lang="ts">
	// Login page - to be implemented in Phase 2
</script>

<main>
	<div class="login-container">
		<h1>Sign In</h1>
		<form>
			<div class="field">
				<label for="email">Email</label>
				<input id="email" type="email" placeholder="Enter your email" />
			</div>
			<div class="field">
				<label for="password">Password</label>
				<input id="password" type="password" placeholder="Enter your password" />
			</div>
			<button type="submit" class="btn-primary">Sign In</button>
		</form>
	</div>
</main>

<style>
	main {
		display: flex;
		justify-content: center;
		align-items: center;
		min-height: 100vh;
	}

	.login-container {
		width: 100%;
		max-width: 400px;
		padding: 2rem;
		background: var(--dd-surface);
		border: 1px solid var(--dd-border);
	}

	h1 {
		font-size: 1.75rem;
		font-weight: 400;
		margin-bottom: 2rem;
	}

	.field {
		margin-bottom: 1.5rem;
	}

	label {
		display: block;
		font-size: 0.875rem;
		color: var(--dd-text-secondary);
		margin-bottom: 0.5rem;
	}

	input {
		width: 100%;
		padding: 0.75rem 1rem;
		background: var(--dd-background);
		border: 1px solid var(--dd-border);
		color: var(--dd-text);
		font-size: 0.875rem;
	}

	input:focus {
		outline: 2px solid var(--dd-primary);
		outline-offset: -2px;
	}

	.btn-primary {
		width: 100%;
		padding: 0.875rem;
		background: var(--dd-primary);
		color: #fff;
		border: none;
		font-size: 1rem;
		cursor: pointer;
	}

	.btn-primary:hover {
		background: #0353e9;
	}
</style>
