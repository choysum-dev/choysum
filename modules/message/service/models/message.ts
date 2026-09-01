// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { BaseModel, Field, Model, type BaseModelCtor } from '@/core/service';
import { getUserId } from '@/core/service/api/context';
import type { Insertable } from '@/core/service/api/input';
import type { FieldSelection } from '@/core/service/api/selection';
import { dial } from '@/core/service/orm/model/model_pool';
import { MessageErrCode, newMessageError, wrapMessageError } from '../error';
import { _lt } from '../i18n';
import PolymorphicRecordModel from '@/core/service/mixins/polymorphic_record_model';
import { publishThreadChangedTip } from '../tips';
import { assertTargetRecordReadable } from '../target_record';
import Notification from './notification';

export {
  TOPIC_MESSAGE_THREAD_CHANGED,
  TOPIC_MESSAGE_NOTIFICATION_USER,
  MESSAGE_POST_TIP_SOURCE,
  __setMessagePublishTipForTest,
} from '../tips';

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

/** V1 Message.Type values. `email` / `note` are placeholders. */
export const MESSAGE_TYPES = ['comment', 'email', 'note'] as const;
export type MessageTypeLiteral = (typeof MESSAGE_TYPES)[number];

/** Owner field name used when Post binds an attachment via document.AttachmentBinding. */
export const MESSAGE_ATTACHMENT_FIELD = 'Attachment';

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
  for (const field of ['Model', 'ResId', 'CreatedAt', 'AuthorUid', 'CompanyId'] as const) {
    if (!next.includes('*') && !next.includes(field)) {
      next = [field, ...next];
    }
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
export default class Message extends PolymorphicRecordModel {
  protected static override polymorphicOrderByField(): string {
    return 'CreatedAt';
  }

  protected static override polymorphicDeniedMessage(): string {
    return 'Message is not allowed for this record';
  }

  protected static override raisePolymorphicInvalidArgument(message: string): never {
    throw newMessageError({ code: MessageErrCode.INVALID_ARGUMENT, message });
  }

  protected static override async assertPolymorphicTargetReadable(model: string, resId: string): Promise<void> {
    await assertTargetRecordReadable(model, resId, this.polymorphicDeniedMessage());
  }

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
   * On success, fans out follower Notifications and best-effort Publishes thread/inbox tips.
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
    await assertTargetRecordReadable(model, resId, 'Message is not allowed for this record');

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

    await Notification.FanOutForMessage(created as Message);
    await publishThreadChangedTip(created as Message);
    return created;
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
