export function createEditContextMenu({ menu, readClipboard, writeClipboard, documentRef = document, windowRef = window }) {
  let target = null;

  function open(event) {
    const nextTarget = editableContextTarget(event.target);
    if (!nextTarget || !menu) return;
    event.preventDefault();
    event.stopPropagation();
    target = nextTarget;
    const canWrite = !target.readOnly && !target.disabled;
    menu.querySelector('[data-edit-command="cut"]').disabled = !canWrite;
    menu.querySelector('[data-edit-command="paste"]').disabled = !canWrite;
    menu.hidden = false;
    const rect = menu.getBoundingClientRect();
    menu.style.left = `${Math.max(8, Math.min(event.clientX, windowRef.innerWidth - rect.width - 8))}px`;
    menu.style.top = `${Math.max(8, Math.min(event.clientY, windowRef.innerHeight - rect.height - 8))}px`;
  }

  async function run(event) {
    const button = event.target.closest?.('[data-edit-command]');
    if (!button || button.disabled || !target) return;
    event.preventDefault();
    event.stopPropagation();
    const command = button.dataset.editCommand;
    try {
      if (command === 'copy') {
        await writeClipboard(selectedEditableText(target, windowRef));
      } else if (command === 'cut') {
        await writeClipboard(selectedEditableText(target, windowRef));
        replaceEditableSelection(target, '', documentRef);
      } else if (command === 'paste') {
        const text = await readClipboard();
        if (text) replaceEditableSelection(target, text, documentRef);
      }
    } finally {
      target.focus();
      hide();
    }
  }

  function hide() {
    if (menu) menu.hidden = true;
    target = null;
  }

  return { hide, open, run };
}

export function editableContextTarget(target) {
  const element = target?.closest?.('input, textarea, [contenteditable="true"]');
  if (!element || element.closest('.terminal-pane')) return null;
  if (element.matches('input')) {
    const type = (element.getAttribute('type') || 'text').toLowerCase();
    if (['button', 'checkbox', 'color', 'file', 'hidden', 'radio', 'range', 'reset', 'submit'].includes(type)) return null;
  }
  return element;
}

export function selectedEditableText(target, windowRef = globalThis) {
  if (target.isContentEditable) return String(windowRef.getSelection?.() ?? '');
  const start = target.selectionStart ?? 0;
  const end = target.selectionEnd ?? start;
  return target.value.slice(start, end);
}

export function replaceEditableSelection(target, text, documentRef = globalThis.document) {
  if (target.isContentEditable) {
    documentRef.execCommand('insertText', false, text);
    target.dispatchEvent(createInputEvent(target, 'insertText', text));
    return;
  }
  const start = target.selectionStart ?? target.value.length;
  const end = target.selectionEnd ?? start;
  target.value = `${target.value.slice(0, start)}${text}${target.value.slice(end)}`;
  const next = start + text.length;
  target.setSelectionRange(next, next);
  target.dispatchEvent(createInputEvent(target, text ? 'insertText' : 'deleteByCut', text));
}

function createInputEvent(target, inputType, data) {
  const InputEventClass = target.ownerDocument?.defaultView?.InputEvent ?? globalThis.InputEvent;
  if (InputEventClass) return new InputEventClass('input', { bubbles: true, inputType, data });
  return { type: 'input', bubbles: true, inputType, data };
}
