// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import common from './common';
import layout from './layout';
import form from './form';

/**
 * English source locale pack used as the baseline and default locale.
 */
const sourceMessages = {
  common,
  layout,
  form,
};

export default sourceMessages;

// Source locale type with precise literal inference.
export type SourceMessages = typeof sourceMessages;

// Recursive helper that widens nested string literals to string.
type DeepStringify<T> = T extends string
  ? string // Convert string literals to plain string.
  : T extends Record<string, any>
    ? { [K in keyof T]: DeepStringify<T[K]> } // Recurse through object members.
    : T; // Leave other types unchanged.

// Translation locale type with all nested string literals widened to string.
export type TranslationMessages = DeepStringify<SourceMessages>;
