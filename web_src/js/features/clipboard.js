const selector = '[data-clipboard-target], [data-clipboard-text]';

// TODO: replace these with toast-style notifications
function onSuccess(btn) {
  if (!btn.dataset.content) return;
  $(btn).popup('destroy');
  btn.dataset.content = btn.dataset.success;
  $(btn).popup('show');
  btn.dataset.content = btn.dataset.original;
}
function onError(btn) {
  if (!btn.dataset.content) return;
  $(btn).popup('destroy');
  btn.dataset.content = btn.dataset.error;
  $(btn).popup('show');
  btn.dataset.content = btn.dataset.original;
}

/**
 * Fallback to use if navigator.clipboard doesn't exist.
 * Achieved via creating a temporary textarea element, selecting the text, and using document.execCommand.
 */
function fallbackCopyToClipboard(text) {
  if (!document.execCommand) return false;

  const tempTextArea = document.createElement('textarea');
  tempTextArea.value = text;

  // avoid scrolling
  tempTextArea.style.top = 0;
  tempTextArea.style.left = 0;
  tempTextArea.style.position = 'fixed';

  document.body.appendChild(tempTextArea);

  tempTextArea.select();

  // if unsecure (not https), there is no navigator.clipboard, but we can still use document.execCommand to copy to clipboard
  const success = document.execCommand('copy');

  document.body.removeChild(tempTextArea);

  return success;
}

export default async function initClipboard() {
  for (const btn of document.querySelectorAll(selector) || []) {
    btn.addEventListener('click', async () => {
      let text;
      if (btn.dataset.clipboardText) {
        text = btn.dataset.clipboardText;
      } else if (btn.dataset.clipboardTarget) {
        text = document.querySelector(btn.dataset.clipboardTarget)?.value;
      }
      if (!text) return;

      try {
        await navigator.clipboard.writeText(text);
        onSuccess(btn);
      } catch {
        if (fallbackCopyToClipboard(text)) {
          onSuccess(btn);
        } else {
          onError(btn);
        }
      }
    });
  }
}
