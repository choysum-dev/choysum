// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { markRaw, shallowRef, type Component, type ShallowRef } from 'vue';

/** How openDialog interacts with the existing managed stack. */
export type DialogMode = 'child' | 'replace' | 'interrupt';

export type DialogOpenOptions = {
  /** Dialog SFC / component (import and pass directly). */
  component: Component;
  /** Extra props merged onto the hosted component (besides modelValue). */
  props?: Record<string, unknown>;
  /** Default child: push. replace/interrupt clear the stack first. */
  mode?: DialogMode;
};

export type DialogEntry = {
  key: string;
  component: Component;
  props: Record<string, unknown>;
  mode: DialogMode;
};

const stack: ShallowRef<DialogEntry[]> = shallowRef([]);
const interruptHandlers = new Set<() => void>();
let keySeq = 0;

function nextKey(): string {
  keySeq += 1;
  return `choy-dialog-${keySeq}`;
}

function runInterruptHandlers(): void {
  for (const handler of interruptHandlers) {
    try {
      handler();
    } catch {
      // Chrome reset must not block opening the interrupt dialog.
    }
  }
}

function setStack(next: DialogEntry[]): void {
  stack.value = next;
}

/** Reactive stack for ChoyDialogHost. */
export function useDialogStack(): ShallowRef<DialogEntry[]> {
  return stack;
}

/**
 * Open a managed dialog. Returns the stack entry key.
 * Host must mount `<ChoyDialogHost />` (App root).
 */
export function openDialog(opts: DialogOpenOptions): string {
  if (!opts?.component) {
    throw new Error('openDialog: component is required');
  }
  const mode: DialogMode = opts.mode || 'child';
  if (mode === 'replace' || mode === 'interrupt') {
    if (mode === 'interrupt') runInterruptHandlers();
    setStack([]);
  }
  const key = nextKey();
  const entry: DialogEntry = {
    key,
    component: markRaw(opts.component),
    props: { ...(opts.props || {}) },
    mode,
  };
  setStack([...stack.value, entry]);
  return key;
}

/** Close one entry by key, or the top entry when key is omitted. */
export function closeDialog(key?: string): void {
  const cur = stack.value;
  if (!cur.length) return;
  if (!key) {
    setStack(cur.slice(0, -1));
    return;
  }
  const next = cur.filter(entry => entry.key !== key);
  if (next.length === cur.length) return;
  setStack(next);
}

/** Close every managed dialog. */
export function closeAllDialogs(): void {
  if (!stack.value.length) return;
  setStack([]);
}

/**
 * Register a chrome reset for interrupt opens (e.g. close user menu).
 * Returns an unsubscribe function.
 */
export function onDialogInterrupt(handler: () => void): () => void {
  interruptHandlers.add(handler);
  return () => {
    interruptHandlers.delete(handler);
  };
}

/** Test helper: clear stack, handlers, and key sequence. */
export function resetDialogServiceForTests(): void {
  setStack([]);
  interruptHandlers.clear();
  keySeq = 0;
}
