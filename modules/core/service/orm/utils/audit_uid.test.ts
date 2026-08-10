// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { withUser } from '../../runtime/context';
import { AuditUidUtils } from './audit_uid';

test('audit uid utils stamps CreatedUid/UpdatedUid from actor on create', () => {
  const out = withUser('U-ACTOR', () => AuditUidUtils.addCreateUids<{ Name: string }>({ Name: 'demo' } as any)) as any;
  expect(out.Name).toBe('demo');
  expect(out.CreatedUid).toBe('U-ACTOR');
  expect(out.UpdatedUid).toBe('U-ACTOR');
  expect(out.DeletedUid).toBeUndefined();
});

test('audit uid utils preserves explicit CreatedUid and overwrites UpdatedUid on create', () => {
  const out = withUser('U-ACTOR', () =>
    AuditUidUtils.addCreateUids<{ Name: string }>({
      Name: 'demo',
      CreatedUid: 'U-PRESET',
      UpdatedUid: 'U-PRESET-U',
    } as any)
  ) as any;
  expect(out.CreatedUid).toBe('U-PRESET');
  expect(out.UpdatedUid).toBe('U-ACTOR');
});

test('audit uid utils addCreateUids is a no-op without actor', () => {
  const input = { Name: 'demo' } as any;
  const out = AuditUidUtils.addCreateUids<{ Name: string }>(input) as any;
  expect(out).toBe(input);
  expect(out.CreatedUid).toBeUndefined();
});

test('audit uid utils addUpdateUid stamps UpdatedUid from actor and strips client value', () => {
  const out = withUser('U-UPD', () =>
    AuditUidUtils.addUpdateUid<{ Name: string }>({ Name: 'demo', UpdatedUid: 'hijack' } as any)
  ) as any;
  expect(out.UpdatedUid).toBe('U-UPD');
});

test('audit uid utils addUpdateUid strips client UpdatedUid without actor', () => {
  const out = AuditUidUtils.addUpdateUid<{ Name: string }>({ Name: 'demo', UpdatedUid: 'hijack' } as any) as any;
  expect(out.UpdatedUid).toBeUndefined();
  expect(out.Name).toBe('demo');
});

test('audit uid utils applyOnUpdate strips CreatedUid and client DeletedUid; stamps UpdatedUid', () => {
  const payload: Record<string, unknown> = {
    Name: 'x',
    CreatedUid: 'hijack',
    DeletedUid: 'hijack-del',
    UpdatedUid: 'stale',
  };
  withUser('U-UPD', () => AuditUidUtils.applyOnUpdate(payload));
  expect(payload.CreatedUid).toBeUndefined();
  expect(payload.DeletedUid).toBeUndefined();
  expect(payload.UpdatedUid).toBe('U-UPD');
  expect(payload.Name).toBe('x');
});

test('audit uid utils applyOnUpdate clears DeletedUid on restore (DeletedAt null)', () => {
  const payload: Record<string, unknown> = {
    DeletedAt: null,
    DeletedUid: 'old-del',
    CreatedUid: 'keep-stripped',
  };
  withUser('U-REST', () => AuditUidUtils.applyOnUpdate(payload));
  expect(payload.CreatedUid).toBeUndefined();
  expect(payload.DeletedUid).toBeNull();
  expect(payload.UpdatedUid).toBe('U-REST');
});

test('audit uid utils applyOnUpdate leaves uid null when no actor', () => {
  const payload: Record<string, unknown> = { Name: 'x', CreatedUid: 'hijack', UpdatedUid: 'hijack-upd' };
  AuditUidUtils.applyOnUpdate(payload);
  expect(payload.CreatedUid).toBeUndefined();
  expect(payload.UpdatedUid).toBeUndefined();
});

test('audit uid utils applyOnSoftDelete stamps DeletedUid and UpdatedUid', () => {
  const payload: Record<string, unknown> = { DeletedAt: new Date(), UpdatedAt: new Date() };
  withUser('U-DEL', () => AuditUidUtils.applyOnSoftDelete(payload));
  expect(payload.DeletedUid).toBe('U-DEL');
  expect(payload.UpdatedUid).toBe('U-DEL');
});

test('audit uid utils applyOnSoftDelete is a no-op without actor', () => {
  const payload: Record<string, unknown> = { DeletedAt: new Date() };
  AuditUidUtils.applyOnSoftDelete(payload);
  expect(payload.DeletedUid).toBeUndefined();
  expect(payload.UpdatedUid).toBeUndefined();
});
