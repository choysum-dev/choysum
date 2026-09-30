// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { choyControlHeightPx } from './choyControlHeight';

describe('choyControlHeightPx', () => {
  test('falls back to 32 when computed style is unavailable', () => {
    expect(choyControlHeightPx()).toBe(32);
    expect(choyControlHeightPx(28)).toBe(28);
  });

  test('reads --choy-control-height when getComputedStyle works', () => {
    const prev = (globalThis as any).getComputedStyle;
    (globalThis as any).getComputedStyle = () => ({
      getPropertyValue: (name: string) => (name === '--choy-control-height' ? '28px' : ''),
    });
    try {
      expect(choyControlHeightPx()).toBe(28);
    } finally {
      (globalThis as any).getComputedStyle = prev;
    }
  });
});
