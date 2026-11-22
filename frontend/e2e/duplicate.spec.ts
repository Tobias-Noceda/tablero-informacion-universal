import { expect, test } from '@playwright/test';

import { account, BASE_URL, boards, cards, signIn, t } from './support/stack';

// U7: a card is duplicated from its toolbar, next to it, with its look, and
// the copy stays after a reload.
test('duplicates a card from its toolbar', async ({ page }) => {
	const ana = await account('ana');
	const board = await boards.create(ana, 'Copies');
	const card = await cards.note(ana, board.id, 'Original note');
	await fetch(`${BASE_URL}/api/v1/post-its/${card.id}/style`, {
		method: 'PATCH',
		headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${ana.token}` },
		body: JSON.stringify({ color: 'green' })
	});

	await signIn(page, ana);
	await page.goto(`/boards/${board.id}`);
	const notes = page.locator('.svelte-flow__node', { hasText: 'Original note' });
	await notes.first().click();
	await page.getByRole('button', { name: t('cards.duplicate') }).click();

	await expect(notes).toHaveCount(2);
	const { postits } = await boards.get(ana, board.id);
	expect(postits).toHaveLength(2);
	const [original, copy] = postits;
	expect(copy.title.text).toBe('Original note');
	expect(copy.color).toBe('green');
	expect(copy.position.x).toBeGreaterThan(original.position.x);

	await page.reload();
	await expect(notes).toHaveCount(2);
});
