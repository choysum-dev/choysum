// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import type { ViewMode } from './viewScopeKeys';

/** Open view-kind id; builtins and apps register via registerViewKind. */
export type ViewKindId = string;

/** How FieldBase / useField should lay out under this kind. */
export type FieldPresentation = 'form' | 'cell' | 'none';

export type ViewKindDef = {
  id: ViewKindId;
  /** Default viewMode when the kind provides scope. */
  defaultMode?: ViewMode;
  /** Field layout contract for useField / FieldBase. */
  fieldPresentation: FieldPresentation;
};

const registry = new Map<ViewKindId, ViewKindDef>();

/** Register or replace a view kind definition. */
export function registerViewKind(def: ViewKindDef): void {
  const id = String(def.id || '').trim();
  if (!id) {
    console.warn('[viewKinds] registerViewKind ignored empty id');
    return;
  }
  registry.set(id, { ...def, id });
}

/** Resolve a registered kind; unknown ids warn and fall back to presentation none. */
export function getViewKind(id: ViewKindId | null | undefined): ViewKindDef {
  const key = String(id ?? '').trim();
  if (key && registry.has(key)) return registry.get(key)!;
  if (key) {
    console.warn(`[viewKinds] unknown view kind "${key}"; fieldPresentation=none`);
  }
  return { id: key || 'unknown', fieldPresentation: 'none', defaultMode: 'display' };
}

export function listViewKinds(): ViewKindDef[] {
  return [...registry.values()];
}

/** Clear registry (tests). */
export function clearViewKindsForTests(): void {
  registry.clear();
}

let builtinsRegistered = false;

/** Idempotent builtin Form/List/Kanban/Chart/Search registration. */
export function registerBuiltinViewKinds(): void {
  if (builtinsRegistered) return;
  builtinsRegistered = true;
  registerViewKind({ id: 'Form', defaultMode: 'edit', fieldPresentation: 'form' });
  registerViewKind({ id: 'List', defaultMode: 'display', fieldPresentation: 'cell' });
  registerViewKind({ id: 'Kanban', defaultMode: 'display', fieldPresentation: 'cell' });
  registerViewKind({ id: 'Chart', defaultMode: 'display', fieldPresentation: 'none' });
  registerViewKind({ id: 'Search', defaultMode: 'display', fieldPresentation: 'none' });
}

/** Test helper to allow re-registering builtins after clear. */
export function resetBuiltinViewKindsRegistrationForTests(): void {
  builtinsRegistered = false;
}
