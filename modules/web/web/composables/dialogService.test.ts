// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { defineComponent } from 'vue';
import {
  closeAllDialogs,
  closeDialog,
  onDialogInterrupt,
  openDialog,
  resetDialogServiceForTests,
  useDialogStack,
} from './dialogService.ts';

const StubA = defineComponent({ name: 'StubA', template: '<div />' });
const StubB = defineComponent({ name: 'StubB', template: '<div />' });

describe('dialogService', () => {
  beforeEach(() => {
    resetDialogServiceForTests();
  });

  test('child pushes without clearing the parent', () => {
    const stack = useDialogStack();
    openDialog({ component: StubA, mode: 'child' });
    openDialog({ component: StubB, mode: 'child' });
    expect(stack.value).toHaveLength(2);
    expect(stack.value[0]?.component).toBe(StubA);
    expect(stack.value[1]?.component).toBe(StubB);
  });

  test('replace clears the stack then opens one entry', () => {
    const stack = useDialogStack();
    openDialog({ component: StubA, mode: 'child' });
    openDialog({ component: StubB, mode: 'replace' });
    expect(stack.value).toHaveLength(1);
    expect(stack.value[0]?.component).toBe(StubB);
    expect(stack.value[0]?.mode).toBe('replace');
  });

  test('replace carries opener from the cleared bottom entry', () => {
    const stack = useDialogStack();
    const pageBtn = document.createElement('button');
    document.body.appendChild(pageBtn);
    pageBtn.focus();
    openDialog({ component: StubA, mode: 'child' });

    const innerBtn = document.createElement('button');
    document.body.appendChild(innerBtn);
    innerBtn.focus();
    openDialog({ component: StubB, mode: 'child' });

    openDialog({ component: StubA, mode: 'replace' });
    expect(stack.value).toHaveLength(1);
    expect(stack.value[0]?.opener).toBe(pageBtn);
    closeDialog();
    expect(document.activeElement).toBe(pageBtn);
    pageBtn.remove();
    innerBtn.remove();
  });

  test('closeAllDialogs restores the outermost opener', () => {
    const pageBtn = document.createElement('button');
    document.body.appendChild(pageBtn);
    pageBtn.focus();
    openDialog({ component: StubA });

    const innerBtn = document.createElement('button');
    document.body.appendChild(innerBtn);
    innerBtn.focus();
    openDialog({ component: StubB });

    const other = document.createElement('button');
    document.body.appendChild(other);
    other.focus();
    closeAllDialogs();
    expect(document.activeElement).toBe(pageBtn);
    pageBtn.remove();
    innerBtn.remove();
    other.remove();
  });

  test('closeDialog by key does not restore opener for a buried entry', () => {
    const pageBtn = document.createElement('button');
    document.body.appendChild(pageBtn);
    pageBtn.focus();
    const k1 = openDialog({ component: StubA });

    const innerBtn = document.createElement('button');
    document.body.appendChild(innerBtn);
    innerBtn.focus();
    openDialog({ component: StubB });

    const other = document.createElement('button');
    document.body.appendChild(other);
    other.focus();
    closeDialog(k1);
    expect(document.activeElement).toBe(other);
    pageBtn.remove();
    innerBtn.remove();
    other.remove();
  });

  test('interrupt clears stack, runs handlers, then opens', () => {
    const stack = useDialogStack();
    let calls = 0;
    const stop = onDialogInterrupt(() => {
      calls += 1;
    });
    openDialog({ component: StubA, mode: 'child' });
    openDialog({ component: StubB, mode: 'interrupt' });
    expect(calls).toBe(1);
    expect(stack.value).toHaveLength(1);
    expect(stack.value[0]?.component).toBe(StubB);
    stop();
  });

  test('openDialog requires component', () => {
    expect(() => openDialog({} as any)).toThrow(/component is required/);
  });

  test('closeDialog removes by key or pops the top', () => {
    const stack = useDialogStack();
    const k1 = openDialog({ component: StubA });
    const k2 = openDialog({ component: StubB });
    closeDialog(k1);
    expect(stack.value).toHaveLength(1);
    expect(stack.value[0]?.key).toBe(k2);
    closeDialog();
    expect(stack.value).toHaveLength(0);
  });

  test('closeAllDialogs empties the stack', () => {
    const stack = useDialogStack();
    openDialog({ component: StubA });
    openDialog({ component: StubB });
    closeAllDialogs();
    expect(stack.value).toHaveLength(0);
  });

  test('closeDialog restores opener focus when available', () => {
    const opener = document.createElement('button');
    document.body.appendChild(opener);
    opener.focus();
    expect(document.activeElement).toBe(opener);

    const other = document.createElement('button');
    document.body.appendChild(other);
    const key = openDialog({ component: StubA });
    other.focus();
    expect(document.activeElement).toBe(other);

    closeDialog(key);
    expect(document.activeElement).toBe(opener);
    opener.remove();
    other.remove();
  });
});
