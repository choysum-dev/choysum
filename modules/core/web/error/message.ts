// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { ChoysumError } from '../../error';

export function errorMessageKey(error: unknown, fallback = 'error.unknown'): string {
  if (error instanceof ChoysumError) {
    return `error.${error.domain}.${error.code}`;
  }
  return fallback;
}
