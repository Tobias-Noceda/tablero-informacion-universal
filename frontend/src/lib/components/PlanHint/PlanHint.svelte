<script lang="ts">
	import { resolve } from '$app/paths';
	import { m } from '$lib/paraglide/messages';
	import { cn } from '$lib/utils';

	interface Props {
		/** What the plan does not allow (any more), e.g. "5 of 5 boards". */
		text: string;
		class?: string;
	}

	// The one line that stands where an entry point the plan does not allow
	// would be. Never a button: it only says why and where the plans are.
	let { text, class: className }: Props = $props();
</script>

<p class={cn('text-xs opacity-70', className)}>
	{text} ·
	<a href={resolve('/pricing')} class="underline hover:opacity-100"
		>{m['billing.hint.see_plans']()}</a
	>
</p>
