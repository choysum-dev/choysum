// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * Pure selection helpers for the company switcher list.
 *
 * Allowed companies stay outside this module. Callers pass only the rows the
 * user may see; ParentId edges outside that set become roots.
 */

export type SwitchCompanyRow = {
  Id: string;
  DisplayName?: string;
  Code?: string;
  Logo?: unknown;
  ParentId?: unknown;
};

export type SwitchCompanyTreeEntry = {
  id: string;
  level: number;
  name: string;
  code: string;
  logo?: unknown;
  childIds: string[];
};

export type SwitchCompanyScopeDraft = {
  activeCompanyId: string;
  enabledCompanyIds: string[];
};

/**
 * Normalize a company id list into unique non-empty values, preserving order.
 */
export function uniqCompanyIds(xs: readonly unknown[]): string[] {
  const out: string[] = [];
  const seen = new Set<string>();
  for (const raw of xs) {
    const id = String(raw ?? '').trim();
    if (!id || seen.has(id)) continue;
    seen.add(id);
    out.push(id);
  }
  return out;
}

/**
 * Compare two company id lists as sets.
 */
export function setEqCompanyIds(a: readonly string[], b: readonly string[]): boolean {
  const sa = new Set(uniqCompanyIds(a));
  const sb = new Set(uniqCompanyIds(b));
  if (sa.size !== sb.size) return false;
  for (const x of sa) if (!sb.has(x)) return false;
  return true;
}

/**
 * Read a ManyToOne / string ParentId into a trimmed company id.
 */
export function normalizeCompanyParentId(parent: unknown): string {
  if (parent == null) return '';
  if (typeof parent === 'string' || typeof parent === 'number') {
    return String(parent).trim();
  }
  if (typeof parent === 'object') {
    const id = (parent as { Id?: unknown }).Id;
    return String(id ?? '').trim();
  }
  // Non-id values (boolean, symbol, …) are treated as missing parents.
  return '';
}

/**
 * Build child lists for every allowed company from ParentId edges.
 * Parents outside the allowed set are ignored so those children become roots.
 */
export function buildCompanyChildrenMap(
  rows: readonly SwitchCompanyRow[],
): Map<string, string[]> {
  const allowed = new Set(rows.map(r => String(r.Id ?? '').trim()).filter(Boolean));
  const children = new Map<string, string[]>();
  for (const id of allowed) children.set(id, []);

  for (const row of rows) {
    const id = String(row.Id ?? '').trim();
    if (!id || !allowed.has(id)) continue;
    const parentId = normalizeCompanyParentId(row.ParentId);
    if (!parentId || parentId === id || !allowed.has(parentId)) continue;
    const list = children.get(parentId) || [];
    if (!list.includes(id)) list.push(id);
    children.set(parentId, list);
  }
  return children;
}

/**
 * Display label used for sorting and search (DisplayName, else Code, else Id).
 */
export function companyDisplayLabel(row: SwitchCompanyRow): string {
  const name = String(row.DisplayName ?? '').trim();
  if (name) return name;
  const code = String(row.Code ?? '').trim();
  if (code) return code;
  return String(row.Id ?? '').trim();
}

/**
 * Flatten allowed companies into a depth-first tree for the switcher list.
 * Roots are companies whose ParentId is missing or outside the allowed set.
 */
export function buildCompanyTreeEntries(
  rows: readonly SwitchCompanyRow[],
): SwitchCompanyTreeEntry[] {
  const byId = new Map<string, SwitchCompanyRow>();
  for (const row of rows) {
    const id = String(row.Id ?? '').trim();
    if (!id) continue;
    byId.set(id, row);
  }
  const children = buildCompanyChildrenMap(rows);

  /** Sort company ids by display label. */
  const sortIds = (ids: string[]): string[] =>
    ids.slice().sort((a, b) => {
      const la = companyDisplayLabel(byId.get(a) || { Id: a });
      const lb = companyDisplayLabel(byId.get(b) || { Id: b });
      return la.localeCompare(lb);
    });

  for (const [parentId, kids] of children) {
    children.set(parentId, sortIds(kids));
  }

  const roots = sortIds(
    Array.from(byId.keys()).filter(id => {
      const parentId = normalizeCompanyParentId(byId.get(id)?.ParentId);
      return !parentId || !byId.has(parentId);
    }),
  );

  const out: SwitchCompanyTreeEntry[] = [];
  const visited = new Set<string>();
  /** Depth-first emit one company and its allowed children, skipping cycles. */
  const walk = (id: string, level: number) => {
    if (visited.has(id)) return;
    visited.add(id);
    const row = byId.get(id)!;
    const childIds = children.get(id) || [];
    out.push({
      id,
      level,
      name: companyDisplayLabel(row),
      code: String(row.Code ?? '').trim(),
      logo: row.Logo,
      childIds,
    });
    for (const childId of childIds) walk(childId, level + 1);
  };
  for (const rootId of roots) walk(rootId, 0);
  // Emit cycle-only nodes that never appear under a root so they stay selectable.
  for (const id of sortIds(Array.from(byId.keys()))) {
    if (!visited.has(id)) walk(id, 0);
  }
  return out;
}

/**
 * Self + allowed descendants in DFS child order.
 */
export function collectCompanyBranchIds(
  companyId: string,
  children: Map<string, string[]>,
): string[] {
  const root = String(companyId ?? '').trim();
  if (!root) return [];
  const out: string[] = [];
  const visited = new Set<string>();
  /** Collect self and descendants once; stop if ParentId forms a cycle. */
  const walk = (id: string) => {
    if (visited.has(id)) return;
    visited.add(id);
    out.push(id);
    for (const childId of children.get(id) || []) walk(childId);
  };
  walk(root);
  return out;
}

/**
 * True when the draft enabled set is a single company, or one fully selected
 * allowed subtree under a shared root (Odoo single-company mode).
 */
export function isSingleCompanyMode(
  enabledCompanyIds: readonly string[],
  children: Map<string, string[]>,
): boolean {
  const enabled = uniqCompanyIds(enabledCompanyIds);
  if (enabled.length <= 1) return true;

  const enabledSet = new Set(enabled);
  /** Return id when it is in the enabled set; otherwise empty. */
  const getActive = (id: string) => (enabledSet.has(id) ? id : '');

  let root: string | undefined;
  for (const companyId of enabled) {
    let cur = companyId;
    let parentWalkGuard = 0;
    const seen = new Set<string>();
    while (parentWalkGuard++ < 64) {
      if (seen.has(cur)) break;
      seen.add(cur);
      let parentId = '';
      for (const [pid, kids] of children) {
        if (kids.includes(cur)) {
          parentId = pid;
          break;
        }
      }
      if (!parentId || !getActive(parentId)) break;
      cur = parentId;
    }
    if (root === undefined) root = cur;
    else if (root !== cur) return false;
  }

  // enabled.length > 1 always assigns root on the first iteration.
  const queue = [...(children.get(root as string) || [])];
  const queued = new Set<string>();
  while (queue.length) {
    const id = queue.pop()!;
    if (queued.has(id)) continue;
    queued.add(id);
    if (!enabledSet.has(id)) return false;
    for (const childId of children.get(id) || []) queue.push(childId);
  }
  return true;
}

/**
 * Enter a company: replace or extend the enabled set, then make it active.
 */
export function logIntoCompany(opts: {
  companyId: string;
  draftActiveCompanyId: string;
  draftEnabledCompanyIds: readonly string[];
  children: Map<string, string[]>;
}): SwitchCompanyScopeDraft {
  const companyId = String(opts.companyId ?? '').trim();
  const draftActive = String(opts.draftActiveCompanyId ?? '').trim();
  let enabled = uniqCompanyIds(opts.draftEnabledCompanyIds);
  if (!companyId) {
    return { activeCompanyId: draftActive, enabledCompanyIds: enabled };
  }

  const branch = collectCompanyBranchIds(companyId, opts.children);

  if (isSingleCompanyMode(enabled, opts.children)) {
    // collectCompanyBranchIds always includes companyId when non-empty.
    enabled = branch;
  } else {
    for (const id of branch) {
      if (!enabled.includes(id)) enabled.push(id);
    }
    enabled = [companyId, ...enabled.filter(id => id !== companyId)];
  }

  return { activeCompanyId: companyId, enabledCompanyIds: enabled };
}

/**
 * Checkbox state for a company branch within the draft enabled set.
 */
export function branchCheckState(
  companyId: string,
  enabledCompanyIds: readonly string[],
  children: Map<string, string[]>,
): boolean | 'indeterminate' {
  const branch = collectCompanyBranchIds(companyId, children);
  if (!branch.length) return false;
  const enabled = new Set(uniqCompanyIds(enabledCompanyIds));
  const selectedCount = branch.filter(id => enabled.has(id)).length;
  if (selectedCount === 0) return false;
  if (selectedCount === branch.length) return true;
  return 'indeterminate';
}

/**
 * Toggle a company branch in the draft enabled set.
 * Refuses to clear the last remaining company. When the active company leaves
 * the set, the first remaining id becomes active.
 */
export function toggleCompanyBranch(opts: {
  companyId: string;
  draftActiveCompanyId: string;
  draftEnabledCompanyIds: readonly string[];
  children: Map<string, string[]>;
}): SwitchCompanyScopeDraft {
  const companyId = String(opts.companyId ?? '').trim();
  const branch = new Set(collectCompanyBranchIds(companyId, opts.children));
  let enabled = uniqCompanyIds(opts.draftEnabledCompanyIds);
  const state = branchCheckState(companyId, enabled, opts.children);

  if (state === true) {
    const next = enabled.filter(id => !branch.has(id));
    if (!next.length) {
      return {
        activeCompanyId: String(opts.draftActiveCompanyId ?? '').trim(),
        enabledCompanyIds: enabled,
      };
    }
    enabled = next;
  } else {
    for (const id of collectCompanyBranchIds(companyId, opts.children)) {
      if (!enabled.includes(id)) enabled.push(id);
    }
  }

  let active = String(opts.draftActiveCompanyId ?? '').trim();
  if (!active || !enabled.includes(active)) {
    active = enabled[0] || '';
  }
  return { activeCompanyId: active, enabledCompanyIds: enabled };
}

/**
 * True when unchecking this company would empty the enabled set.
 */
export function isCompanyBranchToggleDisabled(
  companyId: string,
  enabledCompanyIds: readonly string[],
  children: Map<string, string[]>,
): boolean {
  const enabled = uniqCompanyIds(enabledCompanyIds);
  if (enabled.length <= 1) {
    return branchCheckState(companyId, enabled, children) === true;
  }
  const branch = new Set(collectCompanyBranchIds(companyId, children));
  const remaining = enabled.filter(id => !branch.has(id));
  return remaining.length === 0 && branchCheckState(companyId, enabled, children) === true;
}

/**
 * Select-all control: select every allowed company, or clear back to the
 * current active company when the allowlist is already fully selected.
 */
export function toggleSelectAllCompanies(opts: {
  draftActiveCompanyId: string;
  draftEnabledCompanyIds: readonly string[];
  allowedCompanyIds: readonly string[];
}): SwitchCompanyScopeDraft {
  const allowed = uniqCompanyIds(opts.allowedCompanyIds);
  const enabled = uniqCompanyIds(opts.draftEnabledCompanyIds);
  let active = String(opts.draftActiveCompanyId ?? '').trim();
  if (active && !allowed.includes(active)) active = allowed[0] || '';
  if (!active) active = allowed[0] || '';

  const allSelected =
    allowed.length > 0 && allowed.every(id => enabled.includes(id));
  if (allSelected) {
    const onlyActive = active ? [active] : allowed.slice(0, 1);
    return { activeCompanyId: onlyActive[0] || '', enabledCompanyIds: onlyActive };
  }
  return {
    activeCompanyId: active,
    enabledCompanyIds: allowed.includes(active) ? uniqCompanyIds([active, ...allowed]) : allowed,
  };
}

/**
 * Select-all checkbox visual state against the full allowlist.
 */
export function selectAllCheckState(
  enabledCompanyIds: readonly string[],
  allowedCompanyIds: readonly string[],
): boolean | 'indeterminate' {
  const allowed = uniqCompanyIds(allowedCompanyIds);
  const enabled = new Set(uniqCompanyIds(enabledCompanyIds));
  if (!allowed.length) return false;
  const selected = allowed.filter(id => enabled.has(id)).length;
  if (selected === 0) return false;
  if (selected >= allowed.length) return true;
  return 'indeterminate';
}

/**
 * Keep search hits and their already-loaded ancestors so the tree stays connected.
 */
export function filterCompanyTreeBySearch(
  entries: readonly SwitchCompanyTreeEntry[],
  search: string,
): SwitchCompanyTreeEntry[] {
  const filter = String(search ?? '')
    .toLocaleLowerCase()
    .replace(/\s/g, '');
  if (!filter) return entries.slice();

  const parentById = new Map<string, string>();
  for (const entry of entries) {
    for (const childId of entry.childIds) parentById.set(childId, entry.id);
  }

  const matched = new Set<string>();
  for (const entry of entries) {
    const hay = `${entry.name}${entry.code}${entry.id}`.toLocaleLowerCase().replace(/\s/g, '');
    if (hay.includes(filter)) matched.add(entry.id);
  }

  const keep = new Set<string>();
  for (const id of matched) {
    let cur: string | undefined = id;
    let guard = 0;
    while (cur && !keep.has(cur) && guard++ < 64) {
      keep.add(cur);
      cur = parentById.get(cur);
    }
  }

  return entries.filter(e => keep.has(e.id));
}

/**
 * Ensure active stays inside enabled; empty active falls back to the first enabled id.
 */
export function ensureActiveInEnabledScope(
  activeCompanyId: string,
  enabledCompanyIds: readonly string[],
): SwitchCompanyScopeDraft {
  const enabled = uniqCompanyIds(enabledCompanyIds);
  let active = String(activeCompanyId ?? '').trim();
  if (active && !enabled.includes(active)) {
    enabled.unshift(active);
  }
  if (!active) active = enabled[0] || '';
  return { activeCompanyId: active, enabledCompanyIds: uniqCompanyIds(enabled) };
}
