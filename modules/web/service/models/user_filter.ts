// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { BaseModel, Model, Field } from '@/core/service';
import { Constraint, type ConstraintContext } from '@/core/service/api/constraint';
import type { QueryCondition } from '@/core/service/api/query';
import { ChoysumError, GrpcCode } from '@/core/service/error';
import { createTranslate } from '@/core/service/i18n';

const { _lt } = createTranslate('web', { scope: 'web.model.UserFilter.fields' });
const { _t } = createTranslate('web', { scope: 'web.model.UserFilter' });

/**
 * Persisted Favorites filter for OSearch (Owner Application = web).
 *
 * Align modern Odoo ir.filters: Name is not unique; multiple IsDefault rows are
 * allowed (FE picks newest). ScopeKey is stored as written (FE normalizes routes).
 * UserId default comes from Field metadata; Constraint only rejects foreign owners.
 * Shared write/delete ACL → auth.RoleRecordRule seeds in modules/web/data/bootstrap.json.
 */
@Model('UserFilter', { application: 'web', softDelete: false })
export default class UserFilter extends BaseModel {
  /**
   * Display name of the favorite (not unique; duplicates allowed in the same scope/owner).
   */
  @Field({
    type: 'varchar',
    size: 255,
    notNull: true,
    index: true,
    string: _lt('Name'),
  })
  Name: string;

  /**
   * Route path that scopes Favorites lists (not shown in Favorites UI).
   * Stored as written; FE normalizes before write/query.
   */
  @Field({
    type: 'varchar',
    size: 512,
    notNull: true,
    index: true,
    default: () => '',
    string: _lt('Scope Key'),
    help: _lt('Normalized route path; Favorites UI shows Name only.'),
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
    string: _lt('Application'),
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
    string: _lt('Model Name'),
  })
  ModelName: string;

  /**
   * QueryCondition JSON applied via store.Search (Choysum Condition, not Odoo domain).
   */
  @Field({
    type: 'jsonobject',
    notNull: true,
    default: () => ({}),
    string: _lt('Condition'),
  })
  Condition: QueryCondition<any>;

  /**
   * Owner user; null means shared with all logged-in users.
   * Create omit → current actor (Field default); explicit null stays shared.
   */
  @Field({
    type: 'ManyToOneRef',
    relation: { targetModel: 'auth.User' },
    notNull: false,
    size: 20,
    index: true,
    default: () => UserFilter.userId || null,
    string: _lt('User'),
  })
  UserId?: string | null;

  /**
   * Whether this favorite is a candidate default when the search view opens.
   * Multiple defaults are allowed; FE picks the newest (Odoo-aligned).
   */
  @Field({
    type: 'boolean',
    default: () => false,
    string: _lt('Is Default'),
  })
  IsDefault: boolean;

  /**
   * Reject foreign UserId owners. Does not assign (defaults → Field `default`).
   * Allowed: null (shared), undefined (omitted / untouched), or the current actor.
   */
  @Constraint<UserFilter>(['UserId'])
  static async validateUserFilterConstraint(_self: UserFilter, ctx: ConstraintContext<UserFilter>): Promise<void> {
    const userId = ctx.values.UserId;
    if (userId !== null && userId !== undefined && userId !== this.userId) {
      throw new ChoysumError({
        domain: 'web',
        code: 'PermissionDenied',
        message: _t('Cannot assign a favorite to another user'),
      }).withGrpcCode(GrpcCode.PermissionDenied);
    }
  }
}
