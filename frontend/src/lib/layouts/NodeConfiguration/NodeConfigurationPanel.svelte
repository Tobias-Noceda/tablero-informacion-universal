<script lang="ts">
	import type { Node } from '@xyflow/svelte';

	import * as postItsApi from '$services/post-it';
	import type { Board, SecretRef } from '$types/api';
	import { bindingsFor, refKey } from '$lib/secrets/binding';
	import type { SecretMeta } from '$types/api';
	import { outputs, parameters } from '$components/Nodes/node-map';
	import { analyzeTitle, renderTitle } from '$lib/title';

	import { session } from '$modules/session.svelte';
	import Modal from '$components/Modal/Modal.svelte';
	import Input from '$components/Input/Input.svelte';
	import { m } from '$lib/paraglide/messages';
	import { groupByOrigin } from '$lib/secrets/origin';
	import { getLocale } from '$lib/paraglide/runtime';
	import type { PostItData } from '$layouts/BoardContent/NodeDetail.svelte';
	import { sameBindings, sameParams } from './helpers';
	import Dropdown from '$components/Dropdown/Dropdown.svelte';
	import { cn } from '$lib/utils';

    interface BoardConfigurationProps {
        boardId: string;
        creatingNode: Board['postits'][number] | null;
        paramValues: Record<string, string>;
        usableSecrets: SecretMeta[];
        currentPostIt?: PostItData | null;
        onclose: () => void;
        onPersistNode: (action: 'create' | 'update', node: Node) => void;
    }

	let {
        boardId,
        creatingNode,
        paramValues,
        usableSecrets,
        currentPostIt,
        onclose,
        onPersistNode,
	}: BoardConfigurationProps = $props();

    const scrollClass = cn(
		// Custom scrollbar styles
		'scrollbar-thin scrollbar-thumb-main scrollbar-track-transparent',
		'[&::-webkit-scrollbar]:w-2',
		'[&::-webkit-scrollbar-track]:bg-transparent',
		'[&::-webkit-scrollbar-thumb]:bg-main',
		'[&::-webkit-scrollbar-thumb]:rounded-full',
		'[&::-webkit-scrollbar-thumb]:hover:bg-main-hover'
	);

    const horizontalScrollClass = cn(
		// Custom scrollbar styles
		'scrollbar-thin scrollbar-thumb-main scrollbar-track-transparent',
		'[&::-webkit-scrollbar]:h-2',
		'[&::-webkit-scrollbar-track]:bg-transparent',
		'[&::-webkit-scrollbar-thumb]:bg-main',
		'[&::-webkit-scrollbar-thumb]:rounded-full',
		'[&::-webkit-scrollbar-thumb]:hover:bg-main-hover'
	);

    let originalConfig = $state<{
        title: string;
        paramValues: Record<string, string>;
        bindings: Record<string, SecretRef>;
    } | null>(null);

    const pickable = $derived(groupByOrigin(usableSecrets, boardId, session.userId));
	const byKey = $derived(new Map(usableSecrets.map((s) => [refKey(s), s])));

	let creatingTitle = $state<string>('');
	// A secret parameter holds the picked secret's key, not its name.
	let pickedKeys = $state<Record<string, string>>({});

	const creatingParams = $derived(creatingNode ? (parameters[creatingNode.type!] ?? []) : currentPostIt ? (parameters[currentPostIt.type!] ?? []) : []);
	const missingRequired = $derived(
		creatingParams.some((p) => {
			const value = p.type === 'secret' ? pickedKeys[p.key] : paramValues[p.key];
			return p.default === undefined && !(value ?? '').trim();
		})
	);

    // What the card answers with, usable in its title as {{key}}.
	const creatingOutputs = $derived(creatingNode ? (outputs[creatingNode.type!] ?? []) : currentPostIt ? (outputs[currentPostIt.type!] ?? []) : []);
	// Example values to preview the title; a card without a resource answers
	// with its own params, so those show what the user typed.
	const sampleResponse = $derived(
		Object.fromEntries(
			creatingOutputs.map((o) => [o.key, (paramValues[o.key] ?? '').trim() || o[`example_${getLocale()}`]])
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
			secrets.bindings
		);
		await postItsApi.move(newPostIt.id, creatingNode.position.x, creatingNode.position.y);

		// The create response is the post-it itself: its title is top level.
		const created = newPostIt as unknown as { title?: { text: string; vars: boolean } };
        onPersistNode('create', { ...creatingNode, id: newPostIt.id, data: { title: created.title } } as Node);
		pickedKeys = {};
        creatingTitle = '';
	};

    const updateNode = async () => {
        if (!currentPostIt || missingRequired || titleInvalid) return;

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

        if (!sameParams(creatingParams, params, originalConfig?.paramValues ?? {}) || !sameBindings(secrets.bindings, originalConfig?.bindings ?? {})) {
            await postItsApi.update_well_known_settings(
                currentPostIt.id,
                params,
                secrets.bindings
            );
        }
        if (title !== (currentPostIt.title?.text ?? '')) {
            await postItsApi.update_well_known_title(
                currentPostIt.id,
                { text: title, vars: titleAnalysis.vars }
            );
        }
        onPersistNode('update', { ...currentPostIt, data: { title: { text: title, vars: titleAnalysis.vars } } } as Node);
        pickedKeys = {};
        creatingTitle = '';
    };

    $effect(() => {
        if (currentPostIt) {
            originalConfig = {
                title: currentPostIt.title?.text ?? '',
                paramValues: { ...currentPostIt.paramValues },
                bindings: { ...currentPostIt.bindings },
            };

            creatingTitle = currentPostIt.title?.text ?? '';
            const refOf = (value: string) => {
                const name = value.replace(/^\$/, '');
                return refKey(
                        currentPostIt.bindings?.[name] ?? { scope: { kind: 'board', owner: boardId }, name }
                );
            };
            pickedKeys = Object.fromEntries(
                creatingParams
                        .filter((p) => p.type === 'secret' && currentPostIt.paramValues?.[p.key])
                        .map((p) => [p.key, refOf(currentPostIt.paramValues![p.key])])
            );
        } else {
            originalConfig = null;
            creatingTitle = '';
            pickedKeys = {};
        }
    });
</script>

{#if creatingNode || currentPostIt}
    <Modal
        onclose={() => {
            creatingTitle = '';
            pickedKeys = {};
            onclose();
        }}
        onaccept={() => {
            if (creatingNode) {
                createNode();
            } else if (currentPostIt) {
                updateNode();
            }
        }}
        acceptText={creatingNode ? m['nodes.modify_modal.create.action']() : m['nodes.modify_modal.edit.action']()}
        acceptDisabled={missingRequired || titleInvalid}
    >
        <div class="flex w-full justify-center">
            <h2 class="text-xl font-semibold">{creatingNode ? m['nodes.modify_modal.create.title']() : m['nodes.modify_modal.edit.title']()}</h2>
        </div>
        <Input
            label={m['card_title.label']()}
            placeholder={m['card_title.placeholder']({ example: '{{value}}' })}
            class="focus:*:ring-0! focus:border-red! focus:outline-none!"
            required
            bind:value={creatingTitle}
            maxlength={127}
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
                            title={`${output[`label_${getLocale()}`]}: ${JSON.stringify(sampleResponse[output.key])}`}
                            onclick={() => insertVariable(output.key)}
                        >
                            {`{{${output.key}}}`}
                        </button>
                    {/each}
                </div>
                <span class="text-xs opacity-70">{m['card_title.variables_hint']({ syntax: '{{name}}' })}</span>
            </div>
            <details class={cn("text-xs", horizontalScrollClass)}>
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
                    <span class="font-semibold">{param[`label_${getLocale()}`]}</span>
                    {#if pickable.length === 0}
                        <span class="text-xs text-destructive">{m['secrets.no_credentials']()}</span>
                    {:else}
                        <Dropdown
                            labelClass="font-semibold"
                            placeholder={m['secrets.pick_credential']()}
                            options={Object.fromEntries(
                                pickable.map((group) => [
                                    group.origin,
                                    {
                                        label: m[`secrets.origin_${group.origin}`]() as string,
                                        options: group.secrets.map((secret) => ({
                                            label: `${secret.name} (${secret.provider ?? secret.kind})`,
                                            value: refKey(secret)
                                        }))
                                    }
                                ])
                            )}
                            class="w-full"
                            activatorClass={(isOpen) => cn(
                                "rounded-md border border-border bg-transparent px-3 py-2 w-full!",
                                isOpen ? "outline-none ring-2 ring-main-border border-none" : ""
                            )}
                            drawerClass="rounded-md bg-background!"
                            sectionClass="font-semibold text-foreground!"
                            itemClass={() => "bg-background! rounded-md hover:bg-main-hover!"}
                            bind:value={pickedKeys[param.key]}
                        />
                    {/if}
                </label>
            {:else}
                <Input
                    label={param[`label_${getLocale()}`]!}
                    labelClass="font-semibold"
                    class={cn("focus:*:ring-0 focus:border-main-border focus:outline-none max-h-50!", scrollClass)}
                    placeholder={param[`placeholder_${getLocale()}`]}
                    type={param.type === 'number' ? 'number' : 'text'}
                    required={param.default === undefined}
                    bind:value={paramValues[param.key]}
                    multiline={param.type !== 'number'}
                />
            {/if}
        {/each}
    </Modal>
{/if}
