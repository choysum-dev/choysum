// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/** Representation (C): treat nullish, [], and {} as absent condition. */
export function asPresentCondition<T>(value?: T | null): T | undefined {
  if (value == null) return undefined;
  if (Array.isArray(value) && value.length === 0) return undefined;
  if (typeof value === 'object' && !Array.isArray(value) && Object.keys(value as object).length === 0) return undefined;
  return value as T;
}
