<script lang="ts">
	import type { Node } from '@xyflow/svelte';

	import Button from '$components/Button/Button.svelte';
	import Dropdown from '$components/Dropdown/Dropdown.svelte';
	import Icon from '$components/Icon/Icon.svelte';
	import Input from '$components/Input/Input.svelte';
	import Modal from '$components/Modal/Modal.svelte';
	import type { PostItData } from '$layouts/BoardContent/NodeDetail.svelte';
	import { m } from '$lib/paraglide/messages';
	import { refKey } from '$lib/secrets/binding';
	import { groupByOrigin } from '$lib/secrets/origin';
	import { renderTitle } from '$lib/title';
	import { cn } from '$lib/utils';
	import {
		emptyOneOff,
		ONE_OFF,
		oneOffBody,
		oneOffFrom,
		oneOffProblems,
		outputKeys,
		type OneOffProblem,
		type ValueRow
	} from '$lib/well-knowns/one-off';
	import { session } from '$modules/session.svelte';
	import * as postItsApi from '$services/post-it';
	import type { PostItSettings, SecretMeta, UUID } from '$types/api';

	interface Props {
		boardId: UUID;
		usableSecrets: SecretMeta[];
		/** A card being dropped on the board, where it was dropped. */
		creating?: { id: string; position: { x: number; y: number } } | null;
		/** A card on the board, with what it is stored with. */
		editing?: { node: PostItData; settings: PostItSettings } | null;
		onclose: () => void;
		onPersistNode: (action: 'create' | 'update', node: Node) => void;
	}

	let { boardId, usableSecrets, creating, editing, onclose, onPersistNode }: Props = $props();

	const scrollClass = cn(
		'scrollbar-thin scrollbar-thumb-main scrollbar-track-transparent',
		'[&::-webkit-scrollbar]:w-2',
		'[&::-webkit-scrollbar-track]:bg-transparent',
		'[&::-webkit-scrollbar-thumb]:bg-main',
		'[&::-webkit-scrollbar-thumb]:rounded-full',
		'[&::-webkit-scrollbar-thumb]:hover:bg-main-hover'
	);

	const fieldClass = cn(
		'min-w-0 bg-transparent px-2 py-1 rounded-md',
		'text-sm text-foreground',
		'border-border hover:border-sidebar focus:border-main focus:ring-1 focus:ring-main',
		'focus:outline-none focus:ring-2 focus:ring-main-border'
	);

	// The panel is mounted per card, so what it starts from is fixed.
	// svelte-ignore state_referenced_locally
	const original = editing
		? oneOffFrom(editing.settings, editing.node.title?.text ?? '', boardId)
		: emptyOneOff();
	let draft = $state(structuredClone(original));
	// svelte-ignore state_referenced_locally
	const originalBody = JSON.stringify(oneOffBody(original, usableSecrets, boardId).body);

	let saving = $state(false);
	// Shown once the user tried to save, so an empty form is not all red.
	let touched = $state(false);
	let error = $state('');

	const found = $derived(oneOffProblems(draft, usableSecrets));
	const keys = $derived(outputKeys(draft));
	const sample = $derived(Object.fromEntries(keys.map((key) => [key, `{{${key}}}`])));
	const pickable = $derived(groupByOrigin(usableSecrets, boardId, session.userId));
	const credentialOptions = $derived(
		Object.fromEntries(
			pickable.map((group) => [
				group.origin,
				{
					label: m[`secrets.origin_${group.origin}`]() as string,
					options: group.secrets.map((secret) => ({
						label: `${secret.name} (${secret.provider ?? secret.kind})`,
						value: refKey(secret)
					}))
				}
			])
		)
	);

	function describe(problem: OneOffProblem) {
		switch (problem.code) {
			case 'resource':
				return m['postit_templates.err_resource']();
			case 'outputs':
				return m['postit_templates.err_outputs']();
			case 'output_key':
				return m['postit_templates.err_output_key']();
			case 'output_query':
				return m['postit_templates.err_output_query']();
			case 'row_key':
				return m['one_off.err_row_key']();
			case 'credential':
				return m['one_off.err_credential']();
			case 'duplicate':
				return m['postit_templates.err_duplicate']({ key: problem.value });
			case 'title':
				return m['postit_templates.err_title']({ syntax: '{{name}}' });
		}
	}

	function insertVariable(key: string) {
		const separator = draft.title && !draft.title.endsWith(' ') ? ' ' : '';
		draft.title = `${draft.title}${separator}{{${key}}}`;
	}

	async function save() {
		touched = true;
		if (found.length > 0 || saving) return;
		saving = true;
		error = '';
		const { title, body } = oneOffBody(draft, usableSecrets, boardId);
		try {
			if (creating) {
				const created = await postItsApi.create_one_off(boardId, title, body);
				await postItsApi.move(created.id, creating.position.x, creating.position.y);
				onPersistNode('create', {
					id: created.id,
					type: ONE_OFF,
					position: creating.position,
					data: { title }
				} as Node);
			} else if (editing) {
				if (JSON.stringify(body) !== originalBody) {
					await postItsApi.update_one_off(editing.node.id, body);
				}
				if (title.text !== (editing.node.title?.text ?? '')) {
					await postItsApi.update_well_known_title(editing.node.id, title);
				}
				onPersistNode('update', { ...editing.node, data: { title } } as Node);
			}
		} catch (e) {
			error = m['one_off.error']({
				detail:
					e instanceof postItsApi.PostItError ? e.message : m['postit_templates.error_generic']()
			});
		} finally {
			saving = false;
		}
	}
</script>

{#snippet removeButton(onclick: () => void)}
	<button
		type="button"
		class="shrink-0 cursor-pointer rounded-md p-1 hover:bg-background-hover"
		aria-label={m['postit_templates.remove']()}
		title={m['postit_templates.remove']()}
		{onclick}
	>
		<Icon name="trash" class="h-4 w-4 mx-1 -mt-0.5 text-destructive!" />
	</button>
{/snippet}

{#snippet rowsEditor(label: string, hint: string, rows: ValueRow[])}
	<fieldset class="flex flex-col gap-1">
		<legend class="text-sm font-semibold">{label}</legend>
		<p class="text-xs opacity-70">{hint}</p>
		{#each rows as row, i (i)}
			<div class="flex items-center gap-1">
				<input
					class={fieldClass + ' flex-1'}
					placeholder={m['postit_templates.key']()}
					bind:value={row.key}
				/>
				{#if row.credential}
					{#if pickable.length === 0}
						<span class="flex-1 text-xs text-destructive">{m['secrets.no_credentials']()}</span>
					{:else}
						<div class="flex min-w-0 flex-1">
							<Dropdown
								placeholder={m['secrets.pick_credential']()}
								options={credentialOptions}
								class="w-full!"
								activatorClass={(isOpen) =>
									cn(
										fieldClass,
										'w-full hover:border-sidebar',
										isOpen ? 'outline-none ring-2 ring-main-border border-none' : 'focus:ring-0!'
									)}
								drawerClass="rounded-md bg-background!"
								sectionClass="font-semibold text-foreground!"
								itemClass={() => 'bg-background! rounded-md hover:bg-main-hover!'}
								bind:value={row.secret}
							/>
						</div>
					{/if}
				{:else}
					<input
						class={fieldClass + ' flex-1'}
						placeholder={m['postit_templates.value']()}
						bind:value={row.value}
					/>
				{/if}
				<button
					type="button"
					class={cn(fieldClass, 'shrink-0 cursor-pointer text-xs hover:bg-background-hover')}
					title={row.credential ? m['one_off.toggle_value']() : m['one_off.toggle_credential']()}
					onclick={() => (row.credential = !row.credential)}
				>
					{row.credential ? m['one_off.credential_mode']() : m['one_off.value_mode']()}
				</button>
				{@render removeButton(() => rows.splice(i, 1))}
			</div>
		{/each}
		<Button
			variant="secondary"
			class="self-start px-2 py-0.5 text-xs"
			onclick={() => rows.push({ key: '', value: '', credential: false, secret: '' })}
		>
			{m['postit_templates.add_row']()}
		</Button>
	</fieldset>
{/snippet}

<Modal
	{onclose}
	onaccept={save}
	acceptText={creating
		? m['nodes.modify_modal.create.action']()
		: m['nodes.modify_modal.edit.action']()}
	acceptDisabled={saving || (touched && found.length > 0)}
	dialogClass={cn('max-w-2xl max-h-[90dvh] overflow-y-auto', scrollClass)}
>
	<h2 class="text-lg font-semibold">
		{creating ? m['one_off.create_title']() : m['one_off.edit_title']()}
	</h2>
	<p class="text-xs opacity-70">{m['one_off.hint']()}</p>

	<div class="flex flex-col gap-3 border-t border-main-border pt-3">
		<Input
			label={m['postit_templates.resource']()}
			labelClass="font-semibold"
			placeholder="https://api.example.com/v1/data"
			bind:value={draft.resource}
			required
		/>
		<div class="flex gap-2">
			<label class="flex flex-1 flex-col gap-1 text-sm">
				<span class="font-semibold">{m['postit_templates.response']()}</span>
				<Dropdown
					bind:value={draft.response}
					options={['json', 'html'].map((response) => ({
						value: response,
						label: response.toUpperCase()
					}))}
					class="w-full!"
					activatorClass={(isOpen) =>
						cn(
							fieldClass,
							'w-full text-base hover:border-sidebar',
							isOpen ? 'outline-none ring-2 ring-main-border border-none' : 'focus:ring-0!'
						)}
					drawerClass={cn(fieldClass, 'bg-background mt-1 px-1')}
					itemClass={(isSelected) =>
						cn(
							'hover:bg-background-hover text-foreground text-sm rounded-md',
							isSelected ? 'bg-background-hover' : 'bg-transparent'
						)}
				/>
			</label>
			<label class="flex flex-1 flex-col gap-1 text-sm">
				<span class="font-semibold">{m['postit_templates.rate']()}</span>
				<input
					class={cn(fieldClass, 'px-2 py-1.5')}
					type="number"
					min="600"
					bind:value={draft.rate}
				/>
			</label>
		</div>
		<p class="text-xs opacity-70">{m['postit_templates.rate_hint']()}</p>

		{@render rowsEditor(
			m['postit_templates.queries'](),
			m['one_off.request_hint'](),
			draft.queries
		)}
		{@render rowsEditor(
			m['postit_templates.headers'](),
			m['one_off.request_hint'](),
			draft.headers
		)}

		<fieldset class="flex flex-col gap-1">
			<legend class="text-sm font-semibold">{m['postit_templates.outputs']()}</legend>
			<p class="text-xs opacity-70">
				{m['one_off.outputs_hint']({ example: '.data.temperature' })}
			</p>
			{#each draft.outputs as output, i (i)}
				<div class="flex flex-col gap-1 rounded-md border border-main-border p-2">
					<div class="flex items-center gap-1">
						<input
							class={fieldClass + ' w-28 font-mono'}
							placeholder={m['postit_templates.field_name']()}
							bind:value={output.key}
						/>
						<input
							class={fieldClass + ' flex-1 font-mono'}
							placeholder={m['postit_templates.query']()}
							bind:value={output.query}
						/>
						{@render removeButton(() => draft.outputs.splice(i, 1))}
					</div>
				</div>
			{/each}
			<Button
				variant="secondary"
				class="self-start px-2 py-0.5 text-xs"
				onclick={() => draft.outputs.push({ key: '', query: '' })}
			>
				{m['postit_templates.add_output']()}
			</Button>
		</fieldset>

		<Input
			label={m['card_title.label']()}
			labelClass="font-semibold"
			placeholder={m['card_title.placeholder']({ example: '{{value}}' })}
			bind:value={draft.title}
			maxlength={127}
			required
		/>
		{#if keys.length > 0}
			<div class="flex flex-wrap gap-1">
				{#each keys as key (key)}
					<button
						type="button"
						class="cursor-pointer rounded-md border border-main-border bg-background px-2 py-0.5 font-mono text-xs hover:bg-main-hover"
						onclick={() => insertVariable(key)}
					>
						{`{{${key}}}`}
					</button>
				{/each}
			</div>
			{#if draft.title.trim()}
				<span class="rounded-md bg-main-hover/30 px-2 py-1 text-sm font-bold">
					{renderTitle(draft.title, sample)}
				</span>
			{/if}
		{/if}

		{#if touched && found.length > 0}
			<ul class="flex flex-col gap-0.5 text-xs text-destructive">
				{#each found as problem, i (i)}
					<li>{describe(problem)}</li>
				{/each}
			</ul>
		{/if}
		{#if error}
			<p class="text-sm text-destructive">{error}</p>
		{/if}
	</div>
</Modal>
