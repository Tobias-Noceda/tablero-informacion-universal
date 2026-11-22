import { expect, test } from '@playwright/test';

import { account, boards, signIn, t } from './support/stack';

const VALUE = 'sk_live_E2E_NEVER_SHOWN_42';

// A credential is saved by name, and its value never
// comes back: not on the page, not in any answer the app receives.
test('keeps a credential value out of the page and every response', async ({ page }) => {
	const ana = await account('ana');
	const board = await boards.create(ana, 'Vault');

	const bodies: string[] = [];
	page.on('response', async (res) => {
		if (res.url().includes('/api/')) bodies.push(await res.text().catch(() => ''));
	});

	await signIn(page, ana);
	await page.goto(`/boards/${board.id}`);
	await page.getByRole('button', { name: t('secrets.title') }).click();
	await page.getByRole('button', { name: t('secrets.manage') }).click();
	await page.getByRole('button', { name: t('secrets.add_static') }).click();
	await page.getByPlaceholder('API_KEY').fill('PAYMENTS_KEY');
	await page.locator('input[type="password"]').fill(VALUE);
	await page.getByRole('button', { name: t('secrets.save') }).click();

	await expect(page.getByText('PAYMENTS_KEY').first()).toBeVisible();
	await page.reload();
	await page.getByRole('button', { name: t('secrets.title') }).click();
	await expect(page.getByText('PAYMENTS_KEY').first()).toBeVisible();

	expect(await page.content()).not.toContain(VALUE);
	expect(bodies.some((body) => body.includes(VALUE))).toBe(false);
});
