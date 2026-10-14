// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import {
  computeContainerAdaptiveHeight,
  computeListTableBodyHeight,
  computeViewportAdaptiveHeight,
} from './adaptiveHeightMath';

function rect(partial: Partial<DOMRect>): DOMRect {
  return {
    x: 0,
    y: 0,
    width: 100,
    height: 100,
    top: 0,
    left: 0,
    bottom: 100,
    right: 100,
    toJSON: () => ({}),
    ...partial,
  } as DOMRect;
}

describe('adaptiveHeightMath', () => {
  test('computeListTableBodyHeight floors at 240 but never past viewport', () => {
    expect(computeListTableBodyHeight({ viewport: 0, rowCount: 2, rowHeight: 32, headerHeight: 32 })).toBe(0);
    expect(computeListTableBodyHeight({ viewport: 800, rowCount: 0, rowHeight: 32, headerHeight: 32 })).toBe(800);
    expect(computeListTableBodyHeight({ viewport: 800, rowCount: 2, rowHeight: 32, headerHeight: 32 })).toBe(240);
    expect(computeListTableBodyHeight({ viewport: 200, rowCount: 2, rowHeight: 32, headerHeight: 32 })).toBe(200);
    expect(computeListTableBodyHeight({ viewport: 100, rowCount: 50, rowHeight: 32, headerHeight: 32 })).toBe(100);
  });

  test('computeViewportAdaptiveHeight hard-caps to shell footer', () => {
    const shell = document.createElement('div');
    shell.setAttribute('data-testid', 'choy-shell-scroll');
    shell.getBoundingClientRect = () => rect({ top: 0, bottom: 900 });
    const footer = document.createElement('div');
    footer.setAttribute('data-testid', 'choy-shell-footer');
    footer.getBoundingClientRect = () => rect({ top: 500, bottom: 540 });
    shell.appendChild(footer);
    const page = document.createElement('div');
    page.setAttribute('data-anchor', 'choy.page');
    const wrap = document.createElement('div');
    wrap.closest = ((sel: string) => {
      if (sel === '[data-testid="choy-shell-scroll"]') return shell;
      if (sel === '[data-anchor="choy.page"]') return page;
      return null;
    }) as typeof wrap.closest;
    const orig = window.getComputedStyle;
    (window as any).getComputedStyle = () => ({ paddingBottom: '16px' });
    try {
      expect(
        computeViewportAdaptiveHeight(wrap, 100, { viewportGap: 12, minViewportHeight: 240 }),
      ).toBe(384); // 500 - 100 - 16
    } finally {
      (window as any).getComputedStyle = orig;
    }
  });

  test('computeViewportAdaptiveHeight uses shell bottom when footer missing', () => {
    const shell = document.createElement('div');
    shell.setAttribute('data-testid', 'choy-shell-scroll');
    shell.getBoundingClientRect = () => rect({ top: 0, bottom: 600 });
    const wrap = document.createElement('div');
    wrap.closest = ((sel: string) =>
      sel === '[data-testid="choy-shell-scroll"]' ? shell : null) as typeof wrap.closest;
    expect(
      computeViewportAdaptiveHeight(wrap, 200, { viewportGap: 12, minViewportHeight: 240 }),
    ).toBe(400);
  });

  test('computeViewportAdaptiveHeight falls back to window with min floor', () => {
    const wrap = document.createElement('div');
    wrap.closest = (() => null) as typeof wrap.closest;
    const prev = window.innerHeight;
    Object.defineProperty(window, 'innerHeight', { configurable: true, value: 900 });
    try {
      expect(
        computeViewportAdaptiveHeight(wrap, 800, { viewportGap: 12, minViewportHeight: 240 }),
      ).toBe(240); // max(240, 900-800-12=88)
      expect(
        computeViewportAdaptiveHeight(wrap, 100, { viewportGap: 12, minViewportHeight: 240 }),
      ).toBe(788);
    } finally {
      Object.defineProperty(window, 'innerHeight', { configurable: true, value: prev });
    }
  });

  test('computeContainerAdaptiveHeight subtracts dialog footer', () => {
    const dialog = document.createElement('div');
    dialog.className = 'el-dialog';
    const body = document.createElement('div');
    body.className = 'el-dialog__body';
    body.getBoundingClientRect = () => rect({ top: 50, bottom: 450 });
    const footer = document.createElement('div');
    footer.className = 'el-dialog__footer';
    footer.getBoundingClientRect = () => rect({ height: 40, top: 450, bottom: 490 });
    dialog.appendChild(body);
    dialog.appendChild(footer);
    body.closest = ((sel: string) => (sel === '.el-dialog' ? dialog : null)) as typeof body.closest;
    expect(
      computeContainerAdaptiveHeight(80, body, { minContainerHeight: 160, containerPadding: 8 }),
    ).toBe(322); // 450 - 80 - 40 - 8
  });
});
