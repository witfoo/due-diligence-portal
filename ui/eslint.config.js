import tsParser from '@typescript-eslint/parser';
import tsPlugin from '@typescript-eslint/eslint-plugin';
import securityPlugin from 'eslint-plugin-security';

export default [
	{
		files: ['src/**/*.{js,ts}', 'tests/**/*.{js,ts}'],
		languageOptions: {
			parser: tsParser,
			parserOptions: {
				ecmaVersion: 'latest',
				sourceType: 'module'
			}
		},
		plugins: {
			'@typescript-eslint': tsPlugin,
			security: securityPlugin
		},
		rules: {
			// TypeScript rules
			'@typescript-eslint/no-unused-vars': [
				'warn',
				{ argsIgnorePattern: '^_', varsIgnorePattern: '^_' }
			],

			// Security rules (eslint-plugin-security)
			'security/detect-buffer-noassert': 'warn',
			'security/detect-child-process': 'warn',
			'security/detect-disable-mustache-escape': 'warn',
			'security/detect-eval-with-expression': 'warn',
			'security/detect-new-buffer': 'warn',
			'security/detect-no-csrf-before-method-override': 'warn',
			'security/detect-non-literal-fs-filename': 'warn',
			'security/detect-non-literal-regexp': 'warn',
			'security/detect-non-literal-require': 'warn',
			'security/detect-object-injection': 'off', // False positives with TypeScript
			'security/detect-possible-timing-attacks': 'warn',
			'security/detect-pseudoRandomBytes': 'warn',
			'security/detect-unsafe-regex': 'warn'
		}
	},
	{
		files: ['tests/e2e/**/*.{js,ts}'],
		rules: {
			'security/detect-non-literal-fs-filename': 'off',
			'security/detect-non-literal-require': 'off'
		}
	},
	{
		ignores: ['build/**', '.svelte-kit/**', 'dist/**', 'node_modules/**']
	}
];
