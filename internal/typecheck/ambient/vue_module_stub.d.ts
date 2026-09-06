// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later
// @ts-nocheck

// Fallback `vue` module when type-fetch graphs are missing or hollow.
// Not used when real Vue package types resolve via modules/tsconfig paths.

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
  export function defineProps(props: Record<string, any>): any;
  export function defineEmits<T = any>(): any;
  export function defineEmits(emits: any): any;
  export function defineExpose(exposed: any): void;
  export function defineSlots<T = any>(): T;
  export function defineModel<T = any>(options?: any): any;
  export function defineOptions(options: any): void;
  export function withDefaults<T, D>(props: T, defaults: D): any;
  export function ref<T = any>(value?: T): any;
  export function computed<T = any>(getter: any): any;
  export function watch(...args: any[]): any;
  export function watchEffect(...args: any[]): any;
  export function reactive<T extends object>(target: T): T;
  export function nextTick(fn?: () => void): Promise<void>;
  export function onMounted(fn: () => void): void;
  export function onBeforeUnmount(fn: () => void): void;
  export function onUnmounted(fn: () => void): void;
  export function onBeforeMount(fn: () => void): void;
  export function onUpdated(fn: () => void): void;
  export function onBeforeUpdate(fn: () => void): void;
  export function inject(key: any, defaultValue?: any): any;
  export function provide(key: any, value: any): void;
  export function readonly<T>(target: T): T;
  export function shallowRef<T = any>(value?: T): any;
  export function shallowReactive<T extends object>(target: T): T;
  export function markRaw<T>(value: T): T;
  export function effectScope(detached?: boolean): any;
  export function getCurrentScope(): any;
  export function onScopeDispose(fn: () => void): void;
  export function toValue<T>(source: T): any;
  export function toRef(...args: any[]): any;
  export function toRefs<T extends object>(object: T): any;
  export function unref<T>(ref: T): any;
  export function isRef(value: any): boolean;
  export function h(...args: any[]): any;
  export function resolveComponent(...args: any[]): any;
  export function resolveDirective(...args: any[]): any;
  export function withDirectives(...args: any[]): any;
  export function withModifiers(...args: any[]): any;
  export function withKeys(...args: any[]): any;
  export function mergeProps(...args: any[]): any;
  export function cloneVNode(...args: any[]): any;
  export function isVNode(value: any): boolean;
  export function getCurrentInstance(): any;
  export function useSlots(): any;
  export function useAttrs(): any;
  export function useCssModule(name?: string): any;
  export function useCssVars(fn: any): void;
  export type App = any;
  export type Component = any;
  export type Plugin<Options extends unknown[] = unknown[]> = any;
  export type ComputedRef<T = any> = any;
  export type WritableComputedRef<T = any> = any;
  export type Ref<T = any> = any;
  export type PropType<T = any> = any;
  export type PublicProps = any;
  export type VNode = any;
  export type VNodeProps = any;
  export type Directive = any;
  export type DirectiveBinding = any;
  export type InjectionKey<T = any> = any;
  export type MaybeRefOrGetter<T = any> = any;
  export type MaybeRef<T = any> = any;
  export type ShallowRef<T = any> = any;
  export type ObjectDirective = any;
  export type ExtractPropTypes<T> = any;
  export type DefineComponent<T = any> = any;
  export function createApp(...args: any[]): any;
  export function createVNode(...args: any[]): any;
  export function render(...args: any[]): any;
  export function Transition(...args: any[]): any;
  export function TransitionGroup(...args: any[]): any;
  export function Teleport(...args: any[]): any;
  export function KeepAlive(...args: any[]): any;
  export function Suspense(...args: any[]): any;
  export function Fragment(...args: any[]): any;
  export function Text(...args: any[]): any;
  export function Comment(...args: any[]): any;
}
