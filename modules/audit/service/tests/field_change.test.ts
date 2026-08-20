// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import FieldChange, { assertFieldChangeKind, __setFieldChangeCorrelationReqReaderForTest } from '../models/field_change';
import { AuditErrCode, isAuditError } from '../error';
import { __setFieldChangeTargetAuthForTest } from '../target_record';

const RR_CACHE_KEY = Symbol.for('choysum.recordrule.cache');
const FR_CACHE_KEY = Symbol.for('choysum.fieldrule.cache');
const TEST_USER_ID = 'usr_audit_test';

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
      'audit.FieldChange:read',
      'audit.FieldChange:write',
      'audit.FieldChange:create',
      'audit.FieldChange:delete',
      'FieldChange:read',
      'FieldChange:write',
      'FieldChange:create',
      'FieldChange:delete',
    ],
  };
  jsCtx.identity = { userId: TEST_USER_ID };
  delete (jsCtx as any)[Symbol.for('choysum.ctx.override')];
  delete (jsCtx as any)[Symbol.for('choysum.ctx.frozen')];
  delete (jsCtx as any)[RR_CACHE_KEY];
  delete (jsCtx as any)[FR_CACHE_KEY];
}

async function withAuditScope<T>(fn: () => Promise<T>): Promise<T> {
  resetRequestContext();
  __setFieldChangeTargetAuthForTest(async () => undefined);
  try {
    return await fn();
  } finally {
    __setFieldChangeTargetAuthForTest(undefined);
    __setFieldChangeCorrelationReqReaderForTest(undefined);
  }
}

const OVERSIZED_ACTION_KIND = `action:${'x'.repeat(64)}`; // > 64 chars

async function expectInvalidKind(fn: () => Promise<unknown>): Promise<void> {
  let err: unknown;
  try {
    await fn();
  } catch (e) {
    err = e;
  }
  expect(isAuditError(err)).toBe(true);
  expect((err as any).code).toBe(AuditErrCode.INVALID_KIND);
}

test('audit.FieldChange: assertFieldChangeKind accepts data family and rejects others', () => {
  expect(assertFieldChangeKind('field')).toBe('field');
  expect(assertFieldChangeKind('create')).toBe('create');
  expect(assertFieldChangeKind('unlink')).toBe('unlink');
  expect(assertFieldChangeKind('action:confirm')).toBe('action:confirm');
  expect(() => assertFieldChangeKind('login')).toThrow(/field\|create\|unlink\|action:\*/);
  expect(() => assertFieldChangeKind('')).toThrow(/required/);
  expect(() => assertFieldChangeKind(OVERSIZED_ACTION_KIND)).toThrow(/64 characters/);
});

test('audit.FieldChange: Append field/create and SearchByRecord ordered by At', async () => {
  await withAuditScope(async () => {
    const model = 'partner.Partner';
    const resId = uid('res');

    const earlier = new Date('2026-01-01T00:00:00.000Z');
    const later = new Date('2026-01-02T00:00:00.000Z');

    await FieldChange.Append({
      Model: model,
      ResId: resId,
      Kind: 'create',
      At: earlier,
      NewValue: null,
    });
    await FieldChange.Append({
      Model: model,
      ResId: resId,
      Field: 'Name',
      Kind: 'field',
      OldValue: 'a',
      NewValue: 'b',
      At: later,
    });

    let kindErr: unknown;
    try {
      await FieldChange.Append({
        Model: model,
        ResId: resId,
        Kind: 'login',
      });
    } catch (err) {
      kindErr = err;
    }
    expect(isAuditError(kindErr)).toBe(true);
    expect((kindErr as any).code).toBe(AuditErrCode.INVALID_KIND);

    const rows = await FieldChange.SearchByRecord(model, resId, [
      'Id',
      'Kind',
      'Field',
      'OldValue',
      'NewValue',
      'At',
      'ActorUid',
    ] as any);

    expect(rows.length).toBeGreaterThanOrEqual(2);
    expect(String(rows[0].Kind)).toBe('create');
    expect(String(rows[1].Kind)).toBe('field');
    expect(String(rows[1].Field)).toBe('Name');
    expect(String(rows[1].OldValue)).toBe('a');
    expect(String(rows[1].NewValue)).toBe('b');
    expect(String(rows[0].ActorUid)).toBe(TEST_USER_ID);
  });
});

test('audit.FieldChange: Append/SearchByRecord validate inputs and correlation metadata', async () => {
  await withAuditScope(async () => {
    let nullErr: unknown;
    try {
      await FieldChange.Append(null as any);
    } catch (e) {
      nullErr = e;
    }
    expect(isAuditError(nullErr)).toBe(true);
    expect((nullErr as any).code).toBe(AuditErrCode.INVALID_ARGUMENT);

    let missingErr: unknown;
    try {
      await FieldChange.Append({ Model: '', ResId: '', Kind: 'field' } as any);
    } catch (e) {
      missingErr = e;
    }
    expect(isAuditError(missingErr)).toBe(true);
    expect((missingErr as any).code).toBe(AuditErrCode.INVALID_ARGUMENT);

    let modelOnlyErr: unknown;
    try {
      await FieldChange.Append({ Model: 'base.UoM', ResId: '  ', Kind: 'field' } as any);
    } catch (e) {
      modelOnlyErr = e;
    }
    expect((modelOnlyErr as any).code).toBe(AuditErrCode.INVALID_ARGUMENT);

    let atErr: unknown;
    try {
      await FieldChange.Append({
        Model: 'base.UoM',
        ResId: uid('uom'),
        Kind: 'field',
        At: 'not-a-date',
      } as any);
    } catch (e) {
      atErr = e;
    }
    expect(isAuditError(atErr)).toBe(true);
    expect((atErr as any).code).toBe(AuditErrCode.INVALID_ARGUMENT);

    let searchErr: unknown;
    try {
      await FieldChange.SearchByRecord('', '');
    } catch (e) {
      searchErr = e;
    }
    expect(isAuditError(searchErr)).toBe(true);
    expect((searchErr as any).code).toBe(AuditErrCode.INVALID_ARGUMENT);

    let searchModelErr: unknown;
    try {
      await FieldChange.SearchByRecord('base.UoM', '  ');
    } catch (e) {
      searchModelErr = e;
    }
    expect((searchModelErr as any).code).toBe(AuditErrCode.INVALID_ARGUMENT);

    __setFieldChangeTargetAuthForTest(null);
    let deniedErr: unknown;
    try {
      await FieldChange.SearchByRecord('base.UoM', uid('uom'), ['Id']);
    } catch (e) {
      deniedErr = e;
    }
    expect(isAuditError(deniedErr)).toBe(true);
    expect((deniedErr as any).code).toBe(AuditErrCode.PERMISSION_DENIED);
    __setFieldChangeTargetAuthForTest(async () => undefined);

    const jsCtx = ensureRequestContext();
    jsCtx.req.requestId = 'req_from_camel';
    jsCtx.req.traceId = 'tr_from_camel';
    const withCamel = await FieldChange.Append({
      Model: 'base.UoM',
      ResId: uid('uom'),
      Kind: 'action:ok',
      Field: '',
      CompanyId: '',
    } as any, ['Id', 'Field', 'CompanyId', 'RequestId', 'TraceId', 'Kind'] as any);
    expect((withCamel as any).Field).toBeNull();
    expect((withCamel as any).CompanyId).toBeNull();
    expect(String((withCamel as any).RequestId)).toBe('req_from_camel');
    expect(String((withCamel as any).TraceId)).toBe('tr_from_camel');
    expect(String((withCamel as any).Kind)).toBe('action:ok');

    jsCtx.req = {
      ...jsCtx.req,
      requestId: undefined,
      traceId: undefined,
      RequestId: 'REQ_PASCAL',
      TraceId: 'TR_PASCAL',
    };
    const withPascal = await FieldChange.Append({
      Model: 'base.UoM',
      ResId: uid('uom'),
      Kind: 'unlink',
    } as any, ['RequestId', 'TraceId'] as any);
    expect(String((withPascal as any).RequestId)).toBe('REQ_PASCAL');
    expect(String((withPascal as any).TraceId)).toBe('TR_PASCAL');

    jsCtx.req = {
      ...jsCtx.req,
      RequestId: undefined,
      TraceId: undefined,
      requestId: undefined,
      traceId: undefined,
      trace: { requestId: 'req_nested', traceId: 'tr_nested' },
    };
    const withNested = await FieldChange.Append({
      Model: 'base.UoM',
      ResId: uid('uom'),
      Kind: 'create',
      RequestId: 'req_explicit',
      TraceId: 'tr_explicit',
      CompanyId: 'co_main____________',
      Field: 'Code',
      OldValue: 1 as any,
      NewValue: 2 as any,
    } as any, ['RequestId', 'TraceId', 'CompanyId', 'OldValue', 'NewValue'] as any);
    expect(String((withNested as any).RequestId)).toBe('req_explicit');
    expect(String((withNested as any).TraceId)).toBe('tr_explicit');
    expect(String((withNested as any).CompanyId)).toBe('co_main____________');
    expect(String((withNested as any).OldValue)).toBe('1');
    expect(String((withNested as any).NewValue)).toBe('2');

    // Default At path + nested correlation only.
    resetRequestContext();
    {
      const fresh = ensureRequestContext();
      fresh.req.trace = { requestId: 'req_only_nested', traceId: 'tr_only_nested' };
    }
    const withDefaultAt = await FieldChange.Append({
      Model: 'base.UoM',
      ResId: uid('uom'),
      Kind: 'field',
      Field: null,
      OldValue: null,
      NewValue: null,
    } as any, ['Id', 'At', 'RequestId', 'TraceId', 'Field'] as any);
    expect((withDefaultAt as any).At).toBeTruthy();
    expect(String((withDefaultAt as any).RequestId)).toBe('req_only_nested');
    expect(String((withDefaultAt as any).TraceId)).toBe('tr_only_nested');
    expect((withDefaultAt as any).Field).toBeNull();

    // ActorUid null when request identity is absent.
    resetRequestContext();
    ensureRequestContext().identity = {};
    const noActor = await FieldChange.Append({
      Model: 'base.UoM',
      ResId: uid('uom'),
      Kind: 'create',
    } as any, ['ActorUid'] as any);
    expect((noActor as any).ActorUid == null || (noActor as any).ActorUid === '').toBe(true);

    // Create without Kind hits prepareCreatePayload Kind fallback.
    resetRequestContext();
    let missingKindErr: unknown;
    try {
      await FieldChange.Create({
        Model: 'base.UoM',
        ResId: uid('uom'),
        At: new Date(),
      } as any);
    } catch (e) {
      missingKindErr = e;
    }
    expect(isAuditError(missingKindErr)).toBe(true);
    expect((missingKindErr as any).code).toBe(AuditErrCode.INVALID_KIND);

    // Empty correlation req reader returns {} (ACL still uses live request context).
    resetRequestContext();
    __setFieldChangeCorrelationReqReaderForTest(() => null);
    try {
      const row = await FieldChange.Append({
        Model: 'base.UoM',
        ResId: uid('uom'),
        Kind: 'create',
      } as any, ['RequestId', 'TraceId'] as any);
      expect((row as any).RequestId == null || (row as any).RequestId === '').toBe(true);
      expect((row as any).TraceId == null || (row as any).TraceId === '').toBe(true);
    } finally {
      __setFieldChangeCorrelationReqReaderForTest(undefined);
    }
  });
});

test('audit.FieldChange: oversized Kind rejected on Append/Create/CreateMany', async () => {
  await withAuditScope(async () => {
    await expectInvalidKind(() =>
      FieldChange.Append({
        Model: 'base.UoM',
        ResId: uid('uom'),
        Kind: OVERSIZED_ACTION_KIND,
      } as any)
    );
    await expectInvalidKind(() =>
      FieldChange.Create({
        Model: 'base.UoM',
        ResId: uid('uom'),
        Kind: OVERSIZED_ACTION_KIND,
        At: new Date(),
      } as any)
    );
    await expectInvalidKind(() =>
      FieldChange.CreateMany([
        {
          Model: 'base.UoM',
          ResId: uid('uom'),
          Kind: OVERSIZED_ACTION_KIND,
          At: new Date(),
        },
      ] as any)
    );
  });
});

test('audit.FieldChange: direct Create rejects invalid Kind', async () => {
  await withAuditScope(async () => {
    let err: unknown;
    try {
      await FieldChange.Create({
        Model: 'base.UoM',
        ResId: uid('uom'),
        Kind: 'login',
        At: new Date(),
      } as any);
    } catch (e) {
      err = e;
    }
    expect(isAuditError(err)).toBe(true);
    expect((err as any).code).toBe(AuditErrCode.INVALID_KIND);
  });
});

test('audit.FieldChange: Create/CreateMany stamp ActorUid and canonicalize Kind', async () => {
  await withAuditScope(async () => {
    const created = await FieldChange.Create({
      Model: 'base.UoM',
      ResId: uid('uom'),
      Kind: '  field  ',
      Field: 'Name',
      OldValue: 'a',
      NewValue: 'b',
      ActorUid: 'usr_forged_actor____',
      At: new Date(),
    } as any, ['Id', 'Kind', 'ActorUid'] as any);

    expect(String((created as any).Kind)).toBe('field');
    expect(String((created as any).ActorUid)).toBe(TEST_USER_ID);

    const many = await FieldChange.CreateMany(
      [
        {
          Model: 'base.UoM',
          ResId: uid('uom'),
          Kind: ' create ',
          At: new Date(),
          ActorUid: 'usr_forged_many_____',
        },
      ] as any,
      ['Id', 'Kind', 'ActorUid'] as any
    );
    expect(many.length).toBe(1);
    expect(String((many[0] as any).Kind)).toBe('create');
    expect(String((many[0] as any).ActorUid)).toBe(TEST_USER_ID);

    const emptyMany = await FieldChange.CreateMany(null as any, ['Id'] as any);
    expect(emptyMany).toEqual([]);
  });
});

test('audit.FieldChange: Update/Delete are rejected (append-only)', async () => {
  await withAuditScope(async () => {
    const created = await FieldChange.Append({
      Model: 'base.UoM',
      ResId: uid('uom'),
      Kind: 'field',
      Field: 'Name',
      OldValue: 'x',
      NewValue: 'y',
    });

    async function expectAppendOnly(fn: () => Promise<unknown>): Promise<void> {
      let err: unknown;
      try {
        await fn();
      } catch (e) {
        err = e;
      }
      expect(isAuditError(err)).toBe(true);
      expect((err as any).code).toBe(AuditErrCode.APPEND_ONLY);
    }

    await expectAppendOnly(() => FieldChange.Update(['Id', '=', created.Id] as any, { NewValue: 'z' } as any));
    await expectAppendOnly(() => FieldChange.UpdateById(created.Id, { NewValue: 'z' } as any));
    await expectAppendOnly(() => FieldChange.Delete(['Id', '=', created.Id] as any));
    await expectAppendOnly(() => FieldChange.DeleteById(created.Id));
  });
});
