import type { PostIt } from "$types/api"
import type { Node } from "@xyflow/svelte"

export type PostItWithTitle = PostIt & { title: { text: string, vars: boolean } };

export const mapPostitWithTitle = (postit: PostItWithTitle): Node => {
    return {
        ...postit,
        data: {
            ...postit.data,
            title: postit.title,
        }
    };
};
