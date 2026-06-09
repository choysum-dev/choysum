// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * Symbols for model-internal tracking helpers.
 * These symbols expose internal methods and are not intended for direct external use.
 */
export const MODEL_SYMBOLS = {
  getChangedFields: Symbol('getChangedFields'),
  getOriginalValue: Symbol('getOriginalValue'),
  hasChanged: Symbol('hasChanged'),
  resetChanges: Symbol('resetChanges'),
  collectRelationChanges: Symbol('collectRelationChanges'),
  resetRelationChanges: Symbol('resetRelationChanges'),

  // Internal-use symbol.
  PROXY: Symbol('proxy'),
};
