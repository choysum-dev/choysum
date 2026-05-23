// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

declare module 'vue-virtual-scroller' {
  import { DefineComponent, VNode } from 'vue';

  /** Shared props across scroller implementations */
  interface BaseScrollerProps<T = any> {
    /** Data items rendered by the scroller. */
    items: T[];
    /** Field name used as the unique item key. */
    keyField?: string;
    /** Extra buffer size in pixels; larger values smooth scrolling at the cost of more rendering. */
    buffer?: number;
    /** Per-item height when forcing fixed-size mode in RecycleScroller. */
    itemSize?: number;
    /** Number of off-screen items to prerender for smoother reverse scrolling. */
    prerender?: number;
    /** Whether to enable debug output. */
    debug?: boolean;
    /** class */
    class?: any;
  }

  interface RecycleScrollerProps<T = any> extends BaseScrollerProps<T> {
    /** itemSize is required when using fixed-size mode. */
    itemSize: number;
  }

  interface DynamicScrollerProps<T = any> extends BaseScrollerProps<T> {
    /** Estimated minimum item height used for the initial placeholder render. */
    minItemSize?: number;
  }

  interface DynamicScrollerItemProps<T = any> {
    /** Current item. */
    item: T;
    /** index */
    index: number;
    /** Dependencies that trigger re-measurement when they change. */
    sizeDependencies?: any[];
    /** Whether the current item is active. */
    active?: boolean;
  }

  export const RecycleScroller: DefineComponent<RecycleScrollerProps, () => VNode | VNode[]>;
  export const DynamicScroller: DefineComponent<DynamicScrollerProps, () => VNode | VNode[]>;
  export const DynamicScrollerItem: DefineComponent<DynamicScrollerItemProps, () => VNode | VNode[]>;

  const _default: {
    RecycleScroller: typeof RecycleScroller;
    DynamicScroller: typeof DynamicScroller;
    DynamicScrollerItem: typeof DynamicScrollerItem;
  };
  export default _default;
}
