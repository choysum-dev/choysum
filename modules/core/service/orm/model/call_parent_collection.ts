// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * Invoke a parent collection static with `self` as `this`.
 *
 * Prefer this over `super.M(...)` in HC2 overrides: TypeScript types `super.M`'s
 * `this` as the parent ctor, which collapses open `C` / {@link RowOf}`<C>` to BaseModel
 * and breaks `QueryCondition` / return projection assignability.
 */
export function callParentCollection<R>(method: (...args: never[]) => R, self: unknown, args: readonly unknown[]): R {
  return Reflect.apply(method as (...args: unknown[]) => R, self, args as unknown[]);
}
