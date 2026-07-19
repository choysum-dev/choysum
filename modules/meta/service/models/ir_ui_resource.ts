// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { BaseModel, Compute, Field, Model, SqlCompute } from '@/core/service';
import { sql } from 'kysely';
import IrApplication from './ir_application';
import IrModule from './ir_module';
import IrUiResourceRouteAction from './ir_ui_resource_route_action';
import { normalizeOptionalString, normalizeStringArray, readRefId } from '@/core/service/utils/normalization';
import { normalizePagination, paginateAndWrap } from '@/core/service/utils/pagination';
import { createTranslate, type TermReference } from '@/core/service/i18n';

const { _t } = createTranslate('meta');

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

type IrUiResourceComputeDeps = IrUiResource & Record<'ParentId.ParentPath', unknown>;

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

  @Field({ type: 'jsonobject' })
  TitleText?: TermReference | null;

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
      throw new Error(
        _t('Cycle detected: %s cannot be assigned under one of its descendants', { scope: 'service/models/ir_ui_resource' }, id)
      );
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

  @SqlCompute<IrUiResource>('Childs')
  sqlChilds() {
    const selfTypeRef = this.$sql.col('meta_ir_ui_resource', 'Type');
    const selfIdRef = this.$sql.col('meta_ir_ui_resource', 'Id');

    const dialect = String((globalThis as any)?.$choysum?.db?.dialectName || 'postgres').toLowerCase();

    if (dialect === 'sqlite') {
      return sql<any>`
        (
          select coalesce(
            json_group_array(json(child_row.payload)),
            json('[]')
          )
          from (
            select
              coalesce(c.sequence, 0) as seq,
              c.name as name,
              json_object(
                'Id', c.id,
                'Name', c.name,
                'Type', c.type,
                'Title', c.title,
                'TitleText', json(c.title_text),
                'Sequence', c.sequence,
                'Requires', json(c.requires),
                'Module', c.module,
                'Path', c.path,
                'ParentPath', c.parent_path,
                'UiPath', c.ui_path,
                'DefaultRoles', json(c.default_roles)
              ) as payload
            from meta_ir_ui_resource c
            where ${selfTypeRef} = 'MENU'
              and c.parent_id = ${selfIdRef}

            union all

            select
              coalesce(r.sequence, 0) as seq,
              r.name as name,
              json_object(
                'Id', r.id,
                'Name', r.name,
                'Type', r.type,
                'Title', r.title,
                'TitleText', json(r.title_text),
                'Sequence', r.sequence,
                'Requires', json(r.requires),
                'Module', r.module,
                'Path', r.path,
                'ParentPath', r.parent_path,
                'UiPath', r.ui_path,
                'DefaultRoles', json(r.default_roles)
              ) as payload
            from meta_ir_ui_resource_menu_route mr
            join meta_ir_ui_resource r on r.id = mr.route_ui_resource_id
            where ${selfTypeRef} = 'MENU'
              and mr.menu_ui_resource_id = ${selfIdRef}

            union all

            select
              coalesce(a.sequence, 0) as seq,
              a.name as name,
              json_object(
                'Id', a.id,
                'Name', a.name,
                'Type', a.type,
                'Title', a.title,
                'TitleText', json(a.title_text),
                'Sequence', a.sequence,
                'Requires', json(a.requires),
                'Module', a.module,
                'Path', a.path,
                'ParentPath', a.parent_path,
                'UiPath', a.ui_path,
                'DefaultRoles', json(a.default_roles)
              ) as payload
            from meta_ir_ui_resource_route_action ra
            join meta_ir_ui_resource a on a.id = ra.action_ui_resource_id
            where ${selfTypeRef} = 'ROUTE'
              and ra.route_ui_resource_id = ${selfIdRef}

            order by seq asc, name asc
          ) as child_row
        )
      `;
    }

    return sql<any>`
      (
        select coalesce(
          json_agg(child_row.payload order by child_row.seq asc, child_row.name asc),
          '[]'::json
        )
        from (
          select
            coalesce(c.sequence, 0) as seq,
            c.name as name,
            json_build_object(
              'Id', c.id,
              'Name', c.name,
              'Type', c.type,
              'Title', c.title,
              'TitleText', c.title_text,
              'Sequence', c.sequence,
              'Requires', c.requires,
              'Module', c.module,
              'Path', c.path,
              'ParentPath', c.parent_path,
              'UiPath', c.ui_path,
              'DefaultRoles', c.default_roles
            ) as payload
          from meta_ir_ui_resource c
          where ${selfTypeRef} = 'MENU'
            and c.parent_id = ${selfIdRef}

          union all

          select
            coalesce(r.sequence, 0) as seq,
            r.name as name,
            json_build_object(
              'Id', r.id,
              'Name', r.name,
              'Type', r.type,
              'Title', r.title,
              'TitleText', r.title_text,
              'Sequence', r.sequence,
              'Requires', r.requires,
              'Module', r.module,
              'Path', r.path,
              'ParentPath', r.parent_path,
              'UiPath', r.ui_path,
              'DefaultRoles', r.default_roles
            ) as payload
          from meta_ir_ui_resource_menu_route mr
          join meta_ir_ui_resource r on r.id = mr.route_ui_resource_id
          where ${selfTypeRef} = 'MENU'
            and mr.menu_ui_resource_id = ${selfIdRef}

          union all

          select
            coalesce(a.sequence, 0) as seq,
            a.name as name,
            json_build_object(
              'Id', a.id,
              'Name', a.name,
              'Type', a.type,
              'Title', a.title,
              'TitleText', a.title_text,
              'Sequence', a.sequence,
              'Requires', a.requires,
              'Module', a.module,
              'Path', a.path,
              'ParentPath', a.parent_path,
              'UiPath', a.ui_path,
              'DefaultRoles', a.default_roles
            ) as payload
          from meta_ir_ui_resource_route_action ra
          join meta_ir_ui_resource a on a.id = ra.action_ui_resource_id
          where ${selfTypeRef} = 'ROUTE'
            and ra.route_ui_resource_id = ${selfIdRef}
        ) as child_row
      )
    `;
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
