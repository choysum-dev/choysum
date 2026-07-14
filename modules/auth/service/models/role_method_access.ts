// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { BaseModel, Model, Field } from '@/core/service';
import type { Insertable, Updateable } from '@/core/service/api/input';
import type { FieldSelection } from '@/core/service/api/selection';
import type { QueryCondition } from '@/core/service/api/query';
import Role from './role';
import { normalizeRefId } from '@/core/service/utils/normalization';
import { invalidateAllAuthzCaches } from './_request_cache_invalidation';

/**
 * RoleMethodAccess stores role-level RPC allow and deny overrides at global,
 * application, model, or service scope.
 */
@Model('RoleMethodAccess')
export default class RoleMethodAccess extends BaseModel {
  /**
   * Role that owns this method-access entry.
   */
  @Field({ type: 'ManyToOne', relation: { targetModel: () => Role } })
  RoleId: Role;

  /**
   * Application-level scope when the entry targets an entire application.
   */
  @Field({ type: 'ManyToOneRef', relation: { targetModel: 'meta.IrApplication' }, notNull: false, size: 20, index: true})
  IrApplicationId: string | null;

  /**
   * Model-level scope when the entry targets an entire model.
   */
  @Field({ type: 'ManyToOneRef', relation: { targetModel: 'meta.IrModel' }, notNull: false, size: 20, index: true})
  IrModelId: string | null;

  /**
   * Service-level scope when the entry targets one concrete RPC method surface.
   */
  @Field({
    type: 'ManyToOneRef', relation: { targetModel: 'meta.IrService' },
    notNull: false,
      size: 20,
      index: true,
      checkConstraint: `(
        (deleted_at IS NOT NULL)
        OR (ir_service_id IS NOT NULL AND ir_model_id IS NULL AND ir_application_id IS NULL)
        OR (ir_service_id IS NULL AND ir_model_id IS NOT NULL AND ir_application_id IS NULL)
        OR (ir_service_id IS NULL AND ir_model_id IS NULL AND ir_application_id IS NOT NULL)
        OR (ir_service_id IS NULL AND ir_model_id IS NULL AND ir_application_id IS NULL)
      )`,
  })
  IrServiceId: string | null;

  /**
   * Whether the matched scope is allowed or denied.
   */
  @Field({
    type: 'selection',
    selection: [
      { value: 'allow', label: 'Allow' },
      { value: 'deny', label: 'Deny' },
    ],
    default: () => 'deny',
  })
  Mode: 'allow' | 'deny';

  /**
   * Source of the method-access entry.
   */
  @Field({
    type: 'selection',
    selection: [
      { value: 'manual', label: 'Manual' },
      { value: 'ui', label: 'UI' },
    ],
    default: () => 'manual', size: 16, index: true,
  })
  Source: 'manual' | 'ui';

  /**
   * Validate that the requested scope resolves to service, model, application, or global.
   */
  private static _validateScopeShape(values: Record<string, any>, mode: 'create' | 'update'): void {
    const touchesScope =
      Object.prototype.hasOwnProperty.call(values, 'IrServiceId') ||
      Object.prototype.hasOwnProperty.call(values, 'IrModelId') ||
      Object.prototype.hasOwnProperty.call(values, 'IrApplicationId');
    if (!touchesScope) return;

    if (mode === 'update') {
      const hasAll =
        Object.prototype.hasOwnProperty.call(values, 'IrServiceId') &&
        Object.prototype.hasOwnProperty.call(values, 'IrModelId') &&
        Object.prototype.hasOwnProperty.call(values, 'IrApplicationId');
      if (!hasAll) {
        throw new Error('invalid RoleMethodAccess scope update: must provide IrServiceId/IrModelId/IrApplicationId together');
      }
    }

    const irServiceId = normalizeRefId((values as any).IrServiceId);
    const irModelId = normalizeRefId((values as any).IrModelId);
    const irApplicationId = normalizeRefId((values as any).IrApplicationId);

    const isService = irServiceId != null && irModelId == null && irApplicationId == null;
    const isModel = irServiceId == null && irModelId != null && irApplicationId == null;
    const isApplication = irServiceId == null && irModelId == null && irApplicationId != null;
    const isGlobal = irServiceId == null && irModelId == null && irApplicationId == null;
    if (!isService && !isModel && !isApplication && !isGlobal) {
      throw new Error('invalid RoleMethodAccess scope: must be exactly one of service/model/application/global');
    }

    (values as any).IrServiceId = irServiceId;
    (values as any).IrModelId = irModelId;
    (values as any).IrApplicationId = irApplicationId;
  }

  /**
   * Create one RoleMethodAccess row and invalidate request-scoped auth caches.
   */
  static override async Create<T extends BaseModel>(
    this: { new (...args: any[]): T } & typeof BaseModel,
    value: Partial<Insertable<T & BaseModel>>,
    returnFields?: FieldSelection<T>
  ): Promise<T> {
    RoleMethodAccess._validateScopeShape(value as any, 'create');
    const out = await super.Create(value as any, returnFields as any);
    invalidateAllAuthzCaches();
    return out as unknown as T;
  }

  /**
   * Create multiple RoleMethodAccess rows and invalidate request-scoped auth caches.
   */
  static override async CreateMany<T extends BaseModel>(
    this: { new (...args: any[]): T } & typeof BaseModel,
    values: Partial<Insertable<T & BaseModel>>[],
    returnFields?: FieldSelection<T>
  ): Promise<T[]> {
    for (const v of values || []) RoleMethodAccess._validateScopeShape(v as any, 'create');
    const out = await super.CreateMany(values as any, returnFields as any);
    invalidateAllAuthzCaches();
    return out as unknown as T[];
  }

  /**
   * Update RoleMethodAccess rows and invalidate request-scoped auth caches.
   */
  static override async Update<T extends BaseModel>(
    this: { new (...args: any[]): T } & typeof BaseModel,
    condition: QueryCondition<T>,
    values: Partial<Updateable<T & BaseModel>>,
    returnFields?: FieldSelection<T>,
    options?: any
  ): Promise<Partial<T>[]> {
    RoleMethodAccess._validateScopeShape(values as any, 'update');
    const out = await super.Update(condition as any, values as any, returnFields as any, options as any);
    invalidateAllAuthzCaches();
    return out as unknown as Partial<T>[];
  }

  /**
   * Update one RoleMethodAccess row by Id and invalidate request-scoped auth caches.
   */
  static override async UpdateById<T extends BaseModel>(
    this: { new (...args: any[]): T } & typeof BaseModel,
    id: string,
    values: Partial<Updateable<T & BaseModel>>,
    returnFields?: FieldSelection<T>,
    options?: any
  ): Promise<Partial<T>> {
    RoleMethodAccess._validateScopeShape(values as any, 'update');
    const out = await super.UpdateById(id as any, values as any, returnFields as any, options as any);
    invalidateAllAuthzCaches();
    return out as unknown as Partial<T>;
  }

  /**
   * Delete matching RoleMethodAccess rows and invalidate request-scoped auth caches.
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
   * Delete one RoleMethodAccess row by Id and invalidate request-scoped auth caches.
   */
  static override async DeleteById<T extends BaseModel>(this: { new (...args: any[]): T } & typeof BaseModel, id: string, options?: any): Promise<number> {
    const out = await super.DeleteById(id as any, options as any);
    invalidateAllAuthzCaches();
    return out;
  }
}
