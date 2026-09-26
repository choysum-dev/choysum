// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { formatChoyUtcIso, formatFieldChangeSummary, resolveChoyChatterAuthorLabel } from './chatterHelpers';

const labels = {
  created: 'Record created',
  unlinked: 'Record removed',
  changed: (field: string, oldValue: string, newValue: string) => `${field}:${oldValue}->${newValue}`,
  action: (name: string) => `Action:${name}`,
  fieldFallback: 'Field',
};

test('formatFieldChangeSummary formats create, unlink, field, and action kinds', () => {
  expect(
    formatFieldChangeSummary(
      {
        kind: 'fieldChange',
        id: '1',
        at: 1,
        field: null,
        changeKind: ' create ',
        oldValue: null,
        newValue: null,
        actorUid: null,
      },
      labels,
    ),
  ).toBe('Record created');
  expect(
    formatFieldChangeSummary(
      {
        kind: 'fieldChange',
        id: '2',
        at: 1,
        field: null,
        changeKind: 'unlink',
        oldValue: null,
        newValue: null,
        actorUid: null,
      },
      labels,
    ),
  ).toBe('Record removed');
  expect(
    formatFieldChangeSummary(
      {
        kind: 'fieldChange',
        id: '3',
        at: 1,
        field: 'Name',
        changeKind: 'field',
        oldValue: 'A',
        newValue: 'B',
        actorUid: 'u1',
      },
      labels,
    ),
  ).toBe('Name:A->B');
  expect(
    formatFieldChangeSummary(
      {
        kind: 'fieldChange',
        id: '4',
        at: 1,
        field: null,
        changeKind: 'action:confirm',
        oldValue: null,
        newValue: null,
        actorUid: 'u1',
      },
      labels,
    ),
  ).toBe('Action:confirm');
  expect(
    formatFieldChangeSummary(
      {
        kind: 'fieldChange',
        id: '4b',
        at: 1,
        field: null,
        changeKind: 'Action:Confirm',
        oldValue: null,
        newValue: null,
        actorUid: 'u1',
      },
      labels,
    ),
  ).toBe('Action:Confirm');
  expect(
    formatFieldChangeSummary(
      {
        kind: 'fieldChange',
        id: '4c',
        at: 1,
        field: null,
        changeKind: 'action:',
        oldValue: null,
        newValue: null,
        actorUid: null,
      },
      labels,
    ),
  ).toBe('Action:action:');
});

test('formatFieldChangeSummary normalizes empty values', () => {
  expect(
    formatFieldChangeSummary(
      {
        kind: 'fieldChange',
        id: '6',
        at: 1,
        field: 'Name',
        changeKind: 'field',
        oldValue: '',
        newValue: null,
        actorUid: null,
      },
      labels,
    ),
  ).toBe('Name:—->—');
  expect(
    formatFieldChangeSummary(
      {
        kind: 'fieldChange',
        id: '7',
        at: 1,
        field: null,
        changeKind: 'field',
        oldValue: 'A',
        newValue: 'B',
        actorUid: null,
      },
      labels,
    ),
  ).toBe('Field:A->B');
});

test('formatChoyUtcIso formats finite UTC timestamps', () => {
  expect(formatChoyUtcIso(Date.parse('2024-01-02T03:04:00.000Z'))).toBe('2024-01-02 03:04');
  expect(formatChoyUtcIso(Date.parse('0099-01-01T00:00:00.000Z'))).toBe('0099-01-01 00:00');
  expect(formatChoyUtcIso(Date.UTC(-1, 0, 1, 0, 0))).toBe('-0001-01-01 00:00');
  expect(formatChoyUtcIso(null)).toBe('');
  expect(formatChoyUtcIso(Number.NaN)).toBe('');
});

test('resolveChoyChatterAuthorLabel maps system / you / other', () => {
  expect(resolveChoyChatterAuthorLabel(null)).toBe('System');
  expect(
    resolveChoyChatterAuthorLabel('usr_1', {
      currentUserId: 'usr_1',
      currentUserName: 'Ada',
    }),
  ).toBe('Ada');
  expect(
    resolveChoyChatterAuthorLabel('usr_1', {
      currentUserId: 'usr_1',
      currentUserName: '',
    }),
  ).toBe('You');
  expect(
    resolveChoyChatterAuthorLabel('usr_other', {
      currentUserId: 'usr_1',
    }),
  ).toBe('usr_other');
});
