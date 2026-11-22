import { expect, test, type Page } from '@playwright/test';

import { account, boards, cards, signIn, t } from './support/stack';

const node = (page: Page, text: string) => page.locator('.svelte-flow__node', { hasText: text });

// E2E-07 (RF-15, RNF-01): two people on one board. A change the other makes
// reaches this browser without a reload: a moved card and a new one. The
// time a move takes to show is recorded on the test.
test('shows what another member does, live', async ({ browser }, info) => {
	const owner = await account('owner', 'max');
	const ana = await account('ana');
	const board = await boards.create(owner, 'Shared');
	await boards.share(owner, board.id, ana.email, 'editor');
	const card = await cards.note(owner, board.id, 'Moving note');
	await cards.move(owner, card.id, 100, 100);

	const context = await browser.newContext();
	const page = await context.newPage();
	await signIn(page, ana);
	await page.goto(`/boards/${board.id}`);
	await expect(node(page, 'Moving note')).toBeVisible();

	const sent = Date.now();
	await cards.move(owner, card.id, 420, 260);
	await expect(node(page, 'Moving note')).toHaveAttribute('style', /translate\(420px, 260px\)/);
	const latency = Date.now() - sent;
	info.annotations.push({ type: 'propagation (ms)', description: String(latency) });
	console.log(`a move reached the other member in ${latency} ms`);

	await cards.note(owner, board.id, 'Arrived live');
	await expect(node(page, 'Arrived live')).toBeVisible();
	await context.close();
});

// E2E-08 (RF-13): someone who may only read sees the board but none of the
// tools to change it.
test('keeps a viewer to reading', async ({ page }) => {
	const owner = await account('owner', 'max');
	const vic = await account('vic');
	const board = await boards.create(owner, 'Read only');
	await boards.share(owner, board.id, vic.email, 'viewer');
	await cards.note(owner, board.id, 'Look, do not touch');

	await signIn(page, vic);
	await page.goto(`/boards/${board.id}`);

	await expect(node(page, 'Look, do not touch')).toBeVisible();
	await expect(page.getByRole('button', { name: t('nodes.title') })).toHaveCount(0);
	await expect(page.getByRole('button', { name: t('secrets.title') })).toHaveCount(0);
	await expect(node(page, 'Look, do not touch')).not.toHaveClass(/draggable/);
});
