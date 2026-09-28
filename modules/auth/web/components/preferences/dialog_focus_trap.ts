// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

const NATIVE_FOCUSABLE = new Set(['A', 'BUTTON', 'TEXTAREA', 'INPUT', 'SELECT']);

/** True when the element participates in Tab order inside a dialog. */
export function isTabFocusable(el: HTMLElement): boolean {
  if (el.getAttribute('aria-hidden') === 'true') return false;
  if (el.hasAttribute('hidden')) return false;
  if (
    typeof el.closest === 'function' &&
    (el.closest('[inert]') || el.closest('[hidden]') || el.closest('[aria-hidden="true"]'))
  ) {
    return false;
  }

  const tabindexAttr = el.getAttribute('tabindex');
  if (tabindexAttr != null) {
    // Any negative tabindex removes the element from the tab order.
    const parsed = Number(tabindexAttr);
    if (Number.isFinite(parsed) && parsed < 0) return false;
  }

  const tag = el.tagName;
  const inputType = String(el.getAttribute('type') || (el as HTMLInputElement).type || '').toLowerCase();
  if (tag === 'INPUT' && inputType === 'hidden') return false;

  const disabled =
    el.hasAttribute('disabled') || (el as HTMLButtonElement).disabled === true;

  if (tag === 'A') {
    return !!el.getAttribute('href') && !disabled;
  }
  if (NATIVE_FOCUSABLE.has(tag)) {
    return !disabled;
  }
  // Custom controls: positive/zero tabindex only.
  if (tabindexAttr == null) return false;
  const n = Number(tabindexAttr);
  return Number.isFinite(n) && n >= 0;
}

/** Walk descendants in document order (choysum minimal DOM has no compound selectors). */
function walkElements(root: HTMLElement, visit: (el: HTMLElement) => void): void {
  const kids = root.children;
  for (let i = 0; i < kids.length; i++) {
    const child = kids[i] as HTMLElement;
    visit(child);
    walkElements(child, visit);
  }
}

/** Tab-order focusable descendants inside a modal root (excludes inert/disabled). */
export function listDialogFocusable(root: HTMLElement): HTMLElement[] {
  const out: HTMLElement[] = [];
  walkElements(root, (el) => {
    if (isTabFocusable(el)) out.push(el);
  });
  return out;
}

/**
 * Keep Tab / Shift+Tab cycling inside `root`. Call from a keydown listener on
 * the dialog (or document while open). Non-Tab keys are ignored.
 */
export function trapDialogTabKey(event: KeyboardEvent, root: HTMLElement): void {
  if (event.key !== 'Tab') return;
  const focusable = listDialogFocusable(root);
  if (!focusable.length) {
    event.preventDefault();
    root.focus();
    return;
  }
  const first = focusable[0]!;
  const last = focusable[focusable.length - 1]!;
  const active = document.activeElement as HTMLElement | null;
  const activeIndex = active ? focusable.indexOf(active) : -1;

  if (event.shiftKey) {
    if (activeIndex <= 0) {
      event.preventDefault();
      last.focus();
    }
    return;
  }
  if (activeIndex === -1 || activeIndex >= focusable.length - 1) {
    event.preventDefault();
    first.focus();
  }
}
