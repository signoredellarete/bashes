export function createFieldHelpController(root = document) {
  function close(except = null) {
    root.querySelectorAll('.field-help-trigger[aria-expanded="true"]').forEach((button) => {
      if (button !== except) button.setAttribute('aria-expanded', 'false');
    });
  }

  function toggle(event) {
    event.preventDefault();
    event.stopPropagation();
    const button = event.currentTarget;
    const willOpen = button.getAttribute('aria-expanded') !== 'true';
    close(button);
    button.setAttribute('aria-expanded', willOpen ? 'true' : 'false');
  }

  function bind(buttons) {
    buttons.forEach((button) => {
      button.addEventListener('click', toggle);
      button.addEventListener('keydown', (event) => {
        if (event.key === 'Enter' || event.key === ' ') toggle(event);
      });
    });
  }

  return { bind, close, toggle };
}
