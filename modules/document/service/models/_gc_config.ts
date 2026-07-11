// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { getBackendEnvPositiveInt } from '@/core/service/runtime/env/backend_env';

const DEFAULT_GC_BATCH_SIZE = 200;

export function resolveGcBatchSize(): number {
  return getBackendEnvPositiveInt(['CHOYSUM_DOCUMENT_GC_BATCH_SIZE'], DEFAULT_GC_BATCH_SIZE);
}
