<script lang="ts">
	import './index.css';

	import { getUser } from '$stores/user';

	import * as secretsApi from '$services/secrets';
	import type { SecretMeta, UUID } from '$types/api';

	import { m } from '$lib/paraglide/messages';

	import SidebarElement from './SidebarElement.svelte';
	import Icon from '$components/Icon/Icon.svelte';
	import { setManagingSecrets } from '$stores/sidebar';

	let { board }: { board: UUID } = $props();

	// "board": shared with every member. "mine": what this user keeps here
	// for themselves; nobody else on the board can see or bind it.
	const boardScopes = $derived({
		board: secretsApi.boardScope(board),
		...{ mine: $getUser?.id ? secretsApi.memberScope(board, $getUser?.id) : null }
	});

	let secrets = $state<{ board: SecretMeta[]; mine: SecretMeta[] }>({ board: [], mine: [] });

	async function refresh() {
		try {
			secrets = {
				board: await secretsApi.list(boardScopes.board, $getUser?.id ?? ''),
				mine: boardScopes.mine ? await secretsApi.list(boardScopes.mine, $getUser?.id ?? '') : []
			};
		} catch (e) {
			console.error(e);
		}
	}

	refresh();
</script>

<div class="nodes-container items-start justify-start">
	{#each Object.entries(secrets) as secretScope (secretScope[0])}
		{#each secretScope[1] as secret (secret.name)}
			{#snippet tooltipContent()}
				${secret.name}
			{/snippet}
			<SidebarElement class={`${secret.kind}-${secret.name}`} {tooltipContent}>
				<div class="flex flex-col">
					<code class="text-sm">${secret.name}</code>
					<span class="text-xs opacity-60">
						{secret.provider ?? secret.kind}{secret.flow && !secret.provider
							? ` · ${secret.flow}`
							: ''}
						{#if secret.flow === 'authorization_code'}
							· {secret.authorized ? m['secrets.authorized']() : m['secrets.pending']()}
						{/if}
					</span>
					{#if secretScope[0] === 'mine'}
						<span class="text-xs font-medium opacity-90 text-red-900"> Personal </span>
					{/if}
				</div>
			</SidebarElement>
		{/each}
	{/each}

	<button class="sidebar-button gap-1 mt-2" onclick={() => setManagingSecrets(true)} tabindex="0">
		<Icon name="manage" class="w-5 h-5 text-main-text!" />
		{m['secrets.manage']()}
	</button>
</div>
