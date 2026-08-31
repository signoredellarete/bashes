export const SAVED_PASSWORD_PLACEHOLDER = '••••••••';

export function showSavedPassword(input, saved) {
  if (!input) return;
  input.placeholder = saved ? SAVED_PASSWORD_PLACEHOLDER : '';
}
