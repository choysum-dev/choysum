// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it, vi } from 'vitest';

import { afterLocaleChange, resolveLocaleRemountMode } from './locale_remount';

describe('afterLocaleChange', () => {
  it('defaults to reload', async () => {
    const reload = vi.fn();
    await afterLocaleChange({ mode: 'reload', reload });
    expect(reload).toHaveBeenCalledOnce();
  });

  it('calls remount hook when mode is remount', async () => {
    const remount = vi.fn();
    const reload = vi.fn();
    await afterLocaleChange({ mode: 'remount', remount, reload });
    expect(remount).toHaveBeenCalledOnce();
    expect(reload).not.toHaveBeenCalled();
  });

  it('falls back to reload when remount mode has no hook', async () => {
    const reload = vi.fn();
    await afterLocaleChange({ mode: 'remount', reload });
    expect(reload).toHaveBeenCalledOnce();
  });

  it('resolveLocaleRemountMode defaults to reload', () => {
    expect(resolveLocaleRemountMode()).toBe('reload');
  });
});
