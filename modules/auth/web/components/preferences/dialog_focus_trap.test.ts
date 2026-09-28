// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { isTabFocusable, listDialogFocusable, trapDialogTabKey } from './dialog_focus_trap';

type FakeKeyEvent = {
  key: string;
  shiftKey: boolean;
  defaultPrevented: boolean;
  preventDefault: () => void;
};

function makeKeyEvent(key: string, shiftKey = false): FakeKeyEvent {
  const event: FakeKeyEvent = {
    key,
    shiftKey,
    defaultPrevented: false,
    preventDefault() {
      event.defaultPrevented = true;
    },
  };
  return event;
}

function mountDialog(extraButtons = 0): {
  root: HTMLElement;
  first: HTMLButtonElement;
  middle: HTMLButtonElement | null;
  last: HTMLButtonElement;
} {
  const root = document.createElement('div');
  root.setAttribute('tabindex', '-1');
  const first = document.createElement('button');
  first.setAttribute('type', 'button');
  first.textContent = 'Cancel';
  root.appendChild(first);
  let middle: HTMLButtonElement | null = null;
  for (let i = 0; i < extraButtons; i++) {
    const btn = document.createElement('button');
    btn.setAttribute('type', 'button');
    btn.textContent = `Mid-${i}`;
    root.appendChild(btn);
    if (i === 0) middle = btn;
  }
  const last = document.createElement('button');
  last.setAttribute('type', 'button');
  last.textContent = 'Save';
  root.appendChild(last);
  document.body.appendChild(root);
  return { root, first, middle, last };
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

test('isTabFocusable: rejects aria-hidden controls', () => {
  const btn = document.createElement('button');
  btn.setAttribute('aria-hidden', 'true');
  document.body.appendChild(btn);
  expect(isTabFocusable(btn)).toBe(false);
});

test('isTabFocusable: rejects controls under inert ancestors', () => {
  const host = document.createElement('div');
  host.setAttribute('inert', '');
  const btn = document.createElement('button');
  host.appendChild(btn);
  document.body.appendChild(host);
  expect(isTabFocusable(btn)).toBe(false);
});

test('isTabFocusable: rejects hidden inputs and bare anchors', () => {
  const hiddenInput = document.createElement('input');
  hiddenInput.setAttribute('type', 'hidden');
  document.body.appendChild(hiddenInput);
  expect(isTabFocusable(hiddenInput)).toBe(false);

  const bareAnchor = document.createElement('a');
  document.body.appendChild(bareAnchor);
  expect(isTabFocusable(bareAnchor)).toBe(false);

  const linked = document.createElement('a');
  linked.setAttribute('href', '#');
  document.body.appendChild(linked);
  expect(isTabFocusable(linked)).toBe(true);
});

test('isTabFocusable: custom tabindex and plain elements', () => {
  const custom = document.createElement('div');
  custom.setAttribute('tabindex', '0');
  document.body.appendChild(custom);
  expect(isTabFocusable(custom)).toBe(true);

  const badTab = document.createElement('div');
  badTab.setAttribute('tabindex', 'NaN');
  document.body.appendChild(badTab);
  expect(isTabFocusable(badTab)).toBe(false);

  const plain = document.createElement('div');
  document.body.appendChild(plain);
  expect(isTabFocusable(plain)).toBe(false);
});

test('isTabFocusable: rejects any negative tabindex on native controls', () => {
  const minusOne = document.createElement('button');
  minusOne.setAttribute('tabindex', '-1');
  document.body.appendChild(minusOne);
  expect(isTabFocusable(minusOne)).toBe(false);

  const minusTwo = document.createElement('button');
  minusTwo.setAttribute('tabindex', '-2');
  document.body.appendChild(minusTwo);
  expect(isTabFocusable(minusTwo)).toBe(false);
});

test('trapDialogTabKey wraps forward from last to first', () => {
  const { root, first, last } = mountDialog();
  last.focus();
  const event = makeKeyEvent('Tab');
  trapDialogTabKey(event as unknown as KeyboardEvent, root);
  expect(event.defaultPrevented).toBe(true);
  expect(document.activeElement).toBe(first);
});

test('trapDialogTabKey wraps backward from first to last', () => {
  const { root, first, last } = mountDialog();
  first.focus();
  const event = makeKeyEvent('Tab', true);
  trapDialogTabKey(event as unknown as KeyboardEvent, root);
  expect(event.defaultPrevented).toBe(true);
  expect(document.activeElement).toBe(last);
});

test('trapDialogTabKey from container Tab focuses first control', () => {
  const { root, first } = mountDialog();
  root.focus();
  const event = makeKeyEvent('Tab');
  trapDialogTabKey(event as unknown as KeyboardEvent, root);
  expect(event.defaultPrevented).toBe(true);
  expect(document.activeElement).toBe(first);
});

test('trapDialogTabKey ignores non-Tab keys', () => {
  const { root, first } = mountDialog();
  first.focus();
  const event = makeKeyEvent('Escape');
  trapDialogTabKey(event as unknown as KeyboardEvent, root);
  expect(event.defaultPrevented).toBe(false);
  expect(document.activeElement).toBe(first);
});

test('trapDialogTabKey focuses root when dialog has no tab stops', () => {
  const root = document.createElement('div');
  root.setAttribute('tabindex', '-1');
  document.body.appendChild(root);
  root.focus();
  const event = makeKeyEvent('Tab');
  trapDialogTabKey(event as unknown as KeyboardEvent, root);
  expect(event.defaultPrevented).toBe(true);
  expect(document.activeElement).toBe(root);
});

test('trapDialogTabKey leaves middle controls to the browser', () => {
  const { root, first, middle, last } = mountDialog(1);
  expect(middle).not.toBeNull();
  middle!.focus();
  const forward = makeKeyEvent('Tab');
  trapDialogTabKey(forward as unknown as KeyboardEvent, root);
  expect(forward.defaultPrevented).toBe(false);
  expect(document.activeElement).toBe(middle);

  middle!.focus();
  const backward = makeKeyEvent('Tab', true);
  trapDialogTabKey(backward as unknown as KeyboardEvent, root);
  expect(backward.defaultPrevented).toBe(false);
  expect(document.activeElement).toBe(middle);

  expect(first).not.toBe(last);
});
