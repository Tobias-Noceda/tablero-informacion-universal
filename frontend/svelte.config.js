import adapter from '@sveltejs/adapter-vercel';
import { vitePreprocess } from '@sveltejs/vite-plugin-svelte';

/** @type {import('@sveltejs/kit').Config} */
const config = {
	// Consult https://svelte.dev/docs/kit/integrations
	// for more information about preprocessors
	preprocess: vitePreprocess(),
	compilerOptions: {
		runes: true
	},

	vitePlugin: {
		dynamicCompileOptions({ filename }) {
			if (filename.includes('node_modules')) {
				return {
					runes: undefined
				};
			}
		}
	},

	kit: {
		// adapter-auto only supports some environments, see https://svelte.dev/docs/kit/adapter-auto for a list.
		// If your environment is not supported, or you settled on a specific environment, switch out the adapter.
		// See https://svelte.dev/docs/kit/adapters for more information about adapters.
		adapter: adapter(),
		alias: {
			$assets: 'src/lib/assets',
			$components: 'src/lib/components',
			$modules: 'src/lib/modules',
			$services: 'src/lib/services',
			$stores: 'src/lib/stores',
			$types: 'src/lib/types',
		},
		version: {
			name: process.env.BUILD_VERSION ?? '1',
		}
	}
};

export default config;
