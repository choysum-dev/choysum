// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import Message, {
  TOPIC_MESSAGE_NOTIFICATION_USER,
  TOPIC_MESSAGE_THREAD_CHANGED,
  __setMessagePublishTipForTest,
} from '../models/message';
import Follower from '../models/follower';
import Notification from '../models/notification';
import { MessageErrCode, isMessageError } from '../error';

const RR_CACHE_KEY = Symbol.for('choysum.recordrule.cache');
const FR_CACHE_KEY = Symbol.for('choysum.fieldrule.cache');
const AUTHOR_USER_ID = 'usr_msg_author______';
const FOLLOWER_USER_ID = 'usr_msg_follower____';

function uid(prefix: string): string {
  const xid = (globalThis as any).$choysum?.xid?.New?.();
  if (typeof xid === 'string' && xid.trim()) {
    return xid.trim().slice(0, 20);
  }
  const token = `${Date.now().toString(36)}${Math.random().toString(36).slice(2, 10)}`;
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

const MESSAGE_RECORD_RULES = [
  'message.Message:read',
  'message.Message:write',
  'message.Message:create',
  'message.Message:delete',
  'Message:read',
  'Message:write',
  'Message:create',
  'Message:delete',
  'message.Follower:read',
  'message.Follower:write',
  'message.Follower:create',
  'message.Follower:delete',
  'Follower:read',
  'Follower:write',
  'Follower:create',
  'Follower:delete',
  'message.Notification:read',
  'message.Notification:write',
  'message.Notification:create',
  'message.Notification:delete',
  'Notification:read',
  'Notification:write',
  'Notification:create',
  'Notification:delete',
];

function resetRequestContext(userId: string): void {
  const jsCtx = ensureRequestContext();
  jsCtx.ctx = {};
  jsCtx.req = {
    depth: 0,
    companyMode: 'skip',
    fieldRuleMode: 'skip',
    recordRuleMode: 'allowlist',
    recordRuleAllow: MESSAGE_RECORD_RULES,
  };
  jsCtx.identity = { userId };
  delete (jsCtx as any)[Symbol.for('choysum.ctx.override')];
  delete (jsCtx as any)[Symbol.for('choysum.ctx.frozen')];
  delete (jsCtx as any)[RR_CACHE_KEY];
  delete (jsCtx as any)[FR_CACHE_KEY];
}

async function withMessageScope<T>(userId: string, fn: () => Promise<T>): Promise<T> {
  resetRequestContext(userId);
  return await fn();
}

test('message.Follower: Follow is idempotent and Unfollow removes the row', async () => {
  await withMessageScope(AUTHOR_USER_ID, async () => {
    const model = 'partner.Partner';
    const resId = uid('res');

    const first = await Follower.Follow({ Model: model, ResId: resId });
    const second = await Follower.Follow({ Model: model, ResId: resId });
    expect(String((first as any).Id)).toBe(String((second as any).Id));

    const rows = await Follower.SearchByRecord(model, resId, ['UserId']);
    expect(rows).toHaveLength(1);
    expect(String(rows[0].UserId)).toBe(AUTHOR_USER_ID);

    const deleted = await Follower.Unfollow({ Model: model, ResId: resId });
    expect(deleted).toBe(1);
    expect(await Follower.SearchByRecord(model, resId, ['Id'])).toHaveLength(0);
  });
});

test('message.Message: Post fans out Notification and inbox tip to followers', async () => {
  const published: any[] = [];
  __setMessagePublishTipForTest(event => {
    published.push(event);
  });
  try {
    const model = 'partner.Partner';
    const resId = uid('res');

    await withMessageScope(FOLLOWER_USER_ID, async () => {
      await Follower.Follow({ Model: model, ResId: resId });
    });

    let createdId = '';
    await withMessageScope(AUTHOR_USER_ID, async () => {
      const created = await Message.Post({ Model: model, ResId: resId, Body: 'hello followers' });
      createdId = String((created as any).Id);
      expect(createdId).not.toBe('');
      expect(await Notification.SearchInbox()).toHaveLength(0);
    });

    await withMessageScope(FOLLOWER_USER_ID, async () => {
      const inbox = await Notification.SearchInbox({ unreadOnly: true });
      expect(inbox).toHaveLength(1);
      expect(String(inbox[0].MessageId)).toBe(createdId);
      expect(String(inbox[0].Model)).toBe(model);
      expect(String(inbox[0].ResId)).toBe(resId);
      expect(inbox[0].IsRead).toBe(false);
      expect(String(inbox[0].AuthorUid)).toBe(AUTHOR_USER_ID);

      const threadTips = published.filter(item => item.topic === TOPIC_MESSAGE_THREAD_CHANGED);
      const inboxTips = published.filter(item => item.topic === TOPIC_MESSAGE_NOTIFICATION_USER);
      expect(threadTips).toHaveLength(1);
      expect(inboxTips).toHaveLength(1);
      expect(inboxTips[0].payload.userId).toBe(FOLLOWER_USER_ID);
    });
  } finally {
    __setMessagePublishTipForTest(undefined);
  }
});

test('message.Message: Post does not notify the author follower', async () => {
  __setMessagePublishTipForTest(() => undefined);

  const model = 'partner.Partner';
  const resId = uid('res');

  await withMessageScope(AUTHOR_USER_ID, async () => {
    await Follower.Follow({ Model: model, ResId: resId });
    await Message.Post({ Model: model, ResId: resId, Body: 'self follow' });
    expect(await Notification.SearchInbox()).toHaveLength(0);
  });
});

test('message.Notification: MarkRead and MarkAllRead update inbox rows', async () => {
  __setMessagePublishTipForTest(() => undefined);
  try {
    const model = 'partner.Partner';
    const resId = uid('res');
    const readerId = 'usr_msg_reader______';

    await withMessageScope(readerId, async () => {
      await Notification.MarkAllRead();
      await Follower.Follow({ Model: model, ResId: resId });
    });

    await withMessageScope(AUTHOR_USER_ID, async () => {
      await Message.Post({ Model: model, ResId: resId, Body: 'notify me' });
    });

    await withMessageScope(readerId, async () => {
      const unread = await Notification.SearchInbox({ unreadOnly: true, fields: ['Id', 'IsRead'] });
      expect(unread).toHaveLength(1);
      const id = String(unread[0].Id);
      expect(await Notification.MarkRead([id])).toBe(1);
      expect(await Notification.SearchInbox({ unreadOnly: true })).toHaveLength(0);
    });

    await withMessageScope(AUTHOR_USER_ID, async () => {
      await Message.Post({ Model: model, ResId: resId, Body: 'second ping' });
    });

    await withMessageScope(readerId, async () => {
      expect(await Notification.SearchInbox({ unreadOnly: true })).toHaveLength(1);
      expect(await Notification.MarkAllRead()).toBe(1);
      expect(await Notification.SearchInbox({ unreadOnly: true })).toHaveLength(0);
    });
  } finally {
    __setMessagePublishTipForTest(undefined);
  }
});

test('message.Notification: SearchInbox requires identity', async () => {
  await withMessageScope(AUTHOR_USER_ID, async () => {
    ensureRequestContext().identity = {};
    let err: unknown;
    try {
      await Notification.SearchInbox();
    } catch (e) {
      err = e;
    }
    expect(isMessageError(err)).toBe(true);
    expect((err as any).code).toBe(MessageErrCode.INVALID_ARGUMENT);
  });
});
