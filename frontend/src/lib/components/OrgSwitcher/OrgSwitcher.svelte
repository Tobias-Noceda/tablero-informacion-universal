<script lang="ts">
	import { m } from '$lib/paraglide/messages';
	import { orgs, PERSONAL } from '$stores/org.svelte';
</script>

<select
	class="max-w-48 cursor-pointer truncate rounded-md border border-main-border bg-main px-2 py-1.5 text-sm text-white transition-colors hover:bg-main-hover"
	aria-label={m['orgs.switcher']()}
	value={orgs.selected}
	onchange={(event) => orgs.select(event.currentTarget.value)}
>
	<option value={PERSONAL}>{m['orgs.personal']()}</option>
	{#each orgs.list as org (org.id)}
		<option value={org.id}>{org.name}</option>
	{/each}
</select>
