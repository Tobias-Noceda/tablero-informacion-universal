<script lang="ts">
	import { fromPicker, toPicker, type DateKind } from '$lib/cards/dates';
	import { m } from '$lib/paraglide/messages';
	import { cn } from '$lib/utils';

	interface Props {
		/** `date` stores YYYY-MM-DD; `datetime` stores RFC 3339 in UTC, picked in local time. */
		kind: DateKind;
		/** What the card stores, as its API reads it. */
		value?: string;
		label?: string;
		labelClass?: string;
		required?: boolean;
		class?: string;
	}

	let {
		kind,
		value = $bindable(''),
		label,
		labelClass,
		required = false,
		class: inputClass
	}: Props = $props();

	const id = $props.id();
	const picked = $derived(toPicker(kind, value));
	// A value saved as free text before this field was a date.
	const unreadable = $derived(!!value && !picked);
</script>

<div class="flex w-full flex-col gap-1">
	{#if label}
		<label class={cn('text-sm font-medium text-foreground', labelClass)} for={id}>
			{label}
			{#if required}
				<span class="text-destructive">*</span>
			{/if}
		</label>
	{/if}
	<input
		{id}
		type={kind === 'date' ? 'date' : 'datetime-local'}
		value={picked}
		{required}
		oninput={(event) => (value = fromPicker(kind, event.currentTarget.value))}
		class={cn(
			'w-full rounded-md border border-border bg-background px-3 py-2.5 text-foreground transition-colors',
			'hover:border-sidebar focus:border-main focus:outline-none focus:ring-1 focus:ring-main',
			unreadable && 'border-destructive',
			inputClass
		)}
	/>
	{#if unreadable}
		<p class="text-xs text-destructive">{m['date_input.unreadable']({ value })}</p>
	{:else if kind === 'datetime' && value}
		<p class="text-xs opacity-70">{m['date_input.utc']({ value })}</p>
	{/if}
</div>
