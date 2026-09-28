// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * Entry point for the Choysum Web module.
 *
 * Exports the application instance and the public Choy UI kit surface.
 * Domain modules must import only public Choy* names (and listed helpers)
 * from `@/web` (modules/web/index → kit) — never ui/*, internal/*, Reka, Unovis, or TanStack.
 */

export { default } from './app';
export * from './kit';
