import { describe, expect, it } from 'vitest';
import { analyzeTitle, renderTitle, titleVariables } from './title';

describe('titleVariables', () => {
	it('lists plain variables once', () => {
		expect(titleVariables('{{min}} to {{ max }} ({{min}})')).toEqual({
			names: ['min', 'max'],
			unsupported: false
		});
	});

	it('rejects sections, partials and delimiter changes', () => {
		expect(titleVariables('{{#min}}x{{/min}}').unsupported).toBe(true);
		expect(titleVariables('{{^min}}x{{/min}}').unsupported).toBe(true);
		expect(titleVariables('{{> other}}').unsupported).toBe(true);
		expect(titleVariables('{{=<% %>=}}').unsupported).toBe(true);
	});

	it('flags a template that does not parse', () => {
		expect(titleVariables('{{#min}} never closed').unsupported).toBe(true);
	});
});

describe('analyzeTitle', () => {
	const outputs = ['min', 'max'];

	it('needs vars only when a real output is used', () => {
		expect(analyzeTitle('Plain', outputs).vars).toBe(false);
		expect(analyzeTitle('{{nope}}', outputs)).toMatchObject({
			used: [],
			unknown: ['nope'],
			vars: false
		});
		expect(analyzeTitle('{{min}} {{nope}}', outputs)).toMatchObject({
			used: ['min'],
			unknown: ['nope'],
			vars: true
		});
	});

	it('never needs vars for an unsupported title', () => {
		expect(analyzeTitle('{{#min}}{{max}}{{/min}}', outputs).vars).toBe(false);
	});
});

describe('renderTitle', () => {
	it('fills in values', () => {
		expect(renderTitle('Between {{min}} and {{max}}', { min: 12.4, max: 20 })).toBe(
			'Between 12.4 and 20'
		);
	});

	it('does not escape, Svelte does', () => {
		expect(renderTitle('{{a}} {{{a}}}', { a: '<b>&</b>' })).toBe('<b>&</b> <b>&</b>');
	});

	it('joins arrays and stringifies objects', () => {
		expect(renderTitle('{{list}} / {{obj}}', { list: ['ARS', 'USD'], obj: { a: 1 } })).toBe(
			'ARS, USD / {"a":1}'
		);
	});

	it('leaves missing variables empty', () => {
		expect(renderTitle('[{{nope}}]', { min: 1 })).toBe('[]');
	});

	it('does not reach through the prototype', () => {
		expect(renderTitle('[{{constructor}}][{{toString}}]', {})).toBe('[][]');
	});

	it('returns unsupported or broken titles as they are', () => {
		expect(renderTitle('{{#min}}x{{/min}}', { min: 1 })).toBe('{{#min}}x{{/min}}');
		expect(renderTitle('{{#min}} never closed', { min: 1 })).toBe('{{#min}} never closed');
	});
});
