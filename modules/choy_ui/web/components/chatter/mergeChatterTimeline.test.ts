// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import {
  compareChatterTimelineEntries,
  mergeChatterTimeline,
  parseChatterTimestamp,
} from './mergeChatterTimeline';

test('parseChatterTimestamp parses Date, number, and ISO strings', () => {
  expect(parseChatterTimestamp(new Date('2024-01-01T00:00:00.000Z'))).toBe(
    Date.parse('2024-01-01T00:00:00.000Z'),
  );
  expect(parseChatterTimestamp(1_704_067_200_000)).toBe(1_704_067_200_000);
  expect(parseChatterTimestamp(1.5)).toBeNull();
  expect(parseChatterTimestamp(Number.MAX_SAFE_INTEGER + 1)).toBeNull();
  expect(parseChatterTimestamp('2024-01-01T00:00:00.000Z')).toBe(
    Date.parse('2024-01-01T00:00:00.000Z'),
  );
  // Naive date-time is pinned to UTC (not host-local).
  expect(parseChatterTimestamp('2024-01-01T03:04:00')).toBe(
    Date.parse('2024-01-01T03:04:00.000Z'),
  );
  // Offset without ISO `:` separator is rejected.
  expect(parseChatterTimestamp('2024-01-01T03:04:00+0500')).toBeNull();
  // Protobuf JSON int64 epoch ms arrives as a digit string.
  expect(parseChatterTimestamp('1704067200000')).toBe(1_704_067_200_000);
  expect(parseChatterTimestamp('-1000')).toBe(-1000);
  // Beyond Number.MAX_SAFE_INTEGER must not coerce with precision loss.
  expect(parseChatterTimestamp('9007199254740992')).toBeNull();
  // Safe integers outside ECMAScript Date range must be rejected.
  expect(parseChatterTimestamp(Number.MAX_SAFE_INTEGER)).toBeNull();
  expect(parseChatterTimestamp(String(Number.MAX_SAFE_INTEGER))).toBeNull();
});

test('parseChatterTimestamp returns null for empty or invalid values', () => {
  expect(parseChatterTimestamp(null)).toBeNull();
  expect(parseChatterTimestamp('')).toBeNull();
  expect(parseChatterTimestamp('   ')).toBeNull();
  expect(parseChatterTimestamp(Number.NaN)).toBeNull();
  expect(parseChatterTimestamp(new Date('invalid'))).toBeNull();
  expect(parseChatterTimestamp('not-a-date')).toBeNull();
  // Locale / RFC / space-separated forms are engine-dependent; wire uses ISO-8601.
  expect(parseChatterTimestamp('01/02/2024')).toBeNull();
  expect(parseChatterTimestamp('Mon, 01 Jan 2024 00:00:00 GMT')).toBeNull();
  expect(parseChatterTimestamp('2024-01-01 00:00:00')).toBeNull();
  expect(parseChatterTimestamp('2024-01-01 00:00:00Z')).toBeNull();
});

test('mergeChatterTimeline merges messages and field changes ascending', () => {
  const entries = mergeChatterTimeline(
    [
      {
        Id: 'm2',
        Type: 'comment',
        Body: 'second',
        AuthorUid: 'u1',
        CreatedAt: '2024-01-02T00:00:00.000Z',
      },
      {
        Id: 'm1',
        Type: 'comment',
        Body: 'first',
        AuthorUid: 'u1',
        CreatedAt: '2024-01-01T00:00:00.000Z',
      },
    ],
    [
      {
        Id: 'f1',
        Field: 'Name',
        Kind: 'field',
        OldValue: 'A',
        NewValue: 'B',
        ActorUid: 'u2',
        At: '2024-01-01T12:00:00.000Z',
      },
    ],
  );
  expect(entries.map(entry => `${entry.kind}:${entry.id}`)).toEqual([
    'message:m1',
    'fieldChange:f1',
    'message:m2',
  ]);
});

test('mergeChatterTimeline skips rows without ids or timestamps', () => {
  const entries = mergeChatterTimeline(
    [
      { Id: '', Body: 'x', CreatedAt: '2024-01-01T00:00:00.000Z' },
      { Id: 'm1', Body: 'ok', CreatedAt: '' },
    ],
    [{ Id: 'f1', Kind: 'create', At: null }],
  );
  expect(entries).toEqual([]);
});

test('mergeChatterTimeline keeps a message and field change sharing an id', () => {
  const at = '2024-01-01T00:00:00.000Z';
  const entries = mergeChatterTimeline(
    [{ Id: 'dup', Body: 'm', CreatedAt: at }],
    [{ Id: 'dup', Kind: 'field', Field: 'Name', At: at }],
  );
  expect(entries).toHaveLength(2);
  expect(entries.some(entry => entry.kind === 'message')).toBe(true);
  expect(entries.some(entry => entry.kind === 'fieldChange')).toBe(true);
});

test('mergeChatterTimeline dedupes identical kind:id after sort', () => {
  const at = '2024-01-01T00:00:00.000Z';
  const entries = mergeChatterTimeline(
    [
      { Id: 'm1', Body: 'first', CreatedAt: at },
      { Id: 'm1', Body: 'dup', CreatedAt: at },
    ],
    [],
  );
  expect(entries).toHaveLength(1);
  expect(entries[0]!.kind).toBe('message');
  expect(entries[0]!.kind === 'message' ? entries[0]!.body : '').toBe('first');
});

test('mergeChatterTimeline keeps earliest at when kind:id duplicates differ', () => {
  const entries = mergeChatterTimeline(
    [
      { Id: 'm1', Body: 'later', CreatedAt: '2024-01-02T00:00:00.000Z' },
      { Id: 'm1', Body: 'earlier', CreatedAt: '2024-01-01T00:00:00.000Z' },
    ],
    [],
  );
  expect(entries).toHaveLength(1);
  expect(entries[0]!.kind === 'message' ? entries[0]!.body : '').toBe('earlier');
  expect(entries[0]!.at).toBe(Date.parse('2024-01-01T00:00:00.000Z'));
});

test('compareChatterTimelineEntries tie-breaks fieldChange before message', () => {
  const at = Date.parse('2024-01-01T00:00:00.000Z');
  expect(
    compareChatterTimelineEntries(
      {
        kind: 'fieldChange',
        id: 'f',
        at,
        field: null,
        changeKind: 'create',
        oldValue: null,
        newValue: null,
        actorUid: null,
      },
      {
        kind: 'message',
        id: 'm',
        at,
        type: 'comment',
        body: 'x',
        authorUid: null,
      },
    ),
  ).toBeLessThan(0);
});

test('compareChatterTimelineEntries tie-breaks equal kind by id', () => {
  const at = Date.parse('2024-01-01T00:00:00.000Z');
  expect(
    compareChatterTimelineEntries(
      {
        kind: 'message',
        id: 'm-a',
        at,
        type: 'comment',
        body: 'a',
        authorUid: null,
      },
      {
        kind: 'message',
        id: 'm-b',
        at,
        type: 'comment',
        body: 'b',
        authorUid: null,
      },
    ),
  ).toBeLessThan(0);
  expect(
    compareChatterTimelineEntries(
      {
        kind: 'message',
        id: 'm-b',
        at,
        type: 'comment',
        body: 'b',
        authorUid: null,
      },
      {
        kind: 'message',
        id: 'm-a',
        at,
        type: 'comment',
        body: 'a',
        authorUid: null,
      },
    ),
  ).toBeGreaterThan(0);
});
