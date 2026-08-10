// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { BaseModel, Model, Field } from '@/core/service';
import { Constraint, type ConstraintContext } from '@/core/service/api/constraint';
import type { QueryCondition } from '@/core/service/api/query';
import { ChoysumError, GrpcCode } from '@/core/service/error';
import { createServiceByModel } from '@/core/service/rpc';
import type MetaModel from '@/meta/service/models/model';
import { _lt, _t } from '../i18n';
import { normalizeScopeKey } from './_scope_key';

const SCOPE = 'web.model.SavedFilter';
const MetaModelService = createServiceByModel<typeof MetaModel>('meta.MetaModel');

/**
 * Persisted Favorites filter for OSearch (Owner Application = web).
 *
 * Identity: Application + ModelName. ModelId stores the unique live meta.MetaModel id.
 * IsDefault mutex is scoped by ScopeKey (normalized route path) + UserId.
 * Name is not unique (align modern Odoo ir.filters); Favorites distinguish rows by Id.
 * Field normalize / ModelId / IsDefault mutex → `@Constraint` (uses ctx.mode;
 * Create pre-assigns Id before validation, so `!this.Id` is not a reliable create signal).
 * Shared write/delete ACL (SF11) → auth.RoleRecordRule seeds in modules/web/data/bootstrap.json.
 */
@Model('SavedFilter', { application: 'web', softDelete: false })
export default class SavedFilter extends BaseModel {
  /**
   * Display name of the favorite (not unique; duplicates allowed in the same scope/owner).
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
   * Normalized route path that scopes IsDefault mutex (not shown in Favorites UI).
   */
  @Field({
    type: 'varchar',
    size: 512,
    notNull: true,
    index: true,
    default: () => '',
    string: _lt('Scope Key', { scope: `${SCOPE}.fields` }),
    help: _lt('Normalized route path; Favorites UI shows Name only.', { scope: `${SCOPE}.fields` }),
  })
  ScopeKey: string;

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

  private static _fail(code: string, message: string, grpc: GrpcCode = GrpcCode.InvalidArgument): never {
    throw new ChoysumError({ domain: 'web', code, message }).withGrpcCode(grpc);
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
   * Clear other IsDefault rows the actor may write (SF11). Shared defaults owned by
   * someone else remain; fail so we never leave two shared defaults silently.
   */
  private static async _clearOtherDefaults(
    app: string,
    modelName: string,
    scopeKey: string,
    userId: string | null,
    exceptId?: string
  ): Promise<void> {
    const cond: any = {
      And: [
        ['Application', '=', app],
        ['ModelName', '=', modelName],
        ['ScopeKey', '=', scopeKey],
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
    // Preflight under sudo so a foreign shared default surfaces PermissionDenied
    // instead of a generic record_rule_violation from Update.
    const actor = String(this.userId || '').trim();
    const candidates = await SavedFilter.sudo(
      () => SavedFilter.Search(cond as any, { fields: ['Id', 'UserId', 'CreatedUid'], limit: 50 } as any),
      { hint: 'web.SavedFilter.clearOtherDefaults.preflight' }
    );
    for (const row of candidates || []) {
      const shared = (row as any).UserId == null || (row as any).UserId === '';
      const canWrite = shared
        ? String((row as any).CreatedUid || '').trim() === actor
        : String((row as any).UserId || '').trim() === actor;
      if (!canWrite) {
        this._fail(
          'PermissionDenied',
          _t('Cannot replace another user\'s shared default favorite', { scope: SCOPE }),
          GrpcCode.PermissionDenied
        );
      }
    }
    // No sudo: Record rules must still authorize the clear for rows we expect to write.
    await SavedFilter.Update(cond as any, { IsDefault: false } as any, ['Id'] as any);
    const remaining = await SavedFilter.sudo(
      () => SavedFilter.Search(cond as any, { fields: ['Id'], limit: 1 } as any),
      { hint: 'web.SavedFilter.clearOtherDefaults.check' }
    );
    if (Array.isArray(remaining) && remaining.length > 0) {
      this._fail(
        'PermissionDenied',
        _t('Cannot replace another user\'s shared default favorite', { scope: SCOPE }),
        GrpcCode.PermissionDenied
      );
    }
  }

  private static _mergedField(self: SavedFilter, ctx: ConstraintContext<SavedFilter>, key: string): any {
    if (Object.prototype.hasOwnProperty.call(ctx.values || {}, key)) return (ctx.values as any)[key];
    if (Object.prototype.hasOwnProperty.call(self as any, key)) return (self as any)[key];
    return (ctx.current as any)?.[key];
  }

  /**
   * Normalize ownership, resolve ModelId, and enforce IsDefault mutex.
   * Name is not unique (modern Odoo-aligned). Login is enforced by gRPC AuthN + Method ACL.
   * Static so we can read `ctx.mode` (Create pre-assigns Id before validation).
   */
  @Constraint<SavedFilter>(['Name', 'ScopeKey', 'Application', 'ModelName', 'ModelId', 'UserId', 'IsDefault', 'Condition', 'Active'])
  static async validateSavedFilterConstraint(self: SavedFilter, ctx: ConstraintContext<SavedFilter>): Promise<void> {
    const isCreate = ctx.mode === 'create';
    const values = ctx.values as Record<string, any>;
    const currentId = String((isCreate ? values.Id : SavedFilter._mergedField(self, ctx, 'Id')) || '').trim() || undefined;
    const actor = String(this.userId || '').trim();

    const app = String(SavedFilter._mergedField(self, ctx, 'Application') || '').trim();
    const modelName = String(SavedFilter._mergedField(self, ctx, 'ModelName') || '').trim();
    const name = String(SavedFilter._mergedField(self, ctx, 'Name') || '').trim();
    if (!app || !modelName || !name) {
      SavedFilter._fail('InvalidArgument', _t('Name, Application, and ModelName are required', { scope: SCOPE }));
    }
    values.Application = app;
    values.ModelName = modelName;
    values.Name = name;
    values.ScopeKey = normalizeScopeKey(SavedFilter._mergedField(self, ctx, 'ScopeKey'));

    const modelRows = await MetaModelService.Search(
      { And: [['Application', '=', app], ['Name', '=', modelName]] } as any,
      { fields: ['Id'], limit: 1 } as any
    );
    const modelId = String(modelRows?.[0]?.Id || '').trim();
    if (!modelId) {
      SavedFilter._fail(
        'FailedPrecondition',
        _t('No effective model found for %s.%s', { scope: SCOPE }, app, modelName),
        GrpcCode.FailedPrecondition
      );
    }
    values.ModelId = modelId;

    if (isCreate) {
      // CreatedUid is stamped by repository write prepare (AuditUidUtils) from the request actor.
      const touchedUserId = Object.prototype.hasOwnProperty.call(values, 'UserId');
      if (!touchedUserId) {
        // Only setdefault private ownership when an actor is present; empty actor leaves shared (null).
        if (actor) values.UserId = actor;
      } else {
        values.UserId = SavedFilter._normalizeOwnerUserId(values.UserId, actor);
      }
      if (values.IsDefault == null) values.IsDefault = false;
      if (values.Active == null) values.Active = true;
      if (values.Condition == null) values.Condition = {};
    } else if (Object.prototype.hasOwnProperty.call(values, 'UserId')) {
      values.UserId = SavedFilter._normalizeOwnerUserId(values.UserId, actor);
    }

    const effectiveUserId = (() => {
      const raw = Object.prototype.hasOwnProperty.call(values, 'UserId')
        ? values.UserId
        : SavedFilter._mergedField(self, ctx, 'UserId');
      if (raw == null || raw === '') return null;
      return String(raw).trim();
    })();

    const isDefault = Object.prototype.hasOwnProperty.call(values, 'IsDefault')
      ? values.IsDefault === true
      : SavedFilter._mergedField(self, ctx, 'IsDefault') === true;
    if (isDefault) {
      await SavedFilter._clearOtherDefaults(app, modelName, values.ScopeKey, effectiveUserId, isCreate ? undefined : currentId);
    }
  }
}
