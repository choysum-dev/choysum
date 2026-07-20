// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { BaseModel, Model, Field } from '@/core/service';
import { Onchange } from '@/core/service/api/onchange';
import type { Insertable, Updateable } from '@/core/service/api/input';
import type { FieldSelection } from '@/core/service/api/selection';
import type { QueryCondition } from '@/core/service/api/query';
import { _lt } from '../i18n';
import Role from './role';
import { normalizeRefId } from '@/core/service/utils/normalization';
import { invalidateAllAuthzCaches } from './_request_cache_invalidation';

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
  @Field({
    type: 'ManyToOneRef',
    relation: { targetModel: 'meta.IrApplication' },
    notNull: false,
    size: 20,
    index: true,
    string: _lt('Application', { scope: 'auth.model.RoleFieldRule.fields' }),
  })
  IrApplicationId?: string;

  /**
   * Model-level scope when the rule applies to an entire model.
   */
  @Field({
    type: 'ManyToOneRef',
    relation: { targetModel: 'meta.IrModel' },
    notNull: false,
    size: 20,
    index: true,
    string: _lt('Model', { scope: 'auth.model.RoleFieldRule.fields' }),
  })
  IrModelId?: string;

  /**
   * Field-level scope when the rule applies to one concrete field.
   */
  @Field({
    type: 'ManyToOneRef', relation: { targetModel: 'meta.IrField' },
    notNull: false,
      size: 20,
      index: true,
      checkConstraint: `(
        (
          (deleted_at IS NOT NULL)
          OR (ir_field_id IS NOT NULL AND ir_model_id IS NOT NULL AND ir_application_id IS NULL)
          OR (ir_field_id IS NULL AND ir_model_id IS NOT NULL AND ir_application_id IS NULL)
          OR (ir_field_id IS NULL AND ir_model_id IS NULL AND ir_application_id IS NOT NULL)
          OR (ir_field_id IS NULL AND ir_model_id IS NULL AND ir_application_id IS NULL)
        )
        AND (perm_read IS NOT NULL OR perm_write IS NOT NULL)
      )`,
    string: _lt('Field', { scope: 'auth.model.RoleFieldRule.fields' }),
  })
  IrFieldId?: string;

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
   * Validate that the requested scope resolves to field, model, application, or global.
   */
  private static _validateScopeShape(values: Record<string, any>, mode: 'create' | 'update'): void {
    const touchesScope =
      Object.prototype.hasOwnProperty.call(values, 'IrFieldId') ||
      Object.prototype.hasOwnProperty.call(values, 'IrModelId') ||
      Object.prototype.hasOwnProperty.call(values, 'IrApplicationId');

    const touchesPerm = Object.prototype.hasOwnProperty.call(values, 'PermRead') || Object.prototype.hasOwnProperty.call(values, 'PermWrite');

    if (!touchesScope && !touchesPerm) return;

    if (mode === 'update' && touchesScope) {
      const hasAll =
        Object.prototype.hasOwnProperty.call(values, 'IrFieldId') &&
        Object.prototype.hasOwnProperty.call(values, 'IrModelId') &&
        Object.prototype.hasOwnProperty.call(values, 'IrApplicationId');
      if (!hasAll) {
        throw new Error('invalid RoleFieldRule scope update: must provide IrFieldId/IrModelId/IrApplicationId together');
      }
    }

    if (touchesScope || mode === 'create') {
      const irFieldId = normalizeRefId((values as any).IrFieldId);
      const irModelId = normalizeRefId((values as any).IrModelId);
      const irApplicationId = normalizeRefId((values as any).IrApplicationId);

      const isField = irFieldId != null && irModelId != null && irApplicationId == null;
      const isModel = irFieldId == null && irModelId != null && irApplicationId == null;
      const isApplication = irFieldId == null && irModelId == null && irApplicationId != null;
      const isGlobal = irFieldId == null && irModelId == null && irApplicationId == null;
      if (!isField && !isModel && !isApplication && !isGlobal) {
        throw new Error('invalid RoleFieldRule scope: must be exactly one of field/model/application/global');
      }

      (values as any).IrFieldId = irFieldId;
      (values as any).IrModelId = irModelId;
      (values as any).IrApplicationId = irApplicationId;
    }

    if (touchesPerm || mode === 'create') {
      const permRead = this._normalizePerm((values as any).PermRead);
      const permWrite = this._normalizePerm((values as any).PermWrite);

      (values as any).PermRead = permRead;
      (values as any).PermWrite = permWrite;

      if (permRead == null && permWrite == null) {
        throw new Error('invalid RoleFieldRule: must provide at least one of PermRead/PermWrite');
      }
    }
  }

  /**
   * Create one RoleFieldRule row and invalidate request-scoped auth caches.
   */
  static override async Create<T extends BaseModel>(
    this: { new (...args: any[]): T } & typeof BaseModel,
    value: Partial<Insertable<T & BaseModel>>,
    returnFields?: FieldSelection<T>
  ): Promise<T> {
    RoleFieldRule._validateScopeShape(value as any, 'create');
    const out = await super.Create(value as any, returnFields as any);
    invalidateAllAuthzCaches();
    return out as unknown as T;
  }

  /**
   * Create multiple RoleFieldRule rows and invalidate request-scoped auth caches.
   */
  static override async CreateMany<T extends BaseModel>(
    this: { new (...args: any[]): T } & typeof BaseModel,
    values: Partial<Insertable<T & BaseModel>>[],
    returnFields?: FieldSelection<T>
  ): Promise<T[]> {
    for (const v of values || []) RoleFieldRule._validateScopeShape(v as any, 'create');
    const out = await super.CreateMany(values as any, returnFields as any);
    invalidateAllAuthzCaches();
    return out as unknown as T[];
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
    RoleFieldRule._validateScopeShape(values as any, 'update');
    const out = await super.Update(condition as any, values as any, returnFields as any, options as any);
    invalidateAllAuthzCaches();
    return out as unknown as Partial<T>[];
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
    RoleFieldRule._validateScopeShape(values as any, 'update');
    const out = await super.UpdateById(id as any, values as any, returnFields as any, options as any);
    invalidateAllAuthzCaches();
    return out as unknown as Partial<T>;
  }

  /**
   * Delete matching RoleFieldRule rows and invalidate request-scoped auth caches.
   */
  static override async Delete<T extends BaseModel>(
    this: { new (...args: any[]): T } & typeof BaseModel,
    condition: QueryCondition<T>,
    options?: any
  ): Promise<number> {
    const out = await super.Delete(condition as any, options as any);
    invalidateAllAuthzCaches();
    return out;
  }

  /**
   * Delete one RoleFieldRule row by Id and invalidate request-scoped auth caches.
   */
  static override async DeleteById<T extends BaseModel>(this: { new (...args: any[]): T } & typeof BaseModel, id: string, options?: any): Promise<number> {
    const out = await super.DeleteById(id as any, options as any);
    invalidateAllAuthzCaches();
    return out;
  }

  /**
   * Reset the field scope when the model scope changes and narrow the field picker.
   */
  @Onchange<RoleFieldRule>('IrModelId')
  OnchangeIrModelId() {
    this.IrFieldId = null as any;

    const modelId = this.IrModelId;

    if (modelId) {
      // Narrow the field picker to the selected model
      return {
        condition: [{ field: 'IrFieldId', condition: ['ModelId', '=', modelId] }],
      };
    } else {
      // Fallback to an always-false condition to block selection
      return {
        condition: [{ field: 'IrFieldId', condition: ['Id', '=', '0'] }],
      };
    }
  }
}
