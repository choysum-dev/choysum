// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

// Minimal ambient modules so service scripts can resolve `vue` / `*.vue` without
// installing node_modules/vue into every fixture.
// ComponentPublicInstance intentionally has no string index signature so
// template property access (__VLS_ctx.foo) still typechecks.

declare module "vue" {
  export type ShallowUnwrapRef<T> = T;
  export interface ComponentPublicInstance {
    $el: any;
    $refs: Record<string, any>;
    $slots: Record<string, any>;
    $emit: (...args: any[]) => void;
    $props: Record<string, any>;
    $attrs: Record<string, any>;
  }
  export type GlobalComponents = Record<string, any>;
  export type GlobalDirectives = Record<string, any>;
  export function defineComponent(options: any): any;
  export function defineProps<T = any>(): T;
  export function defineEmits<T = any>(): any;
  export function defineExpose(exposed: any): void;
  export function defineSlots<T = any>(): T;
  export function defineModel<T = any>(options?: any): any;
  export function defineOptions(options: any): void;
  export function withDefaults<T, D>(props: T, defaults: D): any;
}

declare module "vue/jsx-runtime" {
  namespace JSX {
    interface IntrinsicElements {
      [elemName: string]: any;
    }
  }
}

declare module "*.vue" {
  const component: any;
  export default component;
}
