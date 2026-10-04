<script lang="ts">
	import { goto, invalidate } from '$app/navigation';
	import { resolve } from '$app/paths';
	import Button from '$components/Button/Button.svelte';
	import Input from '$components/Input/Input.svelte';
	import { m } from '$lib/paraglide/messages';
	import { session } from '$modules/session.svelte';
	import * as orgsApi from '$services/orgs';
	import { refreshBoards } from '$stores/boards';
	import { orgs } from '$stores/org.svelte';
	import type { OrgMemberSummary, OrgRole } from '$types/api';
	import type { PageProps } from './$types';

	let { data }: PageProps = $props();

	const org = $derived(data.org);
	const isAdmin = $derived(org.role === 'admin');
	const roles: OrgRole[] = ['admin', 'member'];

	let name = $derived(org.name);
	let email = $state('');
	let newRole = $state<OrgRole>('member');
	let error = $state('');

	const errors: Record<string, () => string> = {
		user_not_found: m['orgs.error_user_not_found'],
		invalid_role: m['orgs.error_invalid_role'],
		invalid_name: m['orgs.error_invalid_name'],
		last_admin: m['orgs.error_last_admin'],
		org_not_empty: m['orgs.error_org_not_empty'],
		rate_limited: m['orgs.error_rate_limited']
	};

	function describe(err: unknown) {
		const message = err instanceof orgsApi.OrgError ? errors[err.code] : undefined;
		return (message ?? m['orgs.error_unknown'])();
	}

	// Runs a change, then reloads the organization whether it worked or not, so
	// the page never shows a role the server refused.
	async function attempt(action: () => Promise<unknown>) {
		error = '';
		try {
			await action();
			return true;
		} catch (err) {
			error = describe(err);
			return false;
		} finally {
			await invalidate('app:org');
		}
	}

	async function rename(event: SubmitEvent) {
		event.preventDefault();
		if (await attempt(() => orgsApi.rename(org.id, name.trim()))) await orgs.refresh();
	}

	async function add(event: SubmitEvent) {
		event.preventDefault();
		if (await attempt(() => orgsApi.setMember(org.id, email.trim(), newRole))) email = '';
	}

	// A refused change leaves the role as it was, so the select goes back to it
	// (the reloaded data is the same and would not touch the element).
	async function change(member: OrgMemberSummary, select: HTMLSelectElement) {
		const to = select.value as OrgRole;
		if (!(await attempt(() => orgsApi.setMember(org.id, member.user.email, to)))) {
			select.value = member.role;
		}
	}

	function remove(member: OrgMemberSummary) {
		return attempt(() => orgsApi.removeMember(org.id, member.user.id));
	}

	// Leaving or deleting takes the organization's boards out of the sidebar.
	async function gone() {
		await Promise.all([orgs.refresh(), refreshBoards()]);
		await goto(resolve('/orgs'));
	}

	async function leave() {
		error = '';
		try {
			await orgsApi.removeMember(org.id, session.userId);
			await gone();
		} catch (err) {
			error = describe(err);
		}
	}

	async function del() {
		error = '';
		try {
			await orgsApi.del(org.id);
			await gone();
		} catch (err) {
			error = describe(err);
		}
	}
</script>

<main class="mx-auto flex w-full max-w-lg flex-col gap-6 p-8">
	<a href={resolve('/orgs')} class="text-sm opacity-70 hover:underline">← {m['orgs.back']()}</a>

	{#if isAdmin}
		<form class="flex items-end gap-2" onsubmit={rename}>
			<div class="flex-1">
				<Input id="org-name" label={m['orgs.name']()} bind:value={name} required />
			</div>
			<Button variant="primary" type="submit" disabled={!name.trim() || name.trim() === org.name}>
				{m['orgs.save']()}
			</Button>
		</form>
	{:else}
		<h1 class="text-2xl font-semibold">{org.name}</h1>
	{/if}

	{#if error}
		<p role="alert" class="text-sm text-destructive">{error}</p>
	{/if}

	<section class="flex flex-col gap-2">
		<h2 class="text-sm font-medium">{m['orgs.members']()}</h2>
		<ul class="flex flex-col gap-2">
			{#each org.members as member (member.user.id)}
				<li
					class="flex items-center justify-between gap-2 rounded-md border border-main-border px-3 py-2"
				>
					<div class="flex min-w-0 flex-col">
						<span class="truncate text-sm">
							{member.user.name || member.user.email || member.user.id}
							{#if member.user.id === session.userId}
								<span class="opacity-60">({m['orgs.you']()})</span>
							{/if}
						</span>
						<span class="truncate text-xs opacity-60">{member.user.email}</span>
					</div>
					<div class="flex items-center gap-2">
						{#if isAdmin}
							<select
								class="rounded-md border border-main-border bg-background px-2 py-1 text-sm"
								aria-label={m['orgs.members']()}
								value={member.role}
								onchange={(event) => change(member, event.currentTarget)}
							>
								{#each roles as option (option)}
									<option value={option}>{m[`orgs.role_${option}`]()}</option>
								{/each}
							</select>
							{#if member.user.id !== session.userId}
								<Button variant="destructive" onclick={() => remove(member)}>
									{m['orgs.remove']()}
								</Button>
							{/if}
						{:else}
							<span class="text-sm opacity-70">{m[`orgs.role_${member.role}`]()}</span>
						{/if}
					</div>
				</li>
			{/each}
		</ul>
	</section>

	{#if isAdmin}
		<form class="flex flex-col gap-2 border-t border-main-border pt-4" onsubmit={add}>
			<Input
				id="member-email"
				label={m['orgs.email']()}
				type="email"
				autocomplete="off"
				bind:value={email}
				required
			/>
			<div class="flex gap-2">
				<select
					class="rounded-md border border-main-border bg-background px-2 py-1 text-sm"
					aria-label={m['orgs.members']()}
					bind:value={newRole}
				>
					{#each roles as option (option)}
						<option value={option}>{m[`orgs.role_${option}`]()}</option>
					{/each}
				</select>
				<Button variant="primary" type="submit" disabled={email.trim() === ''}>
					{m['orgs.add']()}
				</Button>
			</div>
		</form>
	{/if}

	<section class="flex flex-col gap-2 border-t border-main-border pt-4">
		<Button variant="destructive" onclick={leave}>{m['orgs.leave']()}</Button>
		{#if isAdmin}
			<Button variant="destructive" onclick={del}>{m['orgs.delete']()}</Button>
			<p class="text-xs opacity-70">{m['orgs.delete_hint']()}</p>
		{/if}
	</section>
</main>
