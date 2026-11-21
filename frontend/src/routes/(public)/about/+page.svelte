<script lang="ts">
	import { getLocale } from '$lib/paraglide/runtime';
	import AboutEn from './AboutEn.svelte';
	import AboutEs from './AboutEs.svelte';
</script>

<!-- Long, linked text reads better whole than as one message per sentence. -->
{#if getLocale() === 'es'}
	<AboutEs />
{:else}
	<AboutEn />
{/if}
