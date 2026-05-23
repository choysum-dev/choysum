// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { GrpcCode, ChoysumError } from '../../error';

export type ErrorAction = 'retry' | 'reauth' | 'forbidden' | 'dismiss';

export function errorAction(error: unknown): ErrorAction {
  if (!(error instanceof ChoysumError)) {
    return 'dismiss';
  }
  if (error.grpcCode === GrpcCode.Unauthenticated) {
    return 'reauth';
  }
  if (error.grpcCode === GrpcCode.PermissionDenied) {
    return 'forbidden';
  }
  if (error.grpcCode === GrpcCode.Unavailable) {
    return 'retry';
  }
  return 'dismiss';
}
