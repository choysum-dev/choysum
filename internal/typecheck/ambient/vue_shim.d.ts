// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later
// @ts-nocheck

// Ambient for Vue JSX when real `vue` package types are available.
// Do not declare `module "vue"` (shadows ref/computed/watch) or `module "*.vue"`
// (shadows named exports such as ViewMode from service-script overlays).

declare module "vue/jsx-runtime" {
  namespace JSX {
    interface IntrinsicElements {
      [elemName: string]: any;
    }
  }
}
