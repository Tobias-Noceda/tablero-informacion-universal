import { describe, expect, it } from 'vitest';
import { filledParams } from './params';

describe('filledParams', () => {
	it('trims what the person typed and leaves out what they left blank', () => {
		expect(filledParams({ text: '  hello ', city: '', unit: '   ' })).toEqual({ text: 'hello' });
	});

	it('keeps a credential reference as it is', () => {
		expect(filledParams({ $credential: '$MY_KEY' })).toEqual({ $credential: '$MY_KEY' });
	});
});
