// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { BaseModel, Model, Field, SqlCompute, type ModelCtor } from '@/core/service';
import type { Insertable, Updateable } from '@/core/service/api/input';
import type { FieldSelection } from '@/core/service/api/selection';
import type { QueryCondition, SearchOptions, SoftDeleteOptions, UpdateOptions } from '@/core/service/api/query';
import { _lt } from '../i18n';
import AuthzMutationModel from '../mixins/authz_mutation_model';
import User from './user/user';
import UserRole from './user_role';
import RoleInheritance from './role_inheritance';
import RoleRecordRule from './role_record_rule';
import RoleMethodAccess from './role_method_access';
import RoleFieldRule from './role_field_rule';
import RoleUiResource from './role_ui_resource';
import { normalizeRefId } from '@/core/service/utils/normalization';
import type MetaUiResource from '@/meta/service/models/ui_resource';
import {
  applyAccessWriteTransformOnCreate,
  applyAccessWriteTransformOnUpdate,
  syncAllowResourceGrants,
  hydrateAccessUiResourceIds,
  wantsAccessField,
} from './_role_ui_projection';

/**
 * Role defines one reusable permission bundle and its derived UI/resource mappings.
 *
 * Extends {@link AuthzMutationModel} so Create/Update/Delete* clear request-scoped
 * authz caches (IsActive / delete change the permission graph even when UI grants
 * are untouched). Browse/Search still hydrate AccessUiResourceIds here.
 */
@Model('Role')
export default class Role extends AuthzMutationModel {
  /**
   * Display name derived from the stable role code.
   */
  @Field({
    type: 'varchar',
    size: 36,
    string: _lt('Display Name', { scope: 'auth.model.Role.fields' }),
  })
  public readonly DisplayName!: string;

  @SqlCompute<Role>('DisplayName')
  sqlDisplayName() {
    return this.$sql.field('Code');
  }

  /**
   * Human-readable role name.
   */
  @Field({
    type: 'varchar',
    size: 100,
    notNull: true,
    translate: true,
    index: 'trigram',
    string: _lt('Name', { scope: 'auth.model.Role.fields' }),
  })
  Name: string;

  /**
   * Stable programmatic role code.
   */
  @Field({
    type: 'varchar',
    size: 50,
    unique: true,
    notNull: true,
    string: _lt('Code', { scope: 'auth.model.Role.fields' }),
    help: _lt('Stable programmatic id; also used as the display name.', {
      scope: 'auth.model.Role.fields',
    }),
  })
  Code: string;

  /**
   * Free-form description shown in management surfaces.
   */
  @Field({
    type: 'varchar',
    size: 255,
    translate: true,
    index: 'trigram',
    string: _lt('Description', { scope: 'auth.model.Role.fields' }),
  })
  Description: string;

  /**
   * Whether the role can still be assigned and evaluated.
   */
  @Field({
    type: 'boolean',
    default: () => true,
    index: true,
    string: _lt('Active', { scope: 'auth.model.Role.fields' }),
  })
  IsActive: boolean;

  /**
   * Whether the role is part of the built-in system baseline.
   */
  @Field({
    type: 'boolean',
    default: () => false,
    index: true,
    string: _lt('Built-in', { scope: 'auth.model.Role.fields' }),
    help: _lt('System roles are seeded and should not be deleted casually.', {
      scope: 'auth.model.Role.fields',
    }),
  })
  IsSystem: boolean;

  /**
   * UI-tree editor projection that only carries allow/resource-level UI resource Ids.
   */
  @Field<MetaUiResource>({
    type: 'ManyToManyRef',
    relation: { targetModel: 'meta.MetaUiResource' },
    string: _lt('Accessible UI Resources', { scope: 'auth.model.Role.fields' }),
    help: _lt('UI tree selection; drives menu and action visibility for this role.', {
      scope: 'auth.model.Role.fields',
    }),
  })
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
    condition: ['IsActive', '=', true],
    string: _lt('Users', { scope: 'auth.model.Role.fields' }),
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
    condition: ['IsActive', '=', true],
    string: _lt('Included Roles', { scope: 'auth.model.Role.fields' }),
    help: _lt('Permissions from included roles are merged into this role.', {
      scope: 'auth.model.Role.fields',
    }),
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
    condition: ['IsActive', '=', true],
    string: _lt('Implied By Roles', { scope: 'auth.model.Role.fields' }),
  })
  ImpliedByRoles: Role[];

  /**
   * Record-rule entries attached to the role.
   */
  @Field({
    type: 'OneToMany',
    relation: { targetModel: () => RoleRecordRule, inverseField: 'RoleId' },
    string: _lt('Record Rules', { scope: 'auth.model.Role.fields' }),
  })
  RecordRules: RoleRecordRule[];

  /**
   * RPC access entries attached to the role.
   */
  @Field({
    type: 'OneToMany',
    relation: { targetModel: () => RoleMethodAccess, inverseField: 'RoleId' },
    string: _lt('Method Accesses', { scope: 'auth.model.Role.fields' }),
  })
  MethodAccesses: RoleMethodAccess[];

  /**
   * Field-rule entries attached to the role.
   */
  @Field({
    type: 'OneToMany',
    relation: { targetModel: () => RoleFieldRule, inverseField: 'RoleId' },
    string: _lt('Field Rules', { scope: 'auth.model.Role.fields' }),
  })
  FieldRules: RoleFieldRule[];

  /**
   * UI resource grant entries attached to the role.
   */
  @Field({
    type: 'OneToMany',
    relation: { targetModel: () => RoleUiResource, inverseField: 'RoleId' },
    string: _lt('UI Resources', { scope: 'auth.model.Role.fields' }),
  })
  UiResources: RoleUiResource[];

  /**
   * Browse one role and hydrate AccessUiResourceIds when requested.
   */
  static override async Browse<T extends BaseModel>(
    this: ModelCtor<T>,
    id: string,
    fields?: FieldSelection<T>,
    options?: SoftDeleteOptions
  ): Promise<T> {
    const row = await super.Browse<T>(id, fields, options);
    if (wantsAccessField(fields)) {
      await hydrateAccessUiResourceIds([row]);
    }
    return row;
  }

  /**
   * Browse multiple roles and hydrate AccessUiResourceIds when requested.
   */
  static override async BrowseMany<T extends BaseModel>(
    this: ModelCtor<T>,
    ids: string[],
    fields?: FieldSelection<T>,
    options?: SoftDeleteOptions
  ): Promise<T[]> {
    const rows = await super.BrowseMany<T>(ids, fields, options);
    if (wantsAccessField(fields)) {
      await hydrateAccessUiResourceIds(rows);
    }
    return rows;
  }

  /**
   * Search roles and hydrate AccessUiResourceIds when requested.
   */
  static override async Search<T extends BaseModel>(
    this: ModelCtor<T>,
    condition: QueryCondition<T> | [] = [],
    options?: SearchOptions<T>
  ): Promise<T[]> {
    const rows = await super.Search<T>(condition, options);
    if (wantsAccessField(options?.fields)) {
      await hydrateAccessUiResourceIds(rows);
    }
    return rows;
  }

  /**
   * Create one role while keeping the UI access projection synchronized.
   */
  static override async Create<T extends BaseModel>(
    this: ModelCtor<T>,
    value: Partial<Insertable<T>>,
    returnFields?: FieldSelection<T>
  ): Promise<T> {
    const payload = { ...(value as Record<string, unknown>) };
    const accessIds = await applyAccessWriteTransformOnCreate(payload);
    const row = await super.Create<T>(payload as Partial<Insertable<T>>, returnFields);
    const roleId = normalizeRefId((row as { Id?: unknown }).Id);
    if (roleId && accessIds) {
      await syncAllowResourceGrants(roleId, accessIds);
      (row as { AccessUiResourceIds?: string[] }).AccessUiResourceIds = [...accessIds];
    } else if (wantsAccessField(returnFields)) {
      await hydrateAccessUiResourceIds([row]);
    }
    return row;
  }

  /**
   * Create multiple roles while keeping the UI access projection synchronized.
   */
  static override async CreateMany<T extends BaseModel>(
    this: ModelCtor<T>,
    values: Partial<Insertable<T>>[],
    returnFields?: FieldSelection<T>
  ): Promise<T[]> {
    const payloads = [...(values || [])].map(v => ({ ...(v as Record<string, unknown>) }));
    const accessList: Array<string[] | null> = [];
    for (const payload of payloads) {
      accessList.push(await applyAccessWriteTransformOnCreate(payload));
    }
    const rows = await super.CreateMany<T>(payloads as Array<Partial<Insertable<T>>>, returnFields);
    for (let i = 0; i < rows.length; i++) {
      const roleId = normalizeRefId((rows[i] as { Id?: unknown }).Id);
      const accessIds = accessList[i];
      if (roleId && accessIds) {
        await syncAllowResourceGrants(roleId, accessIds);
        (rows[i] as { AccessUiResourceIds?: string[] }).AccessUiResourceIds = [...accessIds];
      }
    }
    if (wantsAccessField(returnFields)) {
      const rowsToHydrate = rows.filter((_, i) => accessList[i] == null);
      if (rowsToHydrate.length > 0) {
        await hydrateAccessUiResourceIds(rowsToHydrate);
      }
    }
    return rows;
  }

  /**
   * Update matching roles while keeping the UI access projection synchronized.
   */
  static override async Update<T extends BaseModel>(
    this: ModelCtor<T>,
    condition: QueryCondition<T>,
    values: Partial<Updateable<T>>,
    returnFields?: FieldSelection<T>,
    options?: UpdateOptions
  ): Promise<Partial<T>[]> {
    const payload: Record<string, unknown> = { ...(values as Record<string, unknown>) };
    const shouldHydrateAccess = wantsAccessField(returnFields);
    let roleIdForSync: string | null = null;
    let accessIdsForSync: string[] | null = null;
    if (Object.prototype.hasOwnProperty.call(payload, 'AccessUiResourceIds')) {
      const targetRows = await (super.Search as (
        condition: QueryCondition<T> | [],
        options?: SearchOptions<T>
      ) => Promise<T[]>)(condition, { fields: ['Id'] as unknown as FieldSelection<T> });
      const roleIds = targetRows.map(row => normalizeRefId((row as { Id?: unknown }).Id)).filter(Boolean) as string[];
      if (roleIds.length > 1) {
        throw new Error('Role.Update with AccessUiResourceIds only supports single record update');
      }
      if (roleIds.length === 1) {
        roleIdForSync = roleIds[0];
        accessIdsForSync = await applyAccessWriteTransformOnUpdate(payload, roleIds[0]);
      }
    }

    const rows = await super.Update<T>(condition, payload as Partial<Updateable<T>>, returnFields, options);
    if (roleIdForSync && accessIdsForSync) {
      await syncAllowResourceGrants(roleIdForSync, accessIdsForSync);
      if (rows.length && returnFields != null) {
        rows[0] = await this.Browse<T>(roleIdForSync, returnFields, options);
      }
      if (rows.length && shouldHydrateAccess) {
        (rows[0] as { AccessUiResourceIds?: string[] }).AccessUiResourceIds = [...accessIdsForSync];
      }
    } else if (shouldHydrateAccess) {
      await hydrateAccessUiResourceIds(rows);
    }
    return rows;
  }

  /**
   * Update one role by Id while keeping the UI access projection synchronized.
   */
  static override async UpdateById<T extends BaseModel>(
    this: ModelCtor<T>,
    id: string,
    values: Partial<Updateable<T>>,
    returnFields?: FieldSelection<T>,
    options?: UpdateOptions
  ): Promise<Partial<T>> {
    const payload: Record<string, unknown> = { ...(values as Record<string, unknown>) };
    const accessIds = await applyAccessWriteTransformOnUpdate(payload, id);
    let row = await super.UpdateById<T>(id, payload as Partial<Updateable<T>>, returnFields, options);
    if (accessIds) {
      await syncAllowResourceGrants(id, accessIds);
      if (returnFields != null) {
        row = await this.Browse<T>(id, returnFields, options);
      }
      if (wantsAccessField(returnFields)) {
        (row as { AccessUiResourceIds?: string[] }).AccessUiResourceIds = [...accessIds];
      }
    } else if (wantsAccessField(returnFields)) {
      await hydrateAccessUiResourceIds([row]);
    }
    return row;
  }
}
