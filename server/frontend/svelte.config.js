import adapter from '@sveltejs/adapter-static';

/** @type {import('@sveltejs/kit').Config} */
const config = {
	compilerOptions: {
		runes: ({ filename }) => (filename.split(/[/\\]/).includes('node_modules') ? undefined : true)
	},
	kit: {
		adapter: adapter({
			fallback: 'index.html', // SPA mode
			pages: 'dist',
			assets: 'dist',
			precompress: false,
			strict: false
		}),
		paths: {
			base: '/ui'
		}
	}
};

export default config;
