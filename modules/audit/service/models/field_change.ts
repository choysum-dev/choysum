// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { BaseModel, Field, Model, type BaseModelCtor } from '@/core/service';
import { getCurrentReq, getUserId } from '@/core/service/api/context';
import type { Insertable, Updateable } from '@/core/service/api/input';
import type { FieldSelection } from '@/core/service/api/selection';
import type { QueryCondition, DeleteOptions, UpdateOptions } from '@/core/service/api/query';
import { AuditErrCode, newAuditError } from '../error';
import { _lt } from '../i18n';
import PolymorphicRecordModel from '../mixins/polymorphic_record_model';
import { assertTargetRecordReadable } from '../target_record';
import { publishFieldChangeAppendedTip } from '../tips';

/** Data-family Kind values allowed on FieldChange. */
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
 * Returns the trimmed canonical Kind.
 */
export function assertFieldChangeKind(kind: string): string {
  const normalized = String(kind || '').trim();
  if (!normalized) {
    throw newAuditError({ code: AuditErrCode.INVALID_KIND, message: 'FieldChange.Kind is required' });
  }
  if (normalized.length > 64) {
    throw newAuditError({
      code: AuditErrCode.INVALID_KIND,
      message: 'FieldChange.Kind must not exceed 64 characters',
    });
  }
  if ((FIELD_CHANGE_KINDS as readonly string[]).includes(normalized)) return normalized;
  if (normalized.startsWith('action:')) return normalized;
  throw newAuditError({
    code: AuditErrCode.INVALID_KIND,
    message: `FieldChange.Kind must be field|create|unlink|action:*, got ${normalized}`,
  });
}

type ReqReader = () => unknown;
let correlationReqReader: ReqReader = getCurrentReq;

/**
 * Test-only override for correlation req resolution (does not affect ORM ACL).
 */
export function __setFieldChangeCorrelationReqReaderForTest(reader: ReqReader | undefined): void {
  correlationReqReader = reader || getCurrentReq;
}

function resolveCorrelation(): { requestId?: string; traceId?: string } {
  // Read live req (not getReqMeta): getReqMeta deep-freezes a shallow snapshot and shared
  // nested objects, which can freeze request-scoped service state used by Create.
  const current = correlationReqReader();
  if (!current || typeof current !== 'object') {
    return {};
  }
  const req = current as {
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

type FieldChangeInsert = Partial<Insertable<FieldChange>>;

/**
 * Normalize Kind and force ActorUid from trusted request identity for every create path.
 */
function prepareCreatePayload(value: FieldChangeInsert): FieldChangeInsert {
  const uid = getUserId();
  return {
    ...value,
    Kind: assertFieldChangeKind(value.Kind == null ? '' : String(value.Kind)),
    ActorUid: uid == null || String(uid).trim() === '' ? null : String(uid).trim(),
  };
}

const DEFAULT_APPEND_FIELDS = [
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
] as const satisfies FieldSelection<FieldChange>;

function fieldSelectionWithId(fields: FieldSelection<FieldChange>): FieldSelection<FieldChange> {
  if (fields.includes('*') || fields.includes('Id')) return fields;
  return ['Id', ...fields];
}

/**
 * Append-only compliance field-change history.
 * Table: audit_field_change.
 */
@Model('FieldChange', {
  application: 'audit',
  softDelete: false,
  orderBy: { field: 'At', order: 'asc' },
})
export default class FieldChange extends PolymorphicRecordModel {
  protected static override polymorphicOrderByField(): string {
    return 'At';
  }

  protected static override polymorphicDeniedMessage(): string {
    return 'FieldChange is not allowed for this record';
  }

  protected static override raisePolymorphicInvalidArgument(message: string): never {
    throw newAuditError({ code: AuditErrCode.INVALID_ARGUMENT, message });
  }

  protected static override async assertPolymorphicTargetReadable(model: string, resId: string): Promise<void> {
    await assertTargetRecordReadable(model, resId, this.polymorphicDeniedMessage());
  }

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
   * Appends one FieldChange row (Unary). Rejects non-data-family Kind.
   * ActorUid always comes from trusted request identity.
   * Optional `fields` is forwarded to Create (same FieldSelection contract).
   */
  public static async Append(
    req: AppendFieldChangeReq,
    fields?: FieldSelection<FieldChange>
  ): Promise<FieldChange> {
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

    const kind = String(req.Kind).trim();
    const createValue: FieldChangeInsert = {
      Model: model,
      ResId: resId,
      Field: req.Field == null || req.Field === '' ? null : String(req.Field),
      Kind: kind,
      OldValue: req.OldValue == null ? null : String(req.OldValue),
      NewValue: req.NewValue == null ? null : String(req.NewValue),
      ActorUid: actor,
      At: at,
      CompanyId: req.CompanyId == null || req.CompanyId === '' ? null : String(req.CompanyId),
      RequestId: req.RequestId ?? correlation.requestId ?? null,
      TraceId: req.TraceId ?? correlation.traceId ?? null,
    };
    const returnFields: FieldSelection<FieldChange> = fields ?? [...DEFAULT_APPEND_FIELDS];
    // Always request Id so tip publish does not depend on the caller's projection.
    const createFields = fieldSelectionWithId(returnFields);
    const created = await this.Create(createValue, createFields);
    await publishFieldChangeAppendedTip({
      Id: created.Id,
      Model: model,
      ResId: resId,
      At: at,
    });
    return created;
  }

  /**
   * Create validates Kind, persists the trimmed Kind, and stamps ActorUid from request identity.
   */
  static override async Create<T extends BaseModel>(
    this: BaseModelCtor<T>,
    value: Partial<Insertable<T & BaseModel>>,
    returnFields?: FieldSelection<T>
  ): Promise<T> {
    const payload = prepareCreatePayload(value as FieldChangeInsert);
    return (await super.Create.call(this, payload as Partial<Insertable<T & BaseModel>>, returnFields)) as T;
  }

  /**
   * CreateMany validates Kind, persists trimmed Kind, and stamps ActorUid on every row.
   */
  static override async CreateMany<T extends BaseModel>(
    this: BaseModelCtor<T>,
    values: Partial<Insertable<T & BaseModel>>[],
    returnFields?: FieldSelection<T>
  ): Promise<T[]> {
    const rows = (values || []).map(row => prepareCreatePayload(row as FieldChangeInsert));
    return (await super.CreateMany.call(
      this,
      rows as Partial<Insertable<T & BaseModel>>[],
      returnFields
    )) as T[];
  }

  /** FieldChange is append-only. */
  static override async Update<T extends BaseModel>(
    this: BaseModelCtor<T>,
    _condition: QueryCondition<T>,
    _values: Partial<Updateable<T & BaseModel>>,
    _returnFields?: FieldSelection<T>,
    _options?: UpdateOptions
  ): Promise<never> {
    throw newAuditError({ code: AuditErrCode.APPEND_ONLY, message: 'FieldChange does not support Update' });
  }

  /** FieldChange is append-only. */
  static override async UpdateById<T extends BaseModel>(
    this: BaseModelCtor<T>,
    _id: string,
    _values: Partial<Updateable<T & BaseModel>>,
    _returnFields?: FieldSelection<T>,
    _options?: UpdateOptions
  ): Promise<never> {
    throw newAuditError({ code: AuditErrCode.APPEND_ONLY, message: 'FieldChange does not support UpdateById' });
  }

  /** FieldChange is append-only. */
  static override async Delete<T extends BaseModel>(
    this: BaseModelCtor<T>,
    _condition: QueryCondition<T>,
    _options?: DeleteOptions
  ): Promise<never> {
    throw newAuditError({ code: AuditErrCode.APPEND_ONLY, message: 'FieldChange does not support Delete' });
  }

  /** FieldChange is append-only. */
  static override async DeleteById<T extends BaseModel>(
    this: BaseModelCtor<T>,
    _id: string,
    _options?: DeleteOptions
  ): Promise<never> {
    throw newAuditError({ code: AuditErrCode.APPEND_ONLY, message: 'FieldChange does not support DeleteById' });
  }
}
