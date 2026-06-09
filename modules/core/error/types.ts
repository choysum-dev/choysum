// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * Error code format type.
 */
// export type ErrorCodePattern = Uppercase<string>;
export type ErrorCodePattern = string;

/**
 * Error creation options.
 */
export interface ErrorOptions {
  domain: string;
  code: ErrorCodePattern;
  message: string;
}
