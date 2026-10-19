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