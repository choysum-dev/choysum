// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { BaseModel, Model, Field } from '@/core/service';
import { Constraint, type ConstraintContext } from '@/core/service/api/constraint';
import type { Updateable } from '@/core/service/api/input';
import type { FieldSelection } from '@/core/service/api/selection';
import type { QueryCondition } from '@/core/service/api/query';
import { ChoysumError, GrpcCode } from '@/core/service/error';
import { createServiceByModel } from '@/core/service/rpc';
import type MetaModel from '@/meta/service/models/model';
import { _lt, _t } from '../i18n';
import { resolveEffectiveModelId } from './_resolve_effective_model';

const SCOPE = 'web.model.SavedFilter';

/** Lazy dial so web service can load without auth already registered. */
function roleService(): any {
  return createServiceByModel('auth.Role');
}
function userRoleService(): any {
  return createServiceByModel('auth.UserRole');
}

/**
 * Persisted Favorites filter for OSearch (Owner Application = web).
 *
 * Identity: Application + ModelName. ModelId stores the effective meta.MetaModel id.
 * Field normalize / ModelId / uniqueness / IsDefault mutex → `@Constraint` (uses ctx.mode;
 * Create pre-assigns Id before validation, so `!this.Id` is not a reliable create signal).
 * Shared write/delete ACL (SF11) → Update/Delete overrides (Constraint does not run on delete).
 */
@Model('SavedFilter', { application: 'web', softDelete: false })
export default class SavedFilter extends BaseModel {
  /**
   * Display name of the favorite (unique per Application/ModelName/UserId).
   */
  @Field({
    type: 'varchar',
    size: 255,
    notNull: true,
    index: true,
    string: _lt('Name', { scope: `${SCOPE}.fields` }),
  })
  Name: string;

  /**
   * Target application short name (store.application).
   */
  @Field({
    type: 'varchar',
    size: 64,
    notNull: true,
    index: true,
    string: _lt('Application', { scope: `${SCOPE}.fields` }),
  })
  Application: string;

  /**
   * Target model short name (store.modelName).
   */
  @Field({
    type: 'varchar',
    size: 128,
    notNull: true,
    index: true,
    string: _lt('Model Name', { scope: `${SCOPE}.fields` }),
  })
  ModelName: string;

  /**
   * Effective meta.MetaModel id for Application + ModelName (SF12).
   * notNull is false at the Field/kernel layer because required checks run before
   * `@Constraint` writeback; the constraint always resolves and requires ModelId.
   */
  @Field<MetaModel>({
    type: 'ManyToOneRef',
    relation: { targetModel: 'meta.MetaModel' },
    notNull: false,
    size: 20,
    index: true,
    string: _lt('Model', { scope: `${SCOPE}.fields` }),
  })
  ModelId: string;

  /**
   * QueryCondition JSON applied via store.Search (Choysum Condition, not Odoo domain).
   */
  @Field({
    type: 'jsonobject',
    notNull: true,
    default: () => ({}),
    string: _lt('Condition', { scope: `${SCOPE}.fields` }),
  })
  Condition: QueryCondition<any>;

  /**
   * Optional sort specification JSON.
   */
  @Field({
    type: 'jsonobject',
    notNull: false,
    string: _lt('Sort', { scope: `${SCOPE}.fields` }),
  })
  Sort?: any;

  /**
   * Owner user id; null means shared with all logged-in users.
   */
  @Field({
    type: 'varchar',
    size: 20,
    notNull: false,
    index: true,
    string: _lt('User', { scope: `${SCOPE}.fields` }),
  })
  UserId?: string | null;

  /**
   * Whether this favorite is applied when the search view opens.
   */
  @Field({
    type: 'boolean',
    default: () => false,
    string: _lt('Is Default', { scope: `${SCOPE}.fields` }),
  })
  IsDefault: boolean;

  /**
   * Soft-active flag (table uses hard delete; Active gates visibility).
   */
  @Field({
    type: 'boolean',
    default: () => true,
    string: _lt('Active', { scope: `${SCOPE}.fields` }),
  })
  Active: boolean;

  /**
   * Creator user id for SF11 shared-row write/delete ACL.
   * BaseModel does not yet expose CreateUid audit; persist on this model.
   * notNull is false for the same kernel-before-constraint reason as ModelId;
   * `@Constraint` always stamps CreateUid on create.
   */
  @Field({
    type: 'varchar',
    size: 20,
    notNull: false,
    index: true,
    copy: false,
    string: _lt('Created By', { scope: `${SCOPE}.fields` }),
  })
  CreateUid: string;

  private static _fail(code: string, message: string, grpc: GrpcCode = GrpcCode.InvalidArgument): never {
    throw new ChoysumError({ domain: 'web', code, message }).withGrpcCode(grpc);
  }

  private static _actorId(): string {
    return String(this.userId || '').trim();
  }

  private static async _isSysAdmin(userId: string): Promise<boolean> {
    if (!userId) return false;
    try {
      const roles = await roleService().Search(['Code', '=', 'sys.admin'] as any, { fields: ['Id'], limit: 1 } as any);
      const roleId = String((roles as any)?.[0]?.Id || '').trim();
      if (!roleId) return false;
      const links = await userRoleService().Search(
        {
          And: [
            ['UserId', '=', userId],
            ['RoleId', '=', roleId],
          ],
        } as any,
        { fields: ['Id'], limit: 1 } as any
      );
      return Array.isArray(links) && links.length > 0;
    } catch {
      // Fail closed when auth is not loaded (web does not depend on auth).
      return false;
    }
  }

  private static async _assertCanMutate(row: { UserId?: string | null; CreateUid?: string }): Promise<void> {
    const uid = this._actorId();
    if (!uid) {
      this._fail('PermissionDenied', _t('Authentication required', { scope: SCOPE }), GrpcCode.Unauthenticated);
    }
    const owner = row.UserId == null || row.UserId === '' ? null : String(row.UserId).trim();
    if (owner != null) {
      if (owner !== uid) {
        this._fail('PermissionDenied', _t('You can only modify your own private favorites', { scope: SCOPE }), GrpcCode.PermissionDenied);
      }
      return;
    }
    const createUid = String(row.CreateUid || '').trim();
    if (createUid === uid) return;
    if (await this._isSysAdmin(uid)) return;
    this._fail(
      'PermissionDenied',
      _t('Only the creator or a system administrator can modify shared favorites', { scope: SCOPE }),
      GrpcCode.PermissionDenied
    );
  }

  /**
   * Clear other IsDefault rows. Uses BaseModel.Update to skip SF11 (mutex is not an ACL op).
   */
  private static async _clearOtherDefaults(app: string, modelName: string, userId: string | null, exceptId?: string): Promise<void> {
    const cond: any = {
      And: [
        ['Application', '=', app],
        ['ModelName', '=', modelName],
        ['IsDefault', '=', true],
      ],
    };
    if (userId == null || userId === '') {
      cond.And.push(['UserId', '=', null]);
    } else {
      cond.And.push(['UserId', '=', userId]);
    }
    if (exceptId) {
      cond.And.push(['Id', '!=', exceptId]);
    }
    await (BaseModel.Update as any).call(SavedFilter, cond, { IsDefault: false } as any, ['Id'] as any);
  }

  private static async _assertUniqueName(app: string, modelName: string, userId: string | null, name: string, exceptId?: string): Promise<void> {
    const cond: any = {
      And: [
        ['Application', '=', app],
        ['ModelName', '=', modelName],
        ['Name', '=', name],
      ],
    };
    if (userId == null || userId === '') {
      cond.And.push(['UserId', '=', null]);
    } else {
      cond.And.push(['UserId', '=', userId]);
    }
    if (exceptId) {
      cond.And.push(['Id', '!=', exceptId]);
    }
    const hits = await SavedFilter.Search(cond as any, { fields: ['Id'], limit: 1 } as any);
    if (Array.isArray(hits) && hits.length > 0) {
      this._fail('AlreadyExists', _t('A favorite with this name already exists', { scope: SCOPE }), GrpcCode.AlreadyExists);
    }
  }

  private static _mergedField(self: SavedFilter, ctx: ConstraintContext<SavedFilter>, key: string): any {
    if (Object.prototype.hasOwnProperty.call(ctx.values || {}, key)) return (ctx.values as any)[key];
    if (Object.prototype.hasOwnProperty.call(self as any, key)) return (self as any)[key];
    return (ctx.current as any)?.[key];
  }

  /**
   * Normalize identity / ownership, resolve effective ModelId, enforce uniqueness and IsDefault mutex.
   * Static so we can read `ctx.mode` (Create pre-assigns Id before validation).
   */
  @Constraint<SavedFilter>(['Name', 'Application', 'ModelName', 'ModelId', 'UserId', 'IsDefault', 'Condition', 'Active', 'CreateUid'])
  static async validateSavedFilterConstraint(self: SavedFilter, ctx: ConstraintContext<SavedFilter>): Promise<void> {
    const isCreate = ctx.mode === 'create';
    const values = ctx.values as Record<string, any>;
    const currentId = String((isCreate ? values.Id : SavedFilter._mergedField(self, ctx, 'Id')) || '').trim() || undefined;
    const actor = SavedFilter._actorId();
    if (!actor) {
      SavedFilter._fail('PermissionDenied', _t('Authentication required', { scope: SCOPE }), GrpcCode.Unauthenticated);
    }

    const app = String(SavedFilter._mergedField(self, ctx, 'Application') || '').trim();
    const modelName = String(SavedFilter._mergedField(self, ctx, 'ModelName') || '').trim();
    const name = String(SavedFilter._mergedField(self, ctx, 'Name') || '').trim();
    if (!app || !modelName || !name) {
      SavedFilter._fail('InvalidArgument', _t('Name, Application, and ModelName are required', { scope: SCOPE }));
    }
    values.Application = app;
    values.ModelName = modelName;
    values.Name = name;

    const effectiveId = await resolveEffectiveModelId(app, modelName);
    if (!effectiveId) {
      SavedFilter._fail(
        'FailedPrecondition',
        _t('No effective model found for %s.%s', { scope: SCOPE }, app, modelName),
        GrpcCode.FailedPrecondition
      );
    }
    values.ModelId = effectiveId;

    if (isCreate) {
      values.CreateUid = actor;
      const touchedUserId = Object.prototype.hasOwnProperty.call(values, 'UserId');
      if (!touchedUserId) {
        values.UserId = actor;
      } else if (values.UserId == null || values.UserId === '') {
        values.UserId = null;
      } else {
        values.UserId = String(values.UserId).trim();
      }
      if (values.IsDefault == null) values.IsDefault = false;
      if (values.Active == null) values.Active = true;
      if (values.Condition == null) values.Condition = {};
    } else {
      // CreateUid is immutable.
      values.CreateUid = String((ctx.current as any)?.CreateUid || SavedFilter._mergedField(self, ctx, 'CreateUid') || '').trim();
      if (Object.prototype.hasOwnProperty.call(values, 'UserId')) {
        if (values.UserId === '' || values.UserId == null) values.UserId = null;
        else values.UserId = String(values.UserId).trim();
      }
    }

    const effectiveUserId = (() => {
      const raw = Object.prototype.hasOwnProperty.call(values, 'UserId')
        ? values.UserId
        : SavedFilter._mergedField(self, ctx, 'UserId');
      if (raw == null || raw === '') return null;
      return String(raw).trim();
    })();

    await SavedFilter._assertUniqueName(app, modelName, effectiveUserId, name, isCreate ? undefined : currentId);

    const isDefault = Object.prototype.hasOwnProperty.call(values, 'IsDefault')
      ? values.IsDefault === true
      : SavedFilter._mergedField(self, ctx, 'IsDefault') === true;
    if (isDefault) {
      await SavedFilter._clearOtherDefaults(app, modelName, effectiveUserId, isCreate ? undefined : currentId);
    }
  }

  /**
   * Update SavedFilter rows after SF11 ACL (field rules run via Constraint).
   */
  static override async Update<T extends BaseModel>(
    this: { new (...args: any[]): T } & typeof BaseModel,
    condition: QueryCondition<T>,
    values: Partial<Updateable<T & BaseModel>>,
    returnFields?: FieldSelection<T>,
    options?: any
  ): Promise<Partial<T>[]> {
    const existing = await this.Search(condition as any, {
      fields: ['Id', 'UserId', 'CreateUid'] as any,
      ...(options || {}),
    } as any);
    for (const row of existing || []) {
      await SavedFilter._assertCanMutate(row as any);
    }
    if (!existing || existing.length === 0) return [];
    const patch = { ...(values as any) };
    delete patch.CreateUid;
    return (await super.Update(condition as any, patch as any, returnFields as any, options as any)) as unknown as Partial<T>[];
  }

  /**
   * Update one SavedFilter by id after SF11 ACL.
   */
  static override async UpdateById<T extends BaseModel>(
    this: { new (...args: any[]): T } & typeof BaseModel,
    id: string,
    values: Partial<Updateable<T & BaseModel>>,
    returnFields?: FieldSelection<T>,
    options?: any
  ): Promise<Partial<T>> {
    const row = await this.Browse(id, ['Id', 'UserId', 'CreateUid'] as any);
    if (!row) {
      SavedFilter._fail('NotFound', _t('Favorite not found', { scope: SCOPE }), GrpcCode.NotFound);
    }
    await SavedFilter._assertCanMutate(row as any);
    const patch = { ...(values as any) };
    delete patch.CreateUid;
    return (await super.UpdateById(id, patch as any, returnFields as any, options as any)) as unknown as Partial<T>;
  }

  /**
   * Delete SavedFilter rows matching a condition (SF11).
   */
  static override async Delete<T extends BaseModel>(
    this: { new (...args: any[]): T } & typeof BaseModel,
    condition: QueryCondition<T>,
    options?: any
  ): Promise<number> {
    const existing = await this.Search(condition as any, { fields: ['Id', 'UserId', 'CreateUid'] as any, ...(options || {}) } as any);
    for (const row of existing || []) {
      await SavedFilter._assertCanMutate(row as any);
    }
    if (!existing || existing.length === 0) return 0;
    return await super.Delete(condition as any, options as any);
  }

  /**
   * Delete one SavedFilter by id (SF11).
   */
  static override async DeleteById<T extends BaseModel>(
    this: { new (...args: any[]): T } & typeof BaseModel,
    id: string,
    options?: any
  ): Promise<number> {
    const row = await this.Browse(id, ['Id', 'UserId', 'CreateUid'] as any);
    if (!row) return 0;
    await SavedFilter._assertCanMutate(row as any);
    return await super.DeleteById(id, options as any);
  }
}
