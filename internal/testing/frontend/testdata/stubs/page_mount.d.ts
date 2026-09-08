// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

/**
 * Type surface for `@choysum/page-mount` (FE unit QJS helper).
 * Runtime: esbuild aliases this to stubs/page_mount.js (see host_bundle.go).
 */
import type { Plugin } from 'vue';

export type PageMountOverrides = {
  route?: Record<string, unknown>;
  router?: Record<string, unknown>;
};

/** Matches ChoysumMountGlobalOptions so callers can pass the result as mount `global`. */
export type PageMountGlobal = {
  plugins: Array<Plugin | [Plugin, ...unknown[]]>;
};

export function buildPageMountGlobal(overrides?: PageMountOverrides): PageMountGlobal;

declare const pageMount: {
  buildPageMountGlobal: typeof buildPageMountGlobal;
};

export default pageMount;
