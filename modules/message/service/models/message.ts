// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { BaseModel, Field, Model, type BaseModelCtor } from '@/core/service';
import { getUserId } from '@/core/service/api/context';
import type { Insertable } from '@/core/service/api/input';
import type { FieldSelection } from '@/core/service/api/selection';
import type { QueryCondition, SearchOptions } from '@/core/service/api/query';
import { dial } from '@/core/service/orm/model/model_pool';
import { MessageErrCode, newMessageError, wrapMessageError } from '../error';
import { _lt } from '../i18n';

/** V1 Message.Type values. `email` / `note` are placeholders. */
export const MESSAGE_TYPES = ['comment', 'email', 'note'] as const;
export type MessageTypeLiteral = (typeof MESSAGE_TYPES)[number];

/** Owner field name used when Post binds an attachment via document.AttachmentBinding. */
export const MESSAGE_ATTACHMENT_FIELD = 'Attachment';

/** Frozen tip topic for Form / Chatter thread refresh (matches pkg/bus). */
export const TOPIC_MESSAGE_THREAD_CHANGED = 'message.thread.changed';

/** Tip source stamped on Message.Post publishes. */
export const MESSAGE_POST_TIP_SOURCE = 'message.Post';

type PublishTipFn = (event: {
  topic: string;
  source: string;
  at?: number;
  payload?: Record<string, string>;
}) => void | Promise<void>;

let publishTipOverride: PublishTipFn | null | undefined;

/**
 * Post payload for Message (Unary Post API).
 *
 * AuthorUid is always taken from trusted request identity (`getUserId`), not from
 * the caller payload. After a successful write, Post best-effort Publishes a
 * `message.thread.changed` tip via the host EventBus.
 */
export type PostMessageReq = {
  Model: string;
  ResId: string;
  Body: string;
  Type?: string | null;
  CompanyId?: string | null;
  /** Optional document.AttachmentContent id to bind after create. */
  AttachmentObjectId?: string | null;
  /** Idempotent bind mutation id; generated when AttachmentObjectId is set and omitted. */
  AttachmentMutationId?: string | null;
};

type BindAttachmentFn = (req: {
  attachmentObjectId: string;
  ownerModel: string;
  ownerRecordId: string;
  fieldName: string;
  mutationId: string;
}) => Promise<unknown>;

type DialFn = <T = Record<string, (...args: unknown[]) => unknown>>(fullModelName: string) => T;
type XidNewFn = () => string | null | undefined;

let bindAttachmentOverride: BindAttachmentFn | null | undefined;
let dialOverride: DialFn | undefined;
let xidNewOverride: XidNewFn | undefined;

/**
 * Test-only override for tip Publish.
 * undefined = live $choysum.bus.publish; null = force missing bus; function = stub.
 */
export function __setMessagePublishTipForTest(fn: PublishTipFn | null | undefined): void {
  publishTipOverride = fn;
}

/**
 * Test-only override for document AttachmentBinding.Bind.
 * undefined = live dial; null = force missing Bind; function = stub.
 */
export function __setMessageAttachmentBindForTest(fn: BindAttachmentFn | null | undefined): void {
  bindAttachmentOverride = fn;
}

/**
 * Test-only override for cross-app dial used when Bind override is unset.
 */
export function __setMessageDialForTest(fn: DialFn | undefined): void {
  dialOverride = fn;
}

/**
 * Test-only override for mutation-id xid source (does not affect ORM Id generation).
 */
export function __setMessageXidNewForTest(fn: XidNewFn | undefined): void {
  xidNewOverride = fn;
}

/**
 * Asserts Type is a known V1 message type. Returns the trimmed canonical Type.
 */
export function assertMessageType(type: string): MessageTypeLiteral {
  const normalized = String(type || '').trim();
  if (!normalized) {
    throw newMessageError({ code: MessageErrCode.INVALID_TYPE, message: 'Message.Type is required' });
  }
  if ((MESSAGE_TYPES as readonly string[]).includes(normalized)) {
    return normalized as MessageTypeLiteral;
  }
  throw newMessageError({
    code: MessageErrCode.INVALID_TYPE,
    message: `Message.Type must be comment|email|note, got ${normalized}`,
  });
}

type MessageInsert = Partial<Insertable<Message>>;

function prepareCreatePayload(value: MessageInsert): MessageInsert {
  const uid = getUserId();
  const payload: MessageInsert = {
    ...value,
    AuthorUid: uid == null || String(uid).trim() === '' ? null : String(uid).trim(),
  };
  if (value.Type == null || String(value.Type).trim() === '') {
    // Omit Type so the field default (`comment`) applies.
    delete payload.Type;
  } else {
    payload.Type = assertMessageType(String(value.Type));
  }
  return payload;
}

/**
 * Idempotency key for document.AttachmentBinding.Bind.
 * Prefer host xid as-is (already ≤20 chars); do not prefix+truncate, which drops sequence entropy.
 */
function newMutationId(): string {
  const xid =
    xidNewOverride !== undefined
      ? xidNewOverride()
      : (globalThis as { $choysum?: { xid?: { New?: () => string } } }).$choysum?.xid?.New?.();
  if (typeof xid === 'string' && xid.trim()) {
    return xid.trim().slice(0, 20);
  }
  // Non-crypto fallback uniqueness only (not a secret).
  const token = `${Date.now().toString(36)}${Math.random().toString(36).slice(2, 10)}`;
  return `m${token}`.slice(0, 20);
}

function ensureIdInFields(fields: FieldSelection<Message>): FieldSelection<Message> {
  if (fields.includes('*') || fields.includes('Id')) return fields;
  return ['Id', ...fields];
}

function ensureTipFields(fields: FieldSelection<Message>): FieldSelection<Message> {
  let next = ensureIdInFields(fields);
  if (!next.includes('*') && !next.includes('Model')) {
    next = ['Model', ...next];
  }
  if (!next.includes('*') && !next.includes('ResId')) {
    next = ['ResId', ...next];
  }
  return next;
}

function resolveBind(): BindAttachmentFn | null {
  if (bindAttachmentOverride !== undefined) return bindAttachmentOverride;
  try {
    const dialFn = dialOverride || dial;
    const svc = dialFn<{ Bind?: BindAttachmentFn }>('document.AttachmentBinding');
    if (typeof svc?.Bind !== 'function') return null;
    return svc.Bind.bind(svc);
  } catch {
    return null;
  }
}

function resolvePublishTip(): PublishTipFn | null {
  if (publishTipOverride !== undefined) return publishTipOverride;
  const publish = (globalThis as { $choysum?: { bus?: { publish?: PublishTipFn } } }).$choysum?.bus?.publish;
  return typeof publish === 'function' ? publish.bind((globalThis as any).$choysum.bus) : null;
}

/**
 * Best-effort thread tip after a successful write. Never throws; tip is not authoritative.
 */
async function publishThreadChangedTip(created: Message): Promise<void> {
  const publish = resolvePublishTip();
  if (!publish) return;
  const messageId = String((created as Message).Id || '').trim();
  const model = String((created as Message).Model || '').trim();
  const resId = String((created as Message).ResId || '').trim();
  if (!messageId || !model || !resId) return;
  const createdAt = (created as Message & { CreatedAt?: Date | string | number | null }).CreatedAt;
  let at: number | undefined;
  if (createdAt instanceof Date && !Number.isNaN(createdAt.getTime())) {
    at = createdAt.getTime();
  } else if (typeof createdAt === 'number' && Number.isFinite(createdAt)) {
    at = createdAt;
  } else if (typeof createdAt === 'string' && createdAt.trim()) {
    const parsed = Date.parse(createdAt);
    if (!Number.isNaN(parsed)) at = parsed;
  }
  try {
    await publish({
      topic: TOPIC_MESSAGE_THREAD_CHANGED,
      source: MESSAGE_POST_TIP_SOURCE,
      ...(at != null ? { at } : {}),
      payload: { model, resId, messageId },
    });
  } catch {
    // Tip is best-effort; authoritative state remains the Message row.
  }
}

const DEFAULT_POST_FIELDS = [
  'Id',
  'Type',
  'Body',
  'Model',
  'ResId',
  'AuthorUid',
  'CompanyId',
  'CreatedAt',
] as const satisfies FieldSelection<Message>;

/**
 * Collaboration post on a business record.
 * Table: message_message.
 *
 * Successful Post Publishes a best-effort `message.thread.changed` tip.
 */
@Model('Message', {
  application: 'message',
  softDelete: true,
  companyField: 'CompanyId',
  orderBy: { field: 'CreatedAt', order: 'asc' },
})
export default class Message extends BaseModel {
  @Field({
    type: 'selection',
    selection: [
      { value: 'comment', label: 'comment' },
      { value: 'email', label: 'email' },
      { value: 'note', label: 'note' },
    ],
    size: 16,
    notNull: true,
    index: true,
    default: () => 'comment',
    string: _lt('Type', { scope: 'message.model.Message.fields' }),
  })
  Type: string;

  @Field({
    type: 'text',
    notNull: true,
    string: _lt('Body', { scope: 'message.model.Message.fields' }),
  })
  Body: string;

  @Field({
    type: 'varchar',
    size: 120,
    notNull: true,
    index: true,
    string: _lt('Model', { scope: 'message.model.Message.fields' }),
  })
  Model: string;

  @Field({
    type: 'char',
    size: 20,
    notNull: true,
    index: true,
    string: _lt('Record', { scope: 'message.model.Message.fields' }),
  })
  ResId: string;

  @Field({
    type: 'char',
    size: 20,
    index: true,
    string: _lt('Author', { scope: 'message.model.Message.fields' }),
  })
  AuthorUid: string | null;

  @Field({
    type: 'char',
    size: 20,
    index: true,
    string: _lt('Company', { scope: 'message.model.Message.fields' }),
  })
  CompanyId: string | null;

  /**
   * Binary placeholder so document.AttachmentBinding can own Attachment content.
   * Values are not stored on this column; Bind links AttachmentContent after Post.
   */
  @Field({
    type: 'binary',
    string: _lt('Attachment', { scope: 'message.model.Message.fields' }),
  })
  Attachment: string | null;

  /**
   * Creates one collaboration Message (Unary). Stamps AuthorUid from request identity.
   * Optional AttachmentObjectId dials document.AttachmentBinding.Bind after create.
   * On success, best-effort Publishes `message.thread.changed` (tip failure never rolls back).
   */
  public static async Post(req: PostMessageReq, fields?: FieldSelection<Message>): Promise<Message> {
    if (!req || typeof req !== 'object') {
      throw newMessageError({ code: MessageErrCode.INVALID_ARGUMENT, message: 'Post requires a payload' });
    }
    const model = String(req.Model || '').trim();
    const resId = String(req.ResId || '').trim();
    const body = String(req.Body ?? '');
    if (!model || !resId) {
      throw newMessageError({ code: MessageErrCode.INVALID_ARGUMENT, message: 'Post requires Model and ResId' });
    }
    if (!body.trim()) {
      throw newMessageError({ code: MessageErrCode.INVALID_ARGUMENT, message: 'Post requires a non-empty Body' });
    }

    const type = assertMessageType(req.Type == null || req.Type === '' ? 'comment' : String(req.Type));
    const companyId = req.CompanyId == null || req.CompanyId === '' ? null : String(req.CompanyId);
    const returnFields: FieldSelection<Message> = fields ?? [...DEFAULT_POST_FIELDS];
    const attachmentObjectId = String(req.AttachmentObjectId || '').trim();

    // Resolve Bind before Create so a missing binder does not leave an unbound Message.
    let bind: BindAttachmentFn | null = null;
    if (attachmentObjectId) {
      bind = resolveBind();
      if (!bind) {
        throw newMessageError({
          code: MessageErrCode.ATTACHMENT_BIND_FAILED,
          message: 'document.AttachmentBinding.Bind is not available',
        });
      }
    }

    // Tip needs Id/Model/ResId even when the caller asks for a narrow field set.
    const createFields = ensureTipFields(returnFields);
    const created = await this.Create(
      {
        Type: type,
        Body: body,
        Model: model,
        ResId: resId,
        CompanyId: companyId,
      } as MessageInsert,
      createFields
    );

    if (attachmentObjectId && bind) {
      const ownerRecordId = String((created as Message).Id || '').trim();
      if (!ownerRecordId) {
        throw newMessageError({
          code: MessageErrCode.ATTACHMENT_BIND_FAILED,
          message: 'Message Id is required to bind an attachment',
        });
      }
      const mutationId = String(req.AttachmentMutationId || '').trim() || newMutationId();
      try {
        await bind({
          attachmentObjectId,
          ownerModel: 'message.Message',
          ownerRecordId,
          fieldName: MESSAGE_ATTACHMENT_FIELD,
          mutationId,
        });
      } catch (err) {
        // Compensate unbound Message when Bind fails outside a rolled-back ambient TX.
        try {
          await this.DeleteById(ownerRecordId);
        } catch {
          // Best-effort; surface the Bind failure as the primary error.
        }
        throw wrapMessageError(err, {
          code: MessageErrCode.ATTACHMENT_BIND_FAILED,
          message: err instanceof Error ? err.message : 'Attachment bind failed',
        });
      }
    }

    await publishThreadChangedTip(created as Message);
    return created;
  }

  /**
   * Searches Message rows for one target record, ordered by CreatedAt ascending.
   */
  public static async SearchByRecord(
    model: string,
    resId: string,
    fields?: FieldSelection<Message>
  ): Promise<Partial<Message>[]> {
    const m = String(model || '').trim();
    const id = String(resId || '').trim();
    if (!m || !id) {
      throw newMessageError({
        code: MessageErrCode.INVALID_ARGUMENT,
        message: 'SearchByRecord requires Model and ResId',
      });
    }
    const condition: QueryCondition<Message> = {
      And: [
        ['Model', '=', m],
        ['ResId', '=', id],
      ],
    };
    const options: SearchOptions<Message> = {
      fields,
      orderBy: { field: 'CreatedAt', order: 'asc' },
    };
    return await this.Search(condition, options);
  }

  /**
   * Create stamps Type and AuthorUid from trusted identity.
   */
  static override async Create<T extends BaseModel>(
    this: BaseModelCtor<T>,
    value: Partial<Insertable<T & BaseModel>>,
    returnFields?: FieldSelection<T>
  ): Promise<T> {
    const payload = prepareCreatePayload(value as MessageInsert);
    return (await super.Create.call(this, payload as Partial<Insertable<T & BaseModel>>, returnFields)) as T;
  }

  /**
   * CreateMany stamps Type and AuthorUid on every row.
   */
  static override async CreateMany<T extends BaseModel>(
    this: BaseModelCtor<T>,
    values: Partial<Insertable<T & BaseModel>>[],
    returnFields?: FieldSelection<T>
  ): Promise<T[]> {
    const rows = (values || []).map(row => prepareCreatePayload(row as MessageInsert));
    return (await super.CreateMany.call(
      this,
      rows as Partial<Insertable<T & BaseModel>>[],
      returnFields
    )) as T[];
  }
}
