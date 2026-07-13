// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { BaseModel, Compute, Field, Model } from '@/core/service';
import type { QueryCondition, SearchOptions } from '@/core/service/api/query';
import type { FieldSelection } from '@/core/service/api/selection';
import IrApplication from './ir_application';
import IrModule from './ir_module';
import IrUiResourceMenuRoute from './ir_ui_resource_menu_route';
import IrUiResourceRouteAction from './ir_ui_resource_route_action';
import { normalizeOptionalString, normalizeStringArray, readRefId } from '@/core/service/utils/normalization';
import { normalizePagination, paginateAndWrap } from '@/core/service/utils/pagination';

export type UiResourceType = 'ROUTE' | 'MENU' | 'ACTION';

type EffectiveUiResourceDeclarationKind = 'route' | 'menu' | 'action';

type EffectiveUiResourceRequire = {
  kind: 'rpc';
  model: string;
  method?: string;
};

type EffectiveUiResourceDeclaration = {
  id: string;
  kind: EffectiveUiResourceDeclarationKind;
  title?: string;
  sequence?: number;
  path?: string;
  parentMenu?: string;
  actions?: string[];
  requires: EffectiveUiResourceRequire[];
  defaultRoles: string[];
  override: false;
  module?: string;
  application?: string;
};

type EffectiveUiResourceDeclarationOptions = {
  module?: string;
  application?: string;
  kind?: EffectiveUiResourceDeclarationKind;
  ids?: string[];
  limit?: number;
  offset?: number;
};

type UiResourceChildProjection = {
  Id: string;
  Name?: string;
  Type?: string;
  Title?: string;
  Sequence?: number;
  Requires?: string[] | null;
  Module?: string;
  Path?: string;
  ParentPath?: string | null;
  UiPath?: string;
  DefaultRoles?: string[] | null;
};

type MutableChildCarrier = {
  Childs?: UiResourceChildProjection[];
};

type IrUiResourceComputeDeps = IrUiResource & Record<'ParentId.ParentPath', unknown>;

const uiResourceChildProjectionFields = [
  'Id',
  'Name',
  'Type',
  'Title',
  'Sequence',
  'Requires',
  'Module',
  'Path',
  'ParentPath',
  'UiPath',
  'DefaultRoles',
] as const;

function wantsChildsField(selection: unknown): boolean {
  if (selection == null) return false;
  if (typeof selection === 'string') return selection === 'Childs';
  if (Array.isArray(selection)) return selection.some(item => wantsChildsField(item));
  if (typeof selection === 'object' && Array.isArray((selection as { fields?: unknown[] }).fields)) {
    return wantsChildsField((selection as { fields?: unknown[] }).fields);
  }
  return false;
}

function stripChildsFromSelection<T>(selection: T): T {
  const strip = (value: unknown): unknown => {
    if (value == null) return value;
    if (typeof value === 'string') return value === 'Childs' ? undefined : value;
    if (Array.isArray(value)) {
      return value.map(item => strip(item)).filter(item => item !== undefined);
    }
    if (typeof value === 'object' && Array.isArray((value as { fields?: unknown[] }).fields)) {
      const record = value as Record<string, unknown>;
      return {
        ...record,
        fields: strip((value as { fields?: unknown[] }).fields),
      };
    }
    return value;
  };

  return strip(selection) as T;
}

@Model('IrUiResource', {
  tableName: 'meta_ir_ui_resource',
  parentField: 'ParentId',
  autoMigrate: false,
})
export default class IrUiResource extends BaseModel {
  @Field({ type: 'varchar', size: 255, notNull: true, unique: true, index: true })
  Name!: string;

  @Field({ type: 'varchar', size: 16, notNull: true })
  Type!: UiResourceType;

  @Field({ type: 'varchar', size: 255 })
  Title?: string;

  @Field({ type: 'int', default: 0 })
  Sequence?: number;

  @Field({ type: 'jsonobject' })
  Requires?: string[] | null;

  @Field({ type: 'varchar', size: 255, index: true })
  Module?: string;

  @Field({ type: 'varchar', size: 1024, index: true })
  Path?: string;

  @Field({ type: 'ManyToOne', relation: { targetModel: () => IrUiResource }, notNull: false, index: true })
  ParentId?: IrUiResource;

  @Field({
    type: 'varchar',
    size: 1000,
    indexed: true,
  })
  readonly ParentPath?: string | null;

  @Compute<IrUiResourceComputeDeps>('ParentPath', {
    deps: ['Id', 'Type', 'ParentId', 'ParentId.ParentPath'],
  })
  computeParentPath() {
    const id = String(this.Id || '').trim();
    if (!id) return null;
    if (String(this.Type || '').trim() !== 'MENU') return null;

    const parentRef = this.ParentId;
    const parentPath = parentRef ? String(parentRef.ParentPath || '') : '';
    if (parentPath && parentPath.includes(`${id}/`)) {
      throw new Error(`Cycle detected: ${id} cannot be assigned under one of its descendants`);
    }

    return `${parentPath}${id}/`;
  }

  @Field({ type: 'varchar', size: 512 })
  UiPath?: string;

  @Field({ type: 'jsonobject' })
  DefaultRoles?: string[] | null;

  @Field({ type: 'ManyToOne', relation: { targetModel: () => IrApplication }, notNull: false, index: true })
  IrApplicationId?: IrApplication;

  @Field({ type: 'ManyToOne', relation: { targetModel: () => IrModule }, notNull: false, index: true })
  ModuleId?: IrModule;

  @Field({
    type: 'OneToMany',
    relation: { targetModel: () => IrUiResource },
  })
  readonly Childs?: IrUiResource[];

  private static normalizeChildProjection(row: IrUiResource): UiResourceChildProjection | null {
    const id = String(row.Id || '').trim();
    if (!id) return null;

    return {
      Id: id,
      Name: normalizeOptionalString(row.Name),
      Type: normalizeOptionalString(row.Type),
      Title: normalizeOptionalString(row.Title),
      Sequence: Number(row.Sequence || 0),
      Requires: normalizeStringArray(row.Requires),
      Module: normalizeOptionalString(row.Module),
      Path: normalizeOptionalString(row.Path),
      ParentPath: normalizeOptionalString(row.ParentPath),
      UiPath: normalizeOptionalString(row.UiPath),
      DefaultRoles: normalizeStringArray(row.DefaultRoles),
    };
  }

  private static sortChildRows(rows: UiResourceChildProjection[]): UiResourceChildProjection[] {
    return [...rows].sort((a, b) => {
      const seqA = Number(a.Sequence ?? 0);
      const seqB = Number(b.Sequence ?? 0);
      if (seqA !== seqB) return seqA - seqB;
      return String(a.Name || '').localeCompare(String(b.Name || ''));
    });
  }

  private static async loadChildProjectionMap(ids: string[]): Promise<Map<string, UiResourceChildProjection>> {
    if (!ids.length) return new Map();

    const rows = (await super.Search(
      ['Id', 'in', ids] as any,
      {
        fields: [...uiResourceChildProjectionFields],
        limit: Math.max(1000, ids.length * 4),
      } as any
    )) as IrUiResource[];

    const map = new Map<string, UiResourceChildProjection>();
    for (const row of rows || []) {
      const normalized = this.normalizeChildProjection(row);
      if (!normalized) continue;
      map.set(normalized.Id, normalized);
    }
    return map;
  }

  private static async hydrateChildsField(records: IrUiResource[]): Promise<void> {
    if (!Array.isArray(records) || records.length === 0) return;

    const menuIds: string[] = [];
    const routeIds: string[] = [];
    const childMap = new Map<string, UiResourceChildProjection[]>();

    for (const row of records) {
      const id = String(row.Id || '').trim();
      if (!id) continue;

      const type = String(row.Type || '')
        .trim()
        .toUpperCase();

      childMap.set(id, []);
      if (type === 'MENU') menuIds.push(id);
      if (type === 'ROUTE') routeIds.push(id);
    }

    if (menuIds.length > 0) {
      const directRows = (await super.Search(
        ['ParentId', 'in', menuIds] as any,
        {
          fields: [...uiResourceChildProjectionFields, 'ParentId'],
          limit: Math.max(1000, menuIds.length * 200),
        } as any
      )) as IrUiResource[];

      for (const row of directRows || []) {
        const parentId = readRefId(row.ParentId);
        if (!parentId || !childMap.has(parentId)) continue;
        const normalized = this.normalizeChildProjection(row);
        if (normalized) childMap.get(parentId)!.push(normalized);
      }

      const menuRouteRows = (await IrUiResourceMenuRoute.Search(
        ['MenuUiResourceId', 'in', menuIds] as unknown as QueryCondition<IrUiResourceMenuRoute>,
        {
          fields: ['MenuUiResourceId', 'RouteUiResourceId'],
          limit: Math.max(1000, menuIds.length * 200),
        } as unknown as SearchOptions<IrUiResourceMenuRoute>
      )) as IrUiResourceMenuRoute[];

      const routeIdsFromMenu = Array.from(
        new Set((menuRouteRows || []).map(row => readRefId(row.RouteUiResourceId)).filter((value): value is string => !!value))
      );

      const routeMap = await this.loadChildProjectionMap(routeIdsFromMenu);
      for (const row of menuRouteRows || []) {
        const menuId = readRefId(row.MenuUiResourceId);
        const routeId = readRefId(row.RouteUiResourceId);
        if (!menuId || !routeId || !childMap.has(menuId)) continue;
        const routeProjection = routeMap.get(routeId);
        if (routeProjection) childMap.get(menuId)!.push(routeProjection);
      }
    }

    if (routeIds.length > 0) {
      const routeActionRows = (await IrUiResourceRouteAction.Search(
        ['RouteUiResourceId', 'in', routeIds] as unknown as QueryCondition<IrUiResourceRouteAction>,
        {
          fields: ['RouteUiResourceId', 'ActionUiResourceId'],
          limit: Math.max(1000, routeIds.length * 200),
        } as unknown as SearchOptions<IrUiResourceRouteAction>
      )) as IrUiResourceRouteAction[];

      const actionIds = Array.from(new Set((routeActionRows || []).map(row => readRefId(row.ActionUiResourceId)).filter((value): value is string => !!value)));

      const actionMap = await this.loadChildProjectionMap(actionIds);
      for (const row of routeActionRows || []) {
        const routeId = readRefId(row.RouteUiResourceId);
        const actionId = readRefId(row.ActionUiResourceId);
        if (!routeId || !actionId || !childMap.has(routeId)) continue;
        const actionProjection = actionMap.get(actionId);
        if (actionProjection) childMap.get(routeId)!.push(actionProjection);
      }
    }

    for (const row of records) {
      const id = String(row.Id || '').trim();
      if (!id) continue;

      const dedup = new Map<string, UiResourceChildProjection>();
      for (const child of childMap.get(id) || []) {
        if (!child.Id) continue;
        dedup.set(child.Id, child);
      }

      const sortedChildren = this.sortChildRows(Array.from(dedup.values()));
      (row as MutableChildCarrier).Childs = sortedChildren;
    }
  }

  static override async Browse<T extends BaseModel>(
    this: { new (...args: any[]): T } & typeof BaseModel,
    id: string,
    fields?: FieldSelection<T>,
    options?: any
  ): Promise<T> {
    const shouldHydrateChilds = wantsChildsField(fields);
    const effectiveFields = shouldHydrateChilds ? stripChildsFromSelection(fields) : fields;
    const row = (await super.Browse(id, effectiveFields as any, options as any)) as any;
    if (shouldHydrateChilds) {
      await IrUiResource.hydrateChildsField([row as IrUiResource]);
    }
    return row as T;
  }

  static override async BrowseMany<T extends BaseModel>(
    this: { new (...args: any[]): T } & typeof BaseModel,
    ids: string[],
    fields?: (keyof any)[],
    options?: any
  ): Promise<T[]> {
    const shouldHydrateChilds = wantsChildsField(fields);
    const effectiveFields = shouldHydrateChilds ? stripChildsFromSelection(fields) : fields;
    const rows = (await super.BrowseMany(ids as any, effectiveFields as any, options as any)) as any[];
    if (shouldHydrateChilds) {
      await IrUiResource.hydrateChildsField(rows as IrUiResource[]);
    }
    return rows as T[];
  }

  static override async Search<T extends BaseModel>(
    this: { new (...args: any[]): T } & typeof BaseModel,
    condition: QueryCondition<T> | [] = [],
    options?: SearchOptions<T>
  ): Promise<T[]> {
    const shouldHydrateChilds = wantsChildsField(options?.fields);
    const effectiveOptions = shouldHydrateChilds
      ? ({
          ...(options || {}),
          fields: stripChildsFromSelection(options?.fields),
        } as SearchOptions<T>)
      : options;

    const rows = (await super.Search(condition as any, effectiveOptions as any)) as any[];
    if (shouldHydrateChilds) {
      await IrUiResource.hydrateChildsField(rows as IrUiResource[]);
    }
    return rows as T[];
  }

  private static normalizeRequireToken(token: string): EffectiveUiResourceRequire | null {
    const raw = String(token || '').trim();
    if (!raw.startsWith('rpc:/')) return null;

    const body = raw.slice('rpc:/'.length);
    const slashIndex = body.indexOf('/');
    if (slashIndex <= 0) return null;

    const model = String(body.slice(0, slashIndex) || '').trim();
    const method = String(body.slice(slashIndex + 1) || '').trim();
    if (!model) return null;
    if (!method || method === '*') {
      return { kind: 'rpc', model };
    }
    return { kind: 'rpc', model, method };
  }

  private static normalizeRequires(value: unknown): EffectiveUiResourceRequire[] {
    const seen = new Set<string>();
    const out: EffectiveUiResourceRequire[] = [];
    for (const token of normalizeStringArray(value)) {
      const normalized = this.normalizeRequireToken(token);
      if (!normalized) continue;
      const key = `${normalized.kind}:${normalized.model}:${normalized.method ?? ''}`;
      if (seen.has(key)) continue;
      seen.add(key);
      out.push(normalized);
    }
    return out;
  }

  private static normalizeKind(value: unknown): EffectiveUiResourceDeclarationKind | undefined {
    const raw = String(value || '')
      .trim()
      .toUpperCase();
    switch (raw) {
      case 'ROUTE':
        return 'route';
      case 'MENU':
        return 'menu';
      case 'ACTION':
        return 'action';
      default:
        return undefined;
    }
  }

  private static normalizeKindFilter(value: unknown): UiResourceType | undefined {
    const raw = String(value || '')
      .trim()
      .toLowerCase();
    switch (raw) {
      case 'route':
        return 'ROUTE';
      case 'menu':
        return 'MENU';
      case 'action':
        return 'ACTION';
      default:
        return undefined;
    }
  }

  private static normalizeSequence(value: unknown): number | undefined {
    const normalized = Number(value);
    if (!Number.isFinite(normalized)) return undefined;
    return normalized === 0 ? undefined : normalized;
  }

  static async GetEffectiveDeclarations(options?: EffectiveUiResourceDeclarationOptions): Promise<{
    declarations: EffectiveUiResourceDeclaration[];
    total: number;
    filtered: number;
    offset: number;
    limit?: number;
    returned: number;
  }> {
    const moduleFilter = normalizeOptionalString(options?.module);
    const applicationFilter = normalizeOptionalString(options?.application);
    const kindFilter = this.normalizeKindFilter(options?.kind);
    const idsFilter = normalizeStringArray(options?.ids);
    const pagination = normalizePagination(options);

    const conditionParts: any[] = [];
    if (moduleFilter) conditionParts.push(['Module', '=', moduleFilter]);
    if (applicationFilter) conditionParts.push(['IrApplicationId', '=', applicationFilter]);
    if (kindFilter) conditionParts.push(['Type', '=', kindFilter]);
    if (idsFilter.length > 0) conditionParts.push(['Name', 'in', idsFilter]);
    const condition: any = conditionParts.length <= 1 ? (conditionParts[0] ?? []) : { And: conditionParts };

    const rows = await this.Search(
      condition as any,
      {
        fields: ['Id', 'Name', 'Type', 'Title', 'Sequence', 'Requires', 'Module', 'UiPath', 'DefaultRoles', 'IrApplicationId', 'ParentId'] as any,
        limit: 50000,
      } as any
    );

    const sortedRows = [...(rows || [])].sort((a: any, b: any) => {
      const kindA = String(a?.Type || '');
      const kindB = String(b?.Type || '');
      if (kindA !== kindB) return kindA.localeCompare(kindB);
      return String(a?.Name || '').localeCompare(String(b?.Name || ''));
    });

    const parentIdToName = new Map<string, string>();
    const missingParentIds = new Set<string>();
    for (const row of sortedRows) {
      const id = normalizeOptionalString((row as any)?.Id);
      const name = normalizeOptionalString((row as any)?.Name);
      if (id && name) parentIdToName.set(id, name);
    }
    for (const row of sortedRows) {
      const parentId = readRefId((row as any)?.ParentId);
      if (parentId && !parentIdToName.has(parentId)) {
        missingParentIds.add(parentId);
      }
    }
    if (missingParentIds.size > 0) {
      const parentRows = await this.Search(['Id', 'in', Array.from(missingParentIds)] as any, { fields: ['Id', 'Name'] as any, limit: 5000 } as any);
      for (const row of parentRows || []) {
        const id = normalizeOptionalString((row as any)?.Id);
        const name = normalizeOptionalString((row as any)?.Name);
        if (id && name) parentIdToName.set(id, name);
      }
    }

    const routeRows = sortedRows.filter((row: any) => String((row as any)?.Type || '').trim() === 'ROUTE');
    const routeIds = routeRows.map((row: any) => normalizeOptionalString((row as any)?.Id)).filter((value): value is string => Boolean(value));
    const routeActionsByRouteId = new Map<string, string[]>();
    if (routeIds.length > 0) {
      const relationRows = await IrUiResourceRouteAction.Search(
        ['RouteUiResourceId', 'in', routeIds] as any,
        { fields: ['RouteUiResourceId', 'ActionUiResourceId'] as any, limit: 50000 } as any
      );
      const actionIds = Array.from(
        new Set((relationRows || []).map((row: any) => readRefId((row as any)?.ActionUiResourceId)).filter((value): value is string => Boolean(value)))
      );
      const actionNameById = new Map<string, string>();
      if (actionIds.length > 0) {
        const actionRows = await this.Search(['Id', 'in', actionIds] as any, { fields: ['Id', 'Name'] as any, limit: 50000 } as any);
        for (const row of actionRows || []) {
          const id = normalizeOptionalString((row as any)?.Id);
          const name = normalizeOptionalString((row as any)?.Name);
          if (id && name) actionNameById.set(id, name);
        }
      }
      for (const row of relationRows || []) {
        const routeId = readRefId((row as any)?.RouteUiResourceId);
        const actionId = readRefId((row as any)?.ActionUiResourceId);
        const actionName = actionId ? actionNameById.get(actionId) : undefined;
        if (!routeId || !actionName) continue;
        const existing = routeActionsByRouteId.get(routeId) ?? [];
        if (!existing.includes(actionName)) {
          existing.push(actionName);
          existing.sort();
          routeActionsByRouteId.set(routeId, existing);
        }
      }
    }

    const declarations = sortedRows.map((row: any) => {
      const id = normalizeOptionalString((row as any)?.Id);
      const name = normalizeOptionalString((row as any)?.Name) || '';
      const kind = this.normalizeKind((row as any)?.Type) || 'action';
      const declaration: EffectiveUiResourceDeclaration = {
        id: name,
        kind,
        title: normalizeOptionalString((row as any)?.Title),
        sequence: this.normalizeSequence((row as any)?.Sequence),
        path: normalizeOptionalString((row as any)?.UiPath),
        requires: this.normalizeRequires((row as any)?.Requires),
        defaultRoles: normalizeStringArray((row as any)?.DefaultRoles),
        override: false,
        module: normalizeOptionalString((row as any)?.Module),
        application: readRefId((row as any)?.IrApplicationId),
      };
      if (kind === 'menu') {
        const parentId = readRefId((row as any)?.ParentId);
        declaration.parentMenu = parentId ? parentIdToName.get(parentId) : undefined;
      }
      if (kind === 'route' && id) {
        declaration.actions = [...(routeActionsByRouteId.get(id) ?? [])];
      }
      return declaration;
    });

    return paginateAndWrap(declarations, 'declarations', pagination);
  }
}
