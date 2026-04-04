import { sveltekit } from '@sveltejs/kit/vite';
import { defineConfig } from 'vitest/config';

export default defineConfig({
	plugins: [sveltekit()],
	server: {
		proxy: {
			'/api': {
				target: 'http://localhost:8080',
				changeOrigin: true
			},
			'/health': {
				target: 'http://localhost:8080',
				changeOrigin: true
			}
		}
	},
	test: {
		include: ['src/**/*.{test,spec}.{js,ts}', 'tests/**/*.{test,spec}.{js,ts}'],
		exclude: ['node_modules', 'dist', 'build', 'tests/e2e/**'],
		globals: true,
		environment: 'jsdom',
		alias: {
			$lib: '/src/lib',
			$api: '/src/lib/api',
			$types: '/src/lib/types',
			$components: '/src/lib/components',
			$stores: '/src/lib/stores'
		},
		testTimeout: 10000
	}
});
