// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * Web module install hook and public `@/web` barrel (Choy* UI kit).
 * Does not re-export the web app instance (avoids domain ↔ app cycles).
 */
export function init() {}

export * from './web/kit';
