// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import Message, {
  assertMessageType,
  MESSAGE_ATTACHMENT_FIELD,
  __setMessageAttachmentBindForTest,
  __setMessageDialForTest,
} from '../models/message';
import { MessageErrCode, isMessageError } from '../error';
import { dial } from '@/core/service/orm/model/model_pool';

const RR_CACHE_KEY = Symbol.for('choysum.recordrule.cache');
const FR_CACHE_KEY = Symbol.for('choysum.fieldrule.cache');
const TEST_USER_ID = 'usr_message_test';

function uid(prefix: string): string {
  const xid = (globalThis as any).$choysum?.xid?.New?.();
  const token = typeof xid === 'string' && xid.trim() ? xid.trim() : `${Date.now()}${Math.random()}`;
  return `${prefix}_${token}`.slice(0, 20);
}

function ensureRequestContext(): any {
  const root: any = (globalThis as any).$choysum ?? {};
  if (!root.request) root.request = {};
  if (!root.request.context) root.request.context = {};
  const jsCtx = root.request.context;
  if (!jsCtx.ctx) jsCtx.ctx = {};
  if (!jsCtx.req) jsCtx.req = {};
  if (!jsCtx.identity) jsCtx.identity = {};
  (globalThis as any).$choysum = root;
  return jsCtx;
}

function resetRequestContext(): void {
  const jsCtx = ensureRequestContext();
  jsCtx.ctx = {};
  jsCtx.req = {
    depth: 0,
    companyMode: 'skip',
    fieldRuleMode: 'skip',
    recordRuleMode: 'allowlist',
    recordRuleAllow: [
      'message.Message:read',
      'message.Message:write',
      'message.Message:create',
      'message.Message:delete',
      'Message:read',
      'Message:write',
      'Message:create',
      'Message:delete',
    ],
  };
  jsCtx.identity = { userId: TEST_USER_ID };
  delete (jsCtx as any)[Symbol.for('choysum.ctx.override')];
  delete (jsCtx as any)[Symbol.for('choysum.ctx.frozen')];
  delete (jsCtx as any)[RR_CACHE_KEY];
  delete (jsCtx as any)[FR_CACHE_KEY];
}

function resetMessageTestSeams(): void {
  __setMessageAttachmentBindForTest(undefined);
  __setMessageDialForTest(undefined);
}

async function withMessageScope<T>(fn: () => Promise<T>): Promise<T> {
  resetRequestContext();
  resetMessageTestSeams();
  try {
    return await fn();
  } finally {
    resetMessageTestSeams();
  }
}

test('message.Message: assertMessageType accepts V1 types and rejects others', () => {
  expect(assertMessageType('comment')).toBe('comment');
  expect(assertMessageType('email')).toBe('email');
  expect(assertMessageType('note')).toBe('note');
  expect(() => assertMessageType('')).toThrow(/required/);
  expect(() => assertMessageType('audit')).toThrow(/comment\|email\|note/);
});

test('message.Message: Post creates comment and SearchByRecord orders by CreatedAt', async () => {
  await withMessageScope(async () => {
    const model = 'partner.Partner';
    const resId = uid('res');

    const first = await Message.Post({
      Model: model,
      ResId: resId,
      Body: 'hello',
      Type: 'comment',
    });
    expect(String((first as any).Type)).toBe('comment');
    expect(String((first as any).Body)).toBe('hello');
    expect(String((first as any).AuthorUid)).toBe(TEST_USER_ID);

    await Message.Post({
      Model: model,
      ResId: resId,
      Body: 'second',
    });

    const rows = await Message.SearchByRecord(model, resId, [
      'Id',
      'Type',
      'Body',
      'AuthorUid',
      'CreatedAt',
    ]);
    expect(rows.length).toBeGreaterThanOrEqual(2);
    expect(String(rows[0].Body)).toBe('hello');
    expect(String(rows[1].Body)).toBe('second');
    expect(String(rows[0].AuthorUid)).toBe(TEST_USER_ID);
  });
});

test('message.Message: Post validates inputs and stamps AuthorUid via Create', async () => {
  await withMessageScope(async () => {
    let nullErr: unknown;
    try {
      await Message.Post(null as any);
    } catch (e) {
      nullErr = e;
    }
    expect(isMessageError(nullErr)).toBe(true);
    expect((nullErr as any).code).toBe(MessageErrCode.INVALID_ARGUMENT);

    let missingErr: unknown;
    try {
      await Message.Post({ Model: '', ResId: '', Body: 'x' });
    } catch (e) {
      missingErr = e;
    }
    expect((missingErr as any).code).toBe(MessageErrCode.INVALID_ARGUMENT);

    let emptyBodyErr: unknown;
    try {
      await Message.Post({ Model: 'partner.Partner', ResId: uid('res'), Body: '   ' });
    } catch (e) {
      emptyBodyErr = e;
    }
    expect((emptyBodyErr as any).code).toBe(MessageErrCode.INVALID_ARGUMENT);

    let typeErr: unknown;
    try {
      await Message.Post({ Model: 'partner.Partner', ResId: uid('res'), Body: 'x', Type: 'login' });
    } catch (e) {
      typeErr = e;
    }
    expect((typeErr as any).code).toBe(MessageErrCode.INVALID_TYPE);

    let searchErr: unknown;
    try {
      await Message.SearchByRecord('', '');
    } catch (e) {
      searchErr = e;
    }
    expect((searchErr as any).code).toBe(MessageErrCode.INVALID_ARGUMENT);

    resetRequestContext();
    ensureRequestContext().identity = {};
    const noAuthor = await Message.Post({
      Model: 'partner.Partner',
      ResId: uid('res'),
      Body: 'anon',
    });
    expect((noAuthor as any).AuthorUid == null || (noAuthor as any).AuthorUid === '').toBe(true);
  });
});

test('message.Message: Post binds attachment via document Binding dial seam', async () => {
  await withMessageScope(async () => {
    const binds: any[] = [];
    __setMessageAttachmentBindForTest(async req => {
      binds.push(req);
      return { status: 'active' };
    });

    const row = await Message.Post({
      Model: 'partner.Partner',
      ResId: uid('res'),
      Body: 'with file',
      AttachmentObjectId: 'att_obj_fixture_____',
      AttachmentMutationId: 'mut_fixture________',
    });

    expect(binds.length).toBe(1);
    expect(binds[0].attachmentObjectId).toBe('att_obj_fixture_____');
    expect(binds[0].ownerModel).toBe('message.Message');
    expect(binds[0].ownerRecordId).toBe(String((row as any).Id));
    expect(binds[0].fieldName).toBe(MESSAGE_ATTACHMENT_FIELD);
    expect(binds[0].mutationId).toBe('mut_fixture________');
  });
});

test('message.Message: Post fails closed when attachment Bind is unavailable or throws', async () => {
  await withMessageScope(async () => {
    __setMessageAttachmentBindForTest(null);
    let missingErr: unknown;
    try {
      await Message.Post({
        Model: 'partner.Partner',
        ResId: uid('res'),
        Body: 'x',
        AttachmentObjectId: 'att_missing_________',
      });
    } catch (e) {
      missingErr = e;
    }
    expect(isMessageError(missingErr)).toBe(true);
    expect((missingErr as any).code).toBe(MessageErrCode.ATTACHMENT_BIND_FAILED);

    __setMessageAttachmentBindForTest(async () => {
      throw new Error('bind boom');
    });
    let boomErr: unknown;
    try {
      await Message.Post({
        Model: 'partner.Partner',
        ResId: uid('res'),
        Body: 'x',
        AttachmentObjectId: 'att_boom____________',
      });
    } catch (e) {
      boomErr = e;
    }
    expect(isMessageError(boomErr)).toBe(true);
    expect((boomErr as any).code).toBe(MessageErrCode.ATTACHMENT_BIND_FAILED);

    // Live dial path (no overrides): document.AttachmentBinding.Bind exists but fails
    // without company context — Post must still fail closed as ATTACHMENT_BIND_FAILED.
    __setMessageAttachmentBindForTest(undefined);
    __setMessageDialForTest(undefined);
    let liveErr: unknown;
    try {
      await Message.Post({
        Model: 'partner.Partner',
        ResId: uid('res'),
        Body: 'x',
        AttachmentObjectId: 'att_live_____________',
      });
    } catch (e) {
      liveErr = e;
    }
    expect(isMessageError(liveErr)).toBe(true);
    expect((liveErr as any).code).toBe(MessageErrCode.ATTACHMENT_BIND_FAILED);
  });
});

test('message.Message: dial message.Message exposes Post for cross-app callers', async () => {
  await withMessageScope(async () => {
    const svc = dial<{ Post?: typeof Message.Post }>('message.Message');
    expect(typeof svc?.Post).toBe('function');
    const row = await svc.Post!({
      Model: 'partner.Partner',
      ResId: uid('res'),
      Body: 'via dial',
    });
    expect(String((row as any).Body)).toBe('via dial');
    expect(String((row as any).AuthorUid)).toBe(TEST_USER_ID);
  });
});
