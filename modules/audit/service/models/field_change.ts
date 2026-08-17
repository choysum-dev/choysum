// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { BaseModel, Field, Model } from '@/core/service';
import { getCurrentReq, getUserId } from '@/core/service/api/context';
import type { Insertable, Updateable } from '@/core/service/api/input';
import type { FieldSelection } from '@/core/service/api/selection';
import type { QueryCondition, DeleteOptions, UpdateOptions } from '@/core/service/api/query';
import { AuditErrCode, newAuditError } from '../error';
import { _lt } from '../i18n';

/** Data-family Kind values allowed on FieldChange (AU6). */
export const FIELD_CHANGE_KINDS = ['field', 'create', 'unlink'] as const;
export type FieldChangeKindLiteral = (typeof FIELD_CHANGE_KINDS)[number];

/**
 * Append payload for FieldChange (Unary Append API).
 *
 * ActorUid is always taken from trusted request identity (`getUserId`), not from
 * the caller payload.
 */
export type AppendFieldChangeReq = {
  Model: string;
  ResId: string;
  Field?: string | null;
  Kind: string;
  OldValue?: string | null;
  NewValue?: string | null;
  At?: Date | string | null;
  CompanyId?: string | null;
  RequestId?: string | null;
  TraceId?: string | null;
};

/**
 * Asserts Kind is data-family only: field | create | unlink | action:*.
 */
export function assertFieldChangeKind(kind: string): void {
  const normalized = String(kind || '').trim();
  if (!normalized) {
    throw newAuditError({ code: AuditErrCode.INVALID_KIND, message: 'FieldChange.Kind is required' });
  }
  if ((FIELD_CHANGE_KINDS as readonly string[]).includes(normalized)) return;
  if (normalized.startsWith('action:')) return;
  throw newAuditError({
    code: AuditErrCode.INVALID_KIND,
    message: `FieldChange.Kind must be field|create|unlink|action:*, got ${normalized}`,
  });
}

function resolveCorrelation(): { requestId?: string; traceId?: string } {
  // Read live req (not getReqMeta): getReqMeta deep-freezes a shallow snapshot and shared
  // nested objects, which can freeze request-scoped service state used by Create.
  const req = (getCurrentReq() || {}) as {
    requestId?: unknown;
    RequestId?: unknown;
    traceId?: unknown;
    TraceId?: unknown;
    trace?: { requestId?: unknown; traceId?: unknown };
  };
  const requestId =
    String(req.requestId ?? req.RequestId ?? req.trace?.requestId ?? '').trim() || undefined;
  const traceId = String(req.traceId ?? req.TraceId ?? req.trace?.traceId ?? '').trim() || undefined;
  return { requestId, traceId };
}

function assertCreateKind(value: { Kind?: unknown } | null | undefined): void {
  assertFieldChangeKind(String(value?.Kind ?? ''));
}

/**
 * Append-only compliance field-change history (AU6 / P3-A1).
 * Table: audit_field_change.
 */
@Model('FieldChange', {
  application: 'audit',
  softDelete: false,
  orderBy: { field: 'At', order: 'asc' },
})
export default class FieldChange extends BaseModel {
  @Field({
    type: 'varchar',
    size: 120,
    notNull: true,
    index: true,
    string: _lt('Model', { scope: 'audit.model.FieldChange.fields' }),
  })
  Model: string;

  @Field({
    type: 'char',
    size: 20,
    notNull: true,
    index: true,
    string: _lt('Record', { scope: 'audit.model.FieldChange.fields' }),
  })
  ResId: string;

  @Field({
    type: 'varchar',
    size: 120,
    string: _lt('Field', { scope: 'audit.model.FieldChange.fields' }),
  })
  Field: string | null;

  @Field({
    type: 'varchar',
    size: 64,
    notNull: true,
    index: true,
    string: _lt('Kind', { scope: 'audit.model.FieldChange.fields' }),
  })
  Kind: string;

  @Field({
    type: 'text',
    string: _lt('Old Value', { scope: 'audit.model.FieldChange.fields' }),
  })
  OldValue: string | null;

  @Field({
    type: 'text',
    string: _lt('New Value', { scope: 'audit.model.FieldChange.fields' }),
  })
  NewValue: string | null;

  @Field({
    type: 'char',
    size: 20,
    index: true,
    string: _lt('Actor', { scope: 'audit.model.FieldChange.fields' }),
  })
  ActorUid: string | null;

  @Field({
    type: 'datetime',
    notNull: true,
    index: true,
    string: _lt('At', { scope: 'audit.model.FieldChange.fields' }),
  })
  At: Date;

  @Field({
    type: 'char',
    size: 20,
    index: true,
    string: _lt('Company', { scope: 'audit.model.FieldChange.fields' }),
  })
  CompanyId: string | null;

  @Field({
    type: 'varchar',
    size: 64,
    string: _lt('Request Id', { scope: 'audit.model.FieldChange.fields' }),
  })
  RequestId: string | null;

  @Field({
    type: 'varchar',
    size: 64,
    string: _lt('Trace Id', { scope: 'audit.model.FieldChange.fields' }),
  })
  TraceId: string | null;

  /**
   * Appends one FieldChange row (Unary). Rejects non-data-family Kind (AU6).
   * ActorUid always comes from trusted request identity.
   */
  public static async Append(req: AppendFieldChangeReq): Promise<FieldChange> {
    if (!req || typeof req !== 'object') {
      throw newAuditError({ code: AuditErrCode.INVALID_ARGUMENT, message: 'Append requires a payload' });
    }
    const model = String(req.Model || '').trim();
    const resId = String(req.ResId || '').trim();
    if (!model || !resId) {
      throw newAuditError({ code: AuditErrCode.INVALID_ARGUMENT, message: 'Append requires Model and ResId' });
    }
    assertFieldChangeKind(req.Kind);

    const correlation = resolveCorrelation();
    const actor = String(getUserId() ?? '').trim() || null;
    const at = req.At ? new Date(req.At as string | Date) : new Date();
    if (Number.isNaN(at.getTime())) {
      throw newAuditError({ code: AuditErrCode.INVALID_ARGUMENT, message: 'Append At must be a valid datetime' });
    }

    return (await this.Create(
      {
        Model: model,
        ResId: resId,
        Field: req.Field == null || req.Field === '' ? null : String(req.Field),
        Kind: String(req.Kind).trim(),
        OldValue: req.OldValue == null ? null : String(req.OldValue),
        NewValue: req.NewValue == null ? null : String(req.NewValue),
        ActorUid: actor,
        At: at,
        CompanyId: req.CompanyId == null || req.CompanyId === '' ? null : String(req.CompanyId),
        RequestId: req.RequestId ?? correlation.requestId ?? null,
        TraceId: req.TraceId ?? correlation.traceId ?? null,
      } as any,
      [
        'Id',
        'Model',
        'ResId',
        'Field',
        'Kind',
        'OldValue',
        'NewValue',
        'ActorUid',
        'At',
        'CompanyId',
        'RequestId',
        'TraceId',
      ] as any
    )) as FieldChange;
  }

  /**
   * Searches FieldChange rows for one tracked record, ordered by At ascending.
   */
  public static async SearchByRecord(
    model: string,
    resId: string,
    fields?: FieldSelection<FieldChange>
  ): Promise<Partial<FieldChange>[]> {
    const m = String(model || '').trim();
    const id = String(resId || '').trim();
    if (!m || !id) {
      throw newAuditError({ code: AuditErrCode.INVALID_ARGUMENT, message: 'SearchByRecord requires Model and ResId' });
    }
    return (await this.Search(
      {
        And: [
          ['Model', '=', m],
          ['ResId', '=', id],
        ],
      } as any,
      {
        fields: fields as any,
        orderBy: { field: 'At', order: 'asc' } as any,
      } as any
    )) as Partial<FieldChange>[];
  }

  /**
   * Create validates Kind so direct Create cannot bypass Append's AU6 check.
   */
  static override async Create<T extends BaseModel>(
    this: { new (...args: any[]): T } & typeof BaseModel,
    value: Partial<Insertable<T & BaseModel>>,
    returnFields?: FieldSelection<T>
  ): Promise<T> {
    assertCreateKind(value as { Kind?: unknown });
    return (await super.Create(value as any, returnFields as any)) as unknown as T;
  }

  /**
   * CreateMany validates Kind on every row (AU6).
   */
  static override async CreateMany<T extends BaseModel>(
    this: { new (...args: any[]): T } & typeof BaseModel,
    values: Partial<Insertable<T & BaseModel>>[],
    returnFields?: FieldSelection<T>
  ): Promise<T[]> {
    for (const row of values || []) {
      assertCreateKind(row as { Kind?: unknown });
    }
    return (await super.CreateMany(values as any, returnFields as any)) as unknown as T[];
  }

  /** FieldChange is append-only (AU2). */
  static override async Update<T extends BaseModel>(
    this: { new (...args: any[]): T } & typeof BaseModel,
    _condition: QueryCondition<T>,
    _values: Partial<Updateable<T & BaseModel>>,
    _returnFields?: FieldSelection<T>,
    _options?: UpdateOptions
  ): Promise<never> {
    throw newAuditError({ code: AuditErrCode.APPEND_ONLY, message: 'FieldChange does not support Update' });
  }

  /** FieldChange is append-only (AU2). */
  static override async UpdateById<T extends BaseModel>(
    this: { new (...args: any[]): T } & typeof BaseModel,
    _id: string,
    _values: Partial<Updateable<T & BaseModel>>,
    _returnFields?: FieldSelection<T>,
    _options?: UpdateOptions
  ): Promise<never> {
    throw newAuditError({ code: AuditErrCode.APPEND_ONLY, message: 'FieldChange does not support UpdateById' });
  }

  /** FieldChange is append-only (AU2). */
  static override async Delete<T extends BaseModel>(
    this: { new (...args: any[]): T } & typeof BaseModel,
    _condition: QueryCondition<T>,
    _options?: DeleteOptions
  ): Promise<never> {
    throw newAuditError({ code: AuditErrCode.APPEND_ONLY, message: 'FieldChange does not support Delete' });
  }

  /** FieldChange is append-only (AU2). */
  static override async DeleteById<T extends BaseModel>(
    this: { new (...args: any[]): T } & typeof BaseModel,
    _id: string,
    _options?: DeleteOptions
  ): Promise<never> {
    throw newAuditError({ code: AuditErrCode.APPEND_ONLY, message: 'FieldChange does not support DeleteById' });
  }
}
