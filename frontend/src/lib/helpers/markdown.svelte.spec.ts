// DOMPurify needs a DOM: runs in the browser (client) project.
import { describe, expect, it } from 'vitest';
import { renderMarkdown } from './markdown';

describe('renderMarkdown', () => {
	it('renders headings, emphasis and lists', () => {
		expect(renderMarkdown('# Hi')).toContain('<h1>Hi</h1>');
		expect(renderMarkdown('**b**')).toContain('<strong>b</strong>');
		expect(renderMarkdown('- a\n- b')).toContain('<li>a</li>');
	});

	it('keeps single line breaks', () => {
		expect(renderMarkdown('a\nb')).toContain('<br>');
	});

	it('handles missing text', () => {
		expect(renderMarkdown(undefined)).toBe('');
		expect(renderMarkdown(null)).toBe('');
	});

	it('strips scripts and event handlers', () => {
		const html = renderMarkdown('<script>alert(1)</script><img src="x" onerror="alert(1)">');
		expect(html).not.toContain('<script');
		expect(html).not.toContain('onerror');
	});

	it('drops javascript: links', () => {
		expect(renderMarkdown('[x](javascript:alert(1))')).not.toContain('javascript:');
	});

	it('opens links in a new tab', () => {
		const html = renderMarkdown('[gojq](https://github.com/itchyny/gojq)');
		expect(html).toContain('href="https://github.com/itchyny/gojq"');
		expect(html).toContain('target="_blank"');
		expect(html).toContain('rel="noopener noreferrer"');
	});
});
