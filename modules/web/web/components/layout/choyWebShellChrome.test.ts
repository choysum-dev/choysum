// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import {
  shellMenuTriggerLabel,
  shortAppVersion,
  shouldCloseDrawerOnEscape,
  setDrawerBodyOverflow,
} from './choyWebShellChrome';

describe('choyWebShellChrome', () => {
  test('shortAppVersion truncates long versions and defaults empty to dev', () => {
    expect(shortAppVersion('')).toBe('dev');
    expect(shortAppVersion('  ')).toBe('dev');
    expect(shortAppVersion('1.2.3')).toBe('1.2.3');
    expect(shortAppVersion('1.2.3-alpha.9')).toBe('1.2.3-');
  });

  test('shellMenuTriggerLabel picks mobile vs expand/collapse copy', () => {
    const t = (key: string) => key;
    expect(shellMenuTriggerLabel({ isMobile: true, railCollapsed: false, t })).toBe(
      'layout.header.menu',
    );
    expect(shellMenuTriggerLabel({ isMobile: false, railCollapsed: true, t })).toBe(
      'layout.sidebar.expand',
    );
    expect(shellMenuTriggerLabel({ isMobile: false, railCollapsed: false, t })).toBe(
      'layout.sidebar.collapse',
    );
  });

  test('shouldCloseDrawerOnEscape ignores handled events and non-Escape', () => {
    expect(shouldCloseDrawerOnEscape({ key: 'Escape' }, true)).toBe(true);
    expect(shouldCloseDrawerOnEscape({ key: 'Escape', defaultPrevented: true }, true)).toBe(false);
    expect(shouldCloseDrawerOnEscape({ key: 'Escape' }, false)).toBe(false);
    expect(shouldCloseDrawerOnEscape({ key: 'Enter' }, true)).toBe(false);
  });
});

describe('setDrawerBodyOverflow', () => {
  test('locks and clears document body overflow', () => {
    setDrawerBodyOverflow(true);
    expect(document.body.style.overflow).toBe('hidden');
    setDrawerBodyOverflow(false);
    expect(document.body.style.overflow || '').toBe('');
  });
});

