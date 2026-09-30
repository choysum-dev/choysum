// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * Read `--choy-control-height` as a pixel number for virtual row estimates.
 * Falls back to comfortable density (32) when CSS is unavailable.
 */
export function choyControlHeightPx(fallback = 32): number {
  if (typeof document === 'undefined') return fallback;
  const styleFn = (globalThis as any).getComputedStyle as
    | ((el: Element) => CSSStyleDeclaration)
    | undefined;
  if (typeof styleFn !== 'function') return fallback;
  try {
    const raw = styleFn(document.documentElement).getPropertyValue('--choy-control-height').trim();
    const n = Number.parseFloat(raw);
    return Number.isFinite(n) && n > 0 ? n : fallback;
  } catch {
    return fallback;
  }
}
