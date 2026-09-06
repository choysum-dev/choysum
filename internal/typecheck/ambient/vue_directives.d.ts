// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later
// @ts-nocheck

// Vue GlobalComponents / GlobalDirectives for Go-native typecheck without
// node_modules.
//
// type-fetch .d.ts files load @vue/* via relative paths, so upstream
// `declare module '@vue/runtime-core'` / `https://esm.sh/...` augmentations do
// not merge into import('vue').GlobalComponents. Element Plus global.d.ts is
// also unavailable without npm. This module augmentation supplies built-ins,
// a permissive component index (el-button, …), and Choysum's v-action.
//
// Must be a module (export {}) so this merges into package "vue".

export {};

declare module "vue" {
  export interface GlobalComponents {
    Transition: any;
    TransitionGroup: any;
    Teleport: any;
    KeepAlive: any;
    Suspense: any;
    BaseTransition: any;
    // App-registered / Element Plus template tags (el-icon, ElButton, …).
    [name: string]: any;
  }

  export interface GlobalDirectives {
    vShow: any;
    vOnce: any;
    vMemo: any;
    vText: any;
    vHtml: any;
    // Registered in modules/web/web/directives (v-action).
    vAction: any;
    [name: string]: any;
  }
}
