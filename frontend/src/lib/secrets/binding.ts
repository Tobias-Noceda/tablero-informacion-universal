import type { SecretMeta, SecretRef, UUID } from '$types/api';

// A stable identity for a picked secret; the select option value.
export function refKey(secret: Pick<SecretMeta, 'scope' | 'name'>) {
	return `${secret.scope.kind}|${secret.scope.owner ?? ''}|${secret.name}`;
}

// What a card sends for the secrets picked per parameter: the "$NAME"
// reference, plus a binding for anything the board's own scope cannot
// resolve by name.
export function bindingsFor(picked: Record<string, SecretMeta>, board: UUID) {
	const params: Record<string, string> = {};
	const bindings: Record<string, SecretRef> = {};

	for (const [param, secret] of Object.entries(picked)) {
		params[param] = `$${secret.name}`;

		const inBoard = secret.scope.kind === 'board' && secret.scope.owner === board;
		if (!inBoard) {
			bindings[secret.name] = { scope: secret.scope, name: secret.name };
		}
	}

	return { params, bindings };
}
