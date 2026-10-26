// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/** Primitive snapshot of column declaration props used to skip no-op re-registers. */
export function tableColumnPropSignature(fields: Record<string, unknown>): string {
  return JSON.stringify(fields);
}

/** True when an existing column already matches this signature. */
export function skipTableColumnReregister(
  hasCurrent: boolean,
  signature: string,
  lastSignature: string,
): boolean {
  return hasCurrent && signature === lastSignature;
}

export function onTableColumnSignatureChange(
  signature: string,
  lastSignature: string,
  replace: () => void,
): void {
  if (signature === lastSignature) {
    return;
  }
  replace();
}
