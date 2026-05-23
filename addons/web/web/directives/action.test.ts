// @vitest-environment happy-dom
// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { afterEach, describe, expect, it, vi } from 'vitest';
import type { DirectiveBinding } from 'vue';
import { setGlobalActionChecker, vAction, type ActionBindingValue } from './action';

function mountDirective(value: ActionBindingValue, modifiers: Record<string, boolean> = {}): HTMLButtonElement {
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

  it('hides element by default when permission is denied', () => {
    const checker = vi.fn(() => false);
    const el = mountDirective({ ids: 'auth.action.user_export', hasAction: checker });

    expect(checker).toHaveBeenCalledWith('auth.action.user_export');
    expect(el.style.display).toBe('none');
  });

  it('disables element when using disable modifier', () => {
    const checker = vi.fn(() => false);
    const el = mountDirective({ ids: 'auth.action.user_export', hasAction: checker }, { disable: true });

    expect(el.disabled).toBe(true);
    expect(el.getAttribute('aria-disabled')).toBe('true');
  });

  it('supports OR mode for arrays by default', () => {
    const checker = vi.fn((id?: string) => id === 'auth.action.user_edit');
    const el = mountDirective({ ids: ['auth.action.user_delete', 'auth.action.user_edit'], hasAction: checker });

    expect(el.style.display).not.toBe('none');
  });

  it('supports AND mode for arrays', () => {
    const checker = vi.fn((id?: string) => id === 'auth.action.user_edit');
    const el = mountDirective({ ids: ['auth.action.user_delete', 'auth.action.user_edit'], hasAction: checker }, { and: true });

    expect(el.style.display).toBe('none');
  });

  it('reacts to permission changes on update', () => {
    let allowed = false;
    const checker = vi.fn(() => allowed);
    const el = mountDirective({ ids: 'auth.action.user_edit', hasAction: checker });

    expect(el.style.display).toBe('none');

    allowed = true;
    updateDirective(el, { ids: 'auth.action.user_edit', hasAction: checker });

    expect(el.style.display).toBe('');
  });

  it('uses global checker when binding checker is omitted', () => {
    const checker = vi.fn(() => false);
    setGlobalActionChecker(checker);

    const el = mountDirective('auth.action.user_export');

    expect(checker).toHaveBeenCalledWith('auth.action.user_export');
    expect(el.style.display).toBe('none');
  });

  it('prefers binding checker over global checker', () => {
    const globalChecker = vi.fn(() => false);
    const localChecker = vi.fn(() => true);
    setGlobalActionChecker(globalChecker);

    const el = mountDirective({ ids: 'auth.action.user_export', hasAction: localChecker });

    expect(localChecker).toHaveBeenCalledWith('auth.action.user_export');
    expect(globalChecker).not.toHaveBeenCalled();
    expect(el.style.display).toBe('');
  });
});
