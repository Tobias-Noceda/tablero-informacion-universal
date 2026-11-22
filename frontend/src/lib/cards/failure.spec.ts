import { describe, expect, it } from 'vitest';
import { CardError } from '$services/post-it';
import { failureOf } from './failure';

const answered = (code: string, status?: number) => new CardError(502, code, status, 'detail');

describe('failureOf', () => {
	it('tells what to check from the status the source answered', () => {
		expect(failureOf(answered('source_status', 401))).toEqual({ kind: 'source_auth' });
		expect(failureOf(answered('source_status', 403))).toEqual({ kind: 'source_auth' });
		expect(failureOf(answered('source_status', 404))).toEqual({ kind: 'not_found' });
		expect(failureOf(answered('source_status', 410))).toEqual({ kind: 'not_found' });
		expect(failureOf(answered('source_status', 429))).toEqual({ kind: 'rate_limited' });
		expect(failureOf(answered('source_status', 500))).toEqual({
			kind: 'source_error',
			status: 500
		});
	});

	it('reads every other reason the backend gives', () => {
		expect(failureOf(answered('source_timeout'))).toEqual({ kind: 'timeout' });
		expect(failureOf(answered('source_unreachable'))).toEqual({ kind: 'unreachable' });
		expect(failureOf(answered('source_unreadable'))).toEqual({ kind: 'unreadable' });
		expect(failureOf(answered('query_failed'))).toEqual({ kind: 'query' });
		expect(failureOf(new CardError(403, 'credential_unavailable'))).toEqual({ kind: 'credential' });
		expect(failureOf(new CardError(503, 'card_unavailable'))).toEqual({ kind: 'unavailable' });
	});

	it('tells a dropped connection from an unknown answer', () => {
		expect(failureOf(new CardError(0, 'network'))).toEqual({ kind: 'network' });
		expect(failureOf(new CardError(500))).toEqual({ kind: 'unknown' });
		expect(failureOf(new Error('boom'))).toEqual({ kind: 'unknown' });
	});
});
