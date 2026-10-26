<script lang="ts">
	import { goto } from '$app/navigation';
	import { resolve } from '$app/paths';
	import { page } from '$app/state';
	import { m } from '$lib/paraglide/messages';
	import { landingPath } from '$lib/helpers/routes';
	import { orgs, PERSONAL } from '$stores/org.svelte';
	import { setSidebar } from '$stores/sidebar';
	import Dropdown from '$components/Dropdown/Dropdown.svelte';
	import { cn } from '$lib/utils';

	// Switching organization opens its landing; the URL is the selection.
	function select(value: string) {
		setSidebar(null);
		void goto(resolve(landingPath(value === PERSONAL ? undefined : value)));
	}

	const options = $derived([
		{ value: PERSONAL, label: m['orgs.personal']() },
		...orgs.list.map((org) => ({ value: org.id, label: org.name }))
	]);
</script>

<Dropdown
	class="w-fit"
	activatorClass={() => "w-48! cursor-pointer truncate whitespace-nowrap rounded-md border border-main-border bg-main px-2 py-1.5 text-sm text-white transition-colors hover:bg-main-hover"}
	drawerClass="rounded-md"
	itemClass={() =>
		cn(
			'justify-start!',
			'whitespace-nowrap truncate'
		)
	}
	placeholder={m['orgs.switcher']()}
	{options}
	value={page.params.orgId ?? PERSONAL}
	onchange={select}
/>
