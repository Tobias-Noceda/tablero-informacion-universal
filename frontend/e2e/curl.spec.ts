import { expect, test } from '@playwright/test';

import { account, boards, signIn, t } from './support/stack';

// U3: a cURL command fills in a custom card's request; a credential in it is
// not kept as text, and the request can be tried right away.
test('fills a custom card in from a cURL command', async ({ page }) => {
	const ana = await account('ana');
	const board = await boards.create(ana, 'From docs');
	await signIn(page, ana);
	await page.goto(`/boards/${board.id}`);

	await page.getByRole('button', { name: t('nodes.title') }).click();
	await page
		.locator('aside')
		.getByText(t('one_off.sidebar'))
		.dragTo(page.locator('.svelte-flow__pane'), { targetPosition: { x: 300, y: 200 } });
	const dialog = page.getByRole('dialog');

	await dialog.getByText(t('curl.toggle')).click();
	await dialog
		.getByRole('textbox', { name: t('curl.toggle') })
		.fill(
			`curl -s 'http://203.0.113.10:8080/value?delay=0' \\n  -H 'Accept: application/json' \\n  -H 'Authorization: Bearer sk_secret_123'`
		);
	await dialog.getByRole('button', { name: t('curl.apply') }).click();

	await expect(dialog.getByText(t('curl.applied'))).toBeVisible();
	await expect(dialog.getByText(t('curl.credentials', { names: 'Authorization' }))).toBeVisible();
	await expect(
		dialog.getByPlaceholder('https://api.example.com/v1/data', { exact: true })
	).toHaveValue('http://203.0.113.10:8080/value');
	// Query string rows, then header rows (Authorization as a credential to pick).
	const keys = dialog.getByPlaceholder(t('postit_templates.key'), { exact: true });
	await expect
		.poll(() => keys.evaluateAll((inputs) => inputs.map((i) => (i as HTMLInputElement).value)))
		.toEqual(['delay', 'Accept', 'Authorization']);
	expect(await page.content()).not.toContain('sk_secret_123');

	await dialog.getByRole('button', { name: t('card_preview.try') }).click();
	await expect(dialog.getByText(t('card_preview.answered'))).toBeVisible();
});

test('says when what was pasted is not a cURL command', async ({ page }) => {
	const ana = await account('ana');
	const board = await boards.create(ana, 'From docs');
	await signIn(page, ana);
	await page.goto(`/boards/${board.id}`);

	await page.getByRole('button', { name: t('nodes.title') }).click();
	await page
		.locator('aside')
		.getByText(t('one_off.sidebar'))
		.dragTo(page.locator('.svelte-flow__pane'), { targetPosition: { x: 300, y: 200 } });
	const dialog = page.getByRole('dialog');

	await dialog.getByText(t('curl.toggle')).click();
	await dialog.getByRole('textbox', { name: t('curl.toggle') }).fill('wget https://example.com');
	await dialog.getByRole('button', { name: t('curl.apply') }).click();
	await expect(dialog.getByRole('alert')).toHaveText(t('curl.not_curl'));
});
