// @vitest-environment happy-dom
// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it } from 'vitest';
import { computed, defineComponent, h, provide } from 'vue';
import { mount } from '@vue/test-utils';
import {
  OPageContextKey,
  provideOPageContext,
  resolvePageStore,
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
        provide(OPageContextKey, { store: computed(() => pageStore) });
        return () => h(Child);
      },
    });
    mount(Parent);
    expect(resolved!.value).toBe(pageStore);
  });
});
