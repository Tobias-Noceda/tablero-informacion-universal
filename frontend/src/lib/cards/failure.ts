import { CardError } from '$services/post-it';
import { m } from '$lib/paraglide/messages';

// Why a card could not show fresh data, as what the person should check.
export type CardFailure =
	| { kind: 'source_auth' }
	| { kind: 'not_found' }
	| { kind: 'rate_limited' }
	| { kind: 'source_error'; status: number }
	| { kind: 'timeout' }
	| { kind: 'unreachable' }
	| { kind: 'unreadable' }
	| { kind: 'query' }
	| { kind: 'credential' }
	| { kind: 'unavailable' }
	| { kind: 'network' }
	| { kind: 'unknown' };

const byCode: Record<string, CardFailure> = {
	source_timeout: { kind: 'timeout' },
	source_unreachable: { kind: 'unreachable' },
	source_unreadable: { kind: 'unreadable' },
	query_failed: { kind: 'query' },
	credential_unavailable: { kind: 'credential' },
	card_unavailable: { kind: 'unavailable' },
	network: { kind: 'network' }
};

export function failureOf(err: unknown): CardFailure {
	if (!(err instanceof CardError) || !err.code) return { kind: 'unknown' };
	if (err.code !== 'source_status') return byCode[err.code] ?? { kind: 'unknown' };

	const status = err.sourceStatus ?? 0;
	if (status === 401 || status === 403) return { kind: 'source_auth' };
	if (status === 404 || status === 410) return { kind: 'not_found' };
	if (status === 429) return { kind: 'rate_limited' };
	return { kind: 'source_error', status };
}

export function failureMessage(failure: CardFailure): string {
	switch (failure.kind) {
		case 'source_auth':
			return m['card_failure.source_auth']();
		case 'not_found':
			return m['card_failure.not_found']();
		case 'rate_limited':
			return m['card_failure.rate_limited']();
		case 'source_error':
			return m['card_failure.source_error']({ status: failure.status });
		case 'timeout':
			return m['card_failure.timeout']();
		case 'unreachable':
			return m['card_failure.unreachable']();
		case 'unreadable':
			return m['card_failure.unreadable']();
		case 'query':
			return m['card_failure.query']();
		case 'credential':
			return m['card_failure.credential']();
		case 'unavailable':
			return m['card_failure.unavailable']();
		case 'network':
			return m['card_failure.network']();
		case 'unknown':
			return m['card_failure.unknown']();
	}
}
