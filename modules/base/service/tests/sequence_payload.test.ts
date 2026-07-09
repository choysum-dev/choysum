// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import {
  buildSequenceFormatSnapshot,
  buildSequenceIdempotencyPayload,
  buildSequenceNextResult,
  buildSequencePublicSnapshot,
} from '@/base/service/models/_sequence_payload';

test('base.sequence_payload: buildSequenceFormatSnapshot normalizes nullable prefix/suffix', () => {
  const seq = {
    Prefix: undefined,
    Suffix: null,
    Padding: 6,
  } as any;

  expect(buildSequenceFormatSnapshot(seq)).toEqual({
    Prefix: '',
    Suffix: '',
    Padding: 6,
  });
});

test('base.sequence_payload: buildSequencePublicSnapshot resolves company id from relation object', () => {
  const seq = {
    Id: 'seq_1',
    CompanyId: { Id: 'cmp_1' },
    Code: 'sale.order',
    Prefix: 'SO/',
    Suffix: '',
    Padding: 4,
  } as any;

  expect(buildSequencePublicSnapshot(seq)).toEqual({
    Id: 'seq_1',
    CompanyId: 'cmp_1',
    Code: 'sale.order',
    Prefix: 'SO/',
    Suffix: '',
    Padding: 4,
  });
});

test('base.sequence_payload: buildSequenceIdempotencyPayload builds stable payload shape', () => {
  const expiresAt = new Date('2026-07-10T00:00:00.000Z');
  const seq = {
    Id: 'seq_2',
    CompanyId: 'cmp_2',
    Code: 'account.move',
    Prefix: 'MOVE/',
    Suffix: '-A',
    Padding: 5,
  } as any;

  expect(
    buildSequenceIdempotencyPayload(seq, {
      idempotencyKey: 'idem.k1',
      count: 2,
      dryRun: false,
      rangeStart: 10n,
      rangeEnd: 11n,
      expiresAt,
    })
  ).toEqual({
    CompanyId: 'cmp_2',
    SequenceId: 'seq_2',
    CodeSnapshot: 'account.move',
    FormatSnapshot: {
      Prefix: 'MOVE/',
      Suffix: '-A',
      Padding: 5,
    },
    IdempotencyKey: 'idem.k1',
    Count: 2,
    DryRun: false,
    RangeStart: '10',
    RangeEnd: '11',
    ExpiresAt: expiresAt,
  });
});

test('base.sequence_payload: buildSequenceNextResult wraps sequence snapshot with items and generatedAt', () => {
  const seq = {
    Id: 'seq_3',
    CompanyId: undefined,
    Code: 'invoice',
    Prefix: 'INV/',
    Suffix: '',
    Padding: 4,
  } as any;

  expect(
    buildSequenceNextResult(
      seq,
      [
        { Value: 'INV/0001', Number: 1 },
        { Value: 'INV/0002', Number: 2 },
      ],
      '2026-07-09T00:00:00.000Z'
    )
  ).toEqual({
    Items: [
      { Value: 'INV/0001', Number: 1 },
      { Value: 'INV/0002', Number: 2 },
    ],
    Sequence: {
      Id: 'seq_3',
      CompanyId: undefined,
      Code: 'invoice',
      Prefix: 'INV/',
      Suffix: '',
      Padding: 4,
    },
    GeneratedAt: '2026-07-09T00:00:00.000Z',
  });
});
