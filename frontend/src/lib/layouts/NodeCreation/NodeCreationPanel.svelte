<script lang="ts">
	import {
		useSvelteFlow,
		type Node,
	} from '@xyflow/svelte';

	import * as postItsApi from '$services/post-it';
	import type { Board } from '$types/api';
	import { bindingsFor, refKey } from '$lib/secrets/binding';
	import type { SecretMeta } from '$types/api';
	import { outputs, parameters } from '$components/Nodes/node-map';
	import { analyzeTitle, renderTitle } from '$lib/title';
	import { mouses } from '$stores/mouses.svelte';

	import { getUser } from '$stores/user';
	import Modal from '$components/Modal/Modal.svelte';
	import Input from '$components/Input/Input.svelte';
	import { m } from '$lib/paraglide/messages';
	import { groupByOrigin } from '$lib/secrets/origin';

	let {
        boardId,
        creatingNode,
        paramValues,
        usableSecrets,
        onclose,
        onCreateNode,
	}: {
        boardId: string;
        creatingNode: Board['postits'][number] | null;
        paramValues: Record<string, string>;
        usableSecrets: SecretMeta[];
        onclose: () => void;
        onCreateNode: (node: Node) => void;
	} = $props();

    const pickable = $derived(groupByOrigin(usableSecrets, boardId, $getUser?.id ?? ''));
	const byKey = $derived(new Map(usableSecrets.map((s) => [refKey(s), s])));

	let creatingTitle = $state<string>('');
	// A secret parameter holds the picked secret's key, not its name.
	let pickedKeys = $state<Record<string, string>>({});

	const creatingParams = $derived(creatingNode ? (parameters[creatingNode.type!] ?? []) : []);
	const missingRequired = $derived(
		creatingParams.some((p) => {
			const value = p.type === 'secret' ? pickedKeys[p.key] : paramValues[p.key];
			return p.default === undefined && !(value ?? '').trim();
		})
	);

    // What the card answers with, usable in its title as {{key}}.
	const creatingOutputs = $derived(creatingNode ? (outputs[creatingNode.type!] ?? []) : []);
	// Example values to preview the title; a card without a resource answers
	// with its own params, so those show what the user typed.
	const sampleResponse = $derived(
		Object.fromEntries(
			creatingOutputs.map((o) => [o.key, (paramValues[o.key] ?? '').trim() || o.example])
		)
	);

	const titleAnalysis = $derived(
		analyzeTitle(
			creatingTitle,
			creatingOutputs.map((o) => o.key)
		)
	);
	const titleInvalid = $derived(titleAnalysis.unsupported || titleAnalysis.unknown.length > 0 || !creatingTitle.trim());

    const insertVariable = (key: string) => {
		const separator = creatingTitle && !creatingTitle.endsWith(' ') ? ' ' : '';
		creatingTitle = `${creatingTitle}${separator}{{${key}}}`;
	};

	const { screenToFlowPosition, flowToScreenPosition } = useSvelteFlow();

	mouses.updateMappers(screenToFlowPosition, flowToScreenPosition);

	const createNode = async () => {
		if (!creatingNode || missingRequired || titleInvalid) return;

		const picked = Object.fromEntries(
			creatingParams
				.filter((p) => p.type === 'secret' && byKey.has(pickedKeys[p.key]))
				.map((p) => [p.key, byKey.get(pickedKeys[p.key])!])
		);
		const secrets = bindingsFor(picked, boardId);

		const params = {
			...Object.fromEntries(
				creatingParams
					.filter((p) => p.type !== 'secret')
					.map((p) => [p.key, (paramValues[p.key] ?? '').trim()])
			),
			...secrets.params
		};

		const title = creatingTitle.trim();

		// The backend recomputes vars, this only mirrors it.
		const newPostIt = await postItsApi.create_well_known(
			boardId,
			{ text: title, vars: titleAnalysis.vars },
			creatingNode.type!,
			params,
			$getUser?.id ?? '',
			secrets.bindings
		);
		await postItsApi.move(newPostIt.id, creatingNode.position.x, creatingNode.position.y);

		// The create response is the post-it itself: its title is top level.
		const created = newPostIt as unknown as { title?: { text: string; vars: boolean } };
        onCreateNode({ ...creatingNode, id: newPostIt.id, data: { title: created.title } } as Node);
		pickedKeys = {};
        creatingTitle = '';
	};
</script>

{#if creatingNode}
    <Modal
        onclose={() => {
            creatingTitle = '';
            pickedKeys = {};
            onclose();
        }}
        onaccept={() => {
            if (creatingNode) {
                createNode();
            }
        }}
        acceptText="Create"
        acceptDisabled={missingRequired || titleInvalid}
    >
        <div class="flex w-full justify-center">
            <h2 class="text-xl font-semibold">Create Node</h2>
        </div>
        <Input
            label={m['card_title.label']()}
            placeholder={m['card_title.placeholder']({ example: '{{value}}' })}
            required
            bind:value={creatingTitle}
            labelClass="font-semibold"
        />
        {#if titleAnalysis.unsupported}
            <span class="text-xs text-destructive">{m['card_title.unsupported']({ syntax: '{{name}}' })}</span>
        {:else if titleAnalysis.unknown.length > 0}
            <span class="text-xs text-destructive">
                {m['card_title.unknown']({ names: titleAnalysis.unknown.map((n) => `{{${n}}}`).join(', ') })}
            </span>
        {/if}
        {#if creatingOutputs.length === 0}
            <span class="text-xs opacity-70">{m['card_title.none']()}</span>
        {:else}
            <div class="flex flex-col gap-1 text-sm">
                <span class="font-semibold">{m['card_title.variables']()}</span>
                <div class="flex flex-wrap gap-1">
                    {#each creatingOutputs as output (output.key)}
                        <button
                            type="button"
                            class="rounded-md border border-main-border bg-background px-2 py-0.5 font-mono text-xs hover:bg-main-hover cursor-pointer"
                            title={`${output.label}: ${JSON.stringify(sampleResponse[output.key])}`}
                            onclick={() => insertVariable(output.key)}
                        >
                            {`{{${output.key}}}`}
                        </button>
                    {/each}
                </div>
                <span class="text-xs opacity-70">{m['card_title.variables_hint']({ syntax: '{{name}}' })}</span>
            </div>
            <details class="text-xs">
                <summary class="cursor-pointer">{m['card_title.sample']()}</summary>
                <pre class="mt-1 max-h-40 overflow-auto rounded-md bg-background p-2">{JSON.stringify(sampleResponse, null, 2)}</pre>
            </details>
            {#if titleAnalysis.vars}
                <div class="flex flex-col gap-1 text-sm">
                    <span class="font-semibold">{m['card_title.preview']()}</span>
                    <span class="rounded-md bg-background px-2 py-1 font-bold">{renderTitle(creatingTitle, sampleResponse)}</span>
                </div>
            {/if}
        {/if}
        {#each creatingParams as param (param.key)}
            {#if param.type === 'secret'}
                <label class="flex flex-col gap-1 text-sm">
                    <span class="font-semibold">{param.label}</span>
                    {#if pickable.length === 0}
                        <span class="text-xs text-destructive">{m['secrets.no_credentials']()}</span>
                    {:else}
                        <select
                            class="bg-background border border-main-border rounded-md px-2 py-1"
                            bind:value={pickedKeys[param.key]}
                        >
                            <option value="">{m['secrets.pick_credential']()}</option>
                            {#each pickable as group (group.origin)}
                                <optgroup label={m[`secrets.origin_${group.origin}`]()}>
                                    {#each group.secrets as secret (refKey(secret))}
                                        <option value={refKey(secret)}>{secret.name} ({secret.provider ?? secret.kind})</option>
                                    {/each}
                                </optgroup>
                            {/each}
                        </select>
                    {/if}
                </label>
            {:else}
                <Input
                    label={param.label}
                    labelClass="font-semibold"
                    placeholder={param.placeholder}
                    type={param.type === 'number' ? 'number' : 'text'}
                    required={param.default === undefined}
                    bind:value={paramValues[param.key]}
                />
            {/if}
        {/each}
    </Modal>
{/if}
