<script lang="ts">
  import {
    BaseEdge,
    getStraightPath,
    useInternalNode,
    type EdgeProps,
  } from '@xyflow/svelte';

  import { getEdgeParams } from './utils';

  let { id, source, target, markerEnd, data }: EdgeProps = $props();

  const isSelected = $derived.by(() => data?.isSelected ?? false);

  const sourceNode = $derived.by(() => useInternalNode(source));
  const targetNode = $derived.by(() => useInternalNode(target));

  let path: string | undefined = $derived.by(() => {
    if (sourceNode.current && targetNode.current) {
      const edgeParams = getEdgeParams(sourceNode.current, targetNode.current);
      return getStraightPath({
        sourceX: edgeParams.sx,
        sourceY: edgeParams.sy,
        targetX: edgeParams.tx,
        targetY: edgeParams.ty,
      })[0];
    }
    throw new Error('Source or target node not found');
  });
</script>

<BaseEdge {id} {path} {markerEnd} style={isSelected ? 'stroke: #727272;' : 'stroke: #3e3e3e;'} />
