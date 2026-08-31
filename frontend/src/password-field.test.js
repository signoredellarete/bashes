import { describe, expect, it } from 'vitest';
import { SAVED_PASSWORD_PLACEHOLDER, showSavedPassword } from './password-field.js';

describe('saved password field', () => {
  it('shows a placeholder without changing the field value', () => {
    const input = { value: '', placeholder: '' };
    showSavedPassword(input, true);

    expect(input.value).toBe('');
    expect(input.placeholder).toBe(SAVED_PASSWORD_PLACEHOLDER);
  });

  it('clears the placeholder when no password is stored', () => {
    const input = { value: '', placeholder: SAVED_PASSWORD_PLACEHOLDER };
    showSavedPassword(input, false);

    expect(input.placeholder).toBe('');
  });
});
