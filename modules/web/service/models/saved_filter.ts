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
 * Name is not unique (align modern Odoo ir.filters). Multiple IsDefault rows are allowed;
 * the client picks the newest (UpdatedAt/Id) per private vs shared bucket.
 * Field normalize / ModelId → `@Constraint` (uses ctx.mode; Create pre-assigns Id before
 * validation, so `!this.Id` is not a reliable create signal).
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
   * Normalized route path that scopes Favorites lists (not shown in Favorites UI).
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
   * Whether this favorite is a candidate default when the search view opens.
   * Multiple defaults are allowed; FE picks the newest (Odoo-aligned).
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

  private static _mergedField(self: SavedFilter, ctx: ConstraintContext<SavedFilter>, key: string): any {
    if (Object.prototype.hasOwnProperty.call(ctx.values || {}, key)) return (ctx.values as any)[key];
    if (Object.prototype.hasOwnProperty.call(self as any, key)) return (self as any)[key];
    return (ctx.current as any)?.[key];
  }

  /**
   * Normalize ownership and resolve ModelId. Name is not unique; IsDefault is not mutexed
   * (FE picks newest). Login is enforced by gRPC AuthN + Method ACL.
   * Static so we can read `ctx.mode` (Create pre-assigns Id before validation).
   */
  @Constraint<SavedFilter>(['Name', 'ScopeKey', 'Application', 'ModelName', 'ModelId', 'UserId', 'IsDefault', 'Condition', 'Active'])
  static async validateSavedFilterConstraint(self: SavedFilter, ctx: ConstraintContext<SavedFilter>): Promise<void> {
    const isCreate = ctx.mode === 'create';
    const values = ctx.values as Record<string, any>;
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
  }
}
