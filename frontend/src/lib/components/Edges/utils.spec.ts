import { describe, expect, it } from 'vitest';
import { trimSegment } from './utils';

describe('trimSegment', () => {
	it('pulls each end in along the segment', () => {
		expect(trimSegment({ x: 0, y: 0 }, { x: 10, y: 0 }, 2, 3)).toEqual({
			from: { x: 2, y: 0 },
			to: { x: 7, y: 0 }
		});
	});

	it('leaves an untrimmed end where it was', () => {
		const { from, to } = trimSegment({ x: 0, y: 0 }, { x: 0, y: 10 }, 0, 4);
		expect(from).toEqual({ x: 0, y: 0 });
		expect(to).toEqual({ x: 0, y: 6 });
	});

	it('keeps a segment too short to trim', () => {
		const segment = { from: { x: 0, y: 0 }, to: { x: 3, y: 4 } };
		expect(trimSegment(segment.from, segment.to, 3, 3)).toEqual(segment);
	});
});
