<script lang="ts">
  import { cn } from '$lib/utils';
	import type { Snippet } from 'svelte';

  interface Params {
    variant: 'primary' | 'secondary' | 'tertiary' | 'gray' | 'success' | 'destructive';
    skeleton?: boolean;
    disabled?: boolean;
    children: Snippet;
    onclick: (event: MouseEvent) => void;
    class?: string;
  }

  // Keep reactive object — DO NOT DESTRUCTURE
  let props: Params = $props();

  // reactive derived values
  const variant = $derived(props.variant);
  const skeleton = $derived(props.skeleton ?? false);
  const disabled = $derived(props.disabled ?? false);
  const children = $derived(props.children);
  const onclick = $derived(props.onclick);

  const finalClass = $derived(
    cn(
      'flex items-center justify-center font-semibold rounded-md px-4 py-2',
      props.class,
      'cursor-pointer transition-colors text-base',
      {
        primary: 'bg-main text-main-text hover:bg-main-hover border border-main-border disabled:opacity-50',
        secondary: 'bg-sidebar text-white hover:bg-sidebar-hover border border-sidebar-border disabled:bg-skeleton disabled:border-skeleton disabled:text-gray-500',
        tertiary: 'bg-tertiary text-tertiary-text hover:bg-main-hover hover:text-main-text border border-tertiary-border disabled:opacity-50',
        gray: 'bg-[#6c757d] text-white hover:bg-[#5a6268] border border-[#6c757d] disabled:opacity-50',
        success: 'bg-success text-white hover:bg-success-hover border border-success disabled:opacity-50',
        destructive: 'bg-destructive text-white hover:bg-destructive-hover border border-destructive disabled:opacity-50',
      }[variant],
      skeleton ? '!bg-skeleton animate-pulse !text-transparent cursor-default border border-skeleton' : '',
      disabled ? 'animate-none cursor-not-allowed opacity-60' : '',
      "select-none border-none"
    )
  );
</script>

<button class={finalClass} {onclick} {disabled}>
  {@render children()}
</button>