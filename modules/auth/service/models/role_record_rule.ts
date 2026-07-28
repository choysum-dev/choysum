// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { BaseModel, Model, Field } from '@/core/service';
import type { Insertable, Updateable } from '@/core/service/api/input';
import type { FieldSelection } from '@/core/service/api/selection';
import type { QueryCondition } from '@/core/service/api/query';
import { _lt } from '../i18n';
import Role from './role';
import { normalizeRefId } from '@/core/service/utils/normalization';
import { mutateThenInvalidateAllAuthzCaches } from './_authz_mutation_helpers';
import { assertExclusiveScope } from './_rule_scope_helpers';

/**
 * RecordRule merge operator (Security Algebra §5.4).
 *
 * - grant: matching conditions OR together (open domain)
 * - restrict: matching conditions AND onto the grant domain
 */
export type RoleRecordRuleKind = 'grant' | 'restrict';

/**
 * RoleRecordRule stores record-level condition filters and CRUD permission
 * overrides for a role at global, application, or model scope.
 *
 * Audience axis: `RoleId` null means the rule applies to everyone; non-null
 * means it applies to that role only. This is orthogonal to Kind and to
 * model/app/global scope.
 */
@Model('RoleRecordRule')
export default class RoleRecordRule extends BaseModel {
  /**
   * Role that owns this record-rule entry.
   *
   * Null = applies to all users (audience=everyone). Prefer concrete RoleId
   * for grant packs; null + Kind=grant is allowed but warned.
   */
  @Field({
    type: 'ManyToOne',
    relation: { targetModel: () => Role },
    notNull: false,
    string: _lt('Role', { scope: 'auth.model.RoleRecordRule.fields' }),
  })
  RoleId?: Role; // notNull:false; null/omitted = everyone (avoid `Role | null` — metadata FK parser rejects unions)


  /**
   * Merge operator: grant (OR) or restrict (AND). Default grant for legacy rows.
   */
  @Field({
    type: 'selection',
    selection: [
      { value: 'grant', label: _lt('Grant', { scope: 'auth.model.RoleRecordRule.fields' }) },
      { value: 'restrict', label: _lt('Restrict', { scope: 'auth.model.RoleRecordRule.fields' }) },
    ],
    default: () => 'grant',
    size: 16,
    index: true,
    string: _lt('Kind', { scope: 'auth.model.RoleRecordRule.fields' }),
  })
  Kind: RoleRecordRuleKind;

  /**
   * Application-level scope when the rule targets an entire application.
   */
  @Field({
    type: 'ManyToOneRef',
    relation: { targetModel: 'meta.IrApplication' },
    notNull: false,
    size: 20,
    index: true,
    string: _lt('Application', { scope: 'auth.model.RoleRecordRule.fields' }),
  })
  IrApplicationId: string | null;

  /**
   * Model-level scope when the rule targets one concrete model.
   */
  @Field({
    type: 'ManyToOneRef',
    relation: { targetModel: 'meta.IrModel' },
    notNull: false,
    size: 20,
    index: true,
    checkConstraint: `(
        (deleted_at IS NOT NULL)
        OR (ir_model_id IS NOT NULL AND ir_application_id IS NULL)
        OR (ir_model_id IS NULL AND ir_application_id IS NOT NULL)
        OR (ir_model_id IS NULL AND ir_application_id IS NULL)
      )`,
    string: _lt('Model', { scope: 'auth.model.RoleRecordRule.fields' }),
  })
  IrModelId: string | null;

  /**
   * Condition envelope applied to matching records.
   */
  @Field({
    type: 'jsonobject',
    notNull: false,
    string: _lt('Filter Condition', { scope: 'auth.model.RoleRecordRule.fields' }),
  })
  Condition: QueryCondition<any>;

  /**
   * Whether reads are allowed when this rule matches.
   */
  @Field({
    type: 'boolean',
    default: () => false,
    string: _lt('Read', { scope: 'auth.model.RoleRecordRule.fields' }),
  })
  PermRead: boolean;

  /**
   * Whether writes are allowed when this rule matches.
   */
  @Field({
    type: 'boolean',
    default: () => false,
    string: _lt('Write', { scope: 'auth.model.RoleRecordRule.fields' }),
  })
  PermWrite: boolean;

  /**
   * Whether creates are allowed when this rule matches.
   */
  @Field({
    type: 'boolean',
    default: () => false,
    string: _lt('Create', { scope: 'auth.model.RoleRecordRule.fields' }),
  })
  PermCreate: boolean;

  /**
   * Whether deletes are allowed when this rule matches.
   */
  @Field({
    type: 'boolean',
    default: () => false,
    string: _lt('Delete', { scope: 'auth.model.RoleRecordRule.fields' }),
  })
  PermDelete: boolean;

  /**
   * Normalize Kind and reject unsupported values.
   */
  private static _normalizeKind(v: any): RoleRecordRuleKind {
    const kind = String(v ?? 'grant')
      .trim()
      .toLowerCase();
    if (kind === 'grant' || kind === 'restrict') return kind;
    throw new Error("invalid RoleRecordRule Kind: must be 'grant' or 'restrict'");
  }

  /**
   * Normalize and validate Kind on create/update.
   *
   * On create, if Kind is omitted, leave it unset so the Field `default: () => 'grant'`
   * applies at persistence time (keeps schema default reachable for coverage/runtime).
   */
  private static _validateKind(values: Record<string, any>, mode: 'create' | 'update'): void {
    const touchesKind = Object.prototype.hasOwnProperty.call(values, 'Kind');
    if (!touchesKind) return;
    (values as any).Kind = this._normalizeKind((values as any).Kind);
  }

  /**
   * Normalize RoleId when present (empty / blank → null = everyone).
   */
  private static _normalizeRoleId(values: Record<string, any>, _mode: 'create' | 'update'): void {
    const touchesRole = Object.prototype.hasOwnProperty.call(values, 'RoleId');
    if (!touchesRole) return;

    const raw = (values as any).RoleId;
    if (raw == null || raw === '') {
      (values as any).RoleId = null;
      return;
    }
    if (typeof raw === 'object') {
      const id = normalizeRefId(raw);
      (values as any).RoleId = id ? { Id: id } : null;
      return;
    }
    // normalizeRefId trims whitespace; blank strings become null (everyone).
    const id = normalizeRefId(raw);
    (values as any).RoleId = id;
  }

  /**
   * Warn when Kind=grant applies to everyone (RoleId null) — open-for-all risk.
   *
   * Limitation: this helper only sees the mutation payload. Partial updates that
   * omit Kind or RoleId do not load the persisted row, so e.g. clearing RoleId
   * while Kind stays grant (or setting Kind=grant while RoleId stays null) may
   * not warn. Full create paths and updates that send both fields are covered.
   */
  private static _warnGrantForEveryone(values: Record<string, any>, mode: 'create' | 'update'): void {
    const kind = String((values as any).Kind ?? (mode === 'create' ? 'grant' : ''))
      .trim()
      .toLowerCase();
    if (kind !== 'grant') return;

    const touchesRole = Object.prototype.hasOwnProperty.call(values, 'RoleId');
    if (mode === 'update' && !touchesRole) return;

    const roleId = normalizeRefId((values as any).RoleId);
    // Create without RoleId (or explicit null) ⇒ everyone; update clearing RoleId ⇒ everyone.
    const isEveryone = !roleId;
    if (!isEveryone) return;

    console.warn(
      'RoleRecordRule: Kind=grant with RoleId=null applies to all users (wide-open grant); prefer attaching grants to a concrete role'
    );
  }

  /**
   * Run scope / Kind / RoleId validation before mutating RoleRecordRule rows.
   */
  private static _prepareValues(values: Record<string, any>, mode: 'create' | 'update'): void {
    assertExclusiveScope(values, mode, 'record');
    this._validateKind(values, mode);
    this._normalizeRoleId(values, mode);
    this._warnGrantForEveryone(values, mode);
  }

  /**
   * Create one RoleRecordRule row and invalidate request-scoped auth caches.
   */
  static override async Create<T extends BaseModel>(
    this: { new (...args: any[]): T } & typeof BaseModel,
    value: Partial<Insertable<T & BaseModel>>,
    returnFields?: FieldSelection<T>
  ): Promise<T> {
    RoleRecordRule._prepareValues(value as any, 'create');
    return mutateThenInvalidateAllAuthzCaches(async () => {
      const out = await super.Create(value as any, returnFields as any);
      return out as unknown as T;
    });
  }

  /**
   * Create multiple RoleRecordRule rows and invalidate request-scoped auth caches.
   */
  static override async CreateMany<T extends BaseModel>(
    this: { new (...args: any[]): T } & typeof BaseModel,
    values: Partial<Insertable<T & BaseModel>>[],
    returnFields?: FieldSelection<T>
  ): Promise<T[]> {
    const rows = values || [];
    for (const v of rows) RoleRecordRule._prepareValues(v as any, 'create');
    return mutateThenInvalidateAllAuthzCaches(async () => {
      const out = await super.CreateMany(rows as any, returnFields as any);
      return out as unknown as T[];
    });
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
    RoleRecordRule._prepareValues(values as any, 'update');
    return mutateThenInvalidateAllAuthzCaches(async () => {
      const out = await super.Update(condition as any, values as any, returnFields as any, options as any);
      return out as unknown as Partial<T>[];
    });
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
    RoleRecordRule._prepareValues(values as any, 'update');
    return mutateThenInvalidateAllAuthzCaches(async () => {
      const out = await super.UpdateById(id as any, values as any, returnFields as any, options as any);
      return out as unknown as Partial<T>;
    });
  }

  /**
   * Delete matching RoleRecordRule rows and invalidate request-scoped auth caches.
   */
  static override async Delete<T extends BaseModel>(
    this: { new (...args: any[]): T } & typeof BaseModel,
    condition: QueryCondition<T>,
    options?: any
  ): Promise<number> {
    return mutateThenInvalidateAllAuthzCaches(() => super.Delete(condition as any, options as any));
  }

  /**
   * Delete one RoleRecordRule row by Id and invalidate request-scoped auth caches.
   */
  static override async DeleteById<T extends BaseModel>(this: { new (...args: any[]): T } & typeof BaseModel, id: string, options?: any): Promise<number> {
    return mutateThenInvalidateAllAuthzCaches(() => super.DeleteById(id as any, options as any));
  }
}
