// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { ChoysumError, GrpcCode, isErrorOf } from '@/core/service/error';
import { dial as defaultDial } from './model_pool';

/** Stable domain for {@link assertRecordReadable} failures. */
export const RECORD_PROBE_DOMAIN = 'core' as const;

/** Stable code when a polymorphic Model/ResId target is not Search-readable. */
export const RECORD_NOT_READABLE = 'RECORD_NOT_READABLE' as const;

/**
 * Dial used by {@link assertRecordReadable}.
 * Matches {@link dial}: returns a service instance for a full `app.Model` name.
 */
export type RecordProbeDialFn = <T = Record<string, (...args: unknown[]) => unknown>>(
  fullModelName: string
) => T;

export type AssertRecordReadableOptions = {
  /** Inject dial (tests / alternate service factory). Defaults to model_pool.dial. */
  dial?: RecordProbeDialFn;
  /** Error message when the record is not readable. */
  message?: string;
};

type SearchableService = {
  Search?: (condition: unknown, options?: unknown) => Promise<unknown>;
};

function deny(message: string, cause?: unknown): never {
  const err = new ChoysumError({
    domain: RECORD_PROBE_DOMAIN,
    code: RECORD_NOT_READABLE,
    message,
  }).withGrpcCode(GrpcCode.PermissionDenied);
  if (cause instanceof Error) {
    err.cause = cause;
  } else if (cause !== undefined && cause !== null) {
    err.cause = new Error(String(cause));
  }
  throw err;
}

/**
 * True when `err` is a core {@link RECORD_NOT_READABLE} failure from {@link assertRecordReadable}.
 */
export function isRecordNotReadableError(err: unknown): err is ChoysumError {
  return isErrorOf(err, RECORD_PROBE_DOMAIN, RECORD_NOT_READABLE);
}

/**
 * Confirms the caller can Search the business record identified by `modelFullName` + `resId`.
 *
 * Uses `dial(model).Search(['Id','=',resId])` with `fields: ['Id'], limit: 1`.
 * Does not hang on BaseModel — modules wrap with domain errors (message/audit/document).
 */
export async function assertRecordReadable(
  modelFullName: string,
  resId: string,
  opts?: AssertRecordReadableOptions
): Promise<void> {
  const model = String(modelFullName ?? '').trim();
  const id = String(resId ?? '').trim();
  const message =
    (opts?.message && String(opts.message).trim()) ||
    `record ${model || '(empty)'}/${id || '(empty)'} is not readable`;

  if (!model || !id) {
    deny(message);
  }

  try {
    const dialFn = opts?.dial || defaultDial;
    const svc = dialFn<SearchableService | null | undefined>(model);
    if (!svc || typeof svc.Search !== 'function') {
      deny(message);
    }
    const rows = await svc.Search(['Id', '=', id], { fields: ['Id'], limit: 1 });
    if (Array.isArray(rows) && rows.length > 0) return;
  } catch (err) {
    if (isRecordNotReadableError(err)) throw err;
    deny(message, err);
  }
  deny(message);
}
