// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import FieldChange, { assertFieldChangeKind } from '../models/field_change';
import { AuditErrCode, isAuditError } from '../error';

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
  return fn();
}

test('audit.FieldChange: assertFieldChangeKind accepts data family and rejects others', () => {
  assertFieldChangeKind('field');
  assertFieldChangeKind('create');
  assertFieldChangeKind('unlink');
  assertFieldChangeKind('action:confirm');
  expect(() => assertFieldChangeKind('login')).toThrow(/field\|create\|unlink\|action:\*/);
  expect(() => assertFieldChangeKind('')).toThrow(/required/);
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
