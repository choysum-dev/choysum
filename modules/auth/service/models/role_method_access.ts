// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { BaseModel, Model, Field } from '@/core/service';
import { Onchange } from '@/core/service/api/onchange';
import type { Insertable, Updateable } from '@/core/service/api/input';
import type { FieldSelection } from '@/core/service/api/selection';
import type { QueryCondition } from '@/core/service/api/query';
import { _lt } from '../i18n';
import Role from './role';
import type MetaApplication from '@/meta/service/models/application';
import type MetaModel from '@/meta/service/models/model';
import type MetaService from '@/meta/service/models/service';
import AuthzMutationModel from '../mixins/authz_mutation_model';
import { listLogicalModelSelection, assertLogicalMethods } from './_logical_model_registry';
import { assertExclusiveScope } from './_rule_scope_helpers';

/**
 * RoleMethodAccess stores role-level RPC allow and deny overrides at global,
 * application, model, service, or logical-model scope.
 */
@Model('RoleMethodAccess')
export default class RoleMethodAccess extends AuthzMutationModel {
  /**
   * Role that owns this method-access entry.
   */
  @Field({
    type: 'ManyToOne',
    relation: { targetModel: () => Role },
    string: _lt('Role', { scope: 'auth.model.RoleMethodAccess.fields' }),
  })
  RoleId: Role;

  /**
   * Application-level scope when the entry targets an entire application.
   */
  @Field<MetaApplication>({
    type: 'ManyToOneRef',
    relation: { targetModel: 'meta.MetaApplication' },
    notNull: false,
    size: 20,
    index: true,
    string: _lt('Application', { scope: 'auth.model.RoleMethodAccess.fields' }),
  })
  MetaApplicationId: string | null;

  /**
   * Model-level scope when the entry targets an entire model.
   */
  @Field<MetaModel>({
    type: 'ManyToOneRef',
    relation: { targetModel: 'meta.MetaModel' },
    notNull: false,
    size: 20,
    index: true,
    string: _lt('Model', { scope: 'auth.model.RoleMethodAccess.fields' }),
  })
  MetaModelId: string | null;

  /**
   * Service-level scope when the entry targets one concrete RPC method surface.
   */
  @Field<MetaService>({
    type: 'ManyToOneRef', relation: { targetModel: 'meta.MetaService' },
    notNull: false,
      size: 20,
      index: true,
      checkConstraint: `(
        (deleted_at IS NOT NULL)
        OR (meta_service_id IS NOT NULL AND meta_model_id IS NULL AND meta_application_id IS NULL AND logical_model_name IS NULL)
        OR (meta_service_id IS NULL AND meta_model_id IS NOT NULL AND meta_application_id IS NULL AND logical_model_name IS NULL)
        OR (meta_service_id IS NULL AND meta_model_id IS NULL AND meta_application_id IS NOT NULL AND logical_model_name IS NULL)
        OR (meta_service_id IS NULL AND meta_model_id IS NULL AND meta_application_id IS NULL AND logical_model_name IS NOT NULL)
        OR (meta_service_id IS NULL AND meta_model_id IS NULL AND meta_application_id IS NULL AND logical_model_name IS NULL)
      )`,
    string: _lt('Service', { scope: 'auth.model.RoleMethodAccess.fields' }),
    help: _lt('Leave all scopes empty for a global rule; matching scopes are OR-ed and any deny wins.', {
      scope: 'auth.model.RoleMethodAccess.fields',
    }),
  })
  MetaServiceId: string | null;

  /**
   * Logical model short name (@Model name without application prefix).
   * Mutually exclusive with MetaServiceId / MetaModelId / MetaApplicationId.
   * Options come from core platform-inject self-registration.
   */
  @Field({
    type: 'selection',
    selection: () => listLogicalModelSelection(),
    notNull: false,
    size: 128,
    index: true,
    string: _lt('Logical Model', { scope: 'auth.model.RoleMethodAccess.fields' }),
    help: _lt(
      'Registered short name covering all installed host applications that inject the same model (e.g. TranslationTerm). Mutually exclusive with Application / Model / Service.',
      { scope: 'auth.model.RoleMethodAccess.fields' }
    ),
  })
  LogicalModelName: string | null;

  /**
   * Optional RPC short-name whitelist for LogicalModel scope.
   * Null or empty = all methods on that logical model.
   */
  @Field({
    type: 'jsonobject',
    notNull: false,
    string: _lt('Logical Methods', { scope: 'auth.model.RoleMethodAccess.fields' }),
    help: _lt(
      'Optional method names for logical-model scope (JSON string array). Leave empty to allow or deny all methods on that logical model.',
      { scope: 'auth.model.RoleMethodAccess.fields' }
    ),
  })
  LogicalMethods: string[] | null;

  /**
   * Whether the matched scope is allowed or denied.
   */
  @Field({
    type: 'selection',
    selection: [
      { value: 'allow', label: _lt('Allow', { scope: 'auth.model.RoleMethodAccess.fields' }) },
      { value: 'deny', label: _lt('Deny', { scope: 'auth.model.RoleMethodAccess.fields' }) },
    ],
    default: () => 'deny',
    string: _lt('Mode', { scope: 'auth.model.RoleMethodAccess.fields' }),
    help: _lt('Any matching deny overrides allow across scopes; default is deny.', {
      scope: 'auth.model.RoleMethodAccess.fields',
    }),
  })
  Mode: 'allow' | 'deny';

  /**
   * Source of the method-access entry.
   *
   * UI-Option-A: authority lives in RoleUiResource; Method rows are manual-only.
   * Legacy `ui` values may still exist in DB but are ignored by ACL evaluators and
   * are coerced to `manual` on write (PR-E-4).
   */
  @Field({
    type: 'selection',
    selection: [
      { value: 'manual', label: _lt('Manual', { scope: 'auth.model.RoleMethodAccess.fields' }) },
      { value: 'ui', label: _lt('UI (legacy)', { scope: 'auth.model.RoleMethodAccess.fields' }) },
    ],
    default: () => 'manual',
    size: 16,
    index: true,
    string: _lt('Source', { scope: 'auth.model.RoleMethodAccess.fields' }),
  })
  Source: 'manual' | 'ui';

  /**
   * UI-Option-A: never persist Source=ui (runtime ui-derived ACL replaces materialization).
   */
  private static _coerceSourceManual(values: Record<string, any>, _mode: 'create' | 'update'): void {
    if (!values) return;
    // Only coerce when Source is present in the payload (ui → manual). Create without Source
    // keeps the Field default factory (`manual`) so defaults stay reachable.
    if (!Object.prototype.hasOwnProperty.call(values, 'Source')) return;
    (values as any).Source = 'manual';
  }

  /**
   * Assert LogicalMethods; clear them unless scope is LogicalModel.
   * Call after assertExclusiveScope so LogicalModelName is already asserted.
   *
   * @param previousLogicalModelName Persisted LogicalModelName for update rename checks.
   *   When omitted on an update that touches LogicalModelName without LogicalMethods, treat as a rename.
   */
  private static _assertLogicalMethodsPayload(
    values: Record<string, any>,
    mode: 'create' | 'update',
    previousLogicalModelName?: string | null
  ): void {
    if (!values) return;
    const touchesMethods = Object.prototype.hasOwnProperty.call(values, 'LogicalMethods');
    const touchesLogicalName = Object.prototype.hasOwnProperty.call(values, 'LogicalModelName');

    // Partial update that does not touch scope/methods: leave LogicalMethods alone.
    if (mode === 'update' && !touchesMethods && !touchesLogicalName) return;

    if (touchesMethods) {
      (values as any).LogicalMethods = assertLogicalMethods((values as any).LogicalMethods);
    } else if (mode === 'create') {
      (values as any).LogicalMethods = assertLogicalMethods((values as any).LogicalMethods);
    }

    // When LogicalModelName is present in the payload (create always after assert, or update touching scope),
    // clear methods unless this row is logical scope.
    if (mode === 'create' || touchesLogicalName) {
      const name = String((values as any).LogicalModelName || '').trim();
      if (!name) {
        if ((values as any).LogicalMethods != null) {
          throw new Error('invalid RoleMethodAccess: LogicalMethods requires LogicalModel scope');
        }
        (values as any).LogicalMethods = null;
      } else if (mode === 'update' && !touchesMethods) {
        const previous = String(previousLogicalModelName ?? '').trim();
        // Re-echoing the same logical name (e.g. Mode toggle with full scope payload) is fine.
        // A real rename without an explicit whitelist must fail closed — null would mean all methods.
        if (previous !== name) {
          throw new Error('invalid RoleMethodAccess: LogicalMethods must be provided when LogicalModelName is updated');
        }
      }
    }
    // Methods-only update (no LogicalModelName in payload): normalize and persist.
    // Eval ignores LogicalMethods unless the persisted row is Logical scope.
  }

  private static _prepareValues(
    values: Record<string, any>,
    mode: 'create' | 'update',
    previousLogicalModelName?: string | null
  ): void {
    assertExclusiveScope(values, mode, 'method');
    RoleMethodAccess._assertLogicalMethodsPayload(values, mode, previousLogicalModelName);
    RoleMethodAccess._coerceSourceManual(values, mode);
  }

  /**
   * Whether this update payload needs the persisted LogicalModelName for rename checks.
   */
  private static _needsPreviousLogicalModelName(values: Record<string, any>): boolean {
    if (!values) return false;
    const touchesMethods = Object.prototype.hasOwnProperty.call(values, 'LogicalMethods');
    const touchesLogicalName = Object.prototype.hasOwnProperty.call(values, 'LogicalModelName');
    if (!touchesLogicalName || touchesMethods) return false;
    return Boolean(String((values as any).LogicalModelName || '').trim());
  }

  /**
   * Create one RoleMethodAccess row and invalidate request-scoped auth caches.
   */
  static override async Create<T extends BaseModel>(
    this: { new (...args: any[]): T } & typeof BaseModel,
    value: Partial<Insertable<T & BaseModel>>,
    returnFields?: FieldSelection<T>
  ): Promise<T> {
    RoleMethodAccess._prepareValues(value as any, 'create');
    return (await super.Create(value as any, returnFields as any)) as unknown as T;
  }

  /**
   * Create multiple RoleMethodAccess rows and invalidate request-scoped auth caches.
   */
  static override async CreateMany<T extends BaseModel>(
    this: { new (...args: any[]): T } & typeof BaseModel,
    values: Partial<Insertable<T & BaseModel>>[],
    returnFields?: FieldSelection<T>
  ): Promise<T[]> {
    const rows = values || [];
    for (const v of rows) {
      RoleMethodAccess._prepareValues(v as any, 'create');
    }
    return (await super.CreateMany(rows as any, returnFields as any)) as unknown as T[];
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
    let previousLogicalModelName: string | null | undefined;
    let updateCondition: QueryCondition<T> = condition;
    if (RoleMethodAccess._needsPreviousLogicalModelName(values as any)) {
      // Guard already proved LogicalModelName is a non-empty string after trim.
      const next = String((values as any).LogicalModelName).trim();
      // Prove every matched row already has LogicalModelName === next (no sampling).
      // Null/empty/other names fail Count equality → fail closed (null whitelist = all methods).
      // Pass the same options as super.Update so withDeleted/onlyDeleted stay aligned.
      const matched = Number(await (this as any).Count(condition as any, options as any)) || 0;
      if (matched > 0) {
        const alreadyAtNext =
          Number(
            await (this as any).Count(
              {
                And: [condition as any, ['LogicalModelName', '=', next] as any],
              } as any,
              options as any
            )
          ) || 0;
        if (alreadyAtNext === matched) {
          previousLogicalModelName = next;
          // Couple the write to the proof: rows that race away from `next` are skipped, not renamed.
          updateCondition = {
            And: [condition as any, ['LogicalModelName', '=', next] as any],
          } as any;
        } else {
          previousLogicalModelName = null;
        }
      } else {
        previousLogicalModelName = null;
      }
    }
    RoleMethodAccess._prepareValues(values as any, 'update', previousLogicalModelName);
    return (await super.Update(updateCondition as any, values as any, returnFields as any, options as any)) as unknown as Partial<T>[];
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
    let previousLogicalModelName: string | null | undefined;
    if (RoleMethodAccess._needsPreviousLogicalModelName(values as any)) {
      const existing = await (this as any).Search(['Id', '=', id] as any, {
        fields: ['LogicalModelName'],
        limit: 1,
      } as any);
      previousLogicalModelName = String((existing?.[0] as any)?.LogicalModelName || '').trim() || null;
    }
    RoleMethodAccess._prepareValues(values as any, 'update', previousLogicalModelName);
    return (await super.UpdateById(id as any, values as any, returnFields as any, options as any)) as unknown as Partial<T>;
  }

  /**
   * Selecting a logical model clears Meta* scopes (exclusive shapes).
   */
  @Onchange<RoleMethodAccess>('LogicalModelName')
  OnchangeLogicalModelName() {
    const name = String(this.LogicalModelName || '').trim();
    if (name) {
      this.MetaServiceId = null as any;
      this.MetaModelId = null as any;
      this.MetaApplicationId = null as any;
    } else if (this.LogicalMethods != null) {
      this.LogicalMethods = null as any;
    }
  }

  /**
   * Selecting a concrete Meta scope clears logical-model fields.
   */
  @Onchange<RoleMethodAccess>('MetaServiceId', 'MetaModelId', 'MetaApplicationId')
  OnchangeMetaScopeClearsLogical() {
    const hasMeta =
      Boolean(String(this.MetaServiceId || '').trim()) ||
      Boolean(String(this.MetaModelId || '').trim()) ||
      Boolean(String(this.MetaApplicationId || '').trim());
    if (!hasMeta) return;
    if (String(this.LogicalModelName || '').trim()) this.LogicalModelName = null as any;
    if (this.LogicalMethods != null) this.LogicalMethods = null as any;
  }
}
