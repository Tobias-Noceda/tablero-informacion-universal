<script lang="ts">
	import Button from '$components/Button/Button.svelte';
	import Dropdown from '$components/Dropdown/Dropdown.svelte';
	import Icon from '$components/Icon/Icon.svelte';
	import Input from '$components/Input/Input.svelte';
	import { m } from '$lib/paraglide/messages';
	import { renderTitle } from '$lib/title';
	import { cn } from '$lib/utils';
	import { bodyOf, problems, type Draft, type Problem, type Row } from '$lib/well-knowns/draft';
	import type { PostitTemplateBody } from '$types/api';

	interface Props {
		/** What is being written. The parent owns it, so it outlives the form
		 * (switching tabs keeps a half-written card type). */
		draft: Draft;
		onsave: (body: PostitTemplateBody) => Promise<void>;
		oncancel: () => void;
	}

	let { draft = $bindable(), onsave, oncancel }: Props = $props();

	const PARAM_TYPES = ['string', 'number', 'secret'] as ('string' | 'number' | 'secret')[];

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

	const fieldClass = cn(
		'min-w-0 bg-transparent px-2 py-1 rounded-md',
		'text-sm text-foreground',
		'border-border hover:border-sidebar focus:border-main focus:ring-1 focus:ring-main',
		'focus:outline-none focus:ring-2 focus:ring-main-border',
	);

	function describe(problem: Problem) {
		switch (problem.code) {
			case 'name':
				return m['postit_templates.err_name']();
			case 'resource':
				return m['postit_templates.err_resource']();
			case 'outputs':
				return m['postit_templates.err_outputs']();
			case 'output_key':
				return m['postit_templates.err_output_key']();
			case 'output_label':
				return m['postit_templates.err_output_label']();
			case 'output_query':
				return m['postit_templates.err_output_query']();
			case 'param_key':
				return m['postit_templates.err_param_key']({ key: problem.value || '…' });
			case 'param_label':
				return m['postit_templates.err_param_label']();
			case 'duplicate':
				return m['postit_templates.err_duplicate']({ key: problem.value });
			case 'undeclared':
				return m['postit_templates.err_undeclared']({ value: problem.value });
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
		class="shrink-0 cursor-pointer rounded-md p-1 hover:bg-background-hover"
		aria-label={m['postit_templates.remove']()}
		title={m['postit_templates.remove']()}
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
					placeholder={m['postit_templates.key']()}
					bind:value={row.key}
				/>
				<input
					class={fieldClass + ' flex-1'}
					placeholder={m['postit_templates.value_variable']()}
					bind:value={row.value}
				/>
				{@render removeButton(() => rows.splice(i, 1))}
			</div>
		{/each}
		<Button
			variant="secondary"
			class="self-start px-2 py-0.5 text-xs"
			onclick={() => rows.push({ key: '', value: '' })}
		>
			{m['postit_templates.add_row']()}
		</Button>
	</fieldset>
{/snippet}

<div class="flex flex-col gap-3 border-t border-main-border pt-3">
	<Input
		label={m['postit_templates.name']()}
		labelClass="font-semibold"
		bind:value={draft.name}
		maxlength={64}
		required
	/>
	<Input
		label={m['postit_templates.description']()}
		labelClass="font-semibold"
		bind:value={draft.description}
		maxlength={200}
	/>

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
				options={['json', 'html'].map((response) => {
					return {
						value: response,
						label: response.toUpperCase()
					}
				})}
				class="w-full!"
				activatorClass={(isOpen) => cn( 
					fieldClass,
					'w-full text-base hover:border-sidebar',
					isOpen
						? "outline-none ring-2 ring-main-border border-none"
						: "focus:ring-0!"
				)}
				drawerClass={cn(fieldClass, 'bg-background mt-1 px-1')}
				itemClass={(isSelected) => cn(
					'hover:bg-background-hover text-foreground text-sm rounded-md',
					isSelected
						? 'bg-background-hover'
						: 'bg-transparent'
				)}
			/>
		</label>
		<label class="flex flex-1 flex-col gap-1 text-sm">
			<span class="font-semibold">{m['postit_templates.rate']()}</span>
			<input class={cn(fieldClass, 'px-2 py-1.5')} type="number" min="600" bind:value={draft.rate} />
		</label>
	</div>
	<p class="text-xs opacity-70">{m['postit_templates.rate_hint']()}</p>

	<fieldset class="flex flex-col gap-1">
		<legend class="text-sm font-semibold">{m['postit_templates.parameters']()}</legend>
		<p class="text-xs opacity-70">{m['postit_templates.parameters_hint']({ example: m['postit_templates.var_example']() })}</p>
		{#each draft.parameters as param, i (i)}
			<div class="flex flex-col gap-1 rounded-md border border-main-border p-2">
				<div class="flex items-center gap-1">
					<input
						class={fieldClass + ' w-28 font-mono'}
						placeholder={m['postit_templates.var_example']()}
						bind:value={param.key}
					/>
					<input
						class={fieldClass + ' flex-1'}
						placeholder={m['postit_templates.label']()}
						bind:value={param.label}
					/>
					{@render removeButton(() => draft.parameters.splice(i, 1))}
				</div>
				<div class="flex items-center gap-1">
					<div class="flex w-32">
						<Dropdown
							bind:value={param.type}
							options={PARAM_TYPES.map((type) => {
								return {
									value: type,
									label: m[`postit_templates.type_${type}`]()
								}
							})}
							class="w-full!"
							activatorClass={(isOpen) => cn( 
								fieldClass,
								'w-full text-base hover:border-sidebar',
								isOpen
									? "outline-none ring-2 ring-main-border border-none"
									: "focus:ring-0!"
							)}
							drawerClass={cn(fieldClass, 'bg-background mt-1 px-1')}
							itemClass={(isSelected) => cn(
								'hover:bg-background-hover text-foreground text-sm rounded-md',
								isSelected
									? 'bg-background-hover'
									: 'bg-transparent'
							)}
						/>
					</div>
					<input
						class={cn(fieldClass, 'flex-1 px-2 py-1.5')}
						placeholder={m['postit_templates.placeholder']()}
						bind:value={param.placeholder}
					/>
					{#if param.type !== 'secret'}
						<input
							class={fieldClass + ' flex-1'}
							placeholder={m['postit_templates.default']()}
							bind:value={param.default}
						/>
					{/if}
				</div>
			</div>
		{/each}
		<Button
			variant="secondary"
			class="self-start px-2 py-0.5 text-xs"
			onclick={() =>
				draft.parameters.push({ key: '', label: '', type: 'string', placeholder: '', default: '' })}
		>
			{m['postit_templates.add_parameter']()}
		</Button>
	</fieldset>

	{@render rowsEditor(
		m['postit_templates.queries'](),
		m['postit_templates.request_hint']({ example: m['postit_templates.var_example']() }),
		draft.queries
	)}
	{@render rowsEditor(
		m['postit_templates.headers'](),
		m['postit_templates.request_hint']({ example: '$api_key' }),
		draft.headers
	)}

	<fieldset class="flex flex-col gap-1">
		<legend class="text-sm font-semibold">{m['postit_templates.outputs']()}</legend>
		<p class="text-xs opacity-70">
			{m['postit_templates.outputs_hint']({ example: '.data.temperature' })}
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
						class={fieldClass + ' flex-1'}
						placeholder={m['postit_templates.label']()}
						bind:value={output.label}
					/>
					{@render removeButton(() => draft.outputs.splice(i, 1))}
				</div>
				<div class="flex items-center gap-1">
					<input
						class={fieldClass + ' flex-1 font-mono'}
						placeholder={m['postit_templates.query']()}
						bind:value={output.query}
					/>
					<input
						class={fieldClass + ' flex-1'}
						placeholder={m['postit_templates.example']()}
						bind:value={output.example}
					/>
				</div>
			</div>
		{/each}
		<Button
			variant="secondary"
			class="self-start px-2 py-0.5 text-xs"
			onclick={() => draft.outputs.push({ key: '', label: '', query: '', example: '' })}
		>
			{m['postit_templates.add_output']()}
		</Button>
	</fieldset>

	<Input
		label={m['postit_templates.card_title']()}
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
			{m['postit_templates.save']()}
		</Button>
		<Button variant="destructive" onclick={oncancel}>{m['postit_templates.cancel']()}</Button>
	</div>
</div>
