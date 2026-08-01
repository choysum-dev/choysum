// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { BaseModel, Model, Field } from '@/core/service';
import { Onchange } from '@/core/service/api/onchange';
import type { Insertable, Updateable } from '@/core/service/api/input';
import type { FieldSelection } from '@/core/service/api/selection';
import type { QueryCondition } from '@/core/service/api/query';
import { _lt } from '../i18n';
import Role from './role';
import type MetaApplication from '@/meta/service/models/application';
import type MetaModel from '@/meta/service/models/model';
import type MetaField from '@/meta/service/models/field';
import { mutateThenInvalidateAllAuthzCaches } from './_authz_mutation_helpers';
import { assertExclusiveScope } from './_rule_scope_helpers';

/**
 * RoleFieldRule stores field-level read and write overrides for a role at
 * global, application, model, or field scope.
 */
@Model('RoleFieldRule')
export default class RoleFieldRule extends BaseModel {
  /**
   * Role that owns this field-rule entry.
   */
  @Field({
    type: 'ManyToOne',
    relation: { targetModel: () => Role },
    string: _lt('Role', { scope: 'auth.model.RoleFieldRule.fields' }),
  })
  RoleId: Role;

  /**
   * Application-level scope when the rule applies to an entire application.
   */
  @Field<MetaApplication>({
    type: 'ManyToOneRef',
    relation: { targetModel: 'meta.MetaApplication' },
    notNull: false,
    size: 20,
    index: true,
    string: _lt('Application', { scope: 'auth.model.RoleFieldRule.fields' }),
    help: _lt('Leave all scopes empty for global; otherwise pick one level (field scope also needs a model).', {
      scope: 'auth.model.RoleFieldRule.fields',
    }),
  })
  MetaApplicationId?: string;

  /**
   * Model-level scope when the rule applies to an entire model.
   */
  @Field<MetaModel>({
    type: 'ManyToOneRef',
    relation: { targetModel: 'meta.MetaModel' },
    notNull: false,
    size: 20,
    index: true,
    string: _lt('Model', { scope: 'auth.model.RoleFieldRule.fields' }),
    help: _lt('Leave all scopes empty for global; otherwise pick one level (field scope also needs a model).', {
      scope: 'auth.model.RoleFieldRule.fields',
    }),
  })
  MetaModelId?: string;

  /**
   * Field-level scope when the rule applies to one concrete field.
   */
  @Field<MetaField>({
    type: 'ManyToOneRef', relation: { targetModel: 'meta.MetaField' },
    notNull: false,
      size: 20,
      index: true,
      checkConstraint: `(
        (
          (deleted_at IS NOT NULL)
          OR (meta_field_id IS NOT NULL AND meta_model_id IS NOT NULL AND meta_application_id IS NULL)
          OR (meta_field_id IS NULL AND meta_model_id IS NOT NULL AND meta_application_id IS NULL)
          OR (meta_field_id IS NULL AND meta_model_id IS NULL AND meta_application_id IS NOT NULL)
          OR (meta_field_id IS NULL AND meta_model_id IS NULL AND meta_application_id IS NULL)
        )
        AND (perm_read IS NOT NULL OR perm_write IS NOT NULL)
      )`,
    string: _lt('Field', { scope: 'auth.model.RoleFieldRule.fields' }),
    help: _lt('Leave all scopes empty for global; otherwise pick one level (field scope also needs a model).', {
      scope: 'auth.model.RoleFieldRule.fields',
    }),
  })
  MetaFieldId?: string;

  /**
   * Read permission override for the selected scope.
   */
  @Field({
    type: 'selection',
    selection: [
      { value: 'allow', label: _lt('Allow', { scope: 'auth.model.RoleFieldRule.fields' }) },
      { value: 'deny', label: _lt('Deny', { scope: 'auth.model.RoleFieldRule.fields' }) },
    ],
    notNull: false,
    string: _lt('Read', { scope: 'auth.model.RoleFieldRule.fields' }),
    help: _lt('Override field visibility or editability; at least one of Read/Write must be set.', {
      scope: 'auth.model.RoleFieldRule.fields',
    }),
  })
  PermRead?: 'allow' | 'deny';

  /**
   * Write permission override for the selected scope.
   */
  @Field({
    type: 'selection',
    selection: [
      { value: 'allow', label: _lt('Allow', { scope: 'auth.model.RoleFieldRule.fields' }) },
      { value: 'deny', label: _lt('Deny', { scope: 'auth.model.RoleFieldRule.fields' }) },
    ],
    notNull: false,
    string: _lt('Write', { scope: 'auth.model.RoleFieldRule.fields' }),
    help: _lt('Override field visibility or editability; at least one of Read/Write must be set.', {
      scope: 'auth.model.RoleFieldRule.fields',
    }),
  })
  PermWrite?: 'allow' | 'deny';

  /**
   * Normalize a field-rule permission override.
   */
  private static _normalizePerm(v: any): 'allow' | 'deny' | null {
    if (v == null) return null;
    const s = String(v ?? '')
      .trim()
      .toLowerCase();
    if (!s) return null;
    if (s === 'allow' || s === 'deny') return s;
    throw new Error(`invalid RoleFieldRule perm value: ${String(v)}`);
  }

  /**
   * Normalize PermRead/PermWrite.
   *
   * On update, only rewrite keys present in `values` so a partial patch does not
   * null out the untouched permission column. Create still requires at least one.
   */
  private static _validatePerms(values: Record<string, any>, mode: 'create' | 'update'): void {
    const hasRead = Object.prototype.hasOwnProperty.call(values, 'PermRead');
    const hasWrite = Object.prototype.hasOwnProperty.call(values, 'PermWrite');
    if (!hasRead && !hasWrite && mode !== 'create') return;

    if (hasRead || mode === 'create') {
      (values as any).PermRead = this._normalizePerm((values as any).PermRead);
    }
    if (hasWrite || mode === 'create') {
      (values as any).PermWrite = this._normalizePerm((values as any).PermWrite);
    }

    // Create always materializes both keys; reject empty. Update only rejects when
    // the caller explicitly clears both in the same payload.
    if (mode === 'create' || (hasRead && hasWrite)) {
      if ((values as any).PermRead == null && (values as any).PermWrite == null) {
        throw new Error('invalid RoleFieldRule: must provide at least one of PermRead/PermWrite');
      }
    }
  }

  /**
   * Run scope and permission validation before mutating RoleFieldRule rows.
   */
  private static _prepareValues(values: Record<string, any>, mode: 'create' | 'update'): void {
    assertExclusiveScope(values, mode, 'field');
    this._validatePerms(values, mode);
  }

  /**
   * Create one RoleFieldRule row and invalidate request-scoped auth caches.
   */
  static override async Create<T extends BaseModel>(
    this: { new (...args: any[]): T } & typeof BaseModel,
    value: Partial<Insertable<T & BaseModel>>,
    returnFields?: FieldSelection<T>
  ): Promise<T> {
    RoleFieldRule._prepareValues(value as any, 'create');
    return mutateThenInvalidateAllAuthzCaches(async () => {
      const out = await super.Create(value as any, returnFields as any);
      return out as unknown as T;
    });
  }

  /**
   * Create multiple RoleFieldRule rows and invalidate request-scoped auth caches.
   */
  static override async CreateMany<T extends BaseModel>(
    this: { new (...args: any[]): T } & typeof BaseModel,
    values: Partial<Insertable<T & BaseModel>>[],
    returnFields?: FieldSelection<T>
  ): Promise<T[]> {
    const rows = values || [];
    for (const v of rows) RoleFieldRule._prepareValues(v as any, 'create');
    return mutateThenInvalidateAllAuthzCaches(async () => {
      const out = await super.CreateMany(rows as any, returnFields as any);
      return out as unknown as T[];
    });
  }

  /**
   * Update RoleFieldRule rows and invalidate request-scoped auth caches.
   */
  static override async Update<T extends BaseModel>(
    this: { new (...args: any[]): T } & typeof BaseModel,
    condition: QueryCondition<T>,
    values: Partial<Updateable<T & BaseModel>>,
    returnFields?: FieldSelection<T>,
    options?: any
  ): Promise<Partial<T>[]> {
    RoleFieldRule._prepareValues(values as any, 'update');
    return mutateThenInvalidateAllAuthzCaches(async () => {
      const out = await super.Update(condition as any, values as any, returnFields as any, options as any);
      return out as unknown as Partial<T>[];
    });
  }

  /**
   * Update one RoleFieldRule row by Id and invalidate request-scoped auth caches.
   */
  static override async UpdateById<T extends BaseModel>(
    this: { new (...args: any[]): T } & typeof BaseModel,
    id: string,
    values: Partial<Updateable<T & BaseModel>>,
    returnFields?: FieldSelection<T>,
    options?: any
  ): Promise<Partial<T>> {
    RoleFieldRule._prepareValues(values as any, 'update');
    return mutateThenInvalidateAllAuthzCaches(async () => {
      const out = await super.UpdateById(id as any, values as any, returnFields as any, options as any);
      return out as unknown as Partial<T>;
    });
  }

  /**
   * Delete matching RoleFieldRule rows and invalidate request-scoped auth caches.
   */
  static override async Delete<T extends BaseModel>(
    this: { new (...args: any[]): T } & typeof BaseModel,
    condition: QueryCondition<T>,
    options?: any
  ): Promise<number> {
    return mutateThenInvalidateAllAuthzCaches(() => super.Delete(condition as any, options as any));
  }

  /**
   * Delete one RoleFieldRule row by Id and invalidate request-scoped auth caches.
   */
  static override async DeleteById<T extends BaseModel>(this: { new (...args: any[]): T } & typeof BaseModel, id: string, options?: any): Promise<number> {
    return mutateThenInvalidateAllAuthzCaches(() => super.DeleteById(id as any, options as any));
  }

  /**
   * Reset the field scope when the model scope changes and narrow the field picker.
   */
  @Onchange<RoleFieldRule>('MetaModelId')
  OnchangeMetaModelId() {
    this.MetaFieldId = null as any;

    const modelId = this.MetaModelId;

    if (modelId) {
      // Narrow the field picker to the selected model
      return {
        condition: [{ field: 'MetaFieldId', condition: ['ModelId', '=', modelId] }],
      };
    } else {
      // Fallback to an always-false condition to block selection
      return {
        condition: [{ field: 'MetaFieldId', condition: ['Id', '=', '0'] }],
      };
    }
  }
}
