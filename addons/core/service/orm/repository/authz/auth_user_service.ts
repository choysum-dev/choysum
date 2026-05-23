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

  const msg = String(errRecord?.message ?? '');
  if (/no registered proto files for app\s+auth/i.test(msg)) return true;
  if (/failed to load method descriptor/i.test(msg)) return true;
  if (/unknown service/i.test(msg)) return true;
  if (/unknown method/i.test(msg)) return true;
  if (/(target method does not exist|\u76ee\u6807\u65b9\u6cd5\u4e0d\u5b58\u5728)/i.test(msg)) return true;

  return false;
}
