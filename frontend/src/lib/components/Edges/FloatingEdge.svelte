<script lang="ts">
	import { untrack } from 'svelte';
	import {
		BaseEdge,
		EdgeLabel,
		EdgeToolbar,
		getStraightPath,
		useInternalNode,
		type EdgeProps
	} from '@xyflow/svelte';

	import { m } from '$lib/paraglide/messages';
	import { getStrandEditor } from '$lib/strands/editor';
	import {
		arrowEnds,
		MAX_STRAND_LABEL,
		normalizeLabel,
		resolveStyle,
		strokeColor,
		strokeWidth,
		type StrandData
	} from '$lib/strands/style';
	import StrandToolbar from './StrandToolbar.svelte';
	import { getEdgeParams, trimSegment } from './utils';

	let { id, source, target, data }: EdgeProps = $props();

	const editor = getStrandEditor();

	const strand = $derived((data ?? {}) as StrandData);
	const style = $derived(resolveStyle(strand));
	const isSelected = $derived(strand.isSelected ?? false);
	const canEdit = $derived(editor?.canEdit ?? false);

	const stroke = $derived(strokeColor(style.color));
	const width = $derived(strokeWidth(style.width));
	const ends = $derived(arrowEnds(style.arrow));
	// The arrowhead grows with the line, but less than it.
	const arrowSize = $derived(6 + 3 * width);
	const markerId = $derived(`strand-arrow-${id}`);

	const sourceNode = $derived.by(() => useInternalNode(source));
	const targetNode = $derived.by(() => useInternalNode(target));

	// Border to border: where the line between the centres leaves each card.
	const geometry = $derived.by(() => {
		if (!sourceNode.current || !targetNode.current) return null;
		const { sx, sy, tx, ty } = getEdgeParams(sourceNode.current, targetNode.current);
		const [, labelX, labelY] = getStraightPath({ sourceX: sx, sourceY: sy, targetX: tx, targetY: ty });
		const { from, to } = trimSegment(
			{ x: sx, y: sy },
			{ x: tx, y: ty },
			ends.start ? arrowSize : 0,
			ends.end ? arrowSize : 0
		);
		const [path] = getStraightPath({ sourceX: from.x, sourceY: from.y, targetX: to.x, targetY: to.y });
		return { path, labelX, labelY };
	});

	let editingLabel = $state(false);
	let draft = $state('');
	// Closing the input removes it, which fires its blur while the board is
	// still applying the commit; this plain flag (always current, unlike a
	// read of editingLabel there) makes that blur, or one after Escape, a no-op.
	let labelOpen = false;

	function startEditingLabel() {
		if (!canEdit) return;
		draft = style.label;
		labelOpen = editingLabel = true;
	}

	function closeLabel() {
		labelOpen = editingLabel = false;
	}

	function commitLabel() {
		if (!labelOpen) return;
		closeLabel();
		const label = normalizeLabel(draft);
		if (label !== style.label) editor?.update(id, { label });
	}

	function onLabelKeydown(event: KeyboardEvent) {
		event.stopPropagation();
		if (event.key === 'Enter') commitLabel();
		if (event.key === 'Escape') closeLabel();
	}

	// d3-zoom answers a double click on the canvas with a zoom and stops the
	// event there, before Svelte's delegated handlers run: listen on the
	// element itself.
	const onDoubleClick = (handler: () => void) => (element: HTMLElement) => {
		const listener = (event: MouseEvent) => {
			event.stopPropagation();
			handler();
		};
		element.addEventListener('dblclick', listener);
		return () => element.removeEventListener('dblclick', listener);
	};

	// EdgeLabel portals the input into the label layer after it is created,
	// which drops any focus given before: focus it on the next frame.
	const focusInput = (input: HTMLInputElement) => {
		const frame = requestAnimationFrame(() => input.select());
		return () => cancelAnimationFrame(frame);
	};

	// Leaving the strand ends its edit, keeping what was typed.
	$effect(() => {
		if (!isSelected) untrack(commitLabel);
	});
</script>

{#if geometry}
	<defs>
		<marker
			id={markerId}
			viewBox="0 0 10 10"
			refX="0"
			refY="5"
			markerUnits="userSpaceOnUse"
			markerWidth={arrowSize}
			markerHeight={arrowSize}
			orient="auto-start-reverse"
		>
			<path d="M0,0 L10,5 L0,10 Z" style:fill={stroke} />
		</marker>
	</defs>

	{#if isSelected}
		<path
			d={geometry.path}
			fill="none"
			class="pointer-events-none"
			style:stroke={stroke}
			style:stroke-width="{width + 8}px"
			style:stroke-opacity="0.2"
			style:stroke-linecap="round"
		/>
	{/if}

	<BaseEdge
		{id}
		path={geometry.path}
		markerStart={ends.start ? `url(#${markerId})` : undefined}
		markerEnd={ends.end ? `url(#${markerId})` : undefined}
		style="stroke: {stroke}; stroke-width: {width}px;"
	/>

	{#if editingLabel}
		<EdgeLabel x={geometry.labelX} y={geometry.labelY} transparent class="nodrag nopan">
			<input
				bind:value={draft}
				{@attach focusInput}
				{@attach onDoubleClick(() => {})}
				maxlength={MAX_STRAND_LABEL}
				aria-label={m['strands.label']()}
				placeholder={m['strands.label_placeholder']()}
				class="w-40 rounded-md border bg-(--app-background) px-2 py-0.5 text-center text-xs text-(--app-foreground) outline-none"
				style:border-color={stroke}
				onkeydown={onLabelKeydown}
				onblur={commitLabel}
			/>
		</EdgeLabel>
	{:else if style.label}
		<EdgeLabel x={geometry.labelX} y={geometry.labelY} transparent>
			<button
				type="button"
				class="max-w-48 truncate rounded-md border bg-(--app-background) px-2 py-0.5 text-xs text-(--app-foreground)"
				style:border-color={stroke}
				title={canEdit ? m['strands.label_edit_hint']() : style.label}
				onclick={() => editor?.select(id)}
				{@attach onDoubleClick(startEditingLabel)}
			>
				{style.label}
			</button>
		</EdgeLabel>
	{/if}

	{#if canEdit}
		<EdgeToolbar x={geometry.labelX} y={geometry.labelY} alignY="bottom" isVisible={isSelected} class="pointer-events-none">
			<!-- The gap above the label lets clicks through to it. -->
			<div class="pointer-events-none pb-6">
				<StrandToolbar
					{style}
					onchange={(patch) => editor?.update(id, patch)}
					onlabel={startEditingLabel}
					onremove={() => editor?.remove(id)}
				/>
			</div>
		</EdgeToolbar>
	{/if}
{/if}
