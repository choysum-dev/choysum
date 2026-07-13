// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { BaseModel, Model, Field, Compute } from '@/core/service';
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
import { normalizeRefId } from '@/core/service/utils/normalization';
import {
  applyAccessWriteTransformOnCreate,
  applyAccessWriteTransformOnUpdate,
  syncAllowResourceGrants,
  hydrateAccessUiResourceIds,
  wantsAccessField,
} from './_role_ui_projection';

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
    size: 36,
  })
  public readonly DisplayName!: string;

  @Compute<Role>('DisplayName', {
    deps: ['Code'],
    store: false,
  })
  computeDisplayName() {
    return this.Code;
  }

  /**
   * Human-readable role name.
   */
  @Field({ type: 'varchar', size: 100, unique: true, notNull: true })
  Name: string;

  /**
   * Stable programmatic role code.
   */
  @Field({ type: 'varchar', size: 50, unique: true, notNull: true })
  Code: string;

  /**
   * Free-form description shown in management surfaces.
   */
  @Field({ type: 'varchar', size: 255, index: true })
  Description: string;

  /**
   * Whether the role can still be assigned and evaluated.
   */
  @Field({ type: 'boolean', default: () => true, index: true })
  IsActive: boolean;

  /**
   * Whether the role is part of the built-in system baseline.
   */
  @Field({ type: 'boolean', default: () => false, index: true })
  IsSystem: boolean;

  /**
   * UI-tree editor projection that only carries allow/resource-level UI resource Ids.
   */
  @Field({ type: 'ManyToManyRef', relation: { targetModel: 'meta.IrUiResource' } })
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
   * Browse one role and hydrate AccessUiResourceIds when requested.
   */
  static override async Browse<T extends BaseModel>(
    this: { new (...args: any[]): T } & typeof BaseModel,
    id: string,
    fields?: FieldSelection<T>,
    options?: any
  ): Promise<T> {
    const row = (await super.Browse(id, fields as any, options as any)) as any;
    if (wantsAccessField(fields)) {
      await hydrateAccessUiResourceIds([row]);
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
    if (wantsAccessField(fields)) {
      await hydrateAccessUiResourceIds(rows);
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
    if (wantsAccessField((options as any)?.fields)) {
      await hydrateAccessUiResourceIds(rows);
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
    const accessIds = await applyAccessWriteTransformOnCreate(payload);
    const row = (await super.Create(payload as any, returnFields as any)) as any;
    const roleId = normalizeRefId((row as any)?.Id);
    if (roleId && accessIds) {
      await syncAllowResourceGrants(roleId, accessIds);
      (row as any).AccessUiResourceIds = [...accessIds];
    } else if (wantsAccessField(returnFields)) {
      await hydrateAccessUiResourceIds([row]);
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
      accessList.push(await applyAccessWriteTransformOnCreate(payload));
    }
    const rows = (await super.CreateMany(payloads as any, returnFields as any)) as any[];
    for (let i = 0; i < rows.length; i++) {
      const roleId = normalizeRefId((rows[i] as any)?.Id);
      const accessIds = accessList[i];
      if (roleId && accessIds) {
        await syncAllowResourceGrants(roleId, accessIds);
        (rows[i] as any).AccessUiResourceIds = [...accessIds];
      }
    }
    if (wantsAccessField(returnFields)) {
      const rowsToHydrate = rows.filter((_, i) => accessList[i] == null);
      if (rowsToHydrate.length > 0) {
        await hydrateAccessUiResourceIds(rowsToHydrate);
      }
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
    const shouldHydrateAccess = wantsAccessField(returnFields);
    let roleIdForSync: string | null = null;
    let accessIdsForSync: string[] | null = null;
    if (Object.prototype.hasOwnProperty.call(payload, 'AccessUiResourceIds')) {
      const targetRows = (await super.Search(condition as any, { fields: ['Id'] as any } as any)) as any[];
      const roleIds = targetRows.map(row => normalizeRefId((row as any)?.Id)).filter(Boolean) as string[];
      if (roleIds.length > 1) {
        throw new Error('Role.Update with AccessUiResourceIds only supports single record update');
      }
      if (roleIds.length === 1) {
        roleIdForSync = roleIds[0];
        accessIdsForSync = await applyAccessWriteTransformOnUpdate(payload, roleIds[0]);
      }
    }

    const rows = (await super.Update(condition as any, payload as any, returnFields as any, options as any)) as any[];
    if (roleIdForSync && accessIdsForSync) {
      await syncAllowResourceGrants(roleIdForSync, accessIdsForSync);
      if (rows.length && returnFields != null) {
        rows[0] = (await this.Browse(roleIdForSync as any, returnFields as any, options as any)) as any;
      }
      if (rows.length && shouldHydrateAccess) {
        (rows[0] as any).AccessUiResourceIds = [...accessIdsForSync];
      }
    } else if (shouldHydrateAccess) {
      await hydrateAccessUiResourceIds(rows);
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
    const accessIds = await applyAccessWriteTransformOnUpdate(payload, id);
    let row = (await super.UpdateById(id as any, payload as any, returnFields as any, options as any)) as any;
    if (accessIds) {
      await syncAllowResourceGrants(id, accessIds);
      if (returnFields != null) {
        row = (await this.Browse(id as any, returnFields as any, options as any)) as any;
      }
      if (wantsAccessField(returnFields)) {
        (row as any).AccessUiResourceIds = [...accessIds];
      }
    } else if (wantsAccessField(returnFields)) {
      await hydrateAccessUiResourceIds([row]);
    }
    return row as Partial<T>;
  }
}
