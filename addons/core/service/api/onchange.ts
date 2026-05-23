// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

export { Onchange } from '../orm/decorator/onchange';
export type { OnchangeContext, OnchangeResult } from '../runtime/onchange/types';

/**
 * PreviewModel:
 * Declares the type of this inside Onchange preview contexts while hiding dangerous methods.
 * TypeScript surfaces @deprecated warnings, and the runtime Proxy enforces the same restriction again.
 */
export type PreviewModel<T> = Omit<T, 'update' | 'delete' | 'reload'> & {
  /** @deprecated update() is blocked in preview context. */ update(): never;
  /** @deprecated delete() is blocked in preview context. */ delete(): never;
  /** @deprecated reload() is blocked in preview context. */ reload(): never;
};
