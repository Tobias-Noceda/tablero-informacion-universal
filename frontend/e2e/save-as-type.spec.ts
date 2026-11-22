import { expect, test } from '@playwright/test';

import { account, BASE_URL, boards, cards, signIn, t } from './support/stack';

// U4: a one-off card is saved as a card type from its detail drawer; the
// type's form opens filled in from the card, and once saved the type is
// offered in the cards sidebar.
test('saves a custom card as a card type', async ({ page }) => {
	const ana = await account('ana', 'pro');
	const board = await boards.create(ana, 'Reuse');
	await cards.custom(ana, board.id, 'Price: {{value}}', 'http://203.0.113.10:8080/value?delay=0');

	await signIn(page, ana);
	await page.goto(`/boards/${board.id}`);
	await page.locator('.svelte-flow__node', { hasText: 'Price' }).first().click();
	await page.getByRole('button', { name: t('node_detail.save_as_type') }).click();

	const form = page.getByRole('dialog');
	const name = form.getByLabel(t('postit_templates.name'), { exact: false }).first();
	await expect(name).toHaveValue('Price');
	await name.fill('Fake price');
	await form.getByRole('button', { name: t('postit_templates.save'), exact: true }).click();

	const res = await fetch(`${BASE_URL}/api/v1/users/${ana.id}/postit-templates`, {
		headers: { Authorization: `Bearer ${ana.token}` }
	});
	const saved = await res.json();
	expect(saved).toHaveLength(1);
	expect(saved[0]).toMatchObject({
		name: 'Fake price',
		resource: 'http://203.0.113.10:8080/value?delay=0',
		query: { value: '.value' }
	});

	await page.keyboard.press('Escape');
	await page.getByRole('button', { name: t('nodes.title') }).click();
	await expect(page.locator('aside').getByText('Fake price')).toBeVisible();
});
