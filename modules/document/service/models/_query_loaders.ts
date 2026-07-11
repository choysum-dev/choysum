// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { GrpcCode } from '../error';
import { DocumentErrCode, throwDocumentError } from '../error';

/**
 * Generic single-row loader: Search with limit=1, throw NOT_FOUND if absent.
 */
export async function mustLoadOne<T>(
  searchFn: (condition: unknown, options?: unknown) => Promise<T[]>,
  condition: unknown,
  notFoundMessage: string,
  metadata?: Record<string, unknown>
): Promise<T> {
  const rows = await searchFn(condition, { limit: 1 });
  const record = rows[0] as T | undefined;
  if (!record) {
    throw throwDocumentError(DocumentErrCode.NOT_FOUND, notFoundMessage, GrpcCode.NotFound, metadata ?? {});
  }
  return record;
}
