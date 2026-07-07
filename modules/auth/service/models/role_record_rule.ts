// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { BaseModel, Model, Field } from '@/core/service';
import type { Insertable, Updateable } from '@/core/service/api/input';
import type { FieldSelection } from '@/core/service/api/selection';
import type { QueryCondition } from '@/core/service/api/query';
import Role from './role';
import { normalizeRefId } from './_rule_scope_helpers';
import { invalidateAllAuthzCaches } from './_request_cache_invalidation';

/**
 * RoleRecordRule stores record-level condition filters and CRUD permission
 * overrides for a role at global, application, or model scope.
 */
@Model('RoleRecordRule')
export default class RoleRecordRule extends BaseModel {
  /**
   * Role that owns this record-rule entry.
   */
  @Field({ type: 'ManyToOne', relation: { targetModel: () => Role } })
  RoleId: Role;

  /**
   * Application-level scope when the rule targets an entire application.
   */
  @Field({ type: 'ManyToOneRef', targetModel: 'meta.IrApplication', column: { notNull: false, size: 20, index: true } })
  IrApplicationId: string | null;

  /**
   * Model-level scope when the rule targets one concrete model.
   */
  @Field({
    type: 'ManyToOneRef',
    targetModel: 'meta.IrModel',
    column: {
      notNull: false,
      size: 20,
      index: true,
      checkConstraint: `(
        (deleted_at IS NOT NULL)
        OR (ir_model_id IS NOT NULL AND ir_application_id IS NULL)
        OR (ir_model_id IS NULL AND ir_application_id IS NOT NULL)
        OR (ir_model_id IS NULL AND ir_application_id IS NULL)
      )`,
    },
  })
  IrModelId: string | null;

  /**
   * Condition envelope applied to matching records.
   */
  @Field({ type: 'jsonobject', column: { notNull: false } })
  Condition: QueryCondition<any>;

  /**
   * Whether reads are allowed when this rule matches.
   */
  @Field({ type: 'boolean', column: { default: () => false } })
  PermRead: boolean;

  /**
   * Whether writes are allowed when this rule matches.
   */
  @Field({ type: 'boolean', column: { default: () => false } })
  PermWrite: boolean;

  /**
   * Whether creates are allowed when this rule matches.
   */
  @Field({ type: 'boolean', column: { default: () => false } })
  PermCreate: boolean;

  /**
   * Whether deletes are allowed when this rule matches.
   */
  @Field({ type: 'boolean', column: { default: () => false } })
  PermDelete: boolean;

  /**
   * Validate that the requested scope resolves to model, application, or global.
   */
  private static _validateScopeShape(values: Record<string, any>, mode: 'create' | 'update'): void {
    const touchesScope = Object.prototype.hasOwnProperty.call(values, 'IrModelId') || Object.prototype.hasOwnProperty.call(values, 'IrApplicationId');
    if (!touchesScope) return;

    if (mode === 'update') {
      const hasAll = Object.prototype.hasOwnProperty.call(values, 'IrModelId') && Object.prototype.hasOwnProperty.call(values, 'IrApplicationId');
      if (!hasAll) {
        throw new Error('invalid RoleRecordRule scope update: must provide IrModelId/IrApplicationId together');
      }
    }

    const irModelId = normalizeRefId((values as any).IrModelId);
    const irApplicationId = normalizeRefId((values as any).IrApplicationId);

    const isModel = irModelId != null && irApplicationId == null;
    const isApplication = irModelId == null && irApplicationId != null;
    const isGlobal = irModelId == null && irApplicationId == null;
    if (!isModel && !isApplication && !isGlobal) {
      throw new Error('invalid RoleRecordRule scope: must be exactly one of model/application/global');
    }

    (values as any).IrModelId = irModelId;
    (values as any).IrApplicationId = irApplicationId;
  }

  /**
   * Create one RoleRecordRule row and invalidate request-scoped auth caches.
   */
  static override async Create<T extends BaseModel>(
    this: { new (...args: any[]): T } & typeof BaseModel,
    value: Partial<Insertable<T & BaseModel>>,
    returnFields?: FieldSelection<T>
  ): Promise<T> {
    RoleRecordRule._validateScopeShape(value as any, 'create');
    const out = await super.Create(value as any, returnFields as any);
    invalidateAllAuthzCaches();
    return out as unknown as T;
  }

  /**
   * Create multiple RoleRecordRule rows and invalidate request-scoped auth caches.
   */
  static override async CreateMany<T extends BaseModel>(
    this: { new (...args: any[]): T } & typeof BaseModel,
    values: Partial<Insertable<T & BaseModel>>[],
    returnFields?: FieldSelection<T>
  ): Promise<T[]> {
    for (const v of values || []) RoleRecordRule._validateScopeShape(v as any, 'create');
    const out = await super.CreateMany(values as any, returnFields as any);
    invalidateAllAuthzCaches();
    return out as unknown as T[];
  }

  /**
   * Update RoleRecordRule rows and invalidate request-scoped auth caches.
   */
  static override async Update<T extends BaseModel>(
    this: { new (...args: any[]): T } & typeof BaseModel,
    condition: QueryCondition<T>,
    values: Partial<Updateable<T & BaseModel>>,
    returnFields?: FieldSelection<T>,
    options?: any
  ): Promise<Partial<T>[]> {
    RoleRecordRule._validateScopeShape(values as any, 'update');
    const out = await super.Update(condition as any, values as any, returnFields as any, options as any);
    invalidateAllAuthzCaches();
    return out as unknown as Partial<T>[];
  }

  /**
   * Update one RoleRecordRule row by Id and invalidate request-scoped auth caches.
   */
  static override async UpdateById<T extends BaseModel>(
    this: { new (...args: any[]): T } & typeof BaseModel,
    id: string,
    values: Partial<Updateable<T & BaseModel>>,
    returnFields?: FieldSelection<T>,
    options?: any
  ): Promise<Partial<T>> {
    RoleRecordRule._validateScopeShape(values as any, 'update');
    const out = await super.UpdateById(id as any, values as any, returnFields as any, options as any);
    invalidateAllAuthzCaches();
    return out as unknown as Partial<T>;
  }

  /**
   * Delete matching RoleRecordRule rows and invalidate request-scoped auth caches.
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
   * Delete one RoleRecordRule row by Id and invalidate request-scoped auth caches.
   */
  static override async DeleteById<T extends BaseModel>(this: { new (...args: any[]): T } & typeof BaseModel, id: string, options?: any): Promise<number> {
    const out = await super.DeleteById(id as any, options as any);
    invalidateAllAuthzCaches();
    return out;
  }
}
