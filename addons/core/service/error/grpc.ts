// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { GrpcCode, ChoysumError } from '../../error';

export function serviceErrorCode(error: unknown, fallback = GrpcCode.Unknown): GrpcCode {
  if (error instanceof ChoysumError) {
    return error.grpcCode;
  }
  return fallback;
}
