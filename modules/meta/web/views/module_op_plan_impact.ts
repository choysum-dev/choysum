// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/** One module step in the confirm-dialog plan view. */
export type ModuleOpPlanModuleView = {
  moduleName: string;
  application?: string;
  role?: string;
  effect?: string;
  reason?: string;
  currentVersion?: string;
  targetVersion?: string;
  origin?: string;
};

export function isChangingPlanEffect(effect?: string): boolean {
  const e = String(effect || '').trim();
  return e === 'install' || e === 'upgrade' || e === 'uninstall';
}

export function isSatisfiedPlanEffect(effect?: string): boolean {
  return String(effect || '').trim() === 'already_satisfied';
}

/** Map a Plan ModuleOrder/EnsureModules entry (PascalCase) into the drawer view shape. */
export function mapPlanModuleEntry(entry: any): ModuleOpPlanModuleView {
  return {
    moduleName: String(entry?.ModuleName || '').trim(),
    application: entry?.Application != null ? String(entry.Application).trim() : undefined,
    role: entry?.Role != null ? String(entry.Role) : undefined,
    effect: entry?.Effect != null ? String(entry.Effect) : undefined,
    reason: entry?.Reason != null ? String(entry.Reason) : undefined,
    currentVersion: entry?.CurrentVersion != null ? String(entry.CurrentVersion) : undefined,
    targetVersion: entry?.TargetVersion != null ? String(entry.TargetVersion) : undefined,
    origin: entry?.Origin != null ? String(entry.Origin) : undefined,
  };
}

/**
 * Partition ModuleOrder (+ EnsureModules) into changing vs already-satisfied.
 * Target module is sorted first within the changing list.
 */
export function partitionPlanModules(
  moduleOrder: ModuleOpPlanModuleView[],
  ensureModules: ModuleOpPlanModuleView[] = [],
): { changing: ModuleOpPlanModuleView[]; satisfied: ModuleOpPlanModuleView[] } {
  const seen = new Set<string>();
  const changing: ModuleOpPlanModuleView[] = [];
  const satisfied: ModuleOpPlanModuleView[] = [];
  const push = (item: ModuleOpPlanModuleView) => {
    const name = item.moduleName;
    if (!name || seen.has(name)) return;
    seen.add(name);
    // Legacy plans without Effect: treat as changing so the dialog still lists them.
    if (!item.effect || isChangingPlanEffect(item.effect)) {
      changing.push(item);
    } else if (isSatisfiedPlanEffect(item.effect)) {
      satisfied.push(item);
    } else {
      changing.push(item);
    }
  };
  for (const item of moduleOrder) push(item);
  for (const item of ensureModules) push(item);

  changing.sort((a, b) => {
    const aTarget = a.role === 'target' ? 0 : 1;
    const bTarget = b.role === 'target' ? 0 : 1;
    if (aTarget !== bTarget) return aTarget - bTarget;
    return 0;
  });
  return { changing, satisfied };
}

/** Distinct applications for modules that will actually change (not satisfied). */
export function changedApplicationsFromModules(changing: ModuleOpPlanModuleView[]): string[] {
  const out: string[] = [];
  const seen = new Set<string>();
  for (const item of changing) {
    const app = String(item.application || '').trim();
    if (!app || seen.has(app)) continue;
    seen.add(app);
    out.push(app);
  }
  out.sort();
  return out;
}

/**
 * Show the applications block only when multiple changing modules collapse to
 * fewer applications. Name mismatches are shown inline on the module row instead.
 */
export function shouldShowAffectedApplications(
  changing: ModuleOpPlanModuleView[],
  applications: string[],
): boolean {
  if (!applications.length || !changing.length) return false;
  return changing.length >= 2 && applications.length < changing.length;
}

/** True when Application differs from ModuleName (inline affiliation hint). */
export function moduleShowsApplicationHint(item: ModuleOpPlanModuleView): boolean {
  const app = String(item.application || '').trim();
  const name = String(item.moduleName || '').trim();
  return !!app && !!name && app !== name;
}

/** Display version for the confirm dialog; keep the full string (no ellipsis). */
export function formatPlanVersionDisplay(version?: string): string {
  return String(version || '').trim();
}

/** Whether to render current → target version arrow. */
export function shouldShowPlanVersionArrow(item: ModuleOpPlanModuleView): boolean {
  if (item.effect === 'uninstall') return false;
  const current = String(item.currentVersion || '').trim();
  const target = String(item.targetVersion || '').trim();
  if (!target) return false;
  return current !== target;
}

/** Non-i18n effect token for summaries (never emit legacy reason: root/dependency). */
export function planEffectSummaryToken(effect?: string): string {
  switch (String(effect || '').trim()) {
    case 'install':
      return 'will install';
    case 'upgrade':
      return 'will upgrade';
    case 'uninstall':
      return 'will uninstall';
    case 'already_satisfied':
      return 'already satisfied';
    default:
      return 'will change';
  }
}

/**
 * Condense changing (+ optional satisfied count) into English summary text.
 * Prefer Effect over legacy Reason so progress/result copy stays aligned with the plan dialog.
 */
export function formatModuleOpPlanChangeSummary(
  moduleOrder: ModuleOpPlanModuleView[],
  ensureModules: ModuleOpPlanModuleView[] = [],
): string {
  const { changing, satisfied } = partitionPlanModules(moduleOrder, ensureModules);
  if (!changing.length && !satisfied.length) return '';
  const parts = changing.map(m => `${m.moduleName} (${planEffectSummaryToken(m.effect)})`);
  let out = parts.join(', ');
  if (satisfied.length) {
    const suffix = `${satisfied.length} already satisfied dependency module(s) skipped`;
    out = out ? `${out}; ${suffix}` : suffix;
  }
  return out;
}
