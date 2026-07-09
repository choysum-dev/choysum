// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import type Sequence from './sequence';
import type { SequenceNextItem, SequenceNextResult } from './sequence';

export type SequenceFormatSnapshot = {
  Prefix: string;
  Suffix: string;
  Padding: number;
};

export type SequenceIdempotencyPayload = {
  CompanyId: unknown;
  SequenceId: string;
  CodeSnapshot: string;
  FormatSnapshot: SequenceFormatSnapshot;
  IdempotencyKey: string;
  Count: number;
  DryRun: boolean;
  RangeStart: string;
  RangeEnd: string;
  ExpiresAt: Date;
};

function readSequenceCompanyId(seq: Sequence): unknown {
  return (seq as any).CompanyId?.Id ?? (seq as any).CompanyId;
}

export function buildSequencePublicSnapshot(seq: Sequence): SequenceNextResult['Sequence'] {
  return {
    Id: seq.Id,
    CompanyId: readSequenceCompanyId(seq) as SequenceNextResult['Sequence']['CompanyId'],
    Code: seq.Code,
    Prefix: seq.Prefix,
    Suffix: seq.Suffix,
    Padding: seq.Padding,
  };
}

export function buildSequenceNextResult(seq: Sequence, items: SequenceNextItem[], generatedAt: string): SequenceNextResult {
  return {
    Items: items,
    Sequence: buildSequencePublicSnapshot(seq),
    GeneratedAt: generatedAt,
  };
}

export function buildSequenceFormatSnapshot(seq: Sequence): SequenceFormatSnapshot {
  return {
    Prefix: seq.Prefix ?? '',
    Suffix: seq.Suffix ?? '',
    Padding: seq.Padding,
  };
}

export function buildSequenceIdempotencyPayload(
  seq: Sequence,
  args: {
    idempotencyKey: string;
    count: number;
    dryRun: boolean;
    rangeStart: bigint;
    rangeEnd: bigint;
    expiresAt: Date;
  }
): SequenceIdempotencyPayload {
  return {
    CompanyId: readSequenceCompanyId(seq) ?? null,
    SequenceId: seq.Id,
    CodeSnapshot: seq.Code,
    FormatSnapshot: buildSequenceFormatSnapshot(seq),
    IdempotencyKey: args.idempotencyKey,
    Count: args.count,
    DryRun: args.dryRun,
    RangeStart: args.rangeStart.toString(),
    RangeEnd: args.rangeEnd.toString(),
    ExpiresAt: args.expiresAt,
  };
}
