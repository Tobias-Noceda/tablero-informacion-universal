import DOMPurify from 'dompurify';
import { marked } from 'marked';

// Links in a note are external addresses: open them outside the board.
DOMPurify.addHook('afterSanitizeAttributes', (node) => {
	if (node.tagName === 'A' && node.hasAttribute('href')) {
		node.setAttribute('target', '_blank');
		node.setAttribute('rel', 'noopener noreferrer');
	}
});

/**
 * A note's text as HTML. Boards are shared, so the text may come from
 * another user: the HTML is always sanitized before it reaches `{@html}`.
 */
export function renderMarkdown(text: string | null | undefined): string {
	const html = marked.parse(text ?? '', { async: false, gfm: true, breaks: true });
	return DOMPurify.sanitize(html);
}
