import { describe, expect, it } from 'vitest';
import { withoutQuery } from './speed-insights';

describe('withoutQuery', () => {
	it('drops the query and the fragment, which can carry one-time tokens', () => {
		expect(
			withoutQuery({ type: 'vital', url: 'https://corcheria.company/verify?token=secret' })
		).toEqual({
			type: 'vital',
			url: 'https://corcheria.company/verify'
		});
		expect(
			withoutQuery({
				type: 'vital',
				url: 'https://corcheria.company/oauth2/callback?code=abc&state=xyz#done',
				route: '/oauth2/callback'
			})
		).toEqual({
			type: 'vital',
			url: 'https://corcheria.company/oauth2/callback',
			route: '/oauth2/callback'
		});
	});

	it('keeps the path, which is what a route is measured by', () => {
		const event = {
			type: 'vital' as const,
			url: 'https://corcheria.company/boards/42',
			route: '/boards/[id]'
		};
		expect(withoutQuery(event)).toEqual(event);
	});

	it('drops an event whose address cannot be read rather than send it as is', () => {
		expect(withoutQuery({ type: 'vital', url: 'not a url?token=secret' })).toBeNull();
	});
});
