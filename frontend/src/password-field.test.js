import { describe, expect, it } from 'vitest';
import {
  passwordFieldValue,
  placeCaretAfterSavedPassword,
  SAVED_PASSWORD_MASK,
  showSavedPassword,
} from './password-field.js';

describe('saved password field', () => {
  it('shows a masked value while preserving the keyring password', () => {
    const input = { value: '', placeholder: '' };
    showSavedPassword(input, true);

    expect(input.value).toBe(SAVED_PASSWORD_MASK);
    expect(input.placeholder).toBe('');
    expect(passwordFieldValue(input, true)).toBe('');
  });

  it('uses changed field content as the replacement password', () => {
    const input = { value: 'replacement', placeholder: '' };
    expect(passwordFieldValue(input, true)).toBe('replacement');
  });

  it('clears only the saved-password mask', () => {
    const input = { value: SAVED_PASSWORD_MASK, placeholder: '' };
    showSavedPassword(input, false);

    expect(input.value).toBe('');
  });

  it('places the caret after the saved-password mask', () => {
    const positions = [];
    const input = {
      value: SAVED_PASSWORD_MASK,
      setSelectionRange: (start, end) => positions.push([start, end]),
    };

    expect(placeCaretAfterSavedPassword(input)).toBe(true);
    expect(positions).toEqual([[SAVED_PASSWORD_MASK.length, SAVED_PASSWORD_MASK.length]]);
  });
});
