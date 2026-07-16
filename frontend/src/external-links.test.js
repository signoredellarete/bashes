import { describe, expect, it } from 'vitest';
import { externalHttpURL } from './external-links.js';

describe('external terminal links', () => {
  it('accepts HTTP links and rejects unsafe or invalid protocols', () => {
    expect(externalHttpURL('https://example.test/article')).toBe('https://example.test/article');
    expect(externalHttpURL('http://example.test')).toBe('http://example.test/');
    expect(externalHttpURL('javascript:alert(1)')).toBe('');
    expect(externalHttpURL('not a URL')).toBe('');
  });
});
