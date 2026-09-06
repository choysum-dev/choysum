// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later
// @ts-nocheck

// Minimal `vue` module for fixture ScopeAll runs without node_modules/vue or
// type-fetch path assets. Not used when real Vue types resolve.

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
  export function ref<T = any>(value?: T): any;
  export function computed<T = any>(getter: any): any;
  export function watch(...args: any[]): any;
  export function reactive<T extends object>(target: T): T;
  export function nextTick(fn?: () => void): Promise<void>;
  export function onMounted(fn: () => void): void;
  export function onBeforeUnmount(fn: () => void): void;
  export function inject(key: any, defaultValue?: any): any;
  export function provide(key: any, value: any): void;
  export function readonly<T>(target: T): T;
  export function shallowRef<T = any>(value?: T): any;
  export function markRaw<T>(value: T): T;
  export function effectScope(detached?: boolean): any;
  export function getCurrentScope(): any;
  export function onScopeDispose(fn: () => void): void;
  export function toValue<T>(source: T): any;
  export type App = any;
  export type Component = any;
  export type Plugin = any;
  export type ComputedRef<T = any> = any;
  export type Ref<T = any> = any;
  export type Directive = any;
  export type DirectiveBinding = any;
  export type InjectionKey<T = any> = any;
  export type MaybeRefOrGetter<T = any> = any;
  export type ShallowRef<T = any> = any;
  export type ObjectDirective = any;
  export function createApp(...args: any[]): any;
}
