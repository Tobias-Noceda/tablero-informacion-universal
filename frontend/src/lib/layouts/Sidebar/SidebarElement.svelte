<script lang="ts">
	import './index.css';

	import type { Snippet } from 'svelte';

	type SidebarElementProps = {
		ondragstart?: (event: DragEvent) => void;
		class?: string;
		tooltipContent?: Snippet;
		children: Snippet;
	};

	const { ondragstart, class: className, tooltipContent, children }: SidebarElementProps = $props();

	import { cn } from '$lib/utils';
	import Tooltip from '$components/Tooltip/Tooltip.svelte';
</script>

<div
	class={cn(className, 'sidebar-element')}
	{ondragstart}
	draggable={ondragstart ? true : false}
	role="button"
	tabindex="0"
>
	<Tooltip
		placement="top-right"
		reference="mouse"
		content={tooltipContent || children}
		class="block w-full px-2 py-1.5 text-white text-sm transition-colors truncate select-none!"
		tooltipClass="ml-0.25 mb-0.25 min-w-fit!"
	>
		{@render children()}
	</Tooltip>
</div>
