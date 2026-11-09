import { describe, expect, it } from 'vitest';
import { settingsPanelData } from './model.js';

describe('settings panel model', () => {
  it('marks browsers that are not detected', () => {
    const data = settingsPanelData({
      terminalFontSize: 14,
      terminalScrollback: 100000,
      terminalFontFamily: 'monospace',
      terminalCopyOnSelect: true,
      webAppBrowsers: [{ id: 'firefox' }],
      webAppBrowserSettings: {},
    }, [{ label: 'System monospace', value: 'monospace' }], { min: 8, max: 36 });

    expect(data.browserOptions.find(({ value }) => value === 'firefox')?.label).toBe('Firefox');
    expect(data.browserOptions.find(({ value }) => value === 'chrome')?.label).toBe('Google Chrome (not detected)');
  });

  it('preserves a previously configured terminal font', () => {
    const data = settingsPanelData({
      terminalFontSize: 14,
      terminalScrollback: 100000,
      terminalFontFamily: 'Custom Mono',
      terminalCopyOnSelect: false,
      webAppBrowsers: [],
      webAppBrowserSettings: { preferredBrowserId: 'firefox', customBrowserPath: '/browser' },
    }, [{ label: 'System monospace', value: 'monospace' }], { min: 8, max: 36 });

    expect(data.fontOptions).toContainEqual({ label: 'Previously configured font', value: 'Custom Mono' });
    expect(data.webAppBrowser).toBe('firefox');
    expect(data.customBrowserPath).toBe('/browser');
  });
});
