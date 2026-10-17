<script lang="ts">
	import './layout.css';
	import favicon from '$assets/svgs/corcheria-m2-icono-app.svg';

	let { children } = $props();
</script>

<svelte:head><link rel="icon" href={favicon} /></svelte:head>

{@render children()}
