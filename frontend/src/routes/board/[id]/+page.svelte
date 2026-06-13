<script>
	import './index.css';

	import Flow from './Flow.svelte';
	import DnDProvider from './DnDProvider.svelte';

	import { page } from '$app/state';

	const id = page.params.id;
</script>

<DnDProvider>
	<Flow boardId={id} />
</DnDProvider>
