<script lang="ts">
  import { cn } from '$lib/utils';
  import { iconMap } from './icon-map';

  export type IconNames = keyof typeof iconMap;

  interface Props {
    name: IconNames;
    class?: string;
    // Stroke width only applies if the component detects a stroke-based source SVG
    strokeWidth?: number;
  }

  let { name, class: className = '', strokeWidth = 1.7 }: Props = $props();

  let svgData = $state<{ 
    viewBox: string; 
    body: string; 
    isStrokeStyle: boolean; 
  } | null>(null);

  $effect(() => {
    const src = iconMap[name];
    
    if (typeof src !== 'string') {
      svgData = null;
      return;
    }
    
    fetch(src)
      .then((res) => res.text())
      .then((raw) => {
        // 1. Extract the opening tag for analysis
        const openTagMatch = raw.match(/<svg([^>]*)>/i);
        const rootAttributes = openTagMatch ? openTagMatch[1] : '';

        // 2. Detect Intent: Did source explicitly set fill="none"?
        // If yes, we treat it as line art and will manipulate strokeWidth.
        const isStrokeStyle = /fill\s*=\s*['"]none['"]/i.test(rootAttributes);

        // 3. Extract viewBox
        const viewBoxMatch = rootAttributes.match(/viewBox=["']([^"']+)["']/i);
        const viewBox = viewBoxMatch ? viewBoxMatch[1] : '0 0 24 24';

        // 4. Extract inner body
        const bodyMatch = raw.match(/<svg[^>]*>([\s\S]*?)<\/svg>/i);
        let body = bodyMatch ? bodyMatch[1] : '';

        // 5. Conditional Manipulation
        if (isStrokeStyle) {
          // Standardize line art properties
          body = body.replace(/stroke=["'][^"']*["']/g, 'stroke="currentColor"');
          body = body.replace(/stroke-width=["'][^"']*["']/g, `stroke-width="${strokeWidth}"`);
        } else {
          // For fill/glyph icons, standardize inner fills to inherit currentColor
          body = body.replace(/fill=["'][^"']*["']/g, 'fill="currentColor"');
        }

        svgData = { viewBox, body, isStrokeStyle };
      });
  });

  const finalClass = $derived(cn(
    'inline-block align-middle select-none',
    /\b(w-|h-)/.test(className) ? '' : 'w-4 h-4', // default size
    className
  ));
</script>

{#if svgData}
  <svg 
    viewBox={svgData.viewBox} 
    class={finalClass}
    stroke={svgData.isStrokeStyle ? "currentColor" : "none"}
  >
    {@html svgData.body}
  </svg>
{/if}