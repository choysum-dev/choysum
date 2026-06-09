// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { ChoysumError } from '../../error';

export interface ServiceErrorContext {
  requestId?: string;
  operation?: string;
  resource?: string;
}

export function withServiceErrorContext(error: ChoysumError, context: ServiceErrorContext): ChoysumError {
  const metadata: Record<string, string> = {};
  if (context.requestId) {
    metadata.requestId = context.requestId;
  }
  if (context.operation) {
    metadata.operation = context.operation;
  }
  if (context.resource) {
    metadata.resource = context.resource;
  }
  return error.withMetadata(metadata);
}
