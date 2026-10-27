import tailwindcss from '@tailwindcss/vite';
import adapter from '@sveltejs/adapter-cloudflare';
import { sveltekit } from '@sveltejs/kit/vite';
import { defineConfig } from 'vite';

export default defineConfig({
	plugins: [
		tailwindcss(),
		sveltekit({
			compilerOptions: {
				// Force runes mode for the project, except for libraries. Can be removed in svelte 6.
				runes: ({ filename }) =>
					filename.split(/[/\\]/).includes('node_modules') ? undefined : true
			},
			adapter: process.env.VITEST ? undefined : adapter()
		})
	],
	server: {
		proxy: {
			'/api': {
				target: process.env.PUBLIC_API_ORIGIN ?? 'http://localhost:8080',
				changeOrigin: true
			},
			'/mcp': {
				target: process.env.PUBLIC_API_ORIGIN ?? 'http://localhost:8080',
				changeOrigin: true
			},
			'/healthz': {
				target: process.env.PUBLIC_API_ORIGIN ?? 'http://localhost:8080',
				changeOrigin: true
			}
		}
	}
});
