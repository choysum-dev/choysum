// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { CreateServerApiService } from '../../../rpc';
import { GrpcCode, ChoysumError } from '@/core/service/error';
import type { RecordRuleOp } from '../types';
import { asObjectRecord } from '@/core/utils/object';

type RpcArgMapper = (...args: unknown[]) => unknown;

// Call auth to fetch record-rule and field-rule data.
// IMPORTANT:
// - Do not statically depend on generated service output paths: they may not exist on a fresh install.
// - Do not assume auth is always loaded locally because apps can be deployed independently.
// - Reuse CreateServerApiService smart routing here:
//   1) local pool when the app is running in-process;
//   2) $choysum.grpc.unary for remote gRPC calls.

export const AuthUserService = {
  GetRecordRuleCondition: CreateServerApiService<RpcArgMapper>(
    'auth.User',
    'GetRecordRuleCondition',
    (...args) => {
      return [
        { name: 'model', type: 'string', value: args[0] as string },
        { name: 'op', type: 'string', value: args[1] as string },
      ];
    },
    { name: 'result', type: 'google.protobuf.Value' }
  ),
  GetFieldRuleSpec: CreateServerApiService<RpcArgMapper>(
    'auth.User',
    'GetFieldRuleSpec',
    (...args) => {
      return [{ name: 'model', type: 'string', value: args[0] as string }];
    },
    { name: 'result', type: 'google.protobuf.Value' }
  ),
} as {
  GetRecordRuleCondition(model: string, op: RecordRuleOp): Promise<unknown>;
  GetFieldRuleSpec(model: string): Promise<unknown>;
};

// Shared by not-present (allow) and unavailable (fail-closed) checkers so the
// message subset cannot drift — not-present must stay a strict subset of unavailable.
const MISSING_AUTH_MESSAGE_PATTERNS = [
  /no registered proto files for app\s+auth/i,
  /failed to load method descriptor/i,
  /unknown service/i,
  /unknown method/i,
  /(target method does not exist|\u76ee\u6807\u65b9\u6cd5\u4e0d\u5b58\u5728)/i,
];

function matchesMissingAuthMessage(msg: string): boolean {
  return MISSING_AUTH_MESSAGE_PATTERNS.some((re) => re.test(msg));
}

export function isAuthServiceUnavailable(err: unknown): boolean {
  const errRecord = asObjectRecord(err);
  const code = errRecord?.grpcCode ?? errRecord?.code;
  if (code === GrpcCode.Unimplemented || code === GrpcCode.NotFound || code === GrpcCode.Unavailable) return true;
  if (err instanceof ChoysumError) {
    return err.grpcCode === GrpcCode.Unimplemented || err.grpcCode === GrpcCode.NotFound || err.grpcCode === GrpcCode.Unavailable;
  }

  if (err instanceof TypeError) {
    const msg = String(errRecord?.message ?? err.message ?? '');
    if (/\$choysum|grpc|unary/i.test(msg)) return true;
  }

  return matchesMissingAuthMessage(String(errRecord?.message ?? ''));
}

/**
 * Auth is not part of this deployment (independent apps / unit shards without auth).
 * Distinct from transient Unavailable — missing auth must not fail-closed the whole ORM.
 */
export function isAuthServiceNotPresent(err: unknown): boolean {
  const errRecord = asObjectRecord(err);
  // ChoysumError exposes grpcCode as an own property, so the object-code check covers it.
  const code = errRecord?.grpcCode ?? errRecord?.code;
  if (code === GrpcCode.Unimplemented || code === GrpcCode.NotFound) return true;

  const msg = String(errRecord?.message ?? (err instanceof Error ? err.message : '') ?? '');
  return matchesMissingAuthMessage(msg);
}
