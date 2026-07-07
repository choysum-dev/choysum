// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { BaseModel, Model, Field } from '@/core/service';
import type { Insertable, Updateable } from '@/core/service/api/input';
import type { FieldSelection } from '@/core/service/api/selection';
import type { QueryCondition, SearchOptions } from '@/core/service/api/query';
import User from './user';
import UserRole from './user_role';
import RoleInheritance from './role_inheritance';
import RoleRecordRule from './role_record_rule';
import RoleMethodAccess from './role_method_access';
import RoleFieldRule from './role_field_rule';
import RoleUiResource from './role_ui_resource';
import { normalizeRefId } from './_rule_scope_helpers';

/**
 * Role defines one reusable permission bundle and its derived UI/resource mappings.
 */
@Model('Role')
export default class Role extends BaseModel {
  /**
   * Display name derived from the stable role code.
   */
  @Field({
    type: 'varchar',
    select: {
      expr: ({ field }) => field(Role, 'Code'),
      size: 36,
    },
  })
  public readonly DisplayName!: string;

  /**
   * Human-readable role name.
   */
  @Field({ type: 'varchar', column: { size: 100, unique: true, notNull: true } })
  Name: string;

  /**
   * Stable programmatic role code.
   */
  @Field({ type: 'varchar', column: { size: 50, unique: true, notNull: true } })
  Code: string;

  /**
   * Free-form description shown in management surfaces.
   */
  @Field({ type: 'varchar', column: { size: 255, index: true } })
  Description: string;

  /**
   * Whether the role can still be assigned and evaluated.
   */
  @Field({ type: 'boolean', column: { default: () => true, index: true } })
  IsActive: boolean;

  /**
   * Whether the role is part of the built-in system baseline.
   */
  @Field({ type: 'boolean', column: { default: () => false, index: true } })
  IsSystem: boolean;

  /**
   * UI-tree editor projection that only carries allow/resource-level UI resource Ids.
   */
  @Field({ type: 'ManyToManyRef', targetModel: 'meta.IrUiResource' })
  AccessUiResourceIds: string[];

  /**
   * Users assigned to the role through UserRole.
   */
  @Field({
    type: 'ManyToMany',
    relation: {
      joinModel: () => UserRole,
      targetModel: () => User,
      joinField: 'RoleId',
      inverseJoinField: 'UserId',
    },
  })
  Users: User[];

  /**
   * Roles implied by this role through inheritance.
   */
  @Field({
    type: 'ManyToMany',
    relation: {
      joinModel: () => RoleInheritance,
      targetModel: () => Role,
      joinField: 'ParentRoleId',
      inverseJoinField: 'ChildRoleId',
    },
  })
  ImpliedRoles: Role[];

  /**
   * Roles that imply the current role through inheritance.
   */
  @Field({
    type: 'ManyToMany',
    relation: {
      joinModel: () => RoleInheritance,
      targetModel: () => Role,
      joinField: 'ChildRoleId',
      inverseJoinField: 'ParentRoleId',
    },
  })
  ImpliedByRoles: Role[];

  /**
   * Record-rule entries attached to the role.
   */
  @Field({
    type: 'OneToMany',
    relation: { targetModel: () => RoleRecordRule, inverseField: 'RoleId' },
  })
  RecordRules: RoleRecordRule[];

  /**
   * RPC access entries attached to the role.
   */
  @Field({
    type: 'OneToMany',
    relation: { targetModel: () => RoleMethodAccess, inverseField: 'RoleId' },
  })
  MethodAccesses: RoleMethodAccess[];

  /**
   * Field-rule entries attached to the role.
   */
  @Field({
    type: 'OneToMany',
    relation: { targetModel: () => RoleFieldRule, inverseField: 'RoleId' },
  })
  FieldRules: RoleFieldRule[];

  /**
   * UI resource grant entries attached to the role.
   */
  @Field({
    type: 'OneToMany',
    relation: { targetModel: () => RoleUiResource, inverseField: 'RoleId' },
  })
  UiResources: RoleUiResource[];

  /**
   * Check whether a payload explicitly contains a field.
   */
  private static _hasOwn(obj: Record<string, any>, key: string): boolean {
    return Object.prototype.hasOwnProperty.call(obj, key);
  }

  /**
   * Normalize a scalar or relation payload into a unique Id list.
   */
  private static _normalizeIdList(v: any): string[] {
    if (v == null) return [];
    const arr = Array.isArray(v) ? v : [v];
    const set = new Set<string>();
    for (const it of arr) {
      const id = normalizeRefId(it);
      if (id) set.add(id);
    }
    return Array.from(set);
  }

  /**
   * Read the normalized role Id from a role-related row.
   */
  private static _readRoleId(row: any): string | null {
    return normalizeRefId(row?.RoleId);
  }

  /**
   * Check whether a UI grant row represents an allow/resource-level grant.
   */
  private static _isAllowResourceScope(row: any): boolean {
    const mode = String((row as any)?.Mode ?? 'allow')
      .trim()
      .toLowerCase();
    const uiResourceId = normalizeRefId((row as any)?.IrUiResourceId);
    const appId = normalizeRefId((row as any)?.IrApplicationId);
    return mode === 'allow' && !!uiResourceId && appId == null;
  }

  /**
   * Materialize allow/resource grant payloads from a list of UI resource Ids.
   */
  private static _makeAllowResourceEntries(ids: string[]): Array<Record<string, any>> {
    return ids.map(id => ({
      Mode: 'allow',
      IrApplicationId: null,
      IrUiResourceId: id,
    }));
  }

  /**
   * Extract the replace-array form used by relation write payloads.
   */
  private static _extractUiResourcesArray(v: any): any[] | null {
    if (Array.isArray(v)) return v;
    if (v && typeof v === 'object' && Array.isArray((v as any).replace)) return (v as any).replace;
    return null;
  }

  /**
   * Load persisted UI resource grant rows for one role.
   */
  private static async _loadUiResourcesForRole(roleId: string): Promise<Array<Record<string, any>>> {
    const rows = await RoleUiResource.Search(
      {
        And: [['RoleId', '=', roleId]],
      } as any,
      { fields: ['Id', 'Mode', 'IrApplicationId', 'IrUiResourceId'] as any } as any
    );

    return (rows || []).map((row: any) => ({
      ...(row || {}),
      Id: normalizeRefId(row?.Id) ?? undefined,
      Mode: String((row as any)?.Mode ?? 'allow')
        .trim()
        .toLowerCase(),
      IrApplicationId: normalizeRefId((row as any)?.IrApplicationId),
      IrUiResourceId: normalizeRefId((row as any)?.IrUiResourceId),
    }));
  }

  /**
   * Merge editor access Ids into the persisted UI resource grant shape.
   */
  private static _mergeAccessIntoUiResources(baseRows: Array<Record<string, any>>, accessIds: string[]): Array<Record<string, any>> {
    const preserved = (baseRows || []).filter(row => !this._isAllowResourceScope(row));
    const allowRows = this._makeAllowResourceEntries(accessIds);
    return [...preserved, ...allowRows];
  }

  /**
   * Rewrite AccessUiResourceIds into UiResources for create payloads.
   */
  private static async _applyAccessWriteTransformOnCreate(values: Record<string, any>): Promise<string[] | null> {
    if (!this._hasOwn(values, 'AccessUiResourceIds')) return null;

    const accessIds = this._normalizeIdList(values.AccessUiResourceIds);
    const incomingUiRows = this._extractUiResourcesArray(values.UiResources);
    const baseRows = Array.isArray(incomingUiRows) ? incomingUiRows : [];

    values.UiResources = this._mergeAccessIntoUiResources(baseRows, accessIds);
    delete values.AccessUiResourceIds;
    return accessIds;
  }

  /**
   * Rewrite AccessUiResourceIds into UiResources for update payloads.
   */
  private static async _applyAccessWriteTransformOnUpdate(values: Record<string, any>, roleId: string): Promise<string[] | null> {
    if (!this._hasOwn(values, 'AccessUiResourceIds')) return null;

    const accessIds = this._normalizeIdList(values.AccessUiResourceIds);
    const incomingUiRows = this._extractUiResourcesArray(values.UiResources);
    const baseRows = Array.isArray(incomingUiRows) ? incomingUiRows : await this._loadUiResourcesForRole(roleId);

    values.UiResources = this._mergeAccessIntoUiResources(baseRows, accessIds);
    delete values.AccessUiResourceIds;
    return accessIds;
  }

  /**
   * Persist allow/resource UI grants so they stay aligned with AccessUiResourceIds.
   */
  private static async _syncAllowResourceGrants(roleId: string, accessIds: string[]): Promise<void> {
    const targetIds = this._normalizeIdList(accessIds);
    const targetSet = new Set<string>(targetIds);

    const rows = await this._loadUiResourcesForRole(roleId);
    const allowRows = rows.filter(row => this._isAllowResourceScope(row));

    const existingByResource = new Map<string, string>();
    for (const row of allowRows) {
      const id = normalizeRefId((row as any).Id);
      const resourceId = normalizeRefId((row as any).IrUiResourceId);
      if (!id || !resourceId) continue;
      existingByResource.set(resourceId, id);
    }

    const deleteIds = Array.from(existingByResource.entries())
      .filter(([resourceId]) => !targetSet.has(resourceId))
      .map(([, rowId]) => rowId);

    if (deleteIds.length === 1) {
      await RoleUiResource.DeleteById(deleteIds[0]);
    } else if (deleteIds.length > 1) {
      await RoleUiResource.Delete(['Id', 'in', deleteIds] as any);
    }

    const createRows = targetIds
      .filter(resourceId => !existingByResource.has(resourceId))
      .map(resourceId => ({
        RoleId: { Id: roleId } as any,
        Mode: 'allow',
        IrApplicationId: null,
        IrUiResourceId: resourceId,
      }));

    if (createRows.length) {
      await RoleUiResource.CreateMany(createRows as any);
    }
  }

  /**
   * Build a roleId -> access resource Id map from persisted UI grants.
   */
  private static async _buildAccessMap(roleIds: string[]): Promise<Map<string, string[]>> {
    const out = new Map<string, string[]>();
    if (!roleIds.length) return out;

    const rows = await RoleUiResource.Search(
      {
        And: [['RoleId', 'in', roleIds]],
      } as any,
      { fields: ['RoleId', 'Mode', 'IrApplicationId', 'IrUiResourceId'] as any, limit: Math.max(1000, roleIds.length * 200) } as any
    );

    const map = new Map<string, Set<string>>();
    for (const row of rows || []) {
      const roleId = this._readRoleId(row as any);
      if (!roleId) continue;
      if (!this._isAllowResourceScope(row)) continue;
      const resourceId = normalizeRefId((row as any)?.IrUiResourceId);
      if (!resourceId) continue;
      if (!map.has(roleId)) map.set(roleId, new Set<string>());
      map.get(roleId)!.add(resourceId);
    }

    for (const roleId of roleIds) {
      out.set(roleId, Array.from(map.get(roleId) || []));
    }
    return out;
  }

  /**
   * Hydrate AccessUiResourceIds onto result rows when callers request the field.
   */
  private static async _hydrateAccessUiResourceIds(records: any[]): Promise<void> {
    if (!Array.isArray(records) || records.length === 0) return;
    const roleIds = Array.from(
      new Set(
        records
          .map(row => normalizeRefId((row as any)?.Id))
          .filter(Boolean)
          .map(String)
      )
    );
    if (!roleIds.length) return;

    const accessMap = await this._buildAccessMap(roleIds);
    for (const row of records) {
      const roleId = normalizeRefId((row as any)?.Id);
      if (!roleId) continue;
      (row as any).AccessUiResourceIds = accessMap.get(roleId) || [];
    }
  }

  /**
   * Check whether a field selection asks for AccessUiResourceIds.
   */
  private static _wantsAccessField(selection: any): boolean {
    if (selection == null) return false;
    if (typeof selection === 'string') return selection === 'AccessUiResourceIds';
    if (Array.isArray(selection)) return selection.some(it => this._wantsAccessField(it));
    if (typeof selection === 'object' && Array.isArray((selection as any).fields)) {
      return this._wantsAccessField((selection as any).fields);
    }
    return false;
  }

  /**
   * Browse one role and hydrate AccessUiResourceIds when requested.
   */
  static override async Browse<T extends BaseModel>(
    this: { new (...args: any[]): T } & typeof BaseModel,
    id: string,
    fields?: FieldSelection<T>,
    options?: any
  ): Promise<T> {
    const row = (await super.Browse(id, fields as any, options as any)) as any;
    if (Role._wantsAccessField(fields)) {
      await Role._hydrateAccessUiResourceIds([row]);
    }
    return row as T;
  }

  /**
   * Browse multiple roles and hydrate AccessUiResourceIds when requested.
   */
  static override async BrowseMany<T extends BaseModel>(
    this: { new (...args: any[]): T } & typeof BaseModel,
    ids: string[],
    fields?: (keyof any)[],
    options?: any
  ): Promise<T[]> {
    const rows = (await super.BrowseMany(ids as any, fields as any, options as any)) as any[];
    if (Role._wantsAccessField(fields)) {
      await Role._hydrateAccessUiResourceIds(rows);
    }
    return rows as T[];
  }

  /**
   * Search roles and hydrate AccessUiResourceIds when requested.
   */
  static override async Search<T extends BaseModel>(
    this: { new (...args: any[]): T } & typeof BaseModel,
    condition: QueryCondition<T> | [] = [],
    options?: SearchOptions<T>
  ): Promise<T[]> {
    const rows = (await super.Search(condition as any, options as any)) as any[];
    if (Role._wantsAccessField((options as any)?.fields)) {
      await Role._hydrateAccessUiResourceIds(rows);
    }
    return rows as T[];
  }

  /**
   * Create one role while keeping the UI access projection synchronized.
   */
  static override async Create<T extends BaseModel>(
    this: { new (...args: any[]): T } & typeof BaseModel,
    value: Partial<Insertable<T & BaseModel>>,
    returnFields?: FieldSelection<T>
  ): Promise<T> {
    const payload = { ...(value as any) } as Record<string, any>;
    const accessIds = await Role._applyAccessWriteTransformOnCreate(payload);
    const row = (await super.Create(payload as any, returnFields as any)) as any;
    const roleId = normalizeRefId((row as any)?.Id);
    if (roleId && accessIds) {
      await Role._syncAllowResourceGrants(roleId, accessIds);
      (row as any).AccessUiResourceIds = [...accessIds];
    } else if (Role._wantsAccessField(returnFields)) {
      await Role._hydrateAccessUiResourceIds([row]);
    }
    return row as T;
  }

  /**
   * Create multiple roles while keeping the UI access projection synchronized.
   */
  static override async CreateMany<T extends BaseModel>(
    this: { new (...args: any[]): T } & typeof BaseModel,
    values: Partial<Insertable<T & BaseModel>>[],
    returnFields?: FieldSelection<T>
  ): Promise<T[]> {
    const payloads = [...(values || [])].map(v => ({ ...(v as any) }));
    const accessList: Array<string[] | null> = [];
    for (const payload of payloads) {
      accessList.push(await Role._applyAccessWriteTransformOnCreate(payload));
    }
    const rows = (await super.CreateMany(payloads as any, returnFields as any)) as any[];
    for (let i = 0; i < rows.length; i++) {
      const roleId = normalizeRefId((rows[i] as any)?.Id);
      const accessIds = accessList[i];
      if (roleId && accessIds) {
        await Role._syncAllowResourceGrants(roleId, accessIds);
        (rows[i] as any).AccessUiResourceIds = [...accessIds];
      }
    }
    if (Role._wantsAccessField(returnFields) && accessList.every(ids => ids == null)) {
      await Role._hydrateAccessUiResourceIds(rows);
    }
    return rows as T[];
  }

  /**
   * Update matching roles while keeping the UI access projection synchronized.
   */
  static override async Update<T extends BaseModel>(
    this: { new (...args: any[]): T } & typeof BaseModel,
    condition: QueryCondition<T>,
    values: Partial<Updateable<T & BaseModel>>,
    returnFields?: FieldSelection<T>,
    options?: any
  ): Promise<Partial<T>[]> {
    const payload = { ...(values as any) } as Record<string, any>;
    const shouldHydrateAccess = Role._wantsAccessField(returnFields);
    let roleIdForSync: string | null = null;
    let accessIdsForSync: string[] | null = null;
    if (Role._hasOwn(payload, 'AccessUiResourceIds')) {
      const targetRows = (await super.Search(condition as any, { fields: ['Id'] as any } as any)) as any[];
      const roleIds = targetRows.map(row => normalizeRefId((row as any)?.Id)).filter(Boolean) as string[];
      if (roleIds.length > 1) {
        throw new Error('Role.Update with AccessUiResourceIds only supports single record update');
      }
      if (roleIds.length === 1) {
        roleIdForSync = roleIds[0];
        accessIdsForSync = await Role._applyAccessWriteTransformOnUpdate(payload, roleIds[0]);
      }
    }

    const rows = (await super.Update(condition as any, payload as any, returnFields as any, options as any)) as any[];
    if (roleIdForSync && accessIdsForSync) {
      await Role._syncAllowResourceGrants(roleIdForSync, accessIdsForSync);
      if (rows.length && returnFields != null) {
        rows[0] = (await this.Browse(roleIdForSync as any, returnFields as any, options as any)) as any;
      }
      if (rows.length && shouldHydrateAccess) {
        (rows[0] as any).AccessUiResourceIds = [...accessIdsForSync];
      }
    } else if (shouldHydrateAccess) {
      await Role._hydrateAccessUiResourceIds(rows);
    }
    return rows as Partial<T>[];
  }

  /**
   * Update one role by Id while keeping the UI access projection synchronized.
   */
  static override async UpdateById<T extends BaseModel>(
    this: { new (...args: any[]): T } & typeof BaseModel,
    id: string,
    values: Partial<Updateable<T & BaseModel>>,
    returnFields?: FieldSelection<T>,
    options?: any
  ): Promise<Partial<T>> {
    const payload = { ...(values as any) } as Record<string, any>;
    const accessIds = await Role._applyAccessWriteTransformOnUpdate(payload, id);
    let row = (await super.UpdateById(id as any, payload as any, returnFields as any, options as any)) as any;
    if (accessIds) {
      await Role._syncAllowResourceGrants(id, accessIds);
      if (returnFields != null) {
        row = (await this.Browse(id as any, returnFields as any, options as any)) as any;
      }
      if (Role._wantsAccessField(returnFields)) {
        (row as any).AccessUiResourceIds = [...accessIds];
      }
    } else if (Role._wantsAccessField(returnFields)) {
      await Role._hydrateAccessUiResourceIds([row]);
    }
    return row as Partial<T>;
  }
}
