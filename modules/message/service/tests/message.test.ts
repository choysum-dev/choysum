// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import Message, {
  assertMessageType,
  MESSAGE_ATTACHMENT_FIELD,
  MESSAGE_POST_TIP_SOURCE,
  TOPIC_MESSAGE_THREAD_CHANGED,
  __setMessageAttachmentBindForTest,
  __setMessageDialForTest,
  __setMessagePublishTipForTest,
  __setMessageXidNewForTest,
} from '../models/message';
import { MessageErrCode, isMessageError } from '../error';
import { __setMessageTargetRecordAuthForTest } from '../target_record';
import { dial } from '@/core/service/orm/model/model_pool';

const RR_CACHE_KEY = Symbol.for('choysum.recordrule.cache');
const FR_CACHE_KEY = Symbol.for('choysum.fieldrule.cache');
const TEST_USER_ID = 'usr_message_test';

function uid(prefix: string): string {
  const xid = (globalThis as any).$choysum?.xid?.New?.();
  // Prefer full xid uniqueness; prefix+slice drops trailing sequence entropy.
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
  __setMessagePublishTipForTest(undefined);
  __setMessageXidNewForTest(undefined);
  __setMessageTargetRecordAuthForTest(undefined);
}

async function withMessageScope<T>(fn: () => Promise<T>): Promise<T> {
  resetRequestContext();
  resetMessageTestSeams();
  __setMessageTargetRecordAuthForTest(async () => undefined);
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

    let nullBodyErr: unknown;
    try {
      await Message.Post({ Model: 'partner.Partner', ResId: uid('res'), Body: null as any });
    } catch (e) {
      nullBodyErr = e;
    }
    expect((nullBodyErr as any).code).toBe(MessageErrCode.INVALID_ARGUMENT);

    let nullModelErr: unknown;
    try {
      await Message.Post({ Model: null as any, ResId: uid('res'), Body: 'x' });
    } catch (e) {
      nullModelErr = e;
    }
    expect((nullModelErr as any).code).toBe(MessageErrCode.INVALID_ARGUMENT);

    let nullResErr: unknown;
    try {
      await Message.Post({ Model: 'partner.Partner', ResId: null as any, Body: 'x' });
    } catch (e) {
      nullResErr = e;
    }
    expect((nullResErr as any).code).toBe(MessageErrCode.INVALID_ARGUMENT);

    let typeErr: unknown;
    try {
      await Message.Post({ Model: 'partner.Partner', ResId: uid('res'), Body: 'x', Type: 'login' });
    } catch (e) {
      typeErr = e;
    }
    expect((typeErr as any).code).toBe(MessageErrCode.INVALID_TYPE);

    const defaultType = await Message.Create({
      Model: 'partner.Partner',
      ResId: uid('res'),
      Body: 'via Create',
      CompanyId: null,
    } as any);
    expect(String((defaultType as any).Type)).toBe('comment');

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

test('message.Message: Post and SearchByRecord deny when the target record is unreadable', async () => {
  await withMessageScope(async () => {
    __setMessageTargetRecordAuthForTest(null);
    const model = 'partner.Partner';
    const resId = uid('res');
    let postErr: unknown;
    try {
      await Message.Post({ Model: model, ResId: resId, Body: 'secret' });
    } catch (e) {
      postErr = e;
    }
    expect(isMessageError(postErr)).toBe(true);
    expect((postErr as any).code).toBe(MessageErrCode.PERMISSION_DENIED);

    let searchErr: unknown;
    try {
      await Message.SearchByRecord(model, resId, ['Id']);
    } catch (e) {
      searchErr = e;
    }
    expect(isMessageError(searchErr)).toBe(true);
    expect((searchErr as any).code).toBe(MessageErrCode.PERMISSION_DENIED);
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

    // Whitespace mutation id falls back to generated id; fields already include Id.
    binds.length = 0;
    await Message.Post(
      {
        Model: 'partner.Partner',
        ResId: uid('res'),
        Body: 'blank mutation',
        AttachmentObjectId: 'att_obj_blankmut______',
        AttachmentMutationId: '   ',
      },
      ['Id', 'Body']
    );
    expect(binds.length).toBe(1);
    expect(String(binds[0].mutationId || '').trim()).not.toBe('');

    // Custom fields omitting Id must still resolve ownerRecordId for Bind.
    binds.length = 0;
    const slim = await Message.Post(
      {
        Model: 'partner.Partner',
        ResId: uid('res'),
        Body: 'slim fields',
        AttachmentObjectId: 'att_obj_slim________',
      },
      ['Body', 'Type']
    );
    expect(binds.length).toBe(1);
    expect(String(binds[0].ownerRecordId || '')).not.toBe('');
    expect(String((slim as any).Id || '')).toBe(String(binds[0].ownerRecordId));
  });
});

test('message.Message: Post fails closed when attachment Bind is unavailable or throws', async () => {
  await withMessageScope(async () => {
    const resMissing = uid('res');
    __setMessageAttachmentBindForTest(null);
    let missingErr: unknown;
    try {
      await Message.Post({
        Model: 'partner.Partner',
        ResId: resMissing,
        Body: 'x',
        AttachmentObjectId: 'att_missing_________',
      });
    } catch (e) {
      missingErr = e;
    }
    expect(isMessageError(missingErr)).toBe(true);
    expect((missingErr as any).code).toBe(MessageErrCode.ATTACHMENT_BIND_FAILED);
    // Bind resolved before Create — no orphan Message.
    expect((await Message.SearchByRecord('partner.Partner', resMissing, ['Id'])).length).toBe(0);

    const resBoom = uid('res');
    __setMessageAttachmentBindForTest(async () => {
      throw new Error('bind boom');
    });
    let boomErr: unknown;
    try {
      await Message.Post({
        Model: 'partner.Partner',
        ResId: resBoom,
        Body: 'x',
        AttachmentObjectId: 'att_boom____________',
      });
    } catch (e) {
      boomErr = e;
    }
    expect(isMessageError(boomErr)).toBe(true);
    expect((boomErr as any).code).toBe(MessageErrCode.ATTACHMENT_BIND_FAILED);
    // Bind failure compensates by deleting the created Message.
    expect((await Message.SearchByRecord('partner.Partner', resBoom, ['Id'])).length).toBe(0);

    // Live dial path (no overrides): document.AttachmentBinding.Bind exists but fails
    // without company context — Post must still fail closed as ATTACHMENT_BIND_FAILED.
    const resLive = uid('res');
    __setMessageAttachmentBindForTest(undefined);
    __setMessageDialForTest(undefined);
    let liveErr: unknown;
    try {
      await Message.Post({
        Model: 'partner.Partner',
        ResId: resLive,
        Body: 'x',
        AttachmentObjectId: 'att_live_____________',
      });
    } catch (e) {
      liveErr = e;
    }
    expect(isMessageError(liveErr)).toBe(true);
    expect((liveErr as any).code).toBe(MessageErrCode.ATTACHMENT_BIND_FAILED);
    expect((await Message.SearchByRecord('partner.Partner', resLive, ['Id'])).length).toBe(0);
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

test('message.Message: CreateMany stamps Type/AuthorUid and accepts empty input', async () => {
  await withMessageScope(async () => {
    const many = await Message.CreateMany(
      [
        {
          Model: 'partner.Partner',
          ResId: uid('res'),
          Body: 'many-1',
          Type: '  note  ',
          CompanyId: null,
          AuthorUid: 'usr_forged____________',
        },
        {
          Model: 'partner.Partner',
          ResId: uid('res'),
          Body: 'many-2',
          Type: '   ',
          CompanyId: null,
        },
      ] as any,
      ['Id', 'Type', 'Body', 'AuthorUid'] as any
    );
    expect(many.length).toBe(2);
    expect(String((many[0] as any).Type)).toBe('note');
    expect(String((many[0] as any).AuthorUid)).toBe(TEST_USER_ID);
    expect(String((many[1] as any).Type)).toBe('comment');

    const emptyMany = await Message.CreateMany(null as any, ['Id'] as any);
    expect(emptyMany).toEqual([]);
  });
});

test('message.Message: Post via dialOverride Bind with CompanyId and star fields', async () => {
  await withMessageScope(async () => {
    const companyId = 'cmp_message_fixture_';
    const jsCtx = ensureRequestContext();
    jsCtx.ctx = { activeCompanyId: companyId, enabledCompanyIds: [companyId] };
    delete (jsCtx as any)[Symbol.for('choysum.ctx.frozen')];
    delete (jsCtx as any)[Symbol.for('choysum.ctx.override')];

    const dialBinds: any[] = [];
    __setMessageAttachmentBindForTest(undefined);
    __setMessageDialForTest(() => ({
      Bind: async (req: any) => {
        dialBinds.push(req);
        return { status: 'active' };
      },
    }));
    const withCompany = await Message.Post(
      {
        Model: 'partner.Partner',
        ResId: uid('res'),
        Body: 'company+star',
        CompanyId: companyId,
        AttachmentObjectId: 'att_star_____________',
      },
      ['*']
    );
    expect(String((withCompany as any).CompanyId)).toBe(companyId);
    expect(dialBinds.length).toBe(1);
    expect(String(dialBinds[0].mutationId || '')).not.toBe('');
  });
});

test('message.Message: Post fails closed when dial Bind is missing or dial throws', async () => {
  await withMessageScope(async () => {
    __setMessageAttachmentBindForTest(undefined);
    __setMessageDialForTest(() => ({} as any));
    let noBindErr: unknown;
    try {
      await Message.Post({
        Model: 'partner.Partner',
        ResId: uid('res'),
        Body: 'x',
        AttachmentObjectId: 'att_nobind___________',
      });
    } catch (e) {
      noBindErr = e;
    }
    expect((noBindErr as any).code).toBe(MessageErrCode.ATTACHMENT_BIND_FAILED);

    __setMessageDialForTest(() => {
      throw new Error('dial boom');
    });
    let dialThrowErr: unknown;
    try {
      await Message.Post({
        Model: 'partner.Partner',
        ResId: uid('res'),
        Body: 'x',
        AttachmentObjectId: 'att_dialthrow________',
      });
    } catch (e) {
      dialThrowErr = e;
    }
    expect((dialThrowErr as any).code).toBe(MessageErrCode.ATTACHMENT_BIND_FAILED);
  });
});

test('message.Message: Post surfaces ATTACHMENT_BIND_FAILED when Bind throw is non-Error and Delete fails', async () => {
  await withMessageScope(async () => {
    __setMessageAttachmentBindForTest(async () => {
      throw 'bind string boom';
    });
    const origDeleteById = Message.DeleteById;
    (Message as any).DeleteById = async () => {
      throw new Error('delete boom');
    };
    let nonErrorBindErr: unknown;
    try {
      await Message.Post({
        Model: 'partner.Partner',
        ResId: uid('res'),
        Body: 'x',
        AttachmentObjectId: 'att_nonerr____________',
      });
    } catch (e) {
      nonErrorBindErr = e;
    } finally {
      (Message as any).DeleteById = origDeleteById;
    }
    expect(isMessageError(nonErrorBindErr)).toBe(true);
    expect((nonErrorBindErr as any).code).toBe(MessageErrCode.ATTACHMENT_BIND_FAILED);
    expect(String((nonErrorBindErr as any).message || '')).toMatch(/Attachment bind failed|bind string/i);
  });
});

test('message.Message: Post refuses Bind when Create returns without Id', async () => {
  await withMessageScope(async () => {
    __setMessageAttachmentBindForTest(async () => ({ status: 'active' }));
    const origCreate = Message.Create;
    (Message as any).Create = async function (this: any, value: any, fields?: any) {
      const row = await origCreate.call(this, value, fields);
      (row as any).Id = '';
      return row;
    };
    let missingIdErr: unknown;
    try {
      await Message.Post({
        Model: 'partner.Partner',
        ResId: uid('res'),
        Body: 'x',
        AttachmentObjectId: 'att_noid______________',
      });
    } catch (e) {
      missingIdErr = e;
    } finally {
      (Message as any).Create = origCreate;
    }
    expect((missingIdErr as any).code).toBe(MessageErrCode.ATTACHMENT_BIND_FAILED);
    expect(String((missingIdErr as any).message || '')).toMatch(/Id is required/i);
  });
});

test('message.Message: Post generates mutationId when xid seam is unavailable or blank', async () => {
  await withMessageScope(async () => {
    const noXidBinds: any[] = [];
    __setMessageXidNewForTest(() => undefined);
    __setMessageAttachmentBindForTest(async req => {
      noXidBinds.push(req);
      return { status: 'active' };
    });
    await Message.Post({
      Model: 'partner.Partner',
      ResId: uid('res'),
      Body: 'no-xid',
      AttachmentObjectId: 'att_noxid_____________',
    });
    expect(String(noXidBinds[0].mutationId || '').length).toBeGreaterThan(0);
    expect(String(noXidBinds[0].mutationId).length).toBeLessThanOrEqual(20);

    const blankXidBinds: any[] = [];
    __setMessageXidNewForTest(() => '   ');
    __setMessageAttachmentBindForTest(async req => {
      blankXidBinds.push(req);
      return { status: 'active' };
    });
    await Message.Post({
      Model: 'partner.Partner',
      ResId: uid('res'),
      Body: 'blank-xid',
      AttachmentObjectId: 'att_blankxid__________',
    });
    expect(String(blankXidBinds[0].mutationId || '').length).toBeGreaterThan(0);
    expect(String(blankXidBinds[0].mutationId).length).toBeLessThanOrEqual(20);
  });
});

test('message.Message: Post normalizes blank AuthorUid and empty CompanyId', async () => {
  await withMessageScope(async () => {
    ensureRequestContext().identity = { userId: '   ' };
    const blankAuthor = await Message.Post({
      Model: 'partner.Partner',
      ResId: uid('res'),
      Body: 'blank author',
      CompanyId: '',
    });
    expect((blankAuthor as any).AuthorUid == null || (blankAuthor as any).AuthorUid === '').toBe(true);
    expect((blankAuthor as any).CompanyId == null || (blankAuthor as any).CompanyId === '').toBe(true);
  });
});

test('message.Message: Post publishes message.thread.changed tip after create', async () => {
  await withMessageScope(async () => {
    const published: any[] = [];
    __setMessagePublishTipForTest(event => {
      published.push(event);
    });

    const model = 'partner.Partner';
    const resId = uid('res');
    const created = await Message.Post({
      Model: model,
      ResId: resId,
      Body: 'tip me',
    });

    expect(published).toHaveLength(1);
    expect(published[0].topic).toBe(TOPIC_MESSAGE_THREAD_CHANGED);
    expect(published[0].source).toBe(MESSAGE_POST_TIP_SOURCE);
    expect(published[0].payload).toEqual({
      model,
      resId,
      messageId: String((created as any).Id),
    });
    expect(String(published[0].payload.body || '')).toBe('');
  });
});

test('message.Message: Post tip keeps CreatedAt when return fields omit it', async () => {
  await withMessageScope(async () => {
    const published: any[] = [];
    __setMessagePublishTipForTest(event => {
      published.push(event);
    });

    const created = await Message.Post(
      {
        Model: 'partner.Partner',
        ResId: uid('res'),
        Body: 'narrow fields',
      },
      ['Id', 'Body']
    );

    expect(published).toHaveLength(1);
    expect(typeof published[0].at).toBe('number');
    expect(Number.isFinite(published[0].at)).toBe(true);
    expect(published[0].payload.messageId).toBe(String((created as any).Id));
    expect(published[0].payload.model).toBe('partner.Partner');
  });
});

test('message.Message: Post succeeds when tip Publish fails or bus is missing', async () => {
  await withMessageScope(async () => {
    __setMessagePublishTipForTest(() => {
      throw new Error('bus down');
    });
    const created = await Message.Post({
      Model: 'partner.Partner',
      ResId: uid('res'),
      Body: 'still saved',
    });
    expect(String((created as any).Id || '')).not.toBe('');
    expect((created as any).Body).toBe('still saved');

    __setMessagePublishTipForTest(null);
    const withoutBus = await Message.Post({
      Model: 'partner.Partner',
      ResId: uid('res'),
      Body: 'no bus',
    });
    expect(String((withoutBus as any).Id || '')).not.toBe('');
  });
});

test('message.Message: Post does not publish tip when attachment bind fails', async () => {
  await withMessageScope(async () => {
    const published: any[] = [];
    __setMessagePublishTipForTest(event => {
      published.push(event);
    });
    __setMessageAttachmentBindForTest(async () => {
      throw new Error('bind failed');
    });

    let err: unknown;
    try {
      await Message.Post({
        Model: 'partner.Partner',
        ResId: uid('res'),
        Body: 'with attach',
        AttachmentObjectId: 'att_fail________________',
      });
    } catch (e) {
      err = e;
    }
    expect(isMessageError(err)).toBe(true);
    expect((err as any).code).toBe(MessageErrCode.ATTACHMENT_BIND_FAILED);
    expect(published).toHaveLength(0);
  });
});

test('message.Message: Post skips tip when live bus.publish is missing or non-function', async () => {
  await withMessageScope(async () => {
    const published: any[] = [];
    __setMessagePublishTipForTest(event => {
      published.push(event);
    });

    const root: any = (globalThis as any).$choysum ?? {};
    (globalThis as any).$choysum = root;
    const priorBus = root.bus;
    try {
      root.bus = {};
      __setMessagePublishTipForTest(undefined);

      const noPublish = await Message.Post({
        Model: 'partner.Partner',
        ResId: uid('res'),
        Body: 'no publish fn',
      });
      expect(String((noPublish as any).Id || '')).not.toBe('');
      expect(published).toHaveLength(0);

      root.bus = { publish: 'not-a-function' };
      const nonFn = await Message.Post({
        Model: 'partner.Partner',
        ResId: uid('res2'),
        Body: 'bad publish fn',
      });
      expect(String((nonFn as any).Id || '')).not.toBe('');
      expect(published).toHaveLength(0);
    } finally {
      if (priorBus === undefined) {
        delete root.bus;
      } else {
        root.bus = priorBus;
      }
    }
  });
});

test('message.Message: Post publishes via live $choysum.bus.publish seam', async () => {
  await withMessageScope(async () => {
    const published: any[] = [];
    const root: any = (globalThis as any).$choysum ?? {};
    (globalThis as any).$choysum = root;
    const priorBus = root.bus;
    try {
      root.bus = {
        publish: (event: unknown) => {
          published.push(event);
        },
      };
      __setMessagePublishTipForTest(undefined);

      const model = 'partner.Partner';
      const resId = uid('res');
      const created = await Message.Post({ Model: model, ResId: resId, Body: 'live bus' });

      expect(published).toHaveLength(1);
      expect(published[0].topic).toBe(TOPIC_MESSAGE_THREAD_CHANGED);
      expect(published[0].source).toBe(MESSAGE_POST_TIP_SOURCE);
      expect(published[0].payload.messageId).toBe(String((created as any).Id));
    } finally {
      if (priorBus === undefined) {
        delete root.bus;
      } else {
        root.bus = priorBus;
      }
    }
  });
});

test('message.Message: Post tip resolves CreatedAt from Date, number, and string', async () => {
  await withMessageScope(async () => {
    const published: any[] = [];
    __setMessagePublishTipForTest(event => {
      published.push(event);
    });

    const origCreate = Message.Create;
    const ts = Date.UTC(2024, 0, 15, 12, 0, 0);
    try {
      (Message as any).Create = async function (this: any, value: any, fields?: any) {
        const row = await origCreate.call(this, value, fields);
        (row as any).CreatedAt = new Date(ts);
        return row;
      };
      await Message.Post({ Model: 'partner.Partner', ResId: uid('res1'), Body: 'date-at' }, ['Body']);
      expect(published[0].at).toBe(ts);

      published.length = 0;
      (Message as any).Create = async function (this: any, value: any, fields?: any) {
        const row = await origCreate.call(this, value, fields);
        (row as any).CreatedAt = ts + 1;
        return row;
      };
      await Message.Post({ Model: 'partner.Partner', ResId: uid('res2'), Body: 'number-at' }, ['Body']);
      expect(published[0].at).toBe(ts + 1);

      published.length = 0;
      (Message as any).Create = async function (this: any, value: any, fields?: any) {
        const row = await origCreate.call(this, value, fields);
        (row as any).CreatedAt = new Date(ts + 2).toISOString();
        return row;
      };
      await Message.Post({ Model: 'partner.Partner', ResId: uid('res3'), Body: 'string-at' }, ['Body']);
      expect(published[0].at).toBe(ts + 2);
    } finally {
      (Message as any).Create = origCreate;
    }
  });
});

test('message.Message: Post tip omits at for invalid CreatedAt and skips incomplete rows', async () => {
  await withMessageScope(async () => {
    const published: any[] = [];
    __setMessagePublishTipForTest(event => {
      published.push(event);
    });

    const origCreate = Message.Create;
    try {
      (Message as any).Create = async function (this: any, value: any, fields?: any) {
        const row = await origCreate.call(this, value, fields);
        (row as any).CreatedAt = new Date('not-a-date');
        return row;
      };
      await Message.Post({ Model: 'partner.Partner', ResId: uid('res'), Body: 'bad date' }, ['Body']);
      expect(published[0].at).toBeUndefined();

      published.length = 0;
      (Message as any).Create = async function (this: any, value: any, fields?: any) {
        const row = await origCreate.call(this, value, fields);
        (row as any).CreatedAt = '   ';
        return row;
      };
      await Message.Post({ Model: 'partner.Partner', ResId: uid('res1b'), Body: 'blank date' }, ['Body']);
      expect(published[0].at).toBeUndefined();

      published.length = 0;
      (Message as any).Create = async function (this: any, value: any, fields?: any) {
        const row = await origCreate.call(this, value, fields);
        (row as any).CreatedAt = 'not-parseable';
        return row;
      };
      await Message.Post({ Model: 'partner.Partner', ResId: uid('res1c'), Body: 'bad string date' }, ['Body']);
      expect(published[0].at).toBeUndefined();

      published.length = 0;
      (Message as any).Create = async function (this: any, value: any, fields?: any) {
        const row = await origCreate.call(this, value, fields);
        (row as any).Model = '';
        return row;
      };
      await Message.Post({ Model: 'partner.Partner', ResId: uid('res2'), Body: 'no model' }, ['Body']);
      expect(published).toHaveLength(0);
    } finally {
      (Message as any).Create = origCreate;
    }
  });
});

test('message.Message: Post tip awaits async publish', async () => {
  await withMessageScope(async () => {
    const published: any[] = [];
    __setMessagePublishTipForTest(async event => {
      await Promise.resolve();
      published.push(event);
    });

    await Message.Post({ Model: 'partner.Partner', ResId: uid('res'), Body: 'async tip' });
    expect(published).toHaveLength(1);
  });
});

test('message.Message: Post ensureTipFields keeps explicit Model/ResId/CreatedAt selections', async () => {
  await withMessageScope(async () => {
    const published: any[] = [];
    __setMessagePublishTipForTest(event => {
      published.push(event);
    });

    const star = await Message.Post(
      { Model: 'partner.Partner', ResId: uid('res'), Body: 'star fields' },
      ['*']
    );
    expect(published).toHaveLength(1);
    expect(published[0].payload.messageId).toBe(String((star as any).Id));

    published.length = 0;
    const explicit = await Message.Post(
      { Model: 'partner.Partner', ResId: uid('res2'), Body: 'explicit tip fields' },
      ['Model', 'ResId', 'CreatedAt', 'Body']
    );
    expect(published).toHaveLength(1);
    expect(published[0].payload.messageId).toBe(String((explicit as any).Id));
    expect(typeof published[0].at).toBe('number');
  });
});

test('message.Message: Post ensureTipFields adds only missing columns for narrow selection', async () => {
  await withMessageScope(async () => {
    const published: any[] = [];
    __setMessagePublishTipForTest(event => {
      published.push(event);
    });

    const created = await Message.Post(
      { Model: 'partner.Partner', ResId: uid('res'), Body: 'partial tip fields' },
      ['Model', 'Body']
    );

    expect(published).toHaveLength(1);
    expect(published[0].payload.messageId).toBe(String((created as any).Id));
    expect(published[0].payload.model).toBe('partner.Partner');
    expect(typeof published[0].at).toBe('number');
  });
});
