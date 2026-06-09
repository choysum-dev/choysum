// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { ChoysumError } from '../../error';

export interface UIErrorState {
  code: string;
  message: string;
  details?: Record<string, string>;
}

export function toUIErrorState(error: unknown, fallbackMessage = 'Unexpected error'): UIErrorState {
  if (error instanceof ChoysumError) {
    return {
      code: error.code,
      message: error.message,
      details: error.metadata,
    };
  }
  if (error instanceof Error) {
    return {
      code: 'UNKNOWN',
      message: error.message,
    };
  }
  return {
    code: 'UNKNOWN',
    message: fallbackMessage,
  };
}
