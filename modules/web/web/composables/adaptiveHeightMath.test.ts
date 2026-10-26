// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import {
  collectAdaptiveHeightObserveTargets,
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
  test('computeListTableBodyHeight hugs short content and never past viewport', () => {
    expect(computeListTableBodyHeight({ viewport: 0, rowCount: 2, rowHeight: 32, headerHeight: 32 })).toBe(0);
    expect(computeListTableBodyHeight({ viewport: 800, rowCount: 0, rowHeight: 32, headerHeight: 32 })).toBe(800);
    // 32 + 2*32 + 16 = 112
    expect(computeListTableBodyHeight({ viewport: 800, rowCount: 2, rowHeight: 32, headerHeight: 32 })).toBe(112);
    expect(computeListTableBodyHeight({ viewport: 100, rowCount: 2, rowHeight: 32, headerHeight: 32 })).toBe(100);
    expect(computeListTableBodyHeight({ viewport: 100, rowCount: 50, rowHeight: 32, headerHeight: 32 })).toBe(100);
  });

  test('computeViewportAdaptiveHeight hard-caps to in-flow shell footer', () => {
    const shell = document.createElement('div');
    shell.setAttribute('data-testid', 'choy-shell-scroll');
    // Visible scrollport; footer sits in the last 40px (pinned when content fits).
    shell.getBoundingClientRect = () => rect({ top: 0, bottom: 540 });
    const footer = document.createElement('div');
    footer.setAttribute('data-testid', 'choy-shell-footer');
    footer.getBoundingClientRect = () => rect({ top: 500, bottom: 540, height: 40 });
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
      // (scrollBottom - footerH) - wrapTop - pagePad = (540 - 40) - 100 - 16
      expect(
        computeViewportAdaptiveHeight(wrap, 100, { viewportGap: 12, minViewportHeight: 240 }),
      ).toBe(384);
    } finally {
      (window as any).getComputedStyle = orig;
    }
  });

  test('computeViewportAdaptiveHeight subtracts the main row-gap above an in-flow footer', () => {
    const shell = document.createElement('div');
    shell.setAttribute('data-testid', 'choy-shell-scroll');
    shell.getBoundingClientRect = () => rect({ top: 0, bottom: 540 });
    const footer = document.createElement('div');
    footer.setAttribute('data-testid', 'choy-shell-footer');
    footer.getBoundingClientRect = () => rect({ top: 500, bottom: 540, height: 40 });
    shell.appendChild(footer);
    const wrap = document.createElement('div');
    wrap.closest = ((sel: string) =>
      sel === '[data-testid="choy-shell-scroll"]' ? shell : null) as typeof wrap.closest;
    const orig = window.getComputedStyle;
    (window as any).getComputedStyle = () => ({ rowGap: '16px', gap: '16px', paddingBottom: '0px' });
    try {
      // (scrollBottom - footerH - rowGap) - wrapTop = (540 - 40 - 16) - 100
      expect(
        computeViewportAdaptiveHeight(wrap, 100, { viewportGap: 12, minViewportHeight: 240 }),
      ).toBe(384);
    } finally {
      (window as any).getComputedStyle = orig;
    }
  });

  test('computeViewportAdaptiveHeight ignores drifted footer.top after overflow', () => {
    const shell = document.createElement('div');
    shell.setAttribute('data-testid', 'choy-shell-scroll');
    shell.getBoundingClientRect = () => rect({ top: 0, bottom: 540 });
    const footer = document.createElement('div');
    footer.setAttribute('data-testid', 'choy-shell-footer');
    // Footer has drifted below the visible scrollport (content overflow).
    footer.getBoundingClientRect = () => rect({ top: 700, bottom: 740, height: 40 });
    shell.appendChild(footer);
    const wrap = document.createElement('div');
    wrap.closest = ((sel: string) =>
      sel === '[data-testid="choy-shell-scroll"]' ? shell : null) as typeof wrap.closest;
    // Still caps to visible scrollport: 540 - 40 - 100 = 400 (not 700 - 100).
    expect(
      computeViewportAdaptiveHeight(wrap, 100, { viewportGap: 12, minViewportHeight: 240 }),
    ).toBe(400);
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

  test('computeViewportAdaptiveHeight ignores degenerate zero-size footer', () => {
    const shell = document.createElement('div');
    shell.setAttribute('data-testid', 'choy-shell-scroll');
    shell.getBoundingClientRect = () => rect({ top: 0, bottom: 700 });
    const footer = document.createElement('div');
    footer.setAttribute('data-testid', 'choy-shell-footer');
    footer.getBoundingClientRect = () => rect({ top: 0, bottom: 0, height: 0 });
    shell.appendChild(footer);
    const wrap = document.createElement('div');
    wrap.closest = ((sel: string) =>
      sel === '[data-testid="choy-shell-scroll"]' ? shell : null) as typeof wrap.closest;
    expect(
      computeViewportAdaptiveHeight(wrap, 100, { viewportGap: 12, minViewportHeight: 240 }),
    ).toBe(600); // shell bottom, not footer.top=0
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
    dialog.setAttribute('data-slot', 'dialog-content');
    dialog.getBoundingClientRect = () => rect({ top: 50, bottom: 450 });
    // Inner wrap is the measured container; footer is a sibling under dialog-content.
    const inner = document.createElement('div');
    inner.getBoundingClientRect = () => rect({ top: 50, bottom: 450 });
    const footer = document.createElement('div');
    footer.setAttribute('data-slot', 'dialog-footer');
    footer.getBoundingClientRect = () => rect({ height: 40, top: 450, bottom: 490 });
    // Nested dialog footer must not be subtracted for the outer container.
    const nested = document.createElement('div');
    nested.setAttribute('data-slot', 'dialog-content');
    const nestedFooter = document.createElement('div');
    nestedFooter.setAttribute('data-slot', 'dialog-footer');
    nestedFooter.getBoundingClientRect = () => rect({ height: 99, top: 0, bottom: 99 });
    nested.appendChild(nestedFooter);
    dialog.appendChild(inner);
    dialog.appendChild(footer);
    dialog.appendChild(nested);

    const stubClosest = (owner: HTMLElement, target: HTMLElement) => {
      owner.closest = ((sel: string) =>
        sel === '[data-slot="dialog-content"]' ? target : null) as typeof owner.closest;
    };
    stubClosest(dialog, dialog);
    stubClosest(inner, dialog);
    stubClosest(footer, dialog);
    stubClosest(nested, nested);
    stubClosest(nestedFooter, nested);

    expect(
      computeContainerAdaptiveHeight(80, inner, { minContainerHeight: 160, containerPadding: 8 }),
    ).toBe(322); // 450 - 80 - 40 - 8 (outer footer only)
  });

  test('collectAdaptiveHeightObserveTargets covers shell, dialog, and none', () => {
    expect(collectAdaptiveHeightObserveTargets(null, { mode: 'viewport' })).toEqual([]);

    const wrapNone = document.createElement('div');
    expect(collectAdaptiveHeightObserveTargets(wrapNone, { mode: 'none' })).toEqual([]);
    wrapNone.closest = (() => null) as typeof wrapNone.closest;
    expect(collectAdaptiveHeightObserveTargets(wrapNone, { mode: 'viewport' })).toEqual([]);

    const shell = document.createElement('div');
    shell.setAttribute('data-testid', 'choy-shell-scroll');
    const footer = document.createElement('div');
    footer.setAttribute('data-testid', 'choy-shell-footer');
    shell.appendChild(footer);
    const wrapShell = document.createElement('div');
    wrapShell.closest = ((sel: string) =>
      sel === '[data-testid="choy-shell-scroll"]' ? shell : null) as typeof wrapShell.closest;
    expect(collectAdaptiveHeightObserveTargets(wrapShell, { mode: 'viewport' })).toEqual([
      shell,
      footer,
    ]);

    const content = document.createElement('div');
    content.setAttribute('data-slot', 'dialog-content');
    content.className = 'custom-container';
    document.body.appendChild(content);
    const wrapDialog = document.createElement('div');
    wrapDialog.closest = ((sel: string) => {
      if (sel === '[data-slot="dialog-content"]' || sel === '.custom-container') return content;
      return null;
    }) as typeof wrapDialog.closest;
    expect(
      collectAdaptiveHeightObserveTargets(wrapDialog, {
        mode: 'auto',
        containerSelector: '.custom-container',
      }),
    ).toEqual([content]);

    const wrapParent = document.createElement('div');
    const parent = document.createElement('div');
    parent.appendChild(wrapParent);
    wrapParent.closest = (() => null) as typeof wrapParent.closest;
    expect(collectAdaptiveHeightObserveTargets(wrapParent, { mode: 'container' })).toEqual([parent]);
    content.remove();
  });
});
