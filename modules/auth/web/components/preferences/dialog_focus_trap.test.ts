// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { listDialogFocusable, trapDialogTabKey } from './dialog_focus_trap';

function mountDialog(): { root: HTMLElement; first: HTMLButtonElement; last: HTMLButtonElement } {
  const root = document.createElement('div');
  root.setAttribute('tabindex', '-1');
  const first = document.createElement('button');
  first.setAttribute('type', 'button');
  first.textContent = 'Cancel';
  const last = document.createElement('button');
  last.setAttribute('type', 'button');
  last.textContent = 'Save';
  root.appendChild(first);
  root.appendChild(last);
  document.body.appendChild(root);
  return { root, first, last };
}

afterEach(() => {
  document.body.innerHTML = '';
});

test('listDialogFocusable skips tabindex=-1 container and disabled controls', () => {
  const { root, first, last } = mountDialog();
  const disabled = document.createElement('button');
  disabled.setAttribute('disabled', '');
  root.appendChild(disabled);
  const found = listDialogFocusable(root);
  expect(found.length).toBe(2);
  expect(found[0]).toBe(first);
  expect(found[1]).toBe(last);
});

test('trapDialogTabKey wraps forward from last to first', () => {
  const { root, first, last } = mountDialog();
  last.focus();
  const event = {
    key: 'Tab',
    shiftKey: false,
    defaultPrevented: false,
    preventDefault() {
      this.defaultPrevented = true;
    },
  } as unknown as KeyboardEvent;
  trapDialogTabKey(event, root);
  expect(event.defaultPrevented).toBe(true);
  expect(document.activeElement).toBe(first);
});

test('trapDialogTabKey wraps backward from first to last', () => {
  const { root, first, last } = mountDialog();
  first.focus();
  const event = {
    key: 'Tab',
    shiftKey: true,
    defaultPrevented: false,
    preventDefault() {
      this.defaultPrevented = true;
    },
  } as unknown as KeyboardEvent;
  trapDialogTabKey(event, root);
  expect(event.defaultPrevented).toBe(true);
  expect(document.activeElement).toBe(last);
});

test('trapDialogTabKey from container Tab focuses first control', () => {
  const { root, first } = mountDialog();
  root.focus();
  const event = {
    key: 'Tab',
    shiftKey: false,
    defaultPrevented: false,
    preventDefault() {
      this.defaultPrevented = true;
    },
  } as unknown as KeyboardEvent;
  trapDialogTabKey(event, root);
  expect(event.defaultPrevented).toBe(true);
  expect(document.activeElement).toBe(first);
});

test('trapDialogTabKey ignores non-Tab keys', () => {
  const { root, first } = mountDialog();
  first.focus();
  const event = {
    key: 'Escape',
    shiftKey: false,
    defaultPrevented: false,
    preventDefault() {
      this.defaultPrevented = true;
    },
  } as unknown as KeyboardEvent;
  trapDialogTabKey(event, root);
  expect(event.defaultPrevented).toBe(false);
  expect(document.activeElement).toBe(first);
});
