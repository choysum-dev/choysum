// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { isConnectivityUnreachable } from '@/web/web/stores/connectivityStore';

/**
 * Skip Form save / native submit / settings save while the session is unreachable.
 * Mirrors ChoyActionButton's default connectivity gate for non-button submit paths.
 */
export function runUnlessSessionUnreachable<T>(run: () => T): T | undefined {
  if (isConnectivityUnreachable()) return undefined;
  return run();
}
