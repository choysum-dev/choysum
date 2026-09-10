// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import type { DirectiveBinding } from 'vue';
import { syncFnRecorder } from '@/web/web/__tests__/mountApp';
import { setGlobalActionChecker, vAction, type ActionBindingValue } from './action';

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
    const checker = syncFnRecorder(() => false);
    const el = bindDirective({ ids: 'auth.action.user_export', hasAction: checker });

    expect(checker.calls).toEqual([['auth.action.user_export']]);
    expect(el.style.display).toBe('none');
  });

  test('disables element when using disable modifier', () => {
    const checker = syncFnRecorder(() => false);
    const el = bindDirective({ ids: 'auth.action.user_export', hasAction: checker }, { disable: true });

    expect(el.disabled).toBe(true);
    expect(el.getAttribute('aria-disabled')).toBe('true');
  });

  test('supports OR mode for arrays by default', () => {
    const checker = syncFnRecorder((id?: string) => id === 'auth.action.user_edit');
    const el = bindDirective({ ids: ['auth.action.user_delete', 'auth.action.user_edit'], hasAction: checker });

    expect(el.style.display).not.toBe('none');
  });

  test('supports AND mode for arrays', () => {
    const checker = syncFnRecorder((id?: string) => id === 'auth.action.user_edit');
    const el = bindDirective({ ids: ['auth.action.user_delete', 'auth.action.user_edit'], hasAction: checker }, { and: true });

    expect(el.style.display).toBe('none');
  });

  test('reacts to permission changes on update', () => {
    let allowed = false;
    const checker = syncFnRecorder(() => allowed);
    const el = bindDirective({ ids: 'auth.action.user_edit', hasAction: checker });

    expect(el.style.display).toBe('none');

    allowed = true;
    updateDirective(el, { ids: 'auth.action.user_edit', hasAction: checker });

    expect(el.style.display).toBe('');
  });

  test('uses global checker when binding checker is omitted', () => {
    const checker = syncFnRecorder(() => false);
    setGlobalActionChecker(checker);

    const el = bindDirective('auth.action.user_export');

    expect(checker.calls).toEqual([['auth.action.user_export']]);
    expect(el.style.display).toBe('none');
  });

  test('prefers binding checker over global checker', () => {
    const globalChecker = syncFnRecorder(() => false);
    const localChecker = syncFnRecorder(() => true);
    setGlobalActionChecker(globalChecker);

    const el = bindDirective({ ids: 'auth.action.user_export', hasAction: localChecker });

    expect(localChecker.calls).toEqual([['auth.action.user_export']]);
    expect(globalChecker.calls.length).toBe(0);
    expect(el.style.display).toBe('');
  });
});
