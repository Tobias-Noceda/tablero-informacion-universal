import { describe, expect, it } from 'vitest';

import { fold, matchesCard } from './search';

const dolar = {
	key: 'dolar_oficial',
	name_en: 'Official Dollar',
	name_es: 'Dólar oficial',
	description_en: 'The official exchange rate in Argentina',
	description_es: 'La cotización oficial en Argentina'
};

describe('matchesCard', () => {
	it('matches everything when nothing is typed', () => {
		expect(matchesCard(dolar, '')).toBe(true);
		expect(matchesCard(dolar, '   ')).toBe(true);
	});

	it('finds a name in either language, without case or accents', () => {
		expect(matchesCard(dolar, 'DOLLAR')).toBe(true);
		expect(matchesCard(dolar, 'dolar')).toBe(true);
		expect(matchesCard(dolar, 'Dólar')).toBe(true);
	});

	it('finds the description and the key', () => {
		expect(matchesCard(dolar, 'cotizacion')).toBe(true);
		expect(matchesCard(dolar, 'dolar oficial')).toBe(true);
	});

	it('needs every word typed', () => {
		expect(matchesCard(dolar, 'oficial argentina')).toBe(true);
		expect(matchesCard(dolar, 'oficial bitcoin')).toBe(false);
	});

	it('works without a description', () => {
		expect(
			matchesCard({ ...dolar, description_en: undefined, description_es: undefined }, 'official')
		).toBe(true);
	});
});

describe('fold', () => {
	it('drops case and accents', () => {
		expect(fold('Año Ñandú')).toBe('ano nandu');
	});
});
