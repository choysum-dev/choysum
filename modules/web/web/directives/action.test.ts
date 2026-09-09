// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import type { DirectiveBinding } from 'vue';
import { setGlobalActionChecker, vAction, type ActionBindingValue } from './action';

type CallRecorder = { calls: unknown[][] };

function fnRecorder<T = undefined, A extends unknown[] = unknown[]>(
  impl?: (...args: A) => T | Promise<T>
): CallRecorder & ((...args: A) => T | Promise<T>) {
  const rec: CallRecorder & ((...args: A) => T | Promise<T>) = Object.assign(
    (...args: A) => {
      rec.calls.push(args);
      return impl ? impl(...args) : (undefined as T);
    },
    { calls: [] as unknown[][] }
  );
  return rec;
}

function bindDirective(value: ActionBindingValue, modifiers: Record<string, boolean> = {}): HTMLButtonElement {
  const el = document.createElement('button');
  const binding = { value, modifiers } as DirectiveBinding<ActionBindingValue>;
  (vAction as any).mounted?.(el, binding);
  return el;
}

function updateDirective(el: HTMLButtonElement, value: ActionBindingValue, modifiers: Record<string, boolean> = {}): void {
  const binding = { value, modifiers } as DirectiveBinding<ActionBindingValue>;
  (vAction as any).updated?.(el, binding);
}

describe('v-action directive', () => {
  afterEach(() => {
    setGlobalActionChecker(undefined);
  });

  test('hides element by default when permission is denied', () => {
    const checker = fnRecorder(() => false);
    const el = bindDirective({ ids: 'auth.action.user_export', hasAction: checker });

    expect(checker.calls).toEqual([['auth.action.user_export']]);
    expect(el.style.display).toBe('none');
  });

  test('disables element when using disable modifier', () => {
    const checker = fnRecorder(() => false);
    const el = bindDirective({ ids: 'auth.action.user_export', hasAction: checker }, { disable: true });

    expect(el.disabled).toBe(true);
    expect(el.getAttribute('aria-disabled')).toBe('true');
  });

  test('supports OR mode for arrays by default', () => {
    const checker = fnRecorder((id?: string) => id === 'auth.action.user_edit');
    const el = bindDirective({ ids: ['auth.action.user_delete', 'auth.action.user_edit'], hasAction: checker });

    expect(el.style.display).not.toBe('none');
  });

  test('supports AND mode for arrays', () => {
    const checker = fnRecorder((id?: string) => id === 'auth.action.user_edit');
    const el = bindDirective({ ids: ['auth.action.user_delete', 'auth.action.user_edit'], hasAction: checker }, { and: true });

    expect(el.style.display).toBe('none');
  });

  test('reacts to permission changes on update', () => {
    let allowed = false;
    const checker = fnRecorder(() => allowed);
    const el = bindDirective({ ids: 'auth.action.user_edit', hasAction: checker });

    expect(el.style.display).toBe('none');

    allowed = true;
    updateDirective(el, { ids: 'auth.action.user_edit', hasAction: checker });

    expect(el.style.display).toBe('');
  });

  test('uses global checker when binding checker is omitted', () => {
    const checker = fnRecorder(() => false);
    setGlobalActionChecker(checker);

    const el = bindDirective('auth.action.user_export');

    expect(checker.calls).toEqual([['auth.action.user_export']]);
    expect(el.style.display).toBe('none');
  });

  test('prefers binding checker over global checker', () => {
    const globalChecker = fnRecorder(() => false);
    const localChecker = fnRecorder(() => true);
    setGlobalActionChecker(globalChecker);

    const el = bindDirective({ ids: 'auth.action.user_export', hasAction: localChecker });

    expect(localChecker.calls).toEqual([['auth.action.user_export']]);
    expect(globalChecker.calls.length).toBe(0);
    expect(el.style.display).toBe('');
  });
});
