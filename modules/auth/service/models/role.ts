// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { Model, Field, SqlCompute, type ModelCtor, type RowOf } from '@/core/service';
import type { Insertable, Updateable } from '@/core/service/api/input';
import type { FieldSelection, Projected, RowOrProjected } from '@/core/service/api/selection';
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
  static override async Browse<C extends ModelCtor, F extends FieldSelection<RowOf<C>> | undefined = undefined>(
    this: C,
    id: string,
    fields?: F,
    options?: SoftDeleteOptions
  ): Promise<RowOrProjected<RowOf<C>, F>> {
    const row = await super.Browse<C, F>(id, fields, options);
    if (wantsAccessField(fields)) {
      await hydrateAccessUiResourceIds([row]);
    }
    return row;
  }

  /**
   * Browse multiple roles and hydrate AccessUiResourceIds when requested.
   */
  static override async BrowseMany<C extends ModelCtor, F extends FieldSelection<RowOf<C>> | undefined = undefined>(
    this: C,
    ids: string[],
    fields?: F,
    options?: SoftDeleteOptions
  ): Promise<Array<RowOrProjected<RowOf<C>, F>>> {
    const rows = await super.BrowseMany<C, F>(ids, fields, options);
    if (wantsAccessField(fields)) {
      await hydrateAccessUiResourceIds(rows);
    }
    return rows;
  }

  /**
   * Search roles and hydrate AccessUiResourceIds when requested.
   */
  static override async Search<C extends ModelCtor, F extends FieldSelection<RowOf<C>> | undefined = undefined>(
    this: C,
    condition: QueryCondition<RowOf<C>> | [] = [],
    options?: Omit<SearchOptions<RowOf<C>>, 'fields'> & { fields?: F }
  ): Promise<Array<RowOrProjected<RowOf<C>, F>>> {
    const rows = await super.Search<C, F>(condition, options);
    if (wantsAccessField(options?.fields)) {
      await hydrateAccessUiResourceIds(rows);
    }
    return rows;
  }

  /**
   * Create one role while keeping the UI access projection synchronized.
   */
  static override async Create<C extends ModelCtor, F extends FieldSelection<RowOf<C>> | undefined = undefined>(
    this: C,
    value: Partial<Insertable<RowOf<C>>>,
    returnFields?: F
  ): Promise<RowOrProjected<RowOf<C>, F>> {
    const payload = { ...value };
    const accessIds = await applyAccessWriteTransformOnCreate(payload as Record<string, unknown>);
    const row = await super.Create<C, F>(payload, returnFields);
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
  static override async CreateMany<C extends ModelCtor, F extends FieldSelection<RowOf<C>> | undefined = undefined>(
    this: C,
    values: Partial<Insertable<RowOf<C>>>[],
    returnFields?: F
  ): Promise<Array<RowOrProjected<RowOf<C>, F>>> {
    const payloads = [...(values || [])].map(v => ({ ...v }));
    const accessList: Array<string[] | null> = [];
    for (const payload of payloads) {
      accessList.push(await applyAccessWriteTransformOnCreate(payload as Record<string, unknown>));
    }
    const rows = await super.CreateMany<C, F>(payloads, returnFields);
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
  static override async Update<C extends ModelCtor, F extends FieldSelection<RowOf<C>> | undefined = undefined>(
    this: C,
    condition: QueryCondition<RowOf<C>>,
    values: Partial<Updateable<RowOf<C>>>,
    returnFields?: F,
    options?: UpdateOptions
  ): Promise<Array<F extends FieldSelection<RowOf<C>> ? Projected<RowOf<C>, F> : Partial<RowOf<C>>>> {
    const payload = { ...values };
    const shouldHydrateAccess = wantsAccessField(returnFields);
    let roleIdForSync: string | null = null;
    let accessIdsForSync: string[] | null = null;
    if (Object.prototype.hasOwnProperty.call(payload, 'AccessUiResourceIds')) {
      // Use Role.Search; fields: ['Id'] needs `as never` for the open RowOf<C> slot.
      const targetRows = await this.Search(condition, { fields: ['Id'] as never });
      const roleIds = targetRows.map(row => normalizeRefId((row as { Id?: unknown }).Id)).filter(Boolean) as string[];
      if (roleIds.length > 1) {
        throw new Error('Role.Update with AccessUiResourceIds only supports single record update');
      }
      if (roleIds.length === 1) {
        roleIdForSync = roleIds[0];
        accessIdsForSync = await applyAccessWriteTransformOnUpdate(payload as Record<string, unknown>, roleIds[0]);
      }
    }

    const updated = await super.Update<C, F>(condition, payload, returnFields, options);
    if (roleIdForSync && accessIdsForSync) {
      await syncAllowResourceGrants(roleIdForSync, accessIdsForSync);
      if (updated.length && returnFields != null) {
        updated[0] = await this.Browse(roleIdForSync, returnFields, options);
      }
      if (updated.length && shouldHydrateAccess) {
        (updated[0] as { AccessUiResourceIds?: string[] }).AccessUiResourceIds = [...accessIdsForSync];
      }
    } else if (shouldHydrateAccess) {
      await hydrateAccessUiResourceIds(updated);
    }
    return updated;
  }

  /**
   * Update one role by Id while keeping the UI access projection synchronized.
   */
  static override async UpdateById<C extends ModelCtor, F extends FieldSelection<RowOf<C>> | undefined = undefined>(
    this: C,
    id: string,
    values: Partial<Updateable<RowOf<C>>>,
    returnFields?: F,
    options?: UpdateOptions
  ): Promise<F extends FieldSelection<RowOf<C>> ? Projected<RowOf<C>, F> : Partial<RowOf<C>>> {
    const payload = { ...values };
    const accessIds = await applyAccessWriteTransformOnUpdate(payload as Record<string, unknown>, id);
    let row = await super.UpdateById<C, F>(id, payload, returnFields, options);
    if (accessIds) {
      await syncAllowResourceGrants(id, accessIds);
      if (returnFields != null) {
        row = await this.Browse(id, returnFields, options);
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
