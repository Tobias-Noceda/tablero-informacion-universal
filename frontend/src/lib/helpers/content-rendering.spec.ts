import { describe, expect, it } from 'vitest';
import { githubRepoPath, githubRepoUrl } from './content-rendering';

describe('githubRepoUrl', () => {
	it.each(['itchyny/gojq', '/itchyny/gojq', '//itchyny///gojq/', ' itchyny/gojq '])(
		'links %j to the repository',
		(query) => {
			expect(githubRepoUrl(query)).toBe('https://github.com/itchyny/gojq');
			expect(githubRepoPath(query)).toBe('itchyny/gojq');
		}
	);

	it('keeps dots, dashes and underscores', () => {
		expect(githubRepoUrl('vercel/next.js')).toBe('https://github.com/vercel/next.js');
		expect(githubRepoUrl('my-org/my_repo')).toBe('https://github.com/my-org/my_repo');
	});

	it.each(['gojq language:go', 'gojq', '', '/', 'a/b/c', 'owner/repo?x=1', 42, null, undefined])(
		'does not link %j',
		(query) => {
			expect(githubRepoUrl(query)).toBeNull();
		}
	);
});
