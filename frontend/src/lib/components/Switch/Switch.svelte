<script lang="ts">
	import type { Snippet } from 'svelte';
	import { cn } from '$lib/utils';

	interface Props {
		label?: string;
		/** Accessible name when no visible label is rendered. */
		ariaLabel?: string;
		checked?: boolean;
		disabled?: boolean;
		skeleton?: boolean;
		onchange?: (checked: boolean) => void;
		/** Classes for the wrapper (label + switch). */
		class?: string;
		/** Classes for the track. */
		innerClass?: string;
		/** Shown inside the knob when on / off, e.g. a small icon. */
		checkedIcon?: Snippet;
		uncheckedIcon?: Snippet;
	}

	let {
		label,
		ariaLabel,
		checked = $bindable(false),
		disabled = false,
		skeleton = false,
		onchange,
		class: switchClass,
		innerClass,
		checkedIcon,
		uncheckedIcon
	}: Props = $props();

	const labelId = $props.id();

	function toggle() {
		if (disabled || skeleton) return;
		checked = !checked;
		onchange?.(checked);
	}

	const trackClass = $derived(
		cn(
			'relative inline-flex h-6 w-11 shrink-0 items-center rounded-full p-0.5',
			'transition-colors duration-200 ease-out',
			'focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-main',
			checked ? 'bg-main' : 'bg-black/25',
			disabled ? 'cursor-not-allowed opacity-50' : 'cursor-pointer',
			skeleton && 'animate-pulse cursor-default bg-skeleton',
			innerClass
		)
	);

	const knobClass = $derived(
		cn(
			'pointer-events-none flex h-5 w-5 items-center justify-center rounded-full',
			'bg-white text-[11px] leading-none text-main shadow-md',
			'transition-transform duration-200 ease-out',
			checked ? 'translate-x-5' : 'translate-x-0',
			skeleton && 'bg-skeleton'
		)
	);
</script>

<div class={cn('flex items-center gap-2', switchClass)}>
	{#if label}
		<span id={labelId} class="text-sm font-medium">{label}</span>
	{/if}
	<button
		type="button"
		role="switch"
		class={trackClass}
		aria-checked={checked}
		aria-labelledby={label ? labelId : undefined}
		aria-label={label ? undefined : ariaLabel}
		disabled={disabled || skeleton}
		onclick={toggle}
	>
		<span class={knobClass} aria-hidden="true">
			{#if checked}
				{@render checkedIcon?.()}
			{:else}
				{@render uncheckedIcon?.()}
			{/if}
		</span>
	</button>
</div>
