<script lang="ts">
	import type { Node } from '@xyflow/svelte';

	import Button from '$components/Button/Button.svelte';
	import Dropdown from '$components/Dropdown/Dropdown.svelte';
	import Icon from '$components/Icon/Icon.svelte';
	import Input from '$components/Input/Input.svelte';
	import Modal from '$components/Modal/Modal.svelte';
	import ResponseTree from '$components/ResponseTree/ResponseTree.svelte';
	import type { PostItData } from '$layouts/BoardContent/NodeDetail.svelte';
	import { failureMessage, failureOf } from '$lib/cards/failure';
	import { CurlError, fillFromCurl, parseCurl } from '$lib/well-knowns/curl';
	import { jqPath, lastIndex, outputName, type JsonPath } from '$lib/cards/preview';
	import { m } from '$lib/paraglide/messages';
	import { refKey } from '$lib/secrets/binding';
	import { groupByOrigin } from '$lib/secrets/origin';
	import { renderTitle } from '$lib/title';
	import { cn } from '$lib/utils';
	import {
		bodyCredentialNames,
		emptyOneOff,
		insertCredential,
		MAX_REQUEST_BODY,
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
	import type { OneOffMethod, PostItSettings, SecretMeta, UUID } from '$types/api';

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

	// The body editor, to insert a credential where the cursor is.
	let bodyField = $state<HTMLTextAreaElement>();
	const bodyBytes = $derived(new TextEncoder().encode(draft.body.trim()).length);
	// Said while typing: a body that is not JSON is the usual slip.
	const bodyIsJson = $derived(!draft.body.trim() || bodyCredentialNames(draft.body) !== null);

	function pickBodyCredential(key: string) {
		const secret = usableSecrets.find((s) => refKey(s) === key);
		if (!secret) return;
		const cursor = bodyField?.selectionStart ?? draft.body.length;
		const inserted = insertCredential(draft.body, cursor, secret.name);
		draft.body = inserted.body;
		draft.bodySecrets[secret.name] = key;
		requestAnimationFrame(() => {
			bodyField?.focus();
			bodyField?.setSelectionRange(inserted.cursor, inserted.cursor);
		});
	}

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
			case 'body_json':
				return m['one_off.err_body_json']();
			case 'body_size':
				return m['one_off.err_body_size']({ max: String(MAX_REQUEST_BODY / 1024) });
			case 'body_credential':
				return m['one_off.err_body_credential']({ name: problem.value });
		}
	}

	// A cURL command fills in the request.
	let curlOpen = $state(false);
	let curlText = $state('');
	let curlNotes = $state<string[]>([]);
	let curlError = $state('');

	function applyCurl() {
		curlError = '';
		curlNotes = [];
		try {
			const parsed = parseCurl(curlText);
			draft = fillFromCurl(draft, parsed);
			tried = null;
			curlNotes = [
				m['curl.applied'](),
				...(parsed.credentials.length
					? [m['curl.credentials']({ names: parsed.credentials.join(', ') })]
					: []),
				...parsed.dropped.map((code) =>
					code === 'method'
						? m['curl.dropped_method']()
						: code === 'body_not_json'
							? m['curl.dropped_body']()
							: m['curl.dropped_flags']()
				)
			];
			curlText = '';
			curlOpen = false;
		} catch (e) {
			curlError = e instanceof CurlError ? m[`curl.${e.code}`]() : m['curl.not_curl']();
		}
	}

	// "Try request": the source's whole answer, to pick outputs from.
	let trying = $state(false);
	let tried = $state<{ response: unknown } | null>(null);
	let tryError = $state('');
	let picked = $state('');
	const canTry = $derived(draft.resource.trim() !== '' && draft.response === 'json');

	async function tryRequest() {
		if (!canTry || trying) return;
		trying = true;
		tryError = '';
		picked = '';
		const { body } = oneOffBody(draft, usableSecrets, boardId);
		try {
			const answer = await postItsApi.preview(boardId, {
				resource: body.resource,
				request: body.request,
				response: 'json',
				bindings: body.bindings
			});
			tried = { response: answer.response ?? null };
		} catch (e) {
			tried = null;
			tryError = tryFailure(e);
		} finally {
			trying = false;
		}
	}

	function tryFailure(e: unknown) {
		if (e instanceof postItsApi.CardError) {
			if (e.status === 429) return m['card_preview.too_many']();
			if (e.status === 400) return m['card_preview.invalid']({ detail: e.message });
			if (e.code) return failureMessage(failureOf(e));
		}
		return m['card_failure.unknown']();
	}

	// A value clicked in the answer becomes an output (in the first empty row).
	function pick(path: JsonPath, every: boolean) {
		const index = every ? lastIndex(path) : undefined;
		const name = outputName(path, outputKeys(draft), index !== undefined);
		const output = { key: name, query: jqPath(path, index) };
		const blank = draft.outputs.findIndex((o) => !o.key.trim() && !o.query.trim());
		if (blank >= 0) draft.outputs[blank] = output;
		else draft.outputs.push(output);
		picked = m['card_preview.picked']({ name, variable: `{{${name}}}` });
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

	<details class="rounded-md border border-main-border p-2 text-sm" bind:open={curlOpen}>
		<summary class="cursor-pointer font-semibold">{m['curl.toggle']()}</summary>
		<div class="mt-2 flex flex-col gap-2">
			<p class="text-xs opacity-70">{m['curl.hint']()}</p>
			<textarea
				bind:value={curlText}
				rows="4"
				spellcheck="false"
				aria-label={m['curl.toggle']()}
				class={cn(fieldClass, 'resize-y font-mono text-xs')}
				placeholder={m['curl.placeholder']()}
			></textarea>
			<Button
				variant="secondary"
				class="self-start px-3 py-1 text-sm"
				disabled={!curlText.trim()}
				onclick={applyCurl}
			>
				{m['curl.apply']()}
			</Button>
			{#if curlError}
				<p class="text-xs text-destructive" role="alert">{curlError}</p>
			{/if}
		</div>
	</details>
	{#if curlNotes.length > 0}
		<ul class="flex flex-col gap-0.5 text-xs" aria-live="polite">
			{#each curlNotes as note, i (i)}
				<li class={i === 0 ? 'font-semibold' : ''}>{note}</li>
			{/each}
		</ul>
	{/if}

	<div class="flex flex-col gap-3 border-t border-main-border pt-3">
		<div class="flex items-end gap-2">
			<div class="flex flex-col gap-1 text-sm">
				<span class="font-semibold">{m['one_off.method']()}</span>
				<div class="flex rounded-md border border-border p-0.5" role="radiogroup" aria-label={m['one_off.method']()}>
					{#each ['GET', 'POST'] as const as method (method)}
						<button
							type="button"
							role="radio"
							aria-checked={draft.method === method}
							class={cn(
								'cursor-pointer rounded px-3 py-1 font-mono text-xs transition-colors',
								draft.method === method ? 'bg-main text-main-text' : 'hover:bg-background-hover'
							)}
							onclick={() => (draft.method = method as OneOffMethod)}
						>
							{method}
						</button>
					{/each}
				</div>
			</div>
			<div class="min-w-0 flex-1">
				<Input
					label={m['postit_templates.resource']()}
					labelClass="font-semibold"
					placeholder="https://api.example.com/v1/data"
					bind:value={draft.resource}
					required
				/>
			</div>
		</div>

		{#if draft.method === 'POST'}
			<fieldset class="flex flex-col gap-1">
				<legend class="text-sm font-semibold">{m['one_off.body']()}</legend>
				<p class="text-xs opacity-70">{m['one_off.body_hint']({ example: '"$NAME"' })}</p>
				<textarea
					bind:this={bodyField}
					bind:value={draft.body}
					rows="6"
					spellcheck="false"
					class={cn(fieldClass, 'resize-y font-mono text-xs', !bodyIsJson && 'border-destructive!')}
					placeholder={'{"origin": "EZE", "destination": "NRT"}'}
				></textarea>
				<div class="flex items-center justify-between gap-2 text-xs">
					<span class={cn('opacity-70', !bodyIsJson && 'text-destructive opacity-100')}>
						{bodyIsJson ? m['one_off.body_size']({ used: String(bodyBytes), max: String(MAX_REQUEST_BODY) }) : m['one_off.err_body_json']()}
					</span>
					{#if pickable.length > 0}
						<div class="w-56">
							<Dropdown
								placeholder={m['one_off.body_insert_credential']()}
								options={credentialOptions}
								value=""
								onchange={pickBodyCredential}
								class="w-full!"
								activatorClass={(isOpen) =>
									cn(
										fieldClass,
										'w-full text-xs hover:border-sidebar',
										isOpen ? 'outline-none ring-2 ring-main-border border-none' : 'focus:ring-0!'
									)}
								drawerClass="rounded-md bg-background!"
								sectionClass="font-semibold text-foreground!"
								itemClass={() => 'bg-background! rounded-md hover:bg-main-hover!'}
							/>
						</div>
					{/if}
				</div>
			</fieldset>
		{/if}
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

		<section class="flex flex-col gap-2 rounded-md border border-main-border p-2" aria-live="polite">
			<div class="flex flex-wrap items-center gap-2">
				<Button
					variant="secondary"
					class="px-3 py-1 text-sm"
					onclick={tryRequest}
					disabled={!canTry || trying}
				>
					{trying ? m['card_preview.trying']() : m['card_preview.try']()}
				</Button>
				<p class="min-w-0 flex-1 text-xs opacity-70">
					{!draft.resource.trim()
						? m['card_preview.try_needs_url']()
						: draft.response !== 'json'
							? m['card_preview.try_needs_json']()
							: m['card_preview.try_hint']()}
				</p>
			</div>
			{#if tryError}
				<p class="text-sm text-destructive" role="alert">{tryError}</p>
			{/if}
			{#if tried}
				{#if tried.response === null}
					<p class="text-xs opacity-70">{m['card_preview.empty']()}</p>
				{:else}
					<p class="text-xs font-semibold">{m['card_preview.answered']()}</p>
					<div class={cn('max-h-64 overflow-auto rounded-md border border-border p-2', scrollClass)}>
						<ResponseTree value={tried.response} onpick={pick} />
					</div>
				{/if}
				{#if picked}
					<p class="text-xs font-semibold">{picked}</p>
				{/if}
			{/if}
		</section>

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
