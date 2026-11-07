export function capitalize(str: string): string {
    if (!str) return str;
    return str.charAt(0).toUpperCase() + str.slice(1);
}

export function isLink(value: unknown): value is string {
    return typeof value === 'string' && /^https?:\/\//.test(value);
};

export function isImageLink(value: unknown): value is string {
    return isLink(value) && /\.(jpg|jpeg|png|gif|bmp|webp|svg)$/i.test(value);
};
/**
 * The `owner/repo` a github_repo query names, with leading, trailing and
 * repeated `/` collapsed ("/owner//repo" works in the search the card runs).
 * Null when the query is not one: it is a GitHub search, so it may be free
 * text such as "gojq language:go".
 */
export function githubRepoPath(value: unknown): string | null {
    if (typeof value !== 'string') return null;
    const path = value.trim().replace(/\/+/g, '/').replace(/^\/|\/$/g, '');
    return /^[\w.-]+\/[\w.-]+$/.test(path) ? path : null;
}

/** The repository page of a github_repo query; null when it names none. */
export function githubRepoUrl(value: unknown): string | null {
    const path = githubRepoPath(value);
    return path ? `https://github.com/${path}` : null;
}
