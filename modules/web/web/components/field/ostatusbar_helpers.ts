// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

export type StatusbarMetaOption = { value: string; label: string };

export type StatusbarOption = {
  value: string;
  label: string;
  disabled: boolean;
};

export type StatusbarBeforeChange = (next: string, prev: string | null) => boolean | Promise<boolean>;

export function toStatusbarView(raw: unknown): string | null {
  return raw == null ? null : String(raw);
}

export function fromStatusbarView(v: string | null): string | null {
  return v ?? null;
}

export function normalizeSegmentedModelValue(raw: unknown): string | undefined {
  if (raw == null || raw === '') return undefined;
  return String(raw);
}

export function resolveStatusbarWhitelist(
  statusbarVisible?: string[] | null,
  selection?: string[] | null
): string[] | null {
  if (Array.isArray(statusbarVisible) && statusbarVisible.length > 0) return statusbarVisible;
  if (Array.isArray(selection) && selection.length > 0) return selection;
  return null;
}

export function pickRootOnchangeSelection(
  lastOnchange: { selection?: unknown } | null | undefined,
  field: string
): { values: string[]; disabled?: string[] } | null {
  const raw = lastOnchange?.selection;
  if (!Array.isArray(raw) || raw.length === 0) return null;
  const m = raw.find((s: any) => s && s.field === field);
  if (!m) return null;
  // Only apply a domain when selection is an explicit array (including []).
  // Missing / non-array selection must not be treated as "no options".
  if (!Array.isArray(m.selection)) return null;
  return {
    values: m.selection,
    disabled: Array.isArray(m.disabled) ? m.disabled : undefined,
  };
}

export function currentFromRowRef(rowRef: unknown, leaf: string): string | null {
  if (rowRef == null || !leaf) return null;
  try {
    const row = typeof rowRef === 'function' ? (rowRef as () => unknown)() : rowRef;
    const rec = row && typeof row === 'object' && 'value' in (row as object) ? (row as { value: unknown }).value : row;
    if (rec && typeof rec === 'object' && (rec as any)[leaf] != null) {
      return String((rec as any)[leaf]);
    }
  } catch {
    return null;
  }
  return null;
}

export function currentFromFieldValue(raw: unknown): string | null {
  try {
    return raw != null && raw !== '' ? String(raw) : null;
  } catch {
    return null;
  }
}

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
    const label = item.label == null ? value : String(item.label);
    metaMap.set(value, label);
  }

  let pool: string[];
  // Explicit array (including []) is an authoritative onchange domain.
  // undefined/null means "no onchange filter" → fall back to meta.
  const hasOnchangeDomain = Array.isArray(args.onchangeValues);
  if (hasOnchangeDomain) {
    pool = [];
    const seen = new Set<string>();
    for (const raw of args.onchangeValues as string[]) {
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

  const hasAuthoritativePool = hasOnchangeDomain || metaMap.size > 0;

  let whitelist: string[] | null = null;
  if (Array.isArray(args.whitelist)) {
    whitelist = [];
    for (const raw of args.whitelist) {
      if (raw == null) continue;
      const value = String(raw);
      if (value) whitelist.push(value);
    }
  }

  let values: string[];
  if (whitelist != null && whitelist.length > 0) {
    if (hasAuthoritativePool) {
      const poolSet = new Set(pool);
      values = whitelist.filter(v => poolSet.has(v));
    } else {
      // No meta/onchange pool — still honor whitelist as bare values (labels = values).
      values = [...whitelist];
    }
  } else {
    values = pool;
  }

  const disabledList = Array.isArray(args.onchangeDisabled) ? args.onchangeDisabled : [];
  const disabledSet = new Set(disabledList.map(v => String(v)));
  const out: StatusbarOption[] = [];
  const seen = new Set<string>();
  // Empties are already stripped while building whitelist/pool; only dedupe here.
  for (const value of values) {
    if (seen.has(value)) continue;
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

export function toSegmentedOptions(options: StatusbarOption[]) {
  return options.map(o => ({
    label: o.label,
    value: o.value,
    disabled: o.disabled,
  }));
}

/** Returns an Error when invalid; null when ok / unset. */
export function validateStatusbarValue(
  value: unknown,
  options: Array<{ value: string }>,
  messages: { mustBeString: string; invalid: (v: string) => string }
): Error | null {
  if (value == null || value === '') return null;
  if (typeof value !== 'string') return new Error(messages.mustBeString);
  const ok = options.some(o => o.value === value);
  if (!ok) return new Error(messages.invalid(value));
  return null;
}

export function canSelectStatusbarValue(
  next: string,
  options: StatusbarOption[]
): boolean {
  return options.some(o => o.value === next && !o.disabled);
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

export async function applyStatusbarSelect(args: {
  interactive: boolean;
  pending: boolean;
  nextRaw: string | number | boolean | null | undefined;
  current: string | null;
  options: StatusbarOption[];
  beforeChange?: StatusbarBeforeChange;
  write: (next: string) => void;
}): Promise<'skipped' | 'cancelled' | 'written'> {
  if (!args.interactive || args.pending) return 'skipped';
  const next = args.nextRaw == null ? '' : String(args.nextRaw);
  if (!next) return 'skipped';
  if (next === args.current) return 'skipped';
  if (!canSelectStatusbarValue(next, args.options)) return 'skipped';
  const ok = await gateBeforeChange(args.beforeChange, next, args.current);
  if (!ok) return 'cancelled';
  args.write(next);
  return 'written';
}
