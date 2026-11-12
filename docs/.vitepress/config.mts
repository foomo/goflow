import { defineConfig } from 'vitepress'

// https://vitepress.dev/reference/site-config
export default defineConfig({
	title: 'goflow',
	description: 'Type-safe stream processing for Go',
	lang: "en-US",
	lastUpdated: true,
	appearance: "dark",
	ignoreDeadLinks: true,
	base: '/goflow/',
	sitemap: {
		hostname: 'https://foomo.github.io/goflow',
	},
	themeConfig: {
		// https://vitepress.dev/reference/default-theme-config
		logo: '/logo.png',
		outline: [2, 4],
		nav: [
			{ text: 'Guide', link: '/guide/getting-started' },
			{ text: 'Examples', link: '/examples/' },
			{ text: 'API', link: '/api/reference' },
		],
		sidebar: [
			{
				text: 'Guide',
				items: [
					{ text: 'Getting Started', link: '/guide/getting-started' },
					{ text: 'Operators', link: '/guide/operators' },
					{ text: 'Concurrency', link: '/guide/concurrency' },
					{ text: 'Error Handling', link: '/guide/error-handling' },
					{ text: 'Graceful Shutdown', link: '/guide/graceful-shutdown' },
					{ text: 'Advanced', link: '/guide/advanced' },
				],
			},
			{
				text: 'Examples',
				link: '/examples/',
				items: [
					{ text: 'Parallel HTTP Requests', link: '/examples/parallel-http-requests' },
					{ text: 'File Import with Bulk Inserts', link: '/examples/file-import-with-bulk-inserts' },
					{ text: 'Background Job Queue', link: '/examples/background-job-queue' },
					{ text: 'Event Buffering for a Slow Sink', link: '/examples/event-buffering-for-a-slow-sink' },
					{ text: 'Paginated API Crawl', link: '/examples/paginated-api-crawl' },
					{ text: 'Rate-Limited Calls with Deduplication', link: '/examples/rate-limited-calls-with-deduplication' },
					{ text: 'Merge Multiple Sources', link: '/examples/merge-multiple-sources' },
					{ text: 'Broadcast to Multiple Sinks', link: '/examples/broadcast-to-multiple-sinks' },
					{ text: 'Moving Average and Anomaly Detection', link: '/examples/moving-average-and-anomaly-detection' },
					{ text: 'First Match with Early Exit', link: '/examples/first-match-with-early-exit' },
				],
			},
			{
				text: 'API',
				items: [
					{ text: 'Reference', link: '/api/reference' },
				],
			},
			{
				text: 'Contributing',
				collapsed: true,
				items: [
					{
						text: "Guideline",
						link: '/CONTRIBUTING.md',
					},
					{
						text: "Code of conduct",
						link: '/CODE_OF_CONDUCT.md',
					},
					{
						text: "Security guidelines",
						link: '/SECURITY.md',
					},
				],
			},
		],
		socialLinks: [
			{ icon: 'github', link: 'https://github.com/foomo/goflow' },
		],
		editLink: {
			pattern: 'https://github.com/foomo/goflow/edit/main/docs/:path',
		},
		search: {
			provider: 'local',
		},
		footer: {
			message: 'Made with ♥ <a href="https://www.foomo.org">foomo</a> by <a href="https://www.bestbytes.com">bestbytes</a>',
		},
	},
	markdown: {
		// https://github.com/vuejs/vitepress/discussions/3724
		theme: {
			light: 'catppuccin-latte',
			dark: 'catppuccin-frappe',
		}
	},
	head: [
		['meta', { name: 'theme-color', content: '#ffffff' }],
		['link', { rel: 'icon', href: '/logo.png' }],
		['meta', { name: 'author', content: 'foomo by bestbytes' }],
		// OpenGraph
		['meta', { property: 'og:title', content: 'foomo/goflow' }],
		[
			'meta',
			{
				property: 'og:image',
				content: 'https://github.com/foomo/goflow/blob/main/docs/public/banner.png?raw=true',
			},
		],
		[
			'meta',
			{
				property: 'og:description',
				content: 'Type-safe stream processing for Go',
			},
		],
		['meta', { name: 'twitter:card', content: 'summary_large_image' }],
		[
			'meta',
			{
				name: 'twitter:image',
				content: 'https://github.com/foomo/goflow/blob/main/docs/public/banner.png?raw=true',
			},
		],
		[
			'meta', { name: 'viewport', content: 'width=device-width, initial-scale=1.0, viewport-fit=cover',
			},
		],
	]
})
