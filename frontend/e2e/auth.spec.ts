import { expect, test } from '@playwright/test';
import { randomUUID } from 'node:crypto';

import { account, clientAddress, PASSWORD, signIn, t, verificationToken } from './support/stack';

// E2E-01 (RF-01): sign up with the form, confirm the address with the mailed
// link and land signed in.
test('signs up, verifies the address and lands signed in', async ({ page }) => {
	await page.setExtraHTTPHeaders({ 'X-Forwarded-For': clientAddress() });
	const email = `e2e-form-${randomUUID().slice(0, 8)}@it.test`;

	await page.goto('/register');
	await page.locator('input[name="name"]').fill('Form User');
	await page.locator('input[name="email"]').fill(email);
	await page.locator('input[name="password"]').fill(PASSWORD);
	await page.getByRole('button', { name: t('auth.register') }).click();
	await expect(page.getByRole('heading', { name: t('auth.check_inbox_title') })).toBeVisible();

	await page.goto(`/verify?token=${await verificationToken(email)}`);

	await expect(page).toHaveURL(/\/$/);
	await expect(page.getByRole('heading', { name: t('landing.title_personal') })).toBeVisible();
});

// E2E-02 (RF-03): signing out ends the session; the app sends a protected
// page back to the sign-in form.
test('signs out and keeps protected pages behind the sign-in', async ({ page }) => {
	const ana = await account('ana');
	await signIn(page, ana);
	await page.goto('/');
	await expect(page.getByRole('heading', { name: t('landing.title_personal') })).toBeVisible();

	await page.getByRole('button', { name: t('header.menu') }).click();
	await page.getByRole('menuitem', { name: t('user_info.logout') }).click();
	await expect(page).toHaveURL(/\/login/);

	await page.goto('/profile');
	await expect(page).toHaveURL(/\/login\?next=/);
});
