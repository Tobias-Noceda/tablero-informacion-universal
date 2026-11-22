import { expect, test } from '@playwright/test';

import es from '../messages/es.json' with { type: 'json' };
import { t } from './support/stack';

// E2E-09 (RNF-08): a first visit is in English; the switcher turns the
// sign-in page into Spanish, and its links lead to the Spanish public
// pages (/acerca, /privacidad).
test('switches the language from the sign-in page', async ({ page }) => {
	await page.goto('/login');
	await expect(page.getByRole('button', { name: t('auth.login') }).first()).toBeVisible();

	await page.getByRole('button', { name: 'ES' }).click();
	await expect(page.getByRole('button', { name: es.auth.login }).first()).toBeVisible();

	await page.getByRole('link', { name: es.legal.privacy_policy }).click();
	await expect(page).toHaveURL(/\/privacidad$/);
	await expect(
		page.getByRole('heading', { name: 'Política de privacidad', level: 1 })
	).toBeVisible();
});

// The public pages are plain HTML for Google's review (no scripts): each
// language has its own address, and plain links switch between them.
test('serves each public page in both languages without scripts', async ({ browser }) => {
	const context = await browser.newContext({ javaScriptEnabled: false });
	const page = await context.newPage();

	await page.goto('/privacy');
	await expect(page.getByRole('heading', { name: 'Privacy Policy', level: 1 })).toBeVisible();

	await page.getByRole('link', { name: 'ES', exact: true }).click();
	await expect(page).toHaveURL(/\/privacidad$/);
	await expect(
		page.getByRole('heading', { name: 'Política de privacidad', level: 1 })
	).toBeVisible();
	await expect(page.locator('header').getByRole('link', { name: es.auth.login })).toBeVisible();

	await page.getByRole('link', { name: es.legal.home }).click();
	await expect(page).toHaveURL(/\/acerca$/);

	await page.getByRole('link', { name: 'EN', exact: true }).click();
	await expect(page).toHaveURL(/\/about$/);
	await expect(page.locator('header').getByRole('link', { name: t('auth.login') })).toBeVisible();
	await context.close();
});
