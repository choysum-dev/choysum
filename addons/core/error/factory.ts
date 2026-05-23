// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { ChoysumError, isErrorOf } from './error';
import type { ErrorCodePattern } from './types';

/**
 * Creates domain-specific error helpers.
 *
 * @param domain Error domain name.
 * @returns A set of helpers bound to that domain.
 */
export function createDomainErrorHandlers<T extends ErrorCodePattern = ErrorCodePattern>(domain: string) {
  /**
   * Creates an error for this domain.
   */
  function newError(options: { code: T; message: string }): ChoysumError {
    return new ChoysumError({
      domain,
      code: options.code,
      message: options.message,
    });
  }

  /**
   * Wraps an error for this domain.
   */
  function wrapError(
    err: unknown,
    options: {
      code: T;
      message: string;
    },
    force?: boolean
  ): ChoysumError {
    return ChoysumError.wrap(
      err,
      {
        domain,
        code: options.code,
        message: options.message,
      },
      force
    );
  }

  /**
   * Checks whether an error belongs to this domain.
   */
  function isError(err: unknown, code?: T): err is ChoysumError {
    return isErrorOf(err, domain, code);
  }

  return {
    domain,
    newError,
    wrapError,
    isError,
  } as const;
}

/**
 * Global error helper factory.
 */
export const ErrorFactory = {
  createDomainHandlers: createDomainErrorHandlers,
} as const;
