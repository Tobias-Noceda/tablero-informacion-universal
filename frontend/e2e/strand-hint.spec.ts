import { expect, test, type Page } from '@playwright/test';

import { account, boards, cards, signIn, t } from './support/stack';

const node = (page: Page, text: string) => page.locator('.svelte-flow__node', { hasText: text });

// Drags from just inside the left edge of one card onto the other.
async function drawStrand(page: Page, from: string, to: string) {
	const a = await node(page, from).boundingBox();
	const b = await node(page, to).boundingBox();
	if (!a || !b) throw new Error('cards not on screen');
	await page.mouse.move(a.x + 3, a.y + a.height / 2);
	await page.mouse.down();
	await page.mouse.move(b.x + b.width / 2, b.y + b.height / 2, { steps: 10 });
	await page.mouse.up();
}

async function boardWithTwoCards(name: string) {
	const ana = await account(name);
	const board = await boards.create(ana, 'Links');
	const first = await cards.note(ana, board.id, 'First');
	const second = await cards.note(ana, board.id, 'Second');
	await cards.move(ana, first.id, 0, 0);
	await cards.move(ana, second.id, 400, 0);
	return { ana, board };
}

// U9: the hint shows on a board with two cards and no strand, and goes away
// for good once the first strand is drawn.
test('shows how to draw the first strand until one is drawn', async ({ page }) => {
	const { ana, board } = await boardWithTwoCards('linker');
	await signIn(page, ana);
	await page.goto(`/boards/${board.id}`);

	const hint = page.getByText(t('strand_hint.text'));
	await expect(hint).toBeVisible();

	await drawStrand(page, 'First', 'Second');
	await expect(page.locator('.svelte-flow__edge')).toHaveCount(1);
	await expect(hint).toBeHidden();

	await page.reload();
	await expect(node(page, 'First')).toBeVisible();
	await expect(hint).toBeHidden();
});

test('stays dismissed once the person says they got it', async ({ page }) => {
	const { ana, board } = await boardWithTwoCards('reader');
	await signIn(page, ana);
	await page.goto(`/boards/${board.id}`);

	await page.getByRole('button', { name: t('strand_hint.dismiss') }).click();
	await expect(page.getByText(t('strand_hint.text'))).toBeHidden();

	await page.reload();
	await expect(node(page, 'Second')).toBeVisible();
	await expect(page.getByText(t('strand_hint.text'))).toBeHidden();
});
