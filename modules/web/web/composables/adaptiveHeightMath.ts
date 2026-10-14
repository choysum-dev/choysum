// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * Pure height math for useAdaptiveHeight (unit-tested without Vue lifecycle).
 */

export type AdaptiveHeightMathOptions = {
  viewportGap: number;
  minViewportHeight: number;
  minContainerHeight: number;
  containerPadding: number;
  containerSelector?: string;
};

/** Container mode: fill from wrap top to container bottom, minus dialog footer + pad. */
export function computeContainerAdaptiveHeight(
  wrapRectTop: number,
  container: HTMLElement,
  opts: Pick<AdaptiveHeightMathOptions, 'minContainerHeight' | 'containerPadding'>,
): number {
  const crect = container.getBoundingClientRect();
  const dialog = container.closest('.el-dialog') as HTMLElement | null;
  const footer = dialog?.querySelector('.el-dialog__footer') as HTMLElement | null;
  const footerH = footer ? footer.getBoundingClientRect().height : 0;
  const pad = opts.containerPadding ?? 0;
  const h = Math.floor(crect.bottom - wrapRectTop - footerH - pad);
  return Math.max(opts.minContainerHeight, h);
}

/**
 * Viewport mode: prefer AppShell scroll canvas above the in-flow footer;
 * otherwise distance to the viewport bottom minus gap (with min floor).
 */
export function computeViewportAdaptiveHeight(
  wrap: Element,
  wrapRectTop: number,
  opts: Pick<AdaptiveHeightMathOptions, 'viewportGap' | 'minViewportHeight'>,
): number {
  const shellScroll = wrap.closest('[data-testid="choy-shell-scroll"]') as HTMLElement | null;
  if (shellScroll) {
    const footer = shellScroll.querySelector(
      '[data-testid="choy-shell-footer"]',
    ) as HTMLElement | null;
    const bottom = footer
      ? footer.getBoundingClientRect().top
      : shellScroll.getBoundingClientRect().bottom;
    const page = wrap.closest('[data-anchor="choy.page"]') as HTMLElement | null;
    const pagePadBottom = page
      ? Number.parseFloat(getComputedStyle(page).paddingBottom) || 0
      : 0;
    // Hard-cap to the shell canvas: do not force minViewportHeight past the footer.
    const h = Math.floor(bottom - wrapRectTop - pagePadBottom);
    return Math.max(0, h);
  }

  const vh = window.innerHeight || document.documentElement.clientHeight;
  const gap = opts.viewportGap ?? 0;
  const h = Math.floor(vh - wrapRectTop - gap);
  return Math.max(opts.minViewportHeight, h);
}

/** Cap short list pages so the body does not leave a tall empty void under few rows. */
export function computeListTableBodyHeight(args: {
  viewport: number;
  rowCount: number;
  rowHeight: number;
  headerHeight: number;
}): number {
  const { viewport, rowCount, rowHeight, headerHeight } = args;
  if (!viewport || viewport <= 0) return viewport;
  if (rowCount <= 0) return viewport;
  const contentH = headerHeight + Math.max(rowCount, 4) * rowHeight + 16;
  // Floor at 240 only when the adaptive viewport is tall enough; never exceed it.
  if (contentH < viewport) return Math.min(viewport, Math.max(240, contentH));
  return viewport;
}
