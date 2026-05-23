// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { asObjectRecord } from '../../../utils/object';

export type ValidationBypassCapable = {
  withValidationBypass?: <R>(fn: () => Promise<R>) => Promise<R>;
};

export function getRuntimeErrorMessage(error: unknown): string {
  const root = asObjectRecord(error);
  const nested = asObjectRecord(root?.error);
  const message = nested?.message ?? root?.message;
  if (typeof message === 'string' && message.trim()) return message;
  return String(error);
}

export async function runWithValidationBypass<R>(repo: ValidationBypassCapable, fn: () => Promise<R>): Promise<R> {
  if (typeof repo.withValidationBypass === 'function') {
    return await repo.withValidationBypass(fn);
  }
  return await fn();
}
