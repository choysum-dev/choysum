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
  expect(parseChatterTimestamp('2024-01-01T00:00:00.000Z')).toBe(
    Date.parse('2024-01-01T00:00:00.000Z'),
  );
  // Protobuf JSON int64 epoch ms arrives as a digit string.
  expect(parseChatterTimestamp('1704067200000')).toBe(1_704_067_200_000);
  expect(parseChatterTimestamp('-1000')).toBe(-1000);
  // Beyond Number.MAX_SAFE_INTEGER must not coerce with precision loss.
  expect(parseChatterTimestamp('9007199254740992')).toBeNull();
});

test('parseChatterTimestamp returns null for empty or invalid values', () => {
  expect(parseChatterTimestamp(null)).toBeNull();
  expect(parseChatterTimestamp('')).toBeNull();
  expect(parseChatterTimestamp('   ')).toBeNull();
  expect(parseChatterTimestamp(Number.NaN)).toBeNull();
  expect(parseChatterTimestamp(new Date('invalid'))).toBeNull();
  expect(parseChatterTimestamp('not-a-date')).toBeNull();
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
