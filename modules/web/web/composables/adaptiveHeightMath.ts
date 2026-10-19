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
    const scrollRect = shellScroll.getBoundingClientRect();
    const footer = shellScroll.querySelector(
      '[data-testid="choy-shell-footer"]',
    ) as HTMLElement | null;
    // A hidden footer reports an all-zero rect; treat as absent.
    const footerRect = footer?.getBoundingClientRect() ?? null;
    const footerVisible = !!footerRect && (footerRect.height > 0 || footerRect.top > 0);
    const footerH = footerVisible
      ? footerRect!.height || Math.max(0, footerRect!.bottom - footerRect!.top)
      : 0;
    // In-flow footer sits below a main gap; that gap is not part of the canvas.
    const rowGap = footerVisible ? readRowGapPx(shellScroll) : 0;
    // Cap to the *visible* scrollport minus in-flow footer height. The scroll
    // ancestor (testid choy-shell-scroll) must contain the footer — AppShell
    // puts that testid on <main>. Using footer.top alone inflates height after overflow.
    const bottom = scrollRect.bottom - footerH - rowGap;
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

/** Parsed flex/grid row-gap. `normal` and missing styles count as 0. */
function readRowGapPx(el: HTMLElement): number {
  const style = getComputedStyle(el);
  const raw = style.rowGap && style.rowGap !== 'normal' ? style.rowGap : style.gap;
  const n = Number.parseFloat(raw || '');
  return Number.isFinite(n) ? n : 0;
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
  // Hug content height (plus a small pad); never force a tall min floor past the rows.
  const contentH = headerHeight + rowCount * rowHeight + 16;
  if (contentH < viewport) return contentH;
  return viewport;
}

export type AdaptiveHeightObserveMode = 'auto' | 'viewport' | 'container' | 'none';

/**
 * Elements whose size changes should trigger a recompute (container, shell
 * scroll canvas, and in-flow shell footer when present).
 */
export function collectAdaptiveHeightObserveTargets(
  wrap: Element | null | undefined,
  opts: { mode?: AdaptiveHeightObserveMode; containerSelector?: string },
): HTMLElement[] {
  if (!wrap || opts.mode === 'none') return [];
  const autoContainer = wrap.closest('.el-dialog__body') as HTMLElement | null;
  const shellScroll = wrap.closest('[data-testid="choy-shell-scroll"]') as HTMLElement | null;
  const need =
    opts.mode === 'container' || (opts.mode === 'auto' && !!autoContainer) || !!shellScroll;
  if (!need) return [];

  const bySelector = opts.containerSelector
    ? (wrap.closest(opts.containerSelector) as HTMLElement | null) ||
      (document.querySelector(opts.containerSelector) as HTMLElement | null)
    : null;
  const primary =
    bySelector || autoContainer || shellScroll || (wrap.parentElement as HTMLElement | null);

  const out: HTMLElement[] = [];
  const seen = new Set<HTMLElement>();
  const push = (el: HTMLElement | null | undefined) => {
    if (!el || seen.has(el)) return;
    seen.add(el);
    out.push(el);
  };
  push(primary);
  if (shellScroll) {
    push(shellScroll);
    push(shellScroll.querySelector('[data-testid="choy-shell-footer"]') as HTMLElement | null);
  }
  return out;
}
