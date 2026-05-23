// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import type { RouteRecordRaw } from 'vue-router';
import type { MenuItem } from '../menu';
import { asObjectRecord } from '../../utils/object';
import type { ObjectRecord } from '../../utils/types';

export type ResourceId = string;
export type ResourceKind = 'route' | 'menu' | 'action';

export type ResourceRequire = {
  kind?: 'rpc';
  model: string;
  method?: string;
};

export type NormalizedResourceRequire = {
  kind: 'rpc';
  model: string;
  method?: string;
};

export type ResourceBaseOptions = {
  title?: string;
  sequence?: number;
  requires?: ResourceRequire[];
  defaultRoles?: string[];
  override?: boolean;
};

export type DefineRouteOptions<T extends RouteRecordRaw = RouteRecordRaw> = T &
  ResourceBaseOptions & {
    actions?: string[];
    meta?: ObjectRecord;
  };

export type DefineMenuOptions = ResourceBaseOptions & {
  parentMenu?: string;
  icon?: MenuItem['icon'];
  path?: string;
  children?: MenuItem[];
  meta?: ObjectRecord;
};

export type DefineActionOptions = ResourceBaseOptions;

export type ResourceDeclarationBase = {
  id: ResourceId;
  kind: ResourceKind;
  title?: string;
  sequence?: number;
  requires: NormalizedResourceRequire[];
  defaultRoles: string[];
  override: boolean;
};

export type RouteResourceDeclaration = ResourceDeclarationBase & {
  kind: 'route';
  path?: string;
  actions: string[];
};

export type MenuResourceDeclaration = ResourceDeclarationBase & {
  kind: 'menu';
  path?: string;
  parentMenu?: string;
};

export type ActionResourceDeclaration = ResourceDeclarationBase & {
  kind: 'action';
};

export type ResourceDeclaration = RouteResourceDeclaration | MenuResourceDeclaration | ActionResourceDeclaration;

export type ResourceMeta = ObjectRecord & {
  resourceId: ResourceId;
  resource: ResourceDeclaration;
};

export type DefineModelActionTitles = Partial<Record<'create' | 'edit' | 'delete' | 'copy', string>>;

export type DefineModelActionsOptions = {
  entityTitle?: string;
  titles?: DefineModelActionTitles;
  exclude?: Array<'create' | 'edit' | 'delete' | 'copy'>;
};

export type ModelActions = {
  create?: string;
  edit?: string;
  delete?: string;
  copy?: string;
};

const resourceDeclarations = new Map<ResourceId, ResourceDeclaration>();

function normalizeTitle(value: unknown): string | undefined {
  const normalized = typeof value === 'string' ? value.trim() : '';
  return normalized || undefined;
}

function normalizeSequence(value: unknown): number | undefined {
  const normalized = Number(value);
  return Number.isFinite(normalized) ? normalized : undefined;
}

function normalizeDefaultRoles(value: unknown): string[] {
  if (!Array.isArray(value)) return [];
  return Array.from(new Set(value.map(item => String(item || '').trim()).filter(Boolean)));
}

function normalizeRequires(value: unknown): NormalizedResourceRequire[] {
  if (!Array.isArray(value)) return [];

  const out: NormalizedResourceRequire[] = [];
  const seen = new Set<string>();

  for (const item of value) {
    const requireItem = asObjectRecord(item);
    const model = String(requireItem?.model || '').trim();
    if (!model) continue;
    const method = normalizeTitle(requireItem?.method);
    const kind: 'rpc' = 'rpc';
    const key = `${kind}:${model}:${method ?? ''}`;
    if (seen.has(key)) continue;
    seen.add(key);
    out.push(method ? { kind, model, method } : { kind, model });
  }

  return out;
}

function normalizeActions(value: unknown): string[] {
  if (!Array.isArray(value)) return [];
  return Array.from(new Set(value.map(item => String(item || '').trim()).filter(Boolean)));
}

function cloneDeclaration<T extends ResourceDeclaration>(declaration: T): T {
  return {
    ...declaration,
    requires: declaration.requires.map(item => ({ ...item })),
    defaultRoles: [...declaration.defaultRoles],
    ...(declaration.kind === 'route' ? { actions: [...declaration.actions] } : {}),
  } as T;
}

function registerResourceDeclaration<T extends ResourceDeclaration>(declaration: T): T {
  const cloned = cloneDeclaration(declaration);
  resourceDeclarations.set(cloned.id, cloned);
  return cloned;
}

function withResourceMeta(meta: ObjectRecord | undefined, declaration: ResourceDeclaration): ResourceMeta {
  return {
    ...(meta ?? {}),
    resourceId: declaration.id,
    resource: cloneDeclaration(declaration),
  } as ResourceMeta;
}

export function getResourceDeclaration(id: ResourceId): ResourceDeclaration | undefined {
  const hit = resourceDeclarations.get(id);
  return hit ? cloneDeclaration(hit) : undefined;
}

export function listResourceDeclarations(): ResourceDeclaration[] {
  return Array.from(resourceDeclarations.values()).map(declaration => cloneDeclaration(declaration));
}

export function clearResourceDeclarations(): void {
  resourceDeclarations.clear();
}

export function getResourceDeclarationFromMeta(meta?: ObjectRecord | null): ResourceDeclaration | undefined {
  const declaration = asObjectRecord(meta)?.resource;
  if (!declaration || typeof declaration !== 'object') return undefined;
  return cloneDeclaration(declaration as ResourceDeclaration);
}

export function defineRoute<T extends RouteRecordRaw>(id: ResourceId, config: DefineRouteOptions<T>): T {
  const declaration = registerResourceDeclaration({
    id,
    kind: 'route',
    title: normalizeTitle(config.title),
    sequence: normalizeSequence(config.sequence),
    path: normalizeTitle((config as { path?: unknown })?.path),
    actions: normalizeActions(config.actions),
    requires: normalizeRequires(config.requires),
    defaultRoles: normalizeDefaultRoles(config.defaultRoles),
    override: Boolean(config.override),
  } satisfies RouteResourceDeclaration);

  const { actions: _actions, title, sequence, requires: _requires, defaultRoles: _defaultRoles, override: _override, ...routeConfig } = config;

  const meta = withResourceMeta(routeConfig.meta, declaration);
  if (typeof title === 'string' && title.trim() !== '' && meta.pageTitle == null) {
    meta.pageTitle = title;
  }
  if (Number.isFinite(Number(sequence))) {
    meta.routeSequence = Number(sequence);
  }

  return {
    ...(routeConfig as T),
    meta,
  } as T;
}

export function defineMenu(id: ResourceId, config: DefineMenuOptions): MenuItem {
  const declaration = registerResourceDeclaration({
    id,
    kind: 'menu',
    title: normalizeTitle(config.title) ?? id,
    sequence: normalizeSequence(config.sequence),
    path: normalizeTitle(config.path),
    parentMenu: normalizeTitle(config.parentMenu),
    requires: normalizeRequires(config.requires),
    defaultRoles: normalizeDefaultRoles(config.defaultRoles),
    override: Boolean(config.override),
  } satisfies MenuResourceDeclaration);

  const out: MenuItem = {
    id,
    title: config.title ?? id,
    icon: config.icon,
    path: config.path,
    order: config.sequence,
    children: config.children,
    meta: withResourceMeta(config.meta, declaration),
  };

  if (!out.children || out.children.length === 0) {
    delete out.children;
  }

  return out;
}

export function defineAction(id: ResourceId, config: DefineActionOptions): string {
  registerResourceDeclaration({
    id,
    kind: 'action',
    title: normalizeTitle(config.title),
    sequence: normalizeSequence(config.sequence),
    requires: normalizeRequires(config.requires),
    defaultRoles: normalizeDefaultRoles(config.defaultRoles),
    override: Boolean(config.override),
  } satisfies ActionResourceDeclaration);

  return id;
}

export function defineModelActions(model: string, options: DefineModelActionsOptions = {}): ModelActions {
  const [appRaw, modelRaw] = String(model || '').split('.');
  const app = String(appRaw || '').trim();
  const modelName = String(modelRaw || '').trim();
  if (!app || !modelName) {
    return {};
  }

  const modelSnake = toSnake(modelName);
  const excludes = new Set((options.exclude ?? []).map(v => String(v)));
  const result: ModelActions = {};

  for (const op of ['create', 'edit', 'delete', 'copy'] as const) {
    if (excludes.has(op)) continue;
    result[op] = `${app}.action.${modelSnake}_${op}`;
  }

  return result;
}

function toSnake(input: string): string {
  const value = String(input || '').trim();
  if (!value) return '';
  return value
    .replace(/([a-z0-9])([A-Z])/g, '$1_$2')
    .replace(/[-\s]+/g, '_')
    .toLowerCase();
}
