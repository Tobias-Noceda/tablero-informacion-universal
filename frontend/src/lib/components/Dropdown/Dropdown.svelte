<script lang="ts">
  import { cn } from '$lib/utils';
  import { clickOutside } from '$lib/actions/clickOutside';
	import type { Snippet } from 'svelte';
	import ScrollPagination from '$components/ScrollPagination/ScrollPagination.svelte';
	import type { Paginated } from '$types/api';
	import Avatar from '$components/Avatar/Avatar.svelte';
	import Icon from '$components/Icon/Icon.svelte';

  type SelectOption = {
    value: string;
    label: string;
    icon?: Snippet;
    avatarSrc?: string;
  };

  type GroupedOptions = Record<string, { label: string; options: SelectOption[] }>;

  interface Props {
    label?: string;
    labelClass?: string;
    required?: boolean;
    options?: SelectOption[] | Paginated<SelectOption> | GroupedOptions;
    fetchNextOptions?: () => Promise<Paginated<SelectOption>>;
    value?: string;
    disabled?: boolean;
    skeleton?: boolean;
    placeholder?: string;
    class?: string;
    activatorClass?: (isOpen: boolean) => string;
    drawerClass?: string;
    sectionClass?: string;
    itemClass?: (isSelected: boolean) => string;
    onchange?: (value: string) => void;
    /**
     * Optional slot for rendering a last custom item in the dropdown, such as
     * a "Create new" button. This slot will be rendered after all options and
     * will not be affected by pagination.
     */
    children?: Snippet;
  }

  function isGroupedOptions(options: Props['options']): options is GroupedOptions {
      return (
          typeof options === 'object' &&
          options !== null &&
          !Array.isArray(options) &&
          !('results' in options)
      );
  }

  let {
    label = '',
    labelClass = '',
    required = false,
    options = [],
    fetchNextOptions,
    value = $bindable(),
    disabled = false,
    skeleton = false,
    placeholder = 'Select an option',
    class: selectClass,
    activatorClass,
    drawerClass,
    sectionClass,
    itemClass,
    onchange,
    children
  }: Props = $props();

  let isOpen = $state(false);

  const selectedOption = $derived(
    Array.isArray(options) ? 
      options.find(opt => opt.value === value)
      : isGroupedOptions(options) ? 
        Object.values(options).flatMap(group => group.options).find(opt => opt.value === value)
        : options.results.find(opt => opt.value === value)
  );

  function handleSelect(optionValue: string) {
    value = optionValue;
    isOpen = false;
    onchange?.(optionValue);
  }

  const containerClass = $derived(cn(
    'relative w-60',
    selectClass
  ));

  const buttonClass = $derived(cn(
    'bg-main-hover text-main-text',
    'disabled:opacity-50 disabled:cursor-not-allowed',
    'px-3 py-2.5 rounded-xl transition-colors cursor-pointer',
    'border border-main-border hover:border-sidebar',
    isOpen ? 'border-sidebar' : '',
    activatorClass ? activatorClass(isOpen) : 'w-full',
    'flex items-center justify-between',
    'focus:outline-none focus:ring-none',
    skeleton ? 'bg-skeleton animate-pulse text-transparent cursor-default' : ''
  ));

  const dropdownClass = $derived(cn(
    'absolute z-50 mt-0.5 w-full flex-col',
    'bg-main-hover border border-main-hover rounded-xl shadow-lg',
    'max-h-60 overflow-y-auto',
    // Custom scrollbar styles
    'scrollbar-thin scrollbar-thumb-tertiary scrollbar-track-transparent',
    '[&::-webkit-scrollbar]:w-2',
    '[&::-webkit-scrollbar-track]:bg-transparent',
    '[&::-webkit-scrollbar-thumb]:bg-tertiary',
    '[&::-webkit-scrollbar-thumb]:rounded-full',
    '[&::-webkit-scrollbar-thumb]:hover:bg-tertiary-hover',
    drawerClass,
  ));

  const optionClass = (isSelected: boolean) => cn(
    'w-full justify-start flex items-center text-left',
    'px-3 py-2.5 cursor-pointer transition-colors text-main-text',
    'bg-sidebar hover:bg-sidebar-hover',
    itemClass
      ? itemClass(isSelected)
      : isSelected ? 'bg-sidebar-hover font-medium' : '',
    'overflow-x-hidden'
  );
</script>

<div class="flex flex-col w-full gap-1">
  {#if label}
    <label class={cn("text-sm font-medium text-foreground", labelClass)} for="select">
      {label}
      {#if required}
        <span class="text-red-500">*</span>
      {/if}
    </label>
  {/if}
  <div class={containerClass} use:clickOutside={() => isOpen = false}>
    <button
      type="button"
      class={buttonClass}
      {disabled}
      onclick={() => !disabled && !skeleton && (isOpen = !isOpen)}
    >
      <span class={cn('min-w-0 flex-1 truncate text-start!', selectedOption ? '' : 'text-gray-400')}>
        {selectedOption ? selectedOption.label : placeholder}
      </span>
      <Icon name={isOpen ? 'chevron-up' : 'chevron-down'} class="w-4 h-4 ml-2" />
    </button>

    {#if isOpen && !disabled && !skeleton}
      <div class={dropdownClass}>
        {#if fetchNextOptions && !Array.isArray(options) && !isGroupedOptions(options)}
          <ScrollPagination
            initialItems={options}
            nextFetchFunction={fetchNextOptions}
            key={(option) => option.value}
          >
            {#snippet loading()}
              <div class="px-3 py-2.5">
                <div class="w-full h-4 bg-skeleton animate-pulse rounded-md"></div>
              </div>
            {/snippet}
            {#snippet children(option: SelectOption, index: number)}  
              <button
                tabindex={index}
                class={optionClass(option.value === value)}
                onclick={() => handleSelect(option.value)}
                role="option"
                aria-selected={option.value === value}
              >
                {#if option.icon}
                  {@render option.icon()}
                {:else if option.avatarSrc}
                  <Avatar size="sm" src={option.avatarSrc} class="mr-2" />
                {/if}
                <span class="min-w-0 flex-1 truncate">{option.label}</span>
              </button>
            {/snippet}
          </ScrollPagination>
        {:else if Array.isArray(options)}
          {#each options as option, index (option.value)}
            <button
              tabindex={index}
              class={optionClass(option.value === value)}
              onclick={() => handleSelect(option.value)}
              role="option"
              aria-selected={option.value === value}
            >
              {#if option.icon}
                {@render option.icon()}
              {:else if option.avatarSrc}
                <Avatar size="sm" src={option.avatarSrc} class="mr-2" />
              {/if}
              <span class="min-w-0 flex-1 truncate">{option.label}</span>
            </button>
          {/each}
        {:else if isGroupedOptions(options)}
          {#each Object.entries(options) as [groupKey, group] (groupKey)}
            <div class={cn("px-3 py-2 text-sm font-medium text-gray-500", sectionClass)}>{group.label}</div>
            {#each group.options as option, index (option.value)}
              <button
                tabindex={index}
                class={optionClass(option.value === value)}
                onclick={() => handleSelect(option.value)}
                role="option"
                aria-selected={option.value === value}
              >
                {#if option.icon}
                  {@render option.icon()}
                {:else if option.avatarSrc}
                  <Avatar size="sm" src={option.avatarSrc} class="mr-2" />
                {/if}
                <span class="min-w-0 flex-1 truncate">{option.label}</span>
              </button>
            {/each}
          {/each}
        {/if}
        {#if children}
          {@render children()}
        {/if}
      </div>
    {/if}
  </div>
</div>