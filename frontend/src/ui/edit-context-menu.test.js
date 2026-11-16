import { describe, expect, it, vi } from 'vitest';
import { replaceEditableSelection, selectedEditableText } from './edit-context-menu.js';

function input(value, start, end) {
  const target = {
    value,
    selectionStart: start,
    selectionEnd: end,
    dispatchEvent: vi.fn(),
  };
  target.setSelectionRange = vi.fn((nextStart, nextEnd) => {
    target.selectionStart = nextStart;
    target.selectionEnd = nextEnd;
  });
  return target;
}

describe('editable selection helpers', () => {
  it('reads the selected input text', () => {
    expect(selectedEditableText(input('alpha beta', 6, 10))).toBe('beta');
  });

  it('replaces the selected input text and moves the caret', () => {
    const target = input('alpha beta', 6, 10);
    replaceEditableSelection(target, 'gamma');
    expect(target.value).toBe('alpha gamma');
    expect(target.setSelectionRange).toHaveBeenCalledWith(11, 11);
    expect(target.dispatchEvent).toHaveBeenCalledOnce();
  });
});
