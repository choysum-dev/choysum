// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import Message, {
  TOPIC_MESSAGE_NOTIFICATION_USER,
  TOPIC_MESSAGE_THREAD_CHANGED,
  __setMessagePublishTipForTest,
} from '../models/message';
import Follower, { __setMessageFollowTargetAuthForTest, __setMessageFollowDialForTest } from '../models/follower';
import Notification, { __setNotificationMarkAllReadBatchSizeForTest } from '../models/notification';
import MessageSubtype, { MESSAGE_SUBTYPE_DISCUSSIONS } from '../models/message_subtype';
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
  'message.MessageSubtype:read',
  'message.MessageSubtype:write',
  'message.MessageSubtype:create',
  'message.MessageSubtype:delete',
  'MessageSubtype:read',
  'MessageSubtype:write',
  'MessageSubtype:create',
  'MessageSubtype:delete',
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
  __setMessageFollowTargetAuthForTest(async () => undefined);
  try {
    return await fn();
  } finally {
    __setMessageFollowTargetAuthForTest(undefined);
    __setMessageFollowDialForTest(undefined);
    __setNotificationMarkAllReadBatchSizeForTest(undefined);
  }
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

    const restored = await Follower.Follow({ Model: model, ResId: resId });
    expect(String((restored as any).Id)).toBe(String((first as any).Id));
    expect(await Follower.SearchByRecord(model, resId, ['Id'])).toHaveLength(1);
  });
});

test('message.Follower: Follow denies when target record is unreadable', async () => {
  await withMessageScope(AUTHOR_USER_ID, async () => {
    __setMessageFollowTargetAuthForTest(null);
    let err: unknown;
    try {
      await Follower.Follow({ Model: 'partner.Partner', ResId: uid('res') });
    } catch (e) {
      err = e;
    }
    expect(isMessageError(err)).toBe(true);
    expect((err as any).code).toBe(MessageErrCode.PERMISSION_DENIED);
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

test('message.Notification: MarkAllRead continues past the first Search batch', async () => {
  __setMessagePublishTipForTest(() => undefined);
  try {
    const model = 'partner.Partner';
    const resId = uid('res');
    const readerId = 'usr_msg_batch________';

    await withMessageScope(readerId, async () => {
      await Follower.Follow({ Model: model, ResId: resId });
    });

    await withMessageScope(AUTHOR_USER_ID, async () => {
      await Message.Post({ Model: model, ResId: resId, Body: 'batch-1' });
      await Message.Post({ Model: model, ResId: resId, Body: 'batch-2' });
      await Message.Post({ Model: model, ResId: resId, Body: 'batch-3' });
    });

    await withMessageScope(readerId, async () => {
      __setNotificationMarkAllReadBatchSizeForTest(2);
      expect(await Notification.SearchInbox({ unreadOnly: true })).toHaveLength(3);
      expect(await Notification.MarkAllRead()).toBe(3);
      expect(await Notification.SearchInbox({ unreadOnly: true })).toHaveLength(0);
    });
  } finally {
    __setMessagePublishTipForTest(undefined);
  }
});

test('message.Notification: fan-out skips followers subscribed to a non-discussions subtype', async () => {
  __setMessagePublishTipForTest(() => undefined);
  try {
    const model = 'partner.Partner';
    const resId = uid('res');
    const skippedId = 'usr_msg_skipped_____';
    const includedId = 'usr_msg_included____';
    const typedId = 'usr_msg_typed_______';

    let otherSubtypeId = '';
    await withMessageScope(AUTHOR_USER_ID, async () => {
      const other = await MessageSubtype.Create(
        {
          InternalName: `other_${uid('st')}`,
          Name: 'Other',
          Description: null,
        } as any,
        ['Id'] as any
      );
      otherSubtypeId = String((other as any).Id);
    });

    await withMessageScope(skippedId, async () => {
      await Follower.Follow({ Model: model, ResId: resId, SubtypeId: otherSubtypeId });
    });
    await withMessageScope(includedId, async () => {
      await Follower.Follow({ Model: model, ResId: resId });
    });

    let discussionsId = '';
    await withMessageScope(AUTHOR_USER_ID, async () => {
      const discussions = await MessageSubtype.Search(['InternalName', '=', MESSAGE_SUBTYPE_DISCUSSIONS], {
        fields: ['Id'],
        limit: 1,
      });
      if (discussions.length === 0) {
        const created = await MessageSubtype.Create(
          {
            InternalName: MESSAGE_SUBTYPE_DISCUSSIONS,
            Name: 'Discussions',
            Description: null,
          } as any,
          ['Id'] as any
        );
        discussionsId = String((created as any).Id);
      } else {
        discussionsId = String((discussions[0] as any).Id);
      }
    });

    await withMessageScope(typedId, async () => {
      await Follower.Follow({ Model: model, ResId: resId, SubtypeId: discussionsId });
    });

    await withMessageScope(AUTHOR_USER_ID, async () => {
      await Message.Post({ Model: model, ResId: resId, Body: 'subtype filter' }, ['Body']);
    });

    await withMessageScope(skippedId, async () => {
      expect(await Notification.SearchInbox()).toHaveLength(0);
    });
    await withMessageScope(includedId, async () => {
      const inbox = await Notification.SearchInbox();
      expect(inbox).toHaveLength(1);
      expect(String(inbox[0].AuthorUid)).toBe(AUTHOR_USER_ID);
    });
    await withMessageScope(typedId, async () => {
      expect(await Notification.SearchInbox()).toHaveLength(1);
    });
  } finally {
    __setMessagePublishTipForTest(undefined);
  }
});

test('message.Follower: Follow and Unfollow validate payload and identity', async () => {
  await withMessageScope(AUTHOR_USER_ID, async () => {
    async function expectInvalid(fn: () => Promise<unknown>): Promise<void> {
      let err: unknown;
      try {
        await fn();
      } catch (e) {
        err = e;
      }
      expect(isMessageError(err)).toBe(true);
      expect((err as any).code).toBe(MessageErrCode.INVALID_ARGUMENT);
    }

    await expectInvalid(() => Follower.Follow(null as any));
    await expectInvalid(() => Follower.Follow({ Model: '', ResId: uid('res') }));
    await expectInvalid(() => Follower.Follow({ Model: 'partner.Partner', ResId: '' }));
    await expectInvalid(() => Follower.Unfollow(null as any));
    await expectInvalid(() => Follower.Unfollow({ Model: '', ResId: uid('res') }));
    await expectInvalid(() => Follower.Unfollow({ Model: 'partner.Partner', ResId: '' }));
    await expectInvalid(() => Follower.SearchByRecord('', ''));
    await expectInvalid(() => Follower.SearchByRecord('partner.Partner', '   '));

    ensureRequestContext().identity = {};
    await expectInvalid(() => Follower.Follow({ Model: 'partner.Partner', ResId: uid('res') }));
    await expectInvalid(() => Follower.Unfollow({ Model: 'partner.Partner', ResId: uid('res') }));
  });
});

test('message.Follower: Follow uses live target Search and explicit user/company fields', async () => {
  await withMessageScope(AUTHOR_USER_ID, async () => {
    const companyId = 'cmp_message_fixture_';
    const jsCtx = ensureRequestContext();
    jsCtx.ctx = { activeCompanyId: companyId, enabledCompanyIds: [companyId] };
    delete (jsCtx as any)[Symbol.for('choysum.ctx.frozen')];
    delete (jsCtx as any)[Symbol.for('choysum.ctx.override')];

    const created = await Message.Post({
      Model: 'partner.Partner',
      ResId: uid('res'),
      Body: 'own thread',
      CompanyId: companyId,
    });
    const messageId = String((created as any).Id);
    __setMessageFollowTargetAuthForTest(undefined);

    const followed = await Follower.Follow(
      {
        Model: 'message.Message',
        ResId: messageId,
        UserId: AUTHOR_USER_ID,
        SubtypeId: '',
        CompanyId: companyId,
      },
      ['Id', 'UserId', 'CompanyId']
    );
    expect(String((followed as any).UserId)).toBe(AUTHOR_USER_ID);
    expect(String((followed as any).CompanyId)).toBe(companyId);

    let missingErr: unknown;
    try {
      await Follower.Follow({ Model: 'message.Message', ResId: uid('res') });
    } catch (e) {
      missingErr = e;
    }
    expect((missingErr as any).code).toBe(MessageErrCode.PERMISSION_DENIED);

    let unknownErr: unknown;
    try {
      await Follower.Follow({ Model: 'missing.NoSuchModel', ResId: uid('res') });
    } catch (e) {
      unknownErr = e;
    }
    expect((unknownErr as any).code).toBe(MessageErrCode.PERMISSION_DENIED);

    __setMessageFollowTargetAuthForTest(undefined);
    __setMessageFollowDialForTest(() => ({} as any));
    let noSearchErr: unknown;
    try {
      await Follower.Follow({ Model: 'partner.Partner', ResId: uid('res') });
    } catch (e) {
      noSearchErr = e;
    }
    expect((noSearchErr as any).code).toBe(MessageErrCode.PERMISSION_DENIED);

    __setMessageFollowDialForTest(
      () =>
        ({
          Search: async () => {
            throw { code: MessageErrCode.PERMISSION_DENIED, message: 'denied' };
          },
        }) as any
    );
    let rethrowErr: unknown;
    try {
      await Follower.Follow({ Model: 'partner.Partner', ResId: uid('res') });
    } catch (e) {
      rethrowErr = e;
    }
    expect((rethrowErr as any).code).toBe(MessageErrCode.PERMISSION_DENIED);

    __setMessageFollowDialForTest(
      () =>
        ({
          Search: async () => {
            throw new Error('dial boom');
          },
        }) as any
    );
    let boomErr: unknown;
    try {
      await Follower.Follow({ Model: 'partner.Partner', ResId: uid('res') });
    } catch (e) {
      boomErr = e;
    }
    expect((boomErr as any).code).toBe(MessageErrCode.PERMISSION_DENIED);

    __setMessageFollowDialForTest(() => ({ Search: async () => [{ Id: 'ok' }] }) as any);
    const viaDial = await Follower.Follow({ Model: 'partner.Partner', ResId: uid('okrow') });
    expect(String((viaDial as any).Id || '')).not.toBe('');
  });
});

test('message.Follower: Follow recovers unique conflicts and skips blank Unfollow ids', async () => {
  await withMessageScope(AUTHOR_USER_ID, async () => {
    const model = 'partner.Partner';
    const resId = uid('res');
    const created = await Follower.Follow({ Model: model, ResId: resId });

    const origCreate = Follower.Create;
    const origSearch = Follower.Search;
    let searchCalls = 0;
    (Follower as any).Search = async function (this: any, ...args: any[]) {
      searchCalls += 1;
      if (searchCalls === 1) return [];
      return origSearch.apply(this, args);
    };
    (Follower as any).Create = async () => {
      throw new Error('UNIQUE constraint failed: uidx_message_follower_record_user');
    };
    try {
      const raced = await Follower.Follow({ Model: model, ResId: resId });
      expect(String((raced as any).Id)).toBe(String((created as any).Id));
    } finally {
      (Follower as any).Search = origSearch;
      (Follower as any).Create = origCreate;
    }

    await Follower.Unfollow({ Model: model, ResId: resId });
    searchCalls = 0;
    (Follower as any).Search = async function (this: any, ...args: any[]) {
      searchCalls += 1;
      if (searchCalls === 1) return [];
      return origSearch.apply(this, args);
    };
    (Follower as any).Create = async () => {
      throw { message: 'duplicate key value violates unique constraint' };
    };
    try {
      const restored = await Follower.Follow({ Model: model, ResId: resId });
      expect(String((restored as any).Id)).toBe(String((created as any).Id));
    } finally {
      (Follower as any).Search = origSearch;
      (Follower as any).Create = origCreate;
    }

    (Follower as any).Create = async () => {
      throw new Error('not a unique failure');
    };
    (Follower as any).Search = async () => [];
    let nonUniqueErr: unknown;
    try {
      await Follower.Follow({ Model: model, ResId: uid('res2') });
    } catch (e) {
      nonUniqueErr = e;
    } finally {
      (Follower as any).Search = origSearch;
      (Follower as any).Create = origCreate;
    }
    expect(String((nonUniqueErr as any).message || '')).toMatch(/not a unique failure/);

    (Follower as any).Search = async () => [];
    (Follower as any).Create = async () => {
      throw 'UNIQUE constraint failed';
    };
    let missingRaceErr: unknown;
    try {
      await Follower.Follow({ Model: model, ResId: uid('res3') });
    } catch (e) {
      missingRaceErr = e;
    } finally {
      (Follower as any).Search = origSearch;
      (Follower as any).Create = origCreate;
    }
    expect(String(missingRaceErr || '')).toMatch(/UNIQUE constraint failed/);

    const origUnfollowSearch = Follower.Search;
    (Follower as any).Search = async () => [{ Id: '  ' }, { Id: '' }];
    try {
      expect(await Follower.Unfollow({ Model: model, ResId: resId })).toBe(0);
    } finally {
      (Follower as any).Search = origUnfollowSearch;
    }

    (Follower as any).Search = async () => [
      { Id: String((created as any).Id), DeletedAt: new Date('not-a-date') },
    ];
    try {
      const invalidDate = await Follower.Follow({ Model: model, ResId: resId });
      expect(String((invalidDate as any).Id)).toBe(String((created as any).Id));
    } finally {
      (Follower as any).Search = origUnfollowSearch;
    }
  });
});

test('message.Notification: MarkRead/SearchInbox/FanOut cover remaining branches', async () => {
  __setMessagePublishTipForTest(() => undefined);
  try {
    await Notification.FanOutForMessage({});
    await Notification.FanOutForMessage({ Id: 'm1', Model: '', ResId: 'r1' } as any);

    const model = 'partner.Partner';
    const resId = uid('res');
    const readerId = 'usr_msg_inbox_______';

    await withMessageScope(readerId, async () => {
      await Follower.Follow({ Model: model, ResId: resId });
    });

    await withMessageScope(AUTHOR_USER_ID, async () => {
      await Message.Post({
        Model: model,
        ResId: resId,
        Body: 'company empty',
        CompanyId: '',
      });
      await Message.Post({ Model: model, ResId: resId, Body: 'second for limit' });
    });

    await withMessageScope(readerId, async () => {
      const limited = await Notification.SearchInbox({ limit: 1 });
      expect(limited).toHaveLength(1);
      const unlimitedZero = await Notification.SearchInbox({ limit: 0 });
      expect(unlimitedZero.length).toBeGreaterThanOrEqual(2);
      const all = await Notification.SearchInbox();
      expect(all.length).toBeGreaterThanOrEqual(2);

      let emptyMarkErr: unknown;
      try {
        await Notification.MarkRead([]);
      } catch (e) {
        emptyMarkErr = e;
      }
      expect((emptyMarkErr as any).code).toBe(MessageErrCode.INVALID_ARGUMENT);
      let blankMarkErr: unknown;
      try {
        await Notification.MarkRead(['   ']);
      } catch (e) {
        blankMarkErr = e;
      }
      expect((blankMarkErr as any).code).toBe(MessageErrCode.INVALID_ARGUMENT);
      expect(await Notification.MarkRead(['no_such_notification_'])).toBe(0);

      const firstId = String(all[0].Id);
      expect(await Notification.MarkRead([firstId])).toBe(1);
      expect(await Notification.MarkRead([firstId])).toBe(0);

      __setNotificationMarkAllReadBatchSizeForTest(0);
      expect(await Notification.MarkAllRead()).toBeGreaterThanOrEqual(0);

      const origSearch = Notification.Search;
      (Notification as any).Search = async () => [{ Id: '' }, { Id: '   ' }];
      try {
        expect(await Notification.MarkAllRead()).toBe(0);
      } finally {
        (Notification as any).Search = origSearch;
      }
    });

    await withMessageScope(AUTHOR_USER_ID, async () => {
      ensureRequestContext().identity = { userId: '   ' };
      await Notification.FanOutForMessage({
        Id: uid('msg'),
        Model: model,
        ResId: resId,
        AuthorUid: '   ',
        CompanyId: '',
      } as any);
    });
  } finally {
    __setMessagePublishTipForTest(undefined);
  }
});

