// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * FE helpers for S7 metadata terminology (field_label / selection_label / menu / route / action).
 * Looks up Gateway `metadata` catalog (not vue-i18n messages). Exact scope first, then soft match.
 */

export type MetadataCatalog = Record<
  string, // module
  Record<
    string, // scope
    Record<
      string, // kind
      Record<string, string> // src → value
    >
  >
>;

type MetaLookupOptions = {
  module?: string;
  scope?: string;
  kind: string;
  src: string;
  /** Soft-match: scope must end with this suffix (e.g. `@Name` or `@Type.checking`). */
  scopeSuffix?: string;
};

const FLAG_KEY = 'choysum.web.i18n.metadataTerms';

let catalogRef: MetadataCatalog = {};

/** Replace the in-memory metadata catalog (called after Gateway load). */
export function setMetadataCatalog(catalog: MetadataCatalog | null | undefined): void {
  catalogRef = catalog && typeof catalog === 'object' ? catalog : {};
}

export function getMetadataCatalog(): MetadataCatalog {
  return catalogRef;
}

/** Runtime flag: localStorage / __CHOYSUM_I18N_METADATA_TERMS__ = '0' disables. Default on. */
export function isMetadataTermsEnabled(): boolean {
  if (typeof globalThis === 'undefined') {
    return true;
  }
  const g = globalThis as {
    localStorage?: Storage;
    __CHOYSUM_I18N_METADATA_TERMS__?: string;
  };
  const fromGlobal = String(g.__CHOYSUM_I18N_METADATA_TERMS__ ?? '').trim().toLowerCase();
  if (fromGlobal === '0' || fromGlobal === 'false' || fromGlobal === 'off') {
    return false;
  }
  if (fromGlobal === '1' || fromGlobal === 'true' || fromGlobal === 'on') {
    return true;
  }
  try {
    const fromStorage = String(g.localStorage?.getItem(FLAG_KEY) ?? '')
      .trim()
      .toLowerCase();
    if (fromStorage === '0' || fromStorage === 'false' || fromStorage === 'off') {
      return false;
    }
  } catch {
    // ignore
  }
  return true;
}

function lookupMeta(opts: MetaLookupOptions): string {
  const src = String(opts.src ?? '');
  if (!src || !isMetadataTermsEnabled()) {
    return src;
  }
  const kind = String(opts.kind || '').trim();
  if (!kind) {
    return src;
  }
  const moduleName = String(opts.module || '').trim();
  const scope = String(opts.scope || '').trim();
  const suffix = String(opts.scopeSuffix || '').trim();

  const modules = moduleName ? [moduleName] : Object.keys(catalogRef);
  for (const mod of modules) {
    const byScope = catalogRef[mod];
    if (!byScope) {
      continue;
    }
    if (scope) {
      const hit = byScope[scope]?.[kind]?.[src];
      if (hit) {
        return hit;
      }
    }
    if (suffix) {
      for (const [scp, byKind] of Object.entries(byScope)) {
        if (!scp.endsWith(suffix)) {
          continue;
        }
        const hit = byKind?.[kind]?.[src];
        if (hit) {
          return hit;
        }
      }
    }
    // Soft: any scope in module with same kind+src.
    for (const byKind of Object.values(byScope)) {
      const hit = byKind?.[kind]?.[src];
      if (hit) {
        return hit;
      }
    }
  }
  return src;
}

export function tFieldLabel(opts: {
  src: string;
  prop?: string;
  module?: string;
  path?: string;
}): string {
  const src = String(opts.src ?? '');
  const prop = String(opts.prop || '').trim();
  const path = String(opts.path || '').trim();
  const scope = path && prop ? `${path}@${prop}` : undefined;
  return lookupMeta({
    module: opts.module,
    scope,
    kind: 'field_label',
    src,
    scopeSuffix: prop ? `@${prop}` : undefined,
  });
}

export function tSelectionLabel(opts: {
  src: string;
  field?: string;
  value?: string;
  module?: string;
  path?: string;
}): string {
  const src = String(opts.src ?? '');
  const field = String(opts.field || '').trim();
  const value = String(opts.value || '').trim();
  const path = String(opts.path || '').trim();
  const location = field && value ? `${field}.${value}` : '';
  const scope = path && location ? `${path}@${location}` : undefined;
  return lookupMeta({
    module: opts.module,
    scope,
    kind: 'selection_label',
    src,
    scopeSuffix: location ? `@${location}` : undefined,
  });
}

export function tMenuTitle(opts: {
  src: string;
  menuId: string;
  module?: string;
  path?: string;
}): string {
  const src = String(opts.src ?? '');
  const menuId = String(opts.menuId || '').trim();
  const path = String(opts.path || 'web/menu/menus').trim();
  const moduleName = String(opts.module || moduleFromResourceId(menuId) || '').trim();
  const scope = menuId ? `${path}@${menuId}` : undefined;
  return lookupMeta({
    module: moduleName || undefined,
    scope,
    kind: 'menu',
    src,
    scopeSuffix: menuId ? `@${menuId}` : undefined,
  });
}

export function tRouteTitle(opts: {
  src: string;
  routeId: string;
  module?: string;
  path?: string;
}): string {
  const src = String(opts.src ?? '');
  const routeId = String(opts.routeId || '').trim();
  const path = String(opts.path || 'web/route/routes').trim();
  const moduleName = String(opts.module || moduleFromResourceId(routeId) || '').trim();
  const scope = routeId ? `${path}@${routeId}` : undefined;
  return lookupMeta({
    module: moduleName || undefined,
    scope,
    kind: 'route',
    src,
    scopeSuffix: routeId ? `@${routeId}` : undefined,
  });
}

export function tActionTitle(opts: {
  src: string;
  actionId: string;
  module?: string;
  path?: string;
}): string {
  const src = String(opts.src ?? '');
  const actionId = String(opts.actionId || '').trim();
  const path = String(opts.path || 'web/actions').trim();
  const moduleName = String(opts.module || moduleFromResourceId(actionId) || '').trim();
  const scope = actionId ? `${path}@${actionId}` : undefined;
  return lookupMeta({
    module: moduleName || undefined,
    scope,
    kind: 'action',
    src,
    scopeSuffix: actionId ? `@${actionId}` : undefined,
  });
}

function moduleFromResourceId(id: string): string {
  const part = String(id || '').split('.')[0];
  return part || '';
}
