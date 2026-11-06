<script lang="ts">
	import Button from '$components/Button/Button.svelte';
	import Icon from '$components/Icon/Icon.svelte';
	import Input from '$components/Input/Input.svelte';
	import { m } from '$lib/paraglide/messages';
	import { renderTitle } from '$lib/title';
	import { bodyOf, problems, type Draft, type Problem, type Row } from '$lib/well-knowns/draft';
	import type { CustomWellKnownBody } from '$types/api';

	interface Props {
		/** What is being written. The parent owns it, so it outlives the form
		 * (switching tabs keeps a half-written card type). */
		draft: Draft;
		onsave: (body: CustomWellKnownBody) => Promise<void>;
		oncancel: () => void;
	}

	let { draft = $bindable(), onsave, oncancel }: Props = $props();

	let saving = $state(false);
	// Shown once the user tried to save, so an empty form is not all red.
	let touched = $state(false);

	const found = $derived(problems(draft));
	const outputKeys = $derived(draft.outputs.map((o) => o.key.trim()).filter(Boolean));
	const sample = $derived(
		Object.fromEntries(
			draft.outputs.map((o) => [o.key.trim(), o.example.trim() || `{{${o.key.trim()}}}`])
		)
	);

	const fieldClass =
		'min-w-0 rounded-md border border-main-border bg-transparent px-2 py-1 text-sm focus:outline-none focus:ring-2 focus:ring-main-border';

	function describe(problem: Problem) {
		switch (problem.code) {
			case 'name':
				return m['custom_wk.err_name']();
			case 'resource':
				return m['custom_wk.err_resource']();
			case 'outputs':
				return m['custom_wk.err_outputs']();
			case 'output_key':
				return m['custom_wk.err_output_key']();
			case 'output_label':
				return m['custom_wk.err_output_label']();
			case 'output_query':
				return m['custom_wk.err_output_query']();
			case 'param_key':
				return m['custom_wk.err_param_key']({ key: problem.value || '…' });
			case 'param_label':
				return m['custom_wk.err_param_label']();
			case 'duplicate':
				return m['custom_wk.err_duplicate']({ key: problem.value });
			case 'undeclared':
				return m['custom_wk.err_undeclared']({ value: problem.value });
			case 'title':
				return m['custom_wk.err_title']({ syntax: '{{name}}' });
		}
	}

	function insertVariable(key: string) {
		const separator = draft.title && !draft.title.endsWith(' ') ? ' ' : '';
		draft.title = `${draft.title}${separator}{{${key}}}`;
	}

	async function save() {
		touched = true;
		if (found.length > 0) return;
		saving = true;
		try {
			await onsave(bodyOf(draft));
		} finally {
			saving = false;
		}
	}
</script>

{#snippet removeButton(onclick: () => void)}
	<button
		type="button"
		class="shrink-0 cursor-pointer rounded-md p-1 hover:bg-main-hover"
		aria-label={m['custom_wk.remove']()}
		title={m['custom_wk.remove']()}
		{onclick}
	>
		<Icon name="trash" class="h-4 w-4 mx-1 -mt-0.5 text-destructive!" />
	</button>
{/snippet}

{#snippet rowsEditor(label: string, hint: string, rows: Row[])}
	<fieldset class="flex flex-col gap-1">
		<legend class="text-sm font-semibold">{label}</legend>
		<p class="text-xs opacity-70">{hint}</p>
		{#each rows as row, i (i)}
			<div class="flex items-center gap-1">
				<input
					class={fieldClass + ' flex-1'}
					placeholder={m['custom_wk.key']()}
					bind:value={row.key}
				/>
				<input
					class={fieldClass + ' flex-1'}
					placeholder={m['custom_wk.value_variable']()}
					bind:value={row.value}
				/>
				{@render removeButton(() => rows.splice(i, 1))}
			</div>
		{/each}
		<Button
			variant="gray"
			class="self-start px-2 py-0.5 text-xs"
			onclick={() => rows.push({ key: '', value: '' })}
		>
			{m['custom_wk.add_row']()}
		</Button>
	</fieldset>
{/snippet}

<div class="flex flex-col gap-3 border-t border-main-border pt-3">
	<Input
		label={m['custom_wk.name']()}
		labelClass="font-semibold"
		bind:value={draft.name}
		maxlength={64}
		required
	/>
	<Input
		label={m['custom_wk.description']()}
		labelClass="font-semibold"
		bind:value={draft.description}
		maxlength={200}
	/>

	<Input
		label={m['custom_wk.resource']()}
		labelClass="font-semibold"
		placeholder="https://api.example.com/v1/data"
		bind:value={draft.resource}
		required
	/>
	<div class="flex gap-2">
		<label class="flex flex-1 flex-col gap-1 text-sm">
			<span class="font-semibold">{m['custom_wk.response']()}</span>
			<select class={fieldClass} bind:value={draft.response}>
				<option value="json">JSON</option>
				<option value="html">HTML</option>
			</select>
		</label>
		<label class="flex flex-1 flex-col gap-1 text-sm">
			<span class="font-semibold">{m['custom_wk.rate']()}</span>
			<input class={fieldClass} type="number" min="600" bind:value={draft.rate} />
		</label>
	</div>
	<p class="text-xs opacity-70">{m['custom_wk.rate_hint']()}</p>

	<fieldset class="flex flex-col gap-1">
		<legend class="text-sm font-semibold">{m['custom_wk.parameters']()}</legend>
		<p class="text-xs opacity-70">{m['custom_wk.parameters_hint']({ example: m['custom_wk.var_example']() })}</p>
		{#each draft.parameters as param, i (i)}
			<div class="flex flex-col gap-1 rounded-md border border-main-border p-2">
				<div class="flex items-center gap-1">
					<input
						class={fieldClass + ' w-28 font-mono'}
						placeholder={m['custom_wk.var_example']()}
						bind:value={param.key}
					/>
					<input
						class={fieldClass + ' flex-1'}
						placeholder={m['custom_wk.label']()}
						bind:value={param.label}
					/>
					{@render removeButton(() => draft.parameters.splice(i, 1))}
				</div>
				<div class="flex items-center gap-1">
					<select class={fieldClass} bind:value={param.type}>
						<option value="string">{m['custom_wk.type_string']()}</option>
						<option value="number">{m['custom_wk.type_number']()}</option>
						<option value="secret">{m['custom_wk.type_secret']()}</option>
					</select>
					<input
						class={fieldClass + ' flex-1'}
						placeholder={m['custom_wk.placeholder']()}
						bind:value={param.placeholder}
					/>
					{#if param.type !== 'secret'}
						<input
							class={fieldClass + ' flex-1'}
							placeholder={m['custom_wk.default']()}
							bind:value={param.default}
						/>
					{/if}
				</div>
			</div>
		{/each}
		<Button
			variant="gray"
			class="self-start px-2 py-0.5 text-xs"
			onclick={() =>
				draft.parameters.push({ key: '', label: '', type: 'string', placeholder: '', default: '' })}
		>
			{m['custom_wk.add_parameter']()}
		</Button>
	</fieldset>

	{@render rowsEditor(
		m['custom_wk.queries'](),
		m['custom_wk.request_hint']({ example: m['custom_wk.var_example']() }),
		draft.queries
	)}
	{@render rowsEditor(
		m['custom_wk.headers'](),
		m['custom_wk.request_hint']({ example: '$api_key' }),
		draft.headers
	)}

	<fieldset class="flex flex-col gap-1">
		<legend class="text-sm font-semibold">{m['custom_wk.outputs']()}</legend>
		<p class="text-xs opacity-70">
			{m['custom_wk.outputs_hint']({ example: '.data.temperature' })}
		</p>
		{#each draft.outputs as output, i (i)}
			<div class="flex flex-col gap-1 rounded-md border border-main-border p-2">
				<div class="flex items-center gap-1">
					<input
						class={fieldClass + ' w-28 font-mono'}
						placeholder={m['custom_wk.field_name']()}
						bind:value={output.key}
					/>
					<input
						class={fieldClass + ' flex-1'}
						placeholder={m['custom_wk.label']()}
						bind:value={output.label}
					/>
					{@render removeButton(() => draft.outputs.splice(i, 1))}
				</div>
				<div class="flex items-center gap-1">
					<input
						class={fieldClass + ' flex-1 font-mono'}
						placeholder={m['custom_wk.query']()}
						bind:value={output.query}
					/>
					<input
						class={fieldClass + ' flex-1'}
						placeholder={m['custom_wk.example']()}
						bind:value={output.example}
					/>
				</div>
			</div>
		{/each}
		<Button
			variant="gray"
			class="self-start px-2 py-0.5 text-xs"
			onclick={() => draft.outputs.push({ key: '', label: '', query: '', example: '' })}
		>
			{m['custom_wk.add_output']()}
		</Button>
	</fieldset>

	<Input
		label={m['custom_wk.card_title']()}
		labelClass="font-semibold"
		placeholder={m['card_title.placeholder']({ example: '{{value}}' })}
		bind:value={draft.title}
		maxlength={127}
		required
	/>
	{#if outputKeys.length > 0}
		<div class="flex flex-wrap gap-1">
			{#each outputKeys as key (key)}
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

	<div class="flex gap-2">
		<Button variant="primary" disabled={saving || (touched && found.length > 0)} onclick={save}>
			{m['custom_wk.save']()}
		</Button>
		<Button variant="gray" onclick={oncancel}>{m['custom_wk.cancel']()}</Button>
	</div>
</div>
