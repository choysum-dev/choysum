// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { Code, ConnectError } from '@connectrpc/connect';
import { ErrorInfoSchema } from '../../error';

/** gRPC codes that are business/session outcomes, not transport outages. */
const BUSINESS_CODES = new Set<Code>([
  Code.InvalidArgument,
  Code.NotFound,
  Code.AlreadyExists,
  Code.PermissionDenied,
  Code.FailedPrecondition,
  Code.Unauthenticated,
  Code.OutOfRange,
  Code.ResourceExhausted,
  Code.Aborted,
]);

const FETCH_HINTS = [
  'Failed to fetch',
  'NetworkError',
  'ERR_CONNECTION_REFUSED',
  'Load failed',
  'Network request failed',
  'fetch failed',
];

function messageLooksLikeFetchFailure(message: string): boolean {
  return FETCH_HINTS.some(hint => message.includes(hint));
}

function hasErrorInfo(error: ConnectError): boolean {
  try {
    if (typeof error.findDetails !== 'function') return false;
    const details = error.findDetails(ErrorInfoSchema as any);
    return Array.isArray(details) && details.length > 0;
  } catch {
    return false;
  }
}

/**
 * True when the failure is a transport / connectivity outage rather than a
 * business RPC outcome. Used by lifecycle hooks before normalizeTransportError.
 */
export function isTransportOutage(error: unknown): boolean {
  if (error == null) return false;

  if (error instanceof ConnectError) {
    if (error.code === Code.Canceled) return false;
    if (hasErrorInfo(error)) return false;
    if (BUSINESS_CODES.has(error.code)) return false;
    if (error.code === Code.Unavailable) return true;
    if (error.code === Code.Unknown && messageLooksLikeFetchFailure(error.message || '')) {
      return true;
    }
    // Some runtimes surface refused connections as Unknown without a clear hint.
    if (error.code === Code.DeadlineExceeded) return true;
    return messageLooksLikeFetchFailure(error.message || '');
  }

  if (typeof error === 'object') {
    const record = error as { name?: string; code?: string | number; message?: string };
    if (record.name === 'AbortError' || record.code === 'ABORT_ERR') return false;
    if (record.name === 'TypeError' && messageLooksLikeFetchFailure(String(record.message || ''))) {
      return true;
    }
    return messageLooksLikeFetchFailure(String(record.message || ''));
  }

  return messageLooksLikeFetchFailure(String(error));
}
