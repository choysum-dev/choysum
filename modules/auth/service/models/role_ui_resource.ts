// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { BaseModel, Model, Field } from '@/core/service';
import type { Insertable, Updateable } from '@/core/service/api/input';
import type { FieldSelection } from '@/core/service/api/selection';
import type { QueryCondition } from '@/core/service/api/query';
import { _lt } from '../i18n';
import Role from './role';
import type MetaApplication from '@/meta/service/models/application';
import type MetaUiResource from '@/meta/service/models/ui_resource';
import AuthzMutationModel from '../mixins/authz_mutation_model';
import { assertExclusiveScope } from './_rule_scope_helpers';

/**
 * Permission mode for a role-to-UI-resource grant.
 */
export type RoleUiResourceMode = 'allow' | 'deny';

/**
 * RoleUiResource stores role-level UI permission overrides at global,
 * application, or concrete UI-resource scope.
 */
@Model('RoleUiResource')
export default class RoleUiResource extends AuthzMutationModel {
  /**
   * Role that owns this UI permission override.
   */
  @Field({
    type: 'ManyToOne',
    relation: { targetModel: () => Role },
    string: _lt('Role', { scope: 'auth.model.RoleUiResource.fields' }),
  })
  RoleId: Role;

  /**
   * Whether the matching UI scope is allowed or denied.
   */
  @Field({
    type: 'selection',
    selection: [
      { value: 'allow', label: _lt('Allow', { scope: 'auth.model.RoleUiResource.fields' }) },
      { value: 'deny', label: _lt('Deny', { scope: 'auth.model.RoleUiResource.fields' }) },
    ],
    default: () => 'allow',
    size: 16,
    index: true,
    string: _lt('Mode', { scope: 'auth.model.RoleUiResource.fields' }),
    help: _lt('Allow shows UI resources; deny hides them even if inherited.', {
      scope: 'auth.model.RoleUiResource.fields',
    }),
  })
  Mode: RoleUiResourceMode;

  /**
   * Application-level scope. Mutually exclusive with MetaUiResourceId.
   */
  @Field<MetaApplication>({
    type: 'ManyToOneRef',
    relation: { targetModel: 'meta.MetaApplication' },
    notNull: false,
    size: 20,
    index: true,
    string: _lt('Application Scope', { scope: 'auth.model.RoleUiResource.fields' }),
  })
  MetaApplicationId: string | null;

  /**
   * Concrete UI resource scope. Mutually exclusive with MetaApplicationId.
   */
  @Field<MetaUiResource>({
    type: 'ManyToOneRef', relation: { targetModel: 'meta.MetaUiResource' },
    notNull: false,
    size: 20,
    index: true,
    checkConstraint: `(
        (deleted_at IS NOT NULL)
        OR (meta_ui_resource_id IS NOT NULL AND meta_application_id IS NULL)
        OR (meta_ui_resource_id IS NULL AND meta_application_id IS NOT NULL)
        OR (meta_ui_resource_id IS NULL AND meta_application_id IS NULL)
    )`,
    string: _lt('UI Resource', { scope: 'auth.model.RoleUiResource.fields' }),
  })
  MetaUiResourceId: string | null;

  /**
   * Normalize the permission mode and reject unsupported values.
   */
  private static _normalizeMode(v: any): RoleUiResourceMode {
    const mode = String(v ?? 'allow')
      .trim()
      .toLowerCase();
    if (mode === 'allow' || mode === 'deny') return mode;
    throw new Error("invalid RoleUiResource mode: must be 'allow' or 'deny'");
  }

  /**
   * Normalize and validate the Mode field for create and update operations.
   */
  private static _validateMode(values: Record<string, any>, mode: 'create' | 'update'): void {
    const touchesMode = Object.prototype.hasOwnProperty.call(values, 'Mode');
    if (!touchesMode && mode !== 'create') return;

    (values as any).Mode = this._normalizeMode((values as any).Mode);
  }

  /**
   * Run scope and mode validation before mutating RoleUiResource rows.
   */
  private static _prepareValues(values: Record<string, any>, mode: 'create' | 'update'): void {
    assertExclusiveScope(values, mode, 'ui');
    this._validateMode(values, mode);
  }

  /**
   * Create one RoleUiResource row and invalidate request-scoped auth caches.
   */
  static override async Create<T extends BaseModel>(
    this: { new (...args: any[]): T } & typeof BaseModel,
    value: Partial<Insertable<T & BaseModel>>,
    returnFields?: FieldSelection<T>
  ): Promise<T> {
    RoleUiResource._prepareValues(value as any, 'create');
    return (await super.Create(value as any, returnFields as any)) as unknown as T;
  }

  /**
   * Create multiple RoleUiResource rows and invalidate request-scoped auth caches.
   */
  static override async CreateMany<T extends BaseModel>(
    this: { new (...args: any[]): T } & typeof BaseModel,
    values: Partial<Insertable<T & BaseModel>>[],
    returnFields?: FieldSelection<T>
  ): Promise<T[]> {
    const rows = values || [];
    for (const v of rows) RoleUiResource._prepareValues(v as any, 'create');
    return (await super.CreateMany(rows as any, returnFields as any)) as unknown as T[];
  }

  /**
   * Update RoleUiResource rows and invalidate request-scoped auth caches.
   */
  static override async Update<T extends BaseModel>(
    this: { new (...args: any[]): T } & typeof BaseModel,
    condition: QueryCondition<T>,
    values: Partial<Updateable<T & BaseModel>>,
    returnFields?: FieldSelection<T>,
    options?: any
  ): Promise<Partial<T>[]> {
    RoleUiResource._prepareValues(values as any, 'update');
    return (await super.Update(condition as any, values as any, returnFields as any, options as any)) as unknown as Partial<T>[];
  }

  /**
   * Update one RoleUiResource row by Id and invalidate request-scoped auth caches.
   */
  static override async UpdateById<T extends BaseModel>(
    this: { new (...args: any[]): T } & typeof BaseModel,
    id: string,
    values: Partial<Updateable<T & BaseModel>>,
    returnFields?: FieldSelection<T>,
    options?: any
  ): Promise<Partial<T>> {
    RoleUiResource._prepareValues(values as any, 'update');
    return (await super.UpdateById(id as any, values as any, returnFields as any, options as any)) as unknown as Partial<T>;
  }
}
