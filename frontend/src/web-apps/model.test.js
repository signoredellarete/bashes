import { describe, expect, it } from 'vitest';
import { isWebAppResource, webAppMatchesFilters } from './model.js';

const webApp = {
  id: 'webapp-1',
  name: 'Grafana Internal',
  url: 'https://grafana.internal',
  connectionMode: 'tunnel',
  tags: ['Production', 'Metrics'],
};

describe('Web App model', () => {
  it('recognizes Web Apps without treating SSH resources as Web Apps', () => {
    expect(isWebAppResource(webApp)).toBe(true);
    expect(isWebAppResource({ id: 'host-1', hostname: 'host' })).toBe(false);
  });

  it('filters by name, URL, and any/all tags', () => {
    expect(webAppMatchesFilters(webApp, { query: 'grafana' })).toBe(true);
    expect(webAppMatchesFilters(webApp, { query: 'internal' })).toBe(true);
    expect(webAppMatchesFilters(webApp, { activeTags: ['production', 'missing'], mode: 'any' })).toBe(true);
    expect(webAppMatchesFilters(webApp, { activeTags: ['production', 'missing'], mode: 'all' })).toBe(false);
  });
});
