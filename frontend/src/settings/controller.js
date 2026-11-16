import { settingsPanelData } from './model.js';
import {
  clampNumber,
  DEFAULT_TERMINAL_FONT_FAMILY,
  DEFAULT_TERMINAL_FONT_SIZE,
  DEFAULT_TERMINAL_SCROLLBACK,
  terminalAppearancePreview,
} from '../terminal-settings.js';

export function createSettingsController({
  state,
  panel,
  fontOptions,
  limits,
  loadBrowserSettings,
  getPluginInventory,
  saveBrowserSettings,
  setPluginEnabled,
  confirmPluginEnable,
  persistTerminalSettings,
  applyTerminalSettings,
  restoreFocus,
  notifyError,
  notifySuccess,
}) {
  function data() {
    return settingsPanelData(state, fontOptions, limits);
  }

  async function open() {
    try {
      const [, inventory] = await Promise.all([loadBrowserSettings(), getPluginInventory()]);
      state.pluginInventory = inventory;
    } catch (error) {
      notifyError(`Could not load settings: ${errorMessage(error)}`);
    }
    await panel.open(data());
  }

  function close({ restore = true } = {}) {
    if (restore) applyTerminalSettings(state, { focus: false });
    panel.close();
    restoreFocus();
  }

  function render() {
    panel.update(data());
  }

  function preview(values) {
    applyTerminalSettings(terminalAppearancePreview(
      state,
      values.terminalFontSize,
      values.terminalFontFamily,
    ), { focus: false });
  }

  async function submit(values) {
    if (!await applyPlugins(values.plugins)) return false;
    state.terminalFontSize = clampNumber(
      values.terminalFontSize,
      limits.min,
      limits.max,
      DEFAULT_TERMINAL_FONT_SIZE,
    );
    state.terminalScrollback = clampNumber(values.terminalScrollback, 1000, 500000, DEFAULT_TERMINAL_SCROLLBACK);
    state.terminalFontFamily = values.terminalFontFamily || DEFAULT_TERMINAL_FONT_FAMILY;
    state.terminalCopyOnSelect = values.terminalCopyOnSelect;
    try {
      state.webAppBrowserSettings = await saveBrowserSettings({
        preferredBrowserId: values.webAppBrowser,
        customBrowserPath: values.customBrowserPath.trim(),
      });
    } catch (error) {
      notifyError(`Could not save browser settings: ${errorMessage(error)}`);
      return false;
    }
    persistTerminalSettings(state);
    applyTerminalSettings(state);
    notifySuccess('Settings saved.');
    close({ restore: false });
    return true;
  }

  function adjustFontSize(delta) {
    state.terminalFontSize = clampNumber(
      state.terminalFontSize + delta,
      limits.min,
      limits.max,
      DEFAULT_TERMINAL_FONT_SIZE,
    );
    persistTerminalSettings(state);
    applyTerminalSettings(state);
    if (panel.isOpen()) render();
  }

  async function loadPlugins() {
    state.pluginInventory = await getPluginInventory();
  }

  async function applyPlugins(requested = []) {
    const current = new Map((state.pluginInventory.plugins ?? []).map((plugin) => [plugin.id, plugin]));
    const changes = requested.filter((plugin) => current.get(plugin.id)?.enabled !== plugin.enabled);
    const enabling = changes.filter((plugin) => plugin.enabled);
    if (enabling.length > 0 && !await confirmPluginEnable(enabling)) return false;

    const applied = [];
    try {
      for (const plugin of changes) {
        await setPluginEnabled(plugin.id, plugin.enabled);
        applied.push({ id: plugin.id, enabled: !plugin.enabled });
      }
      await loadPlugins();
      return true;
    } catch (error) {
      for (const plugin of applied.reverse()) {
        try {
          await setPluginEnabled(plugin.id, plugin.enabled);
        } catch {
          // The inventory reload below exposes any rollback failure.
        }
      }
      try {
        await loadPlugins();
      } catch {
        // Preserve the original settings error.
      }
      notifyError(`Could not save plugin settings: ${errorMessage(error)}`);
      return false;
    }
  }

  return { adjustFontSize, close, open, preview, render, submit };
}

function errorMessage(error) {
  return error?.message ?? String(error);
}
