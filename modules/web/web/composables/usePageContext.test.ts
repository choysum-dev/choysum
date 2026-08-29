// @vitest-environment happy-dom
// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it } from 'vitest';
import { defineComponent, h } from 'vue';
import { mount } from '@vue/test-utils';
import {
  provideOPageContext,
  resolvePageStore,
  useOptionalPageStore,
  useRegisterPageActionTarget,
  useResolvedOptionalPageStore,
} from './usePageContext';

describe('usePageContext', () => {
  it('provides store for descendants', () => {
    const store = { storeId: 'page-store' };
    let seen: unknown = null;
    const Child = defineComponent({
      setup() {
        seen = resolvePageStore(undefined, 'Child');
        return () => h('div');
      },
    });
    const Parent = defineComponent({
      setup() {
        provideOPageContext({ store });
        return () => h(Child);
      },
    });
    mount(Parent);
    expect(seen).toBe(store);
  });

  it('prefers an explicit prop store over the page store', () => {
    const pageStore = { storeId: 'page' };
    const propStore = { storeId: 'prop' };
    let seen: unknown = null;
    const Child = defineComponent({
      setup() {
        seen = resolvePageStore(propStore, 'Child');
        return () => h('div');
      },
    });
    const Parent = defineComponent({
      setup() {
        provideOPageContext({ store: pageStore });
        return () => h(Child);
      },
    });
    mount(Parent);
    expect(seen).toBe(propStore);
  });

  it('throws when neither prop nor page store is available', () => {
    expect(() => resolvePageStore(undefined, 'Missing')).toThrow(/Missing requires a store/);
  });

  it('soft-resolves optional store from the page context', () => {
    const pageStore = { storeId: 'soft' };
    let resolved: { value: unknown } | null = null;
    const Child = defineComponent({
      setup() {
        resolved = useResolvedOptionalPageStore(() => undefined);
        return () => h('div');
      },
    });
    const Parent = defineComponent({
      setup() {
        provideOPageContext({ store: pageStore });
        return () => h(Child);
      },
    });
    mount(Parent);
    expect(resolved!.value).toBe(pageStore);
  });

  it('registers and unregisters a page action target', async () => {
    const store = { storeId: 'page' };
    const target = { refresh: () => undefined, selectedItems: [] as Array<{ Id?: string }> };
    let ctx: ReturnType<typeof provideOPageContext> | null = null;
    const Child = defineComponent({
      setup() {
        useRegisterPageActionTarget({ store, target });
        return () => h('div');
      },
    });
    const Parent = defineComponent({
      setup() {
        ctx = provideOPageContext({ store });
        return () => h(Child);
      },
    });
    const wrapper = mount(Parent);
    expect(ctx!.actionTarget.value).toBe(target);
    wrapper.unmount();
    expect(ctx!.actionTarget.value).toBeNull();
  });

  it('skips auto-register when the view store differs from the page store', () => {
    const pageStore = { storeId: 'page' };
    const viewStore = { storeId: 'other' };
    const target = { refresh: () => undefined };
    let ctx: ReturnType<typeof provideOPageContext> | null = null;
    const Child = defineComponent({
      setup() {
        useRegisterPageActionTarget({ store: viewStore, target });
        return () => h('div');
      },
    });
    const Parent = defineComponent({
      setup() {
        ctx = provideOPageContext({ store: pageStore });
        return () => h(Child);
      },
    });
    mount(Parent);
    expect(ctx!.actionTarget.value).toBeNull();
  });

  it('respects enabled false to opt out of registration', () => {
    const store = { storeId: 'page' };
    const target = { refresh: () => undefined };
    let ctx: ReturnType<typeof provideOPageContext> | null = null;
    const Child = defineComponent({
      setup() {
        useRegisterPageActionTarget({ store, target, enabled: false });
        return () => h('div');
      },
    });
    const Parent = defineComponent({
      setup() {
        ctx = provideOPageContext({ store });
        return () => h(Child);
      },
    });
    mount(Parent);
    expect(ctx!.actionTarget.value).toBeNull();
  });

  it('forces registration when enabled is true even if stores differ', () => {
    const pageStore = { storeId: 'page' };
    const viewStore = { storeId: 'other' };
    const target = { refresh: () => undefined };
    let ctx: ReturnType<typeof provideOPageContext> | null = null;
    const Child = defineComponent({
      setup() {
        useRegisterPageActionTarget({ store: viewStore, target, enabled: true });
        return () => h('div');
      },
    });
    const Parent = defineComponent({
      setup() {
        ctx = provideOPageContext({ store: pageStore });
        return () => h(Child);
      },
    });
    mount(Parent);
    expect(ctx!.actionTarget.value).toBe(target);
  });

  it('no-ops unregister when the target is not the current action target', () => {
    const store = { storeId: 'page' };
    const kept = { refresh: () => undefined };
    const other = { refresh: () => undefined };
    let ctx: ReturnType<typeof provideOPageContext> | null = null;
    const Parent = defineComponent({
      setup() {
        ctx = provideOPageContext({ store });
        return () => h('div');
      },
    });
    mount(Parent);
    ctx!.registerActionTarget(kept);
    ctx!.unregisterActionTarget(other);
    expect(ctx!.actionTarget.value).toBe(kept);
  });

  it('treats a nullish page store getter as null', () => {
    let seen: unknown = 'unset';
    const Child = defineComponent({
      setup() {
        seen = useOptionalPageStore().value;
        return () => h('div');
      },
    });
    const Parent = defineComponent({
      setup() {
        provideOPageContext({ store: () => undefined });
        return () => h(Child);
      },
    });
    mount(Parent);
    expect(seen).toBeNull();
  });

  it('skips registration when no page context is provided', () => {
    const store = { storeId: 'orphan' };
    const target = { refresh: () => undefined };
    const Orphan = defineComponent({
      setup() {
        useRegisterPageActionTarget({ store, target, enabled: true });
        return () => h('div');
      },
    });
    expect(() => mount(Orphan)).not.toThrow();
  });
});
