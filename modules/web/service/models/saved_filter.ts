// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { BaseModel, Model, Field } from '@/core/service';
import type { Insertable, Updateable } from '@/core/service/api/input';
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
 * Shared rows use UserId = null; SF11 write/delete requires CreateUid == me or sys.admin.
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
   */
  @Field<MetaModel>({
    type: 'ManyToOneRef',
    relation: { targetModel: 'meta.MetaModel' },
    notNull: true,
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
   */
  @Field({
    type: 'varchar',
    size: 20,
    notNull: true,
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

  private static async _assertCanMutateShared(row: { UserId?: string | null; CreateUid?: string }): Promise<void> {
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
    await (super.Update as any).call(this, cond, { IsDefault: false } as any, ['Id'] as any);
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
    const hits = await this.Search(cond as any, { fields: ['Id'], limit: 1 } as any);
    if (Array.isArray(hits) && hits.length > 0) {
      this._fail('AlreadyExists', _t('A favorite with this name already exists', { scope: SCOPE }), GrpcCode.AlreadyExists);
    }
  }

  private static async _prepareCreate(values: Record<string, any>): Promise<void> {
    const actor = this._actorId();
    if (!actor) {
      this._fail('PermissionDenied', _t('Authentication required', { scope: SCOPE }), GrpcCode.Unauthenticated);
    }

    const app = String(values.Application || '').trim();
    const modelName = String(values.ModelName || '').trim();
    const name = String(values.Name || '').trim();
    if (!app || !modelName || !name) {
      this._fail('InvalidArgument', _t('Name, Application, and ModelName are required', { scope: SCOPE }));
    }
    values.Application = app;
    values.ModelName = modelName;
    values.Name = name;

    const effectiveId = await resolveEffectiveModelId(app, modelName);
    if (!effectiveId) {
      this._fail(
        'FailedPrecondition',
        _t('No effective model found for %s.%s', { scope: SCOPE }, app, modelName),
        GrpcCode.FailedPrecondition
      );
    }
    values.ModelId = effectiveId;

    if (Object.prototype.hasOwnProperty.call(values, 'UserId')) {
      const raw = values.UserId;
      if (raw == null || raw === '') {
        values.UserId = null;
      } else {
        values.UserId = String(raw).trim();
      }
    } else {
      values.UserId = actor;
    }

    values.CreateUid = actor;
    if (values.IsDefault == null) values.IsDefault = false;
    if (values.Active == null) values.Active = true;
    if (values.Condition == null) values.Condition = {};

    await this._assertUniqueName(app, modelName, values.UserId ?? null, name);

    if (values.IsDefault) {
      await this._clearOtherDefaults(app, modelName, values.UserId ?? null);
    }
  }

  private static async _prepareUpdate(id: string, values: Record<string, any>): Promise<void> {
    const row = await this.Browse(id, ['Id', 'Application', 'ModelName', 'Name', 'UserId', 'CreateUid', 'IsDefault'] as any);
    if (!row) {
      this._fail('NotFound', _t('Favorite not found', { scope: SCOPE }), GrpcCode.NotFound);
    }
    await this._assertCanMutateShared(row as any);

    // CreateUid is immutable.
    if (Object.prototype.hasOwnProperty.call(values, 'CreateUid')) {
      delete values.CreateUid;
    }

    const app = Object.prototype.hasOwnProperty.call(values, 'Application')
      ? String(values.Application || '').trim()
      : String((row as any).Application || '').trim();
    const modelName = Object.prototype.hasOwnProperty.call(values, 'ModelName')
      ? String(values.ModelName || '').trim()
      : String((row as any).ModelName || '').trim();
    if (!app || !modelName) {
      this._fail('InvalidArgument', _t('Application and ModelName are required', { scope: SCOPE }));
    }
    if (Object.prototype.hasOwnProperty.call(values, 'Application')) values.Application = app;
    if (Object.prototype.hasOwnProperty.call(values, 'ModelName')) values.ModelName = modelName;

    const touchesIdentity =
      Object.prototype.hasOwnProperty.call(values, 'Application') || Object.prototype.hasOwnProperty.call(values, 'ModelName');
    if (touchesIdentity || Object.prototype.hasOwnProperty.call(values, 'ModelId')) {
      const effectiveId = await resolveEffectiveModelId(app, modelName);
      if (!effectiveId) {
        this._fail(
          'FailedPrecondition',
          _t('No effective model found for %s.%s', { scope: SCOPE }, app, modelName),
          GrpcCode.FailedPrecondition
        );
      }
      values.ModelId = effectiveId;
    }

    let userId: string | null = (row as any).UserId == null || (row as any).UserId === '' ? null : String((row as any).UserId).trim();
    if (Object.prototype.hasOwnProperty.call(values, 'UserId')) {
      const raw = values.UserId;
      userId = raw == null || raw === '' ? null : String(raw).trim();
      values.UserId = userId;
    }

    const name = Object.prototype.hasOwnProperty.call(values, 'Name')
      ? String(values.Name || '').trim()
      : String((row as any).Name || '').trim();
    if (Object.prototype.hasOwnProperty.call(values, 'Name')) {
      if (!name) this._fail('InvalidArgument', _t('Name is required', { scope: SCOPE }));
      values.Name = name;
    }

    const nameTouched =
      Object.prototype.hasOwnProperty.call(values, 'Name') ||
      Object.prototype.hasOwnProperty.call(values, 'UserId') ||
      touchesIdentity;
    if (nameTouched) {
      await this._assertUniqueName(app, modelName, userId, name, String((row as any).Id));
    }

    if (values.IsDefault === true) {
      await this._clearOtherDefaults(app, modelName, userId, String((row as any).Id));
    }
  }

  /**
   * Create a SavedFilter after resolving effective ModelId and enforcing defaults/ACL.
   */
  static override async Create<T extends BaseModel>(
    this: { new (...args: any[]): T } & typeof BaseModel,
    value: Partial<Insertable<T & BaseModel>>,
    returnFields?: FieldSelection<T>
  ): Promise<T> {
    await SavedFilter._prepareCreate(value as any);
    return (await super.Create(value as any, returnFields as any)) as unknown as T;
  }

  /**
   * Create many SavedFilter rows with the same Create validations.
   */
  static override async CreateMany<T extends BaseModel>(
    this: { new (...args: any[]): T } & typeof BaseModel,
    values: Partial<Insertable<T & BaseModel>>[],
    returnFields?: FieldSelection<T>
  ): Promise<T[]> {
    const rows = values || [];
    for (const v of rows) await SavedFilter._prepareCreate(v as any);
    return (await super.CreateMany(rows as any, returnFields as any)) as unknown as T[];
  }

  /**
   * Update SavedFilter rows (SF11 + identity / default exclusivity).
   * Delegates per-row so CreateUid / default exclusivity stay correct.
   */
  static override async Update<T extends BaseModel>(
    this: { new (...args: any[]): T } & typeof BaseModel,
    condition: QueryCondition<T>,
    values: Partial<Updateable<T & BaseModel>>,
    returnFields?: FieldSelection<T>,
    options?: any
  ): Promise<Partial<T>[]> {
    const existing = await this.Search(condition as any, { fields: ['Id'] as any, ...(options || {}) } as any);
    const out: Partial<T>[] = [];
    for (const row of existing || []) {
      const updated = await (this as any).UpdateById(String((row as any).Id), { ...(values as any) }, returnFields as any, options as any);
      out.push(updated as Partial<T>);
    }
    return out;
  }

  /**
   * Update one SavedFilter by id.
   */
  static override async UpdateById<T extends BaseModel>(
    this: { new (...args: any[]): T } & typeof BaseModel,
    id: string,
    values: Partial<Updateable<T & BaseModel>>,
    returnFields?: FieldSelection<T>,
    options?: any
  ): Promise<Partial<T>> {
    const patch = { ...(values as any) };
    await SavedFilter._prepareUpdate(String(id), patch);
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
      await SavedFilter._assertCanMutateShared(row as any);
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
    await SavedFilter._assertCanMutateShared(row as any);
    return await super.DeleteById(id, options as any);
  }
}
