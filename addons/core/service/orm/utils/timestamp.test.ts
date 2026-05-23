// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

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
