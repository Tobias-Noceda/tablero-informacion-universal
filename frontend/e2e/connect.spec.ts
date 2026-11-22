import { expect, test, type Page } from '@playwright/test';

import { account, boards, cards, signIn } from './support/stack';

const node = (page: Page, text: string) => page.locator('.svelte-flow__node', { hasText: text });

// A strand drawn from a card's edge can be dropped anywhere on the other
// card: not only on its edge, nor only near its centre.
test('drops a strand anywhere on the other card', async ({ page }) => {
	const ana = await account('ana');
	const board = await boards.create(ana, 'Links');
	const from = await cards.note(ana, board.id, 'From here');
	const to = await cards.note(ana, board.id, 'To there');
	await cards.move(ana, from.id, 0, 0);
	await cards.move(ana, to.id, 400, 0);
	await signIn(page, ana);
	await page.goto(`/boards/${board.id}`);

	const source = (await node(page, 'From here').boundingBox())!;
	const target = (await node(page, 'To there').boundingBox())!;
	await page.mouse.move(source.x + 3, source.y + source.height / 2);
	await page.mouse.down();
	// Inside B, away from its edge strip and from its centre.
	await page.mouse.move(target.x + target.width * 0.3, target.y + target.height * 0.3, {
		steps: 12
	});
	await page.mouse.up();

	await expect(page.locator('.svelte-flow__edge')).toHaveCount(1);
	expect((await boards.get(ana, board.id)).strands).toHaveLength(1);
});
