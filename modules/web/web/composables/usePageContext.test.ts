// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { defineComponent, h } from 'vue';

import { flushPromises, mountApp } from '@/web/web/__tests__/mountApp';
import {
  provideOPageContext,
  resolvePageStore,
  useOptionalPageStore,
  useRegisterPageActionTarget,
  useResolvedOptionalPageStore,
} from './usePageContext';

describe('usePageContext', () => {
  test('provides store for descendants', () => {
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
    const { unmount } = mountApp(Parent);
    expect(seen).toBe(store);
    unmount();
  });

  test('provides store to default-slot content (OPage page pattern)', () => {
    const store = { storeId: 'slot-store' };
    let seen: unknown = null;
    const Child = defineComponent({
      setup() {
        seen = resolvePageStore(undefined, 'SlotChild');
        return () => h('div');
      },
    });
    const PageShell = defineComponent({
      setup(_, { slots }) {
        provideOPageContext({ store });
        return () => h('div', slots.default?.());
      },
    });
    const Page = defineComponent({
      setup() {
        return () => h(PageShell, null, { default: () => h(Child) });
      },
    });
    const { unmount } = mountApp(Page);
    expect(seen).toBe(store);
    unmount();
  });

  test('prefers an explicit prop store over the page store', () => {
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
    const { unmount } = mountApp(Parent);
    expect(seen).toBe(propStore);
    unmount();
  });

  test('throws when neither prop nor page store is available', () => {
    let threw: unknown;
    const Orphan = defineComponent({
      setup() {
        try {
          resolvePageStore(undefined, 'Missing');
        } catch (err) {
          threw = err;
        }
        return () => h('div');
      },
    });
    const { unmount } = mountApp(Orphan);
    expect(String((threw as Error)?.message || threw)).toMatch(/Missing requires a store/);
    unmount();
  });

  test('soft-resolves optional store from the page context', () => {
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
    const { unmount } = mountApp(Parent);
    expect(resolved!.value).toBe(pageStore);
    unmount();
  });

  test('registers and unregisters a page action target', async () => {
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
    const { unmount } = mountApp(Parent);
    expect(ctx!.actionTarget.value).toBe(target);
    unmount();
    await flushPromises();
    expect(ctx!.actionTarget.value).toBeNull();
  });

  test('skips auto-register when the view store differs from the page store', () => {
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
    const { unmount } = mountApp(Parent);
    expect(ctx!.actionTarget.value).toBeNull();
    unmount();
  });

  test('respects enabled false to opt out of registration', () => {
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
    const { unmount } = mountApp(Parent);
    expect(ctx!.actionTarget.value).toBeNull();
    unmount();
  });

  test('forces registration when enabled is true even if stores differ', () => {
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
    const { unmount } = mountApp(Parent);
    expect(ctx!.actionTarget.value).toBe(target);
    unmount();
  });

  test('no-ops unregister when the target is not the current action target', () => {
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
    const { unmount } = mountApp(Parent);
    ctx!.registerActionTarget(kept);
    ctx!.unregisterActionTarget(other);
    expect(ctx!.actionTarget.value).toBe(kept);
    unmount();
  });

  test('treats a nullish page store getter as null', () => {
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
    const { unmount } = mountApp(Parent);
    expect(seen).toBeNull();
    unmount();
  });

  test('skips registration when no page context is provided', () => {
    const store = { storeId: 'orphan' };
    const target = { refresh: () => undefined };
    const Orphan = defineComponent({
      setup() {
        useRegisterPageActionTarget({ store, target, enabled: true });
        return () => h('div');
      },
    });
    expect(() => {
      const { unmount } = mountApp(Orphan);
      unmount();
    }).not.toThrow();
  });
});
