// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * Parse a 1-based form-pager current draft; clamps into 1…total.
 * Returns null when the draft is empty or not a positive integer.
 */
export function parseFormPagerCurrentDraft(draft: string, total: number): number | null {
  const t = Math.max(0, Math.trunc(Number(total) || 0));
  if (t <= 0) return null;
  const raw = String(draft ?? '').trim();
  if (!raw) return null;
  if (!/^\d+$/.test(raw)) return null;
  const n = Number.parseInt(raw, 10);
  if (!Number.isFinite(n) || n <= 0) return null;
  return Math.min(n, t);
}

/**
 * Width in `ch` for the editable current input only.
 * Sized from the widest of draft / current / total digit runs, plus padding
 * slack for Input border + horizontal padding — not from List's range floor
 * (that 10ch minimum made short indexes look like a sparse empty box).
 */
export function formPagerCurrentInputCh(opts: {
  draft?: string;
  currentLabel: string;
  totalLabel: string;
}): number {
  const draftLen = String(opts.draft ?? '').trim().length;
  const currentLen = String(opts.currentLabel ?? '').length;
  const totalLen = String(opts.totalLabel ?? '').length;
  const digits = Math.max(draftLen, currentLen, totalLen, 2);
  return Math.min(12, digits + 3);
}

/**
 * Resolve the native input from a template ref (plain element or component `$el`).
 */
export function resolvePagerInputEl(ref: unknown): HTMLInputElement | null {
  if (!ref || typeof ref !== 'object') return null;
  const r = ref as HTMLInputElement & { $el?: unknown };
  if (typeof r.focus === 'function' && !r.$el) {
    return r as HTMLInputElement;
  }
  const el = r.$el;
  return el instanceof HTMLInputElement ? el : null;
}
