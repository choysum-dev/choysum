// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import type { RouteRecordRaw } from 'vue-router';
import type { MenuItem } from '../menu';
import { asObjectRecord, isObjectRecord } from '../../utils/object';
import type { ObjectRecord } from '../../utils/types';
import { isTermReference, createTermReference, type TermReference } from '../../service/i18n';
import { assertNonNegativeInt, normalizeOptionalString } from '../../service/utils/normalization';

export type ResourceId = string;
export type ResourceKind = 'route' | 'menu' | 'action';
export type ResourceTitle = string | TermReference;

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
  title?: ResourceTitle;
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
  titleText?: TermReference;
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

export type DefineModelActionTitles = Partial<Record<'create' | 'edit' | 'delete' | 'copy', ResourceTitle>>;

export type DefineModelActionsOptions = {
  entityTitle?: ResourceTitle;
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
  return normalizeOptionalString(value);
}

function normalizeResourceTitle(value: unknown): { title?: string; titleText?: TermReference } {
  if (isTermReference(value)) {
    const title = normalizeTitle(value.src);
    return title ? { title, titleText: { ...value } } : {};
  }
  const title = normalizeTitle(value);
  return title ? { title } : {};
}

function normalizeSequence(value: unknown): number | undefined {
  if (value === undefined || value === null || (typeof value === 'string' && value.trim() === '')) {
    return undefined;
  }
  return assertNonNegativeInt(value);
}

function normalizeStringListField(value: unknown, errorCode: string): string[] {
  if (value == null) return [];
  if (!Array.isArray(value)) {
    throw new Error(errorCode);
  }

  const out: string[] = [];
  const seen = new Set<string>();
  for (const item of value) {
    if (typeof item !== 'string') {
      throw new Error(errorCode);
    }
    const normalized = item.trim();
    if (!normalized || seen.has(normalized)) continue;
    seen.add(normalized);
    out.push(normalized);
  }
  return out;
}

function normalizeDefaultRoles(value: unknown): string[] {
  return normalizeStringListField(value, 'invalid_resource_default_roles');
}

function normalizeRequires(value: unknown): NormalizedResourceRequire[] {
  if (value == null) return [];
  if (!Array.isArray(value)) {
    throw new Error('invalid_resource_requires');
  }

  const out: NormalizedResourceRequire[] = [];
  const seen = new Set<string>();

  for (const item of value) {
    if (!isObjectRecord(item)) {
      throw new Error('invalid_resource_requires');
    }
    if (typeof item.model !== 'string' || (item.method != null && typeof item.method !== 'string')) {
      throw new Error('invalid_resource_requires');
    }
    if (item.kind != null && item.kind !== 'rpc') {
      throw new Error('invalid_resource_requires');
    }
    const model = normalizeOptionalString(item.model);
    if (!model) {
      throw new Error('invalid_resource_requires');
    }
    const method = normalizeOptionalString(item.method);
    const kind: 'rpc' = 'rpc';
    const key = `${kind}:${model}:${method ?? ''}`;
    if (seen.has(key)) continue;
    seen.add(key);
    out.push(method ? { kind, model, method } : { kind, model });
  }

  return out;
}

function normalizeActions(value: unknown): string[] {
  return normalizeStringListField(value, 'invalid_resource_actions');
}

function cloneDeclaration<T extends ResourceDeclaration>(declaration: T): T {
  return {
    ...declaration,
    requires: (declaration.requires ?? []).map(item => ({ ...item })),
    defaultRoles: [...(declaration.defaultRoles ?? [])],
    ...(declaration.titleText ? { titleText: { ...declaration.titleText } } : {}),
    ...(declaration.kind === 'route' ? { actions: [...(declaration.actions ?? [])] } : {}),
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
  const normalizedTitle = normalizeResourceTitle(config.title);
  const sequence = normalizeSequence(config.sequence);
  const declaration = registerResourceDeclaration({
    id,
    kind: 'route',
    ...normalizedTitle,
    sequence,
    path: normalizeTitle((config as { path?: unknown })?.path),
    actions: normalizeActions(config.actions),
    requires: normalizeRequires(config.requires),
    defaultRoles: normalizeDefaultRoles(config.defaultRoles),
    override: Boolean(config.override),
  } satisfies RouteResourceDeclaration);

  const { actions: _actions, title, sequence: _sequence, requires: _requires, defaultRoles: _defaultRoles, override: _override, ...routeConfig } = config;

  const meta = withResourceMeta(routeConfig.meta, declaration);
  if (normalizedTitle.title && meta.pageTitle == null) {
    meta.pageTitle = normalizedTitle.title;
  }
  if (normalizedTitle.titleText && meta.pageTitleText == null) {
    meta.pageTitleText = normalizedTitle.titleText;
  }
  if (sequence !== undefined) {
    meta.routeSequence = sequence;
  }

  return {
    ...(routeConfig as T),
    meta,
  } as T;
}

export function defineMenu(id: ResourceId, config: DefineMenuOptions): MenuItem {
  const normalizedTitle = normalizeResourceTitle(config.title);
  const sequence = normalizeSequence(config.sequence);
  const declaration = registerResourceDeclaration({
    id,
    kind: 'menu',
    title: normalizedTitle.title ?? id,
    titleText: normalizedTitle.titleText,
    sequence,
    path: normalizeTitle(config.path),
    parentMenu: normalizeTitle(config.parentMenu),
    requires: normalizeRequires(config.requires),
    defaultRoles: normalizeDefaultRoles(config.defaultRoles),
    override: Boolean(config.override),
  } satisfies MenuResourceDeclaration);

  const out: MenuItem = {
    id,
    title: normalizedTitle.title ?? id,
    titleText: normalizedTitle.titleText,
    icon: config.icon,
    path: config.path,
    order: sequence,
    children: config.children,
    meta: withResourceMeta(config.meta, declaration),
  };

  if (!out.children || out.children.length === 0) {
    delete out.children;
  }

  return out;
}

export function defineAction(id: ResourceId, config: DefineActionOptions): string {
  const normalizedTitle = normalizeResourceTitle(config.title);
  registerResourceDeclaration({
    id,
    kind: 'action',
    ...normalizedTitle,
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
  const normalizedEntityTitle = normalizeResourceTitle(options.entityTitle);
  const titleOverrides = Object.fromEntries(
    Object.entries(options.titles ?? {})
      .map(([op, title]) => [op, normalizeResourceTitle(title)] as const)
      .filter(([, normalized]) => Boolean(normalized.title))
  ) as Partial<Record<'create' | 'edit' | 'delete' | 'copy', ReturnType<typeof normalizeResourceTitle>>>;
  const prefixes: Record<'create' | 'edit' | 'delete' | 'copy', string> = {
    create: 'Create ',
    edit: 'Edit ',
    delete: 'Delete ',
    copy: 'Copy ',
  };
  const result: ModelActions = {};

  for (const op of ['create', 'edit', 'delete', 'copy'] as const) {
    if (excludes.has(op)) continue;
    const id = `${app}.action.${modelSnake}_${op}`;
    result[op] = id;

    const override = titleOverrides[op];
    const titleConfig = override?.title
      ? override
      : normalizedEntityTitle.title
        ? {
            title: `${prefixes[op]}${normalizedEntityTitle.title}`,
            titleText: normalizedEntityTitle.titleText
              ? createTermReference(normalizedEntityTitle.titleText.module, `${prefixes[op]}${normalizedEntityTitle.titleText.src}`, {
                  scope: normalizedEntityTitle.titleText.scope,
                  kind: normalizedEntityTitle.titleText.kind,
                })
              : undefined,
          }
        : {};

    registerResourceDeclaration({
      id,
      kind: 'action',
      ...titleConfig,
      requires: [],
      defaultRoles: [],
      override: false,
    } satisfies ActionResourceDeclaration);
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
