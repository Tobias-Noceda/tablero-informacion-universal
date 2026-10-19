<script lang="ts">
	import { getLocale, locales, type Locale } from '$lib/paraglide/runtime';
	import { cn } from '$lib/utils';
	import { switchLocale } from '$modules/locale';

	const languageClass = (locale: Locale) =>
		cn(
			'px-4 py-2 rounded-md text-white hover:bg-main-hover transition-colors text-main-text text-sm font-medium',
			getLocale() === locale ? 'bg-main-hover' : '',
			'cursor-pointer'
		);
</script>

<div class="flex flex-row items-center gap-3">
	{#each locales as locale (locale)}
		<button type="button" class={languageClass(locale)} onclick={() => switchLocale(locale)}>
			{locale.toUpperCase()}
		</button>
	{/each}
</div>
