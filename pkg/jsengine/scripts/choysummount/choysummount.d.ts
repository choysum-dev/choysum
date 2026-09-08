// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

/**
 * Type surface for `@choysum/test-utils` (FE unit QJS host).
 * Runtime: esbuild aliases this to choysummount.js (see host_bundle.go).
 * Keep in sync with FrozenVTUSubsetAPIs / choysummount.js.
 */
import type { Component, Plugin } from 'vue';

export type ChoysumMountStubs =
  | true
  | string[]
  | Record<string, true | Component | Record<string, unknown>>;

export type ChoysumMountGlobalOptions = {
  plugins?: Array<Plugin | [Plugin, unknown?]>;
  provide?: Record<string | symbol, unknown>;
  components?: Record<string, Component>;
};

export type ChoysumMountOptions = {
  props?: Record<string, unknown>;
  stubs?: ChoysumMountStubs;
  shallow?: boolean;
  global?: ChoysumMountGlobalOptions;
};

export type ChoysumDOMWrapper = {
  exists(): boolean;
  element: Element | null;
  text(): string;
  trigger(eventName: string): Promise<void>;
};

export type ChoysumWrapper = {
  element: Element;
  vm: unknown;
  find(selector: string): ChoysumDOMWrapper;
  trigger(eventName: string): Promise<void>;
  text(): string;
  unmount(): void;
};

export function mount(component: Component | Record<string, unknown>, options?: ChoysumMountOptions): ChoysumWrapper;
export function shallowMount(component: Component | Record<string, unknown>, options?: ChoysumMountOptions): ChoysumWrapper;
export function flushPromises(): Promise<void>;

declare const choysumMount: {
  mount: typeof mount;
  shallowMount: typeof shallowMount;
  flushPromises: typeof flushPromises;
};

export default choysumMount;
