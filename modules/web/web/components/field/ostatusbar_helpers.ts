// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

export type StatusbarMetaOption = { value: string; label: string };

export type StatusbarOption = {
  value: string;
  label: string;
  disabled: boolean;
};

export type StatusbarBeforeChange = (next: string, prev: string | null) => boolean | Promise<boolean>;

/**
 * Resolve visible statusbar options (D5):
 * meta → optional onchange filter → whitelist order → ensure current value is present.
 */
export function resolveStatusbarOptions(args: {
  meta: StatusbarMetaOption[];
  whitelist?: string[] | null;
  current?: string | null;
  onchangeValues?: string[] | null;
  onchangeDisabled?: string[] | null;
}): StatusbarOption[] {
  const meta = Array.isArray(args.meta) ? args.meta : [];
  const metaMap = new Map<string, string>();
  for (const item of meta) {
    const value = item?.value != null ? String(item.value) : '';
    if (!value || metaMap.has(value)) continue;
    metaMap.set(value, String(item.label ?? value));
  }

  let pool: string[];
  if (Array.isArray(args.onchangeValues) && args.onchangeValues.length > 0) {
    pool = [];
    const seen = new Set<string>();
    for (const raw of args.onchangeValues) {
      const value = raw != null ? String(raw) : '';
      if (!value || seen.has(value)) continue;
      if (metaMap.size > 0 && !metaMap.has(value)) continue;
      seen.add(value);
      pool.push(value);
    }
  } else if (metaMap.size > 0) {
    pool = [...metaMap.keys()];
  } else {
    pool = [];
  }

  const whitelist = Array.isArray(args.whitelist)
    ? args.whitelist.map(v => (v != null ? String(v) : '')).filter(Boolean)
    : null;

  let values: string[];
  if (whitelist && whitelist.length > 0) {
    const poolSet = new Set(pool);
    if (pool.length > 0) {
      values = whitelist.filter(v => poolSet.has(v));
    } else {
      // No meta/onchange pool — still honor whitelist as bare values (labels = values).
      values = [...whitelist];
    }
  } else {
    values = pool;
  }

  const disabledSet = new Set((args.onchangeDisabled || []).map(v => String(v)));
  const out: StatusbarOption[] = [];
  const seen = new Set<string>();
  for (const value of values) {
    if (!value || seen.has(value)) continue;
    seen.add(value);
    out.push({
      value,
      label: metaMap.get(value) ?? value,
      disabled: disabledSet.has(value),
    });
  }

  const current = args.current != null && args.current !== '' ? String(args.current) : null;
  if (current && !seen.has(current)) {
    out.push({
      value: current,
      label: metaMap.get(current) ?? current,
      disabled: disabledSet.has(current),
    });
  }

  return out;
}

/** D7: no hook → allow; only explicit `true` proceeds; throw/reject → cancel. */
export async function gateBeforeChange(
  beforeChange: StatusbarBeforeChange | undefined,
  next: string,
  prev: string | null
): Promise<boolean> {
  if (!beforeChange) return true;
  try {
    return (await beforeChange(next, prev)) === true;
  } catch {
    return false;
  }
}
