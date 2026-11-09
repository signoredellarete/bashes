const BROWSER_NAMES = new Map([
  ['firefox', 'Firefox'],
  ['chrome', 'Google Chrome'],
  ['chromium', 'Chromium'],
  ['edge', 'Microsoft Edge'],
]);

export function settingsPanelData(state, terminalFontOptions, limits) {
  const fontOptions = terminalFontOptions.map(({ label, value }) => ({ label, value }));
  if (!terminalFontOptions.some(({ value }) => value === state.terminalFontFamily)) {
    fontOptions.push({ label: 'Previously configured font', value: state.terminalFontFamily });
  }

  const detected = new Set(state.webAppBrowsers.map((browser) => browser.id));
  const browserOptions = [{ label: 'Automatic (Firefox preferred)', value: '' }];
  for (const [id, name] of BROWSER_NAMES) {
    browserOptions.push({
      label: `${name}${detected.has(id) ? '' : ' (not detected)'}`,
      value: id,
    });
  }

  return {
    terminalFontSize: state.terminalFontSize,
    terminalScrollback: state.terminalScrollback,
    terminalFontFamily: state.terminalFontFamily,
    terminalCopyOnSelect: state.terminalCopyOnSelect,
    webAppBrowser: state.webAppBrowserSettings.preferredBrowserId || '',
    customBrowserPath: state.webAppBrowserSettings.customBrowserPath || '',
    fontOptions,
    browserOptions,
    minTerminalFontSize: limits.min,
    maxTerminalFontSize: limits.max,
  };
}
