// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { pinViewportWidth } from './pinViewport';

describe('pinViewportWidth', () => {
  afterEach(() => {
    pinViewportWidth(1280);
  });

  test('pins width and ignores non-width media clauses', () => {
    pinViewportWidth(500);
    expect(window.innerWidth).toBe(500);
    // Reassignable (writable: true).
    (window as any).innerWidth = 501;
    expect(window.innerWidth).toBe(501);
    pinViewportWidth(500);

    expect(window.matchMedia('(max-width: 768px)').matches).toBe(true);
    expect(
      window.matchMedia('(max-width: 768px) and (prefers-color-scheme: dark)').matches,
    ).toBe(true);
    expect(window.matchMedia('(min-width: 1024px)').matches).toBe(false);
    // VueUse-style fractional max-width thresholds (tablet band).
    pinViewportWidth(900);
    expect(window.matchMedia('(min-width: 768px) and (max-width: 1023.9px)').matches).toBe(
      true,
    );
    expect(window.matchMedia('(min-width: 1024px)').matches).toBe(false);
  });
});
