// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { BaseModel, Model, Field } from '@/core/service';
import { Constraint, type ConstraintContext } from '@/core/service/api/constraint';
import type { QueryCondition } from '@/core/service/api/query';
import { ChoysumError, GrpcCode } from '@/core/service/error';
import type MetaModel from '@/meta/service/models/model';
import { _lt, _t } from '../i18n';
import { resolveEffectiveModelId } from './_resolve_effective_model';

const SCOPE = 'web.model.SavedFilter';

/**
 * Persisted Favorites filter for OSearch (Owner Application = web).
 *
 * Identity: Application + ModelName. ModelId stores the effective meta.MetaModel id.
 * Field normalize / ModelId / uniqueness / IsDefault mutex → `@Constraint` (uses ctx.mode;
 * Create pre-assigns Id before validation, so `!this.Id` is not a reliable create signal).
 * Shared write/delete ACL (SF11) → auth.RoleRecordRule seeds in modules/web/data/bootstrap.json.
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
   * Creator user id for SF11 shared-row write/delete Record rules.
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

  /**
   * Owner may be null (shared) or the current actor. Other user ids are rejected.
   */
  private static _normalizeOwnerUserId(raw: unknown, actor: string): string | null {
    if (raw == null || raw === '') return null;
    const id = String(raw).trim();
    if (!id) return null;
    if (id === actor) return actor;
    this._fail('PermissionDenied', _t('Cannot assign a favorite to another user', { scope: SCOPE }), GrpcCode.PermissionDenied);
  }

  /**
   * Clear other IsDefault rows under sudo so Record rules do not block the mutex.
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
    await SavedFilter.sudo(
      () => SavedFilter.Update(cond as any, { IsDefault: false } as any, ['Id'] as any),
      { hint: 'web.SavedFilter.clearOtherDefaults' }
    );
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
      } else {
        values.UserId = SavedFilter._normalizeOwnerUserId(values.UserId, actor);
      }
      if (values.IsDefault == null) values.IsDefault = false;
      if (values.Active == null) values.Active = true;
      if (values.Condition == null) values.Condition = {};
    } else {
      // CreateUid is immutable.
      values.CreateUid = String((ctx.current as any)?.CreateUid || SavedFilter._mergedField(self, ctx, 'CreateUid') || '').trim();
      if (Object.prototype.hasOwnProperty.call(values, 'UserId')) {
        values.UserId = SavedFilter._normalizeOwnerUserId(values.UserId, actor);
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
}
