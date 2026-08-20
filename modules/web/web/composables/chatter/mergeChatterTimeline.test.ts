// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it } from 'vitest';
import {
  compareChatterTimelineEntries,
  mergeChatterTimeline,
  parseChatterTimestamp,
} from './mergeChatterTimeline';

describe('parseChatterTimestamp', () => {
  it('parses Date, number, and ISO strings', () => {
    expect(parseChatterTimestamp(new Date('2024-01-01T00:00:00.000Z'))).toBe(Date.parse('2024-01-01T00:00:00.000Z'));
    expect(parseChatterTimestamp(1_704_067_200_000)).toBe(1_704_067_200_000);
    expect(parseChatterTimestamp('2024-01-01T00:00:00.000Z')).toBe(Date.parse('2024-01-01T00:00:00.000Z'));
  });

  it('returns null for empty or invalid values', () => {
    expect(parseChatterTimestamp(null)).toBeNull();
    expect(parseChatterTimestamp('')).toBeNull();
    expect(parseChatterTimestamp('   ')).toBeNull();
    expect(parseChatterTimestamp(Number.NaN)).toBeNull();
    expect(parseChatterTimestamp(new Date('invalid'))).toBeNull();
    expect(parseChatterTimestamp('not-a-date')).toBeNull();
  });
});

describe('mergeChatterTimeline', () => {
  it('merges messages and field changes by timestamp ascending', () => {
    const entries = mergeChatterTimeline(
      [
        { Id: 'm2', Type: 'comment', Body: 'second', AuthorUid: 'u1', CreatedAt: '2024-01-02T00:00:00.000Z' },
        { Id: 'm1', Type: 'comment', Body: 'first', AuthorUid: 'u1', CreatedAt: '2024-01-01T00:00:00.000Z' },
      ],
      [{ Id: 'f1', Field: 'Name', Kind: 'field', OldValue: 'A', NewValue: 'B', ActorUid: 'u2', At: '2024-01-01T12:00:00.000Z' }]
    );

    expect(entries.map(entry => `${entry.kind}:${entry.id}`)).toEqual(['message:m1', 'fieldChange:f1', 'message:m2']);
  });

  it('skips rows without ids or timestamps', () => {
    const entries = mergeChatterTimeline(
      [{ Id: '', Body: 'x', CreatedAt: '2024-01-01T00:00:00.000Z' }, { Id: 'm1', Body: 'ok', CreatedAt: '' }],
      [{ Id: 'f1', Kind: 'create', At: null }]
    );
    expect(entries).toEqual([]);
  });

  it('normalizes nullable fields and tie-breaks equal timestamps', () => {
    const at = '2024-01-01T00:00:00.000Z';
    const entries = mergeChatterTimeline(
      [
        {
          Id: 'm-b',
          Type: '  ',
          Body: null,
          AuthorUid: '  ',
          CreatedAt: at,
        },
      ],
      [
        {
          Id: 'f-a',
          Field: '  ',
          Kind: '  ',
          OldValue: null,
          NewValue: null,
          ActorUid: '  ',
          At: at,
        },
      ]
    );

    expect(entries).toEqual([
      {
        kind: 'fieldChange',
        id: 'f-a',
        at: Date.parse(at),
        field: null,
        changeKind: 'field',
        oldValue: null,
        newValue: null,
        actorUid: null,
      },
      {
        kind: 'message',
        id: 'm-b',
        at: Date.parse(at),
        type: 'comment',
        body: '',
        authorUid: null,
      },
    ]);
  });

  it('accepts Date and numeric timestamps', () => {
    const at = new Date('2024-06-01T00:00:00.000Z');
    const entries = mergeChatterTimeline(
      [{ Id: 'm1', Body: 'hi', AuthorUid: 'u1', CreatedAt: at.getTime() }],
      [{ Id: 'f1', Kind: 'create', At: at }]
    );
    expect(entries).toHaveLength(2);
    expect(entries.every(entry => entry.at === at.getTime())).toBe(true);
    expect(entries.find(entry => entry.kind === 'message')).toMatchObject({ authorUid: 'u1' });
  });

  it('tie-breaks equal timestamps by kind and id', () => {
    const at = '2024-01-01T00:00:00.000Z';
    const entries = mergeChatterTimeline(
      [
        { Id: 'm-b', Body: 'b', CreatedAt: at },
        { Id: 'm-a', Body: 'a', CreatedAt: at },
      ],
      []
    );
    expect(entries.map(entry => entry.id)).toEqual(['m-a', 'm-b']);
  });

  it('normalizes actor and field metadata on field-change rows', () => {
    const entries = mergeChatterTimeline(
      [],
      [
        {
          Id: 'f1',
          Field: '  ',
          Kind: '  ',
          OldValue: 'A',
          NewValue: 'B',
          ActorUid: ' actor ',
          At: '2024-01-01T00:00:00.000Z',
        },
      ]
    );
    expect(entries[0]).toMatchObject({
      field: null,
      changeKind: 'field',
      actorUid: 'actor',
    });
  });

  it('accepts null timelines and defaults blank kinds', () => {
    expect(mergeChatterTimeline(null, null)).toEqual([]);
    expect(mergeChatterTimeline(undefined, undefined)).toEqual([]);
    const at = '2024-01-01T00:00:00.000Z';
    const entries = mergeChatterTimeline(
      [null as any, { Id: 'm1', Body: 'x', CreatedAt: at }],
      [null as any, { Id: 'f1', At: at } as any, { Id: 'f2', Kind: undefined as any, Field: 'Name', At: at }]
    );
    expect(entries.some(entry => entry.kind === 'fieldChange' && entry.id === 'f1')).toBe(true);
    expect(entries.some(entry => entry.kind === 'fieldChange' && entry.id === 'f2' && entry.changeKind === 'field')).toBe(
      true
    );
    expect(entries.some(entry => entry.kind === 'message' && entry.id === 'm1')).toBe(true);
  });

  it('orders same-timestamp field changes before messages in either compare direction', () => {
    const at = '2024-03-01T00:00:00.000Z';
    const entries = mergeChatterTimeline(
      [
        { Id: 'm-z', Body: 'z', CreatedAt: at },
        { Id: 'm-a', Body: 'a', CreatedAt: at },
      ],
      [
        { Id: 'f-z', Kind: 'create', At: at },
        { Id: 'f-a', Kind: 'create', At: at },
      ]
    );
    expect(entries.map(entry => `${entry.kind}:${entry.id}`)).toEqual([
      'fieldChange:f-a',
      'fieldChange:f-z',
      'message:m-a',
      'message:m-z',
    ]);

    const fieldChange = entries[0]!;
    const message = entries[2]!;
    expect(compareChatterTimelineEntries(fieldChange, message)).toBeLessThan(0);
    expect(compareChatterTimelineEntries(message, fieldChange)).toBeGreaterThan(0);
    expect(compareChatterTimelineEntries(message, { ...message, at: message.at + 1 })).toBeLessThan(0);
    expect(compareChatterTimelineEntries(message, { ...message, id: 'm-b' })).toBeLessThan(0);
  });

  it('rejects non-finite numeric timestamps', () => {
    expect(parseChatterTimestamp(Number.POSITIVE_INFINITY)).toBeNull();
    expect(parseChatterTimestamp(undefined)).toBeNull();
  });
});
