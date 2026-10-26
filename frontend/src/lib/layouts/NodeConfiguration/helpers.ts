import { refKey } from "$lib/secrets/binding";
import type { SecretRef } from "$types/api";

// Key order and whitespace aside, the same settings.
export const sameParams = (creatingParams: { key: string }[], a: Record<string, string>, b: Record<string, string>) =>
        creatingParams.every((p) => (a[p.key] ?? '').trim() === (b[p.key] ?? '').trim());

// Same names bound to the same secrets; scope objects can differ in key order.
export const sameBindings = (a: Record<string, SecretRef>, b: Record<string, SecretRef>) => {
        const keys = (r: Record<string, SecretRef>) =>
                Object.entries(r).map(([name, ref]) => `${name}=${refKey(ref)}`).sort().join(',');
        return keys(a) === keys(b);
};