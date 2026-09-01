export const SAVED_PASSWORD_MASK = '••••••••';

export function showSavedPassword(input, saved) {
  if (!input) return;
  input.placeholder = '';
  if (saved) {
    input.value = SAVED_PASSWORD_MASK;
  } else if (input.value === SAVED_PASSWORD_MASK) {
    input.value = '';
  }
}

export function passwordFieldValue(input, hadSavedPassword) {
  if (!input) return '';
  return hadSavedPassword && input.value === SAVED_PASSWORD_MASK ? '' : input.value;
}

export function placeCaretAfterSavedPassword(input) {
  if (!input || input.value !== SAVED_PASSWORD_MASK || typeof input.setSelectionRange !== 'function') return false;
  input.setSelectionRange(input.value.length, input.value.length);
  return true;
}
