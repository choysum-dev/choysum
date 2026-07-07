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
 * Permission mode for a role-to-UI-resource grant.
 */
export type RoleUiResourceMode = 'allow' | 'deny';

/**
 * RoleUiResource stores role-level UI permission overrides at global,
 * application, or concrete UI-resource scope.
 */
@Model('RoleUiResource')
export default class RoleUiResource extends BaseModel {
  /**
   * Role that owns this UI permission override.
   */
  @Field({ type: 'ManyToOne', relation: { targetModel: () => Role } })
  RoleId: Role;

  /**
   * Whether the matching UI scope is allowed or denied.
   */
  @Field({
    type: 'selection',
    selection: [
      { value: 'allow', label: 'Allow' },
      { value: 'deny', label: 'Deny' },
    ],
    column: { default: () => 'allow', size: 16, index: true },
  })
  Mode: RoleUiResourceMode;

  /**
   * Application-level scope. Mutually exclusive with IrUiResourceId.
   */
  @Field({ type: 'ManyToOneRef', targetModel: 'meta.IrApplication', column: { notNull: false, size: 20, index: true } })
  IrApplicationId: string | null;

  /**
   * Concrete UI resource scope. Mutually exclusive with IrApplicationId.
   */
  @Field({
    type: 'ManyToOneRef',
    targetModel: 'meta.IrUiResource',
    column: {
      notNull: false,
      size: 20,
      index: true,
      checkConstraint: `(
        (deleted_at IS NOT NULL)
        OR (ir_ui_resource_id IS NOT NULL AND ir_application_id IS NULL)
        OR (ir_ui_resource_id IS NULL AND ir_application_id IS NOT NULL)
        OR (ir_ui_resource_id IS NULL AND ir_application_id IS NULL)
      )`,
    },
  })
  IrUiResourceId: string | null;

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
   * Validate that the UI-resource scope is one of resource, application, or global.
   */
  private static _validateScopeShape(values: Record<string, any>, mode: 'create' | 'update'): void {
    const touchesScope = Object.prototype.hasOwnProperty.call(values, 'IrUiResourceId') || Object.prototype.hasOwnProperty.call(values, 'IrApplicationId');
    if (!touchesScope) return;

    if (mode === 'update') {
      const hasAll = Object.prototype.hasOwnProperty.call(values, 'IrUiResourceId') && Object.prototype.hasOwnProperty.call(values, 'IrApplicationId');
      if (!hasAll) {
        throw new Error('invalid RoleUiResource scope update: must provide IrUiResourceId/IrApplicationId together');
      }
    }

    const irUiResourceId = normalizeRefId((values as any).IrUiResourceId);
    const irApplicationId = normalizeRefId((values as any).IrApplicationId);

    const isResource = irUiResourceId != null && irApplicationId == null;
    const isApplication = irUiResourceId == null && irApplicationId != null;
    const isGlobal = irUiResourceId == null && irApplicationId == null;
    if (!isResource && !isApplication && !isGlobal) {
      throw new Error('invalid RoleUiResource scope: must be exactly one of resource/application/global');
    }

    (values as any).IrUiResourceId = irUiResourceId;
    (values as any).IrApplicationId = irApplicationId;
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
   * Create one RoleUiResource row and invalidate request-scoped auth caches.
   */
  static override async Create<T extends BaseModel>(
    this: { new (...args: any[]): T } & typeof BaseModel,
    value: Partial<Insertable<T & BaseModel>>,
    returnFields?: FieldSelection<T>
  ): Promise<T> {
    RoleUiResource._validateScopeShape(value as any, 'create');
    RoleUiResource._validateMode(value as any, 'create');
    const out = await super.Create(value as any, returnFields as any);
    invalidateAllAuthzCaches();
    return out as unknown as T;
  }

  /**
   * Create multiple RoleUiResource rows and invalidate request-scoped auth caches.
   */
  static override async CreateMany<T extends BaseModel>(
    this: { new (...args: any[]): T } & typeof BaseModel,
    values: Partial<Insertable<T & BaseModel>>[],
    returnFields?: FieldSelection<T>
  ): Promise<T[]> {
    for (const v of values || []) {
      RoleUiResource._validateScopeShape(v as any, 'create');
      RoleUiResource._validateMode(v as any, 'create');
    }
    const out = await super.CreateMany(values as any, returnFields as any);
    invalidateAllAuthzCaches();
    return out as unknown as T[];
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
    RoleUiResource._validateScopeShape(values as any, 'update');
    RoleUiResource._validateMode(values as any, 'update');
    const out = await super.Update(condition as any, values as any, returnFields as any, options as any);
    invalidateAllAuthzCaches();
    return out as unknown as Partial<T>[];
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
    RoleUiResource._validateScopeShape(values as any, 'update');
    RoleUiResource._validateMode(values as any, 'update');
    const out = await super.UpdateById(id as any, values as any, returnFields as any, options as any);
    invalidateAllAuthzCaches();
    return out as unknown as Partial<T>;
  }

  /**
   * Delete matching RoleUiResource rows and invalidate request-scoped auth caches.
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
   * Delete one RoleUiResource row by Id and invalidate request-scoped auth caches.
   */
  static override async DeleteById<T extends BaseModel>(this: { new (...args: any[]): T } & typeof BaseModel, id: string, options?: any): Promise<number> {
    const out = await super.DeleteById(id as any, options as any);
    invalidateAllAuthzCaches();
    return out;
  }
}
