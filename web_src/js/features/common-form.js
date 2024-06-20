import $ from 'jquery';
import {handleGlobalEnterQuickSubmit} from './comp/QuickSubmit.js';

export function initGlobalFormDirtyLeaveConfirm() {
  // Warn users that try to leave a page after entering data into a form.
  // Except on sign-in pages, and for forms marked as 'ignore-dirty'.
  if (!$('.user.signin').length) {
    $('form:not(.ignore-dirty)').areYouSure();
  }
}

export function initGlobalEnterQuickSubmit() {
  document.addEventListener('keydown', (e) => {
    if (e.key !== 'Enter') return;
    const hasCtrlOrMeta = ((e.ctrlKey || e.metaKey) && !e.altKey);
    if (hasCtrlOrMeta && e.target.matches('textarea')) {
      if (handleGlobalEnterQuickSubmit(e.target)) {
        e.preventDefault();
      }
    } else if (e.target.matches('input') && !e.target.closest('form')) {
      // input in a normal form could handle Enter key by default, so we only handle the input outside a form
      // eslint-disable-next-line unicorn/no-lonely-if
      if (handleGlobalEnterQuickSubmit(e.target)) {
        e.preventDefault();
      }
    }
  });
}
