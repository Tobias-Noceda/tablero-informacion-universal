import type { CardStyle, PostIt } from "$types/api"
import type { Node } from "@xyflow/svelte"
import { cardNodeFields, withoutStyle } from "$lib/cards/style";

export type PostItWithTitle = PostIt & CardStyle & { title: { text: string, vars: boolean } };

// A board card as a canvas node: its size is the node's own (Svelte Flow lays
// the card out with it), its colour and border go in its data.
export const mapPostitWithTitle = (postit: PostItWithTitle): Node => {
    const { look, ...size } = cardNodeFields(postit);
    return {
        ...withoutStyle(postit),
        ...size,
        data: {
            ...postit.data,
            title: postit.title,
            look,
        }
    };
};
