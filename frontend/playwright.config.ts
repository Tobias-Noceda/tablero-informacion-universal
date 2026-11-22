import { defineConfig, devices } from '@playwright/test';

// End-to-end tests against the whole app behind the Docker ALB (see e2e/README.md).
// E2E_BASE_URL points them elsewhere; the helpers in e2e/support also reach
// the stack's logs and database through `docker compose`.
export default defineConfig({
	testDir: './e2e',
	// The tests share one stack: one at a time keeps them independent.
	workers: 1,
	fullyParallel: false,
	forbidOnly: !!process.env.CI,
	retries: process.env.CI ? 1 : 0,
	timeout: 60_000,
	expect: { timeout: 10_000 },
	reporter: process.env.CI ? [['list'], ['html', { open: 'never' }]] : 'list',
	use: {
		baseURL: process.env.E2E_BASE_URL ?? 'http://localhost',
		locale: 'en-US',
		trace: 'retain-on-failure',
		screenshot: 'only-on-failure'
	},
	projects: [{ name: 'chromium', use: { ...devices['Desktop Chrome'] } }]
});
