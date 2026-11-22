import { expect, test, type Page } from '@playwright/test';

import { account, boards, signIn, t } from './support/stack';

// The fake source of perf/docker-compose.perf.yaml (start the stack with it):
// the backend's SSRF guard lets 203.0.113.0/24 through.
const SOURCE = 'http://203.0.113.10:8080/value?delay=0';

async function openOneOff(page: Page) {
	await page.getByRole('button', { name: t('nodes.title') }).click();
	await page
		.locator('aside')
		.getByText(t('one_off.sidebar'))
		.dragTo(page.locator('.svelte-flow__pane'), { targetPosition: { x: 300, y: 200 } });
	return page.getByRole('dialog');
}

test('builds a card by trying the request and clicking the value', async ({ page }) => {
	const ana = await account('ana');
	const board = await boards.create(ana, 'Try it');
	await signIn(page, ana);
	await page.goto(`/boards/${board.id}`);

	const dialog = await openOneOff(page);
	const tryButton = dialog.getByRole('button', { name: t('card_preview.try') });
	await expect(tryButton).toBeDisabled();

	await dialog.getByPlaceholder('https://api.example.com/v1/data', { exact: true }).fill(SOURCE);
	await tryButton.click();
	await expect(dialog.getByText(t('card_preview.answered'))).toBeVisible();

	// {"value": <hits>, "served_at": <ms>}: pick the value.
	await dialog.getByText('value:').locator('xpath=following-sibling::button[1]').click();
	await expect(dialog.getByPlaceholder(t('postit_templates.field_name')).first()).toHaveValue(
		'value'
	);
	await expect(dialog.getByPlaceholder(t('postit_templates.query')).first()).toHaveValue('.value');

	await dialog.getByRole('button', { name: '{{value}}' }).click();
	await dialog.getByRole('button', { name: t('nodes.modify_modal.create.action') }).click();

	await expect(dialog).toBeHidden();
	await expect(page.locator('.svelte-flow__node').first()).toContainText(/\d+/);
});

test('says why a tried request failed', async ({ page }) => {
	const ana = await account('ana');
	const board = await boards.create(ana, 'Try it');
	await signIn(page, ana);
	await page.goto(`/boards/${board.id}`);

	const dialog = await openOneOff(page);
	await dialog
		.getByPlaceholder('https://api.example.com/v1/data', { exact: true })
		.fill('http://127.0.0.1:9/value');
	await dialog.getByRole('button', { name: t('card_preview.try') }).click();

	await expect(dialog.getByRole('alert')).toHaveText(t('card_failure.unreachable'));
	await expect(page.locator('.svelte-flow__node')).toHaveCount(0);
});
