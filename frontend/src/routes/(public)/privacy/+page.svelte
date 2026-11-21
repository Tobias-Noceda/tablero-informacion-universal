<script lang="ts">
	import { getLocale } from '$lib/paraglide/runtime';
	import PrivacyEn from './PrivacyEn.svelte';
	import PrivacyEs from './PrivacyEs.svelte';
</script>

<!-- Long, linked text reads better whole than as one message per sentence. -->
{#if getLocale() === 'es'}
	<PrivacyEs />
{:else}
	<PrivacyEn />
{/if}
