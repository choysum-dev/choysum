// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * Generates a unique error ID.
 * Compatible with QuickJS and browser environments.
 */
export function generateErrorId(): string {
  // Use the injected $choysum.xid in QuickJS runtimes.
  if (typeof $choysum !== 'undefined' && $choysum.xid && typeof $choysum.xid.New === 'function') {
    return $choysum.xid.New();
  }

  // Use a simplified fallback in browsers.
  return 'err_' + Math.random().toString(36).substring(2, 10) + Date.now().toString(36);
}

/**
 * Validates the error code format.
 */
export function validateErrorCode(code: string): asserts code is Uppercase<string> {
  if (!/^[A-Z_]+$/.test(code)) {
    throw new Error(`Invalid error code format: "${code}". Only uppercase letters (A-Z) and underscores (_) are allowed.`);
  }
}
