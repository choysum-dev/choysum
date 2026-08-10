// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { withUser } from '../../runtime/context';
import { TimestampUtils } from './timestamp';

test('timestamp utils addTimestamps fills CreatedAt and UpdatedAt when missing', () => {
  const before = Date.now();
  const out = TimestampUtils.addTimestamps<{ Name: string }>({ Name: 'demo' } as any) as any;
  const after = Date.now();

  expect(out.Name).toBe('demo');
  expect(out.CreatedAt instanceof Date).toBe(true);
  expect(out.UpdatedAt instanceof Date).toBe(true);
  expect(out.CreatedAt.getTime() >= before).toBe(true);
  expect(out.CreatedAt.getTime() <= after).toBe(true);
  expect(out.UpdatedAt.getTime() >= before).toBe(true);
  expect(out.UpdatedAt.getTime() <= after).toBe(true);
  expect(out.CreatedUid).toBeUndefined();
  expect(out.UpdatedUid).toBeUndefined();
});

test('timestamp utils preserves explicit timestamps for create and update payloads', () => {
  const createdAt = new Date('2026-01-01T00:00:00.000Z');
  const updatedAt = new Date('2026-01-02T00:00:00.000Z');

  const created = TimestampUtils.addTimestamps<{ Name: string }>({
    Name: 'demo',
    CreatedAt: createdAt,
    UpdatedAt: updatedAt,
  } as any) as any;

  const updated = TimestampUtils.addUpdateTimestamp<{ Name: string }>({
    Name: 'demo',
    UpdatedAt: updatedAt,
  } as any) as any;

  expect(created.CreatedAt).toBe(createdAt);
  expect(created.UpdatedAt).toBe(updatedAt);
  expect(updated.UpdatedAt).toBe(updatedAt);
});

test('timestamp utils addUpdateTimestamp adds UpdatedAt when absent', () => {
  const before = Date.now();
  const updated = TimestampUtils.addUpdateTimestamp<{ Name: string }>({ Name: 'demo' } as any) as any;
  const after = Date.now();

  expect(updated.Name).toBe('demo');
  expect(updated.UpdatedAt instanceof Date).toBe(true);
  expect(updated.UpdatedAt.getTime() >= before).toBe(true);
  expect(updated.UpdatedAt.getTime() <= after).toBe(true);
});

test('timestamp utils stamps CreatedUid/UpdatedUid from actor on create', () => {
  const out = withUser('U-ACTOR', () => TimestampUtils.addTimestamps<{ Name: string }>({ Name: 'demo' } as any)) as any;
  expect(out.CreatedUid).toBe('U-ACTOR');
  expect(out.UpdatedUid).toBe('U-ACTOR');
  expect(out.DeletedUid).toBeUndefined();
});

test('timestamp utils preserves explicit CreatedUid/UpdatedUid on create', () => {
  const out = withUser('U-ACTOR', () =>
    TimestampUtils.addTimestamps<{ Name: string }>({
      Name: 'demo',
      CreatedUid: 'U-PRESET',
      UpdatedUid: 'U-PRESET-U',
    } as any)
  ) as any;
  expect(out.CreatedUid).toBe('U-PRESET');
  expect(out.UpdatedUid).toBe('U-PRESET-U');
});

test('timestamp utils addUpdateTimestamp stamps UpdatedUid from actor', () => {
  const out = withUser('U-UPD', () => TimestampUtils.addUpdateTimestamp<{ Name: string }>({ Name: 'demo' } as any)) as any;
  expect(out.UpdatedUid).toBe('U-UPD');
});

test('applyAuditUidOnUpdate strips CreatedUid and client DeletedUid; stamps UpdatedUid', () => {
  const payload: Record<string, unknown> = {
    Name: 'x',
    CreatedUid: 'hijack',
    DeletedUid: 'hijack-del',
    UpdatedUid: 'stale',
  };
  withUser('U-UPD', () => TimestampUtils.applyAuditUidOnUpdate(payload));
  expect(payload.CreatedUid).toBeUndefined();
  expect(payload.DeletedUid).toBeUndefined();
  expect(payload.UpdatedUid).toBe('U-UPD');
  expect(payload.Name).toBe('x');
});

test('applyAuditUidOnUpdate clears DeletedUid on restore (DeletedAt null)', () => {
  const payload: Record<string, unknown> = {
    DeletedAt: null,
    DeletedUid: 'old-del',
    CreatedUid: 'keep-stripped',
  };
  withUser('U-REST', () => TimestampUtils.applyAuditUidOnUpdate(payload));
  expect(payload.CreatedUid).toBeUndefined();
  expect(payload.DeletedUid).toBeNull();
  expect(payload.UpdatedUid).toBe('U-REST');
});

test('applyAuditUidOnUpdate leaves uid null when no actor', () => {
  const payload: Record<string, unknown> = { Name: 'x', CreatedUid: 'hijack' };
  TimestampUtils.applyAuditUidOnUpdate(payload);
  expect(payload.CreatedUid).toBeUndefined();
  expect(payload.UpdatedUid).toBeUndefined();
});
