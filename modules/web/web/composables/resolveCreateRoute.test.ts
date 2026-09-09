// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { defineComponent, h, nextTick, ref } from 'vue';
import { createMemoryHistory, createRouter, type Router } from 'vue-router';

import { fnRecorder, mountApp } from '@/web/web/__tests__/mountApp';
import {
  deriveCreateRouteName,
  resolveCreateRouteLocation,
  useResolvedCreateAction,
} from './resolveCreateRoute';

const EmptyRoute = { render: () => h('div') };

describe('deriveCreateRouteName', () => {
  test('maps List/Detail/Kanban stems to Create', () => {
    expect(deriveCreateRouteName('PartnerList')).toBe('PartnerCreate');
    expect(deriveCreateRouteName('PartnerDetail')).toBe('PartnerCreate');
    expect(deriveCreateRouteName('TokenKanban')).toBe('TokenCreate');
    expect(deriveCreateRouteName('FieldRuleList')).toBe('FieldRuleCreate');
  });

  test('keeps Create names for form New on the create screen', () => {
    expect(deriveCreateRouteName('PartnerCreate')).toBe('PartnerCreate');
  });

  test('returns undefined for non-surface names', () => {
    expect(deriveCreateRouteName('MetaModuleListTable')).toBeUndefined();
    expect(deriveCreateRouteName('MetaModuleHistory')).toBeUndefined();
    expect(deriveCreateRouteName('login')).toBeUndefined();
    expect(deriveCreateRouteName('')).toBeUndefined();
    expect(deriveCreateRouteName(undefined)).toBeUndefined();
    expect(deriveCreateRouteName(null)).toBeUndefined();
  });
});

describe('resolveCreateRouteLocation', () => {
  test('returns a named location when the Create route matches', () => {
    const resolve = fnRecorder(() => ({ name: 'PartnerCreate', matched: [{ path: '/partner/partners/new' }] }));
    const router = { resolve } as unknown as Router;
    expect(resolveCreateRouteLocation(router, 'PartnerList')).toEqual({ name: 'PartnerCreate' });
    expect(resolve.calls).toEqual([[{ name: 'PartnerCreate' }]]);
  });

  test('returns undefined when resolve has no match', () => {
    const router = {
      resolve: fnRecorder(() => ({ name: 'MetaModuleCreate', matched: [] })),
    } as unknown as Router;
    expect(resolveCreateRouteLocation(router, 'MetaModuleList')).toBeUndefined();
  });

  test('returns undefined when resolved name mismatches', () => {
    const router = {
      resolve: fnRecorder(() => ({ name: 'Other', matched: [{ path: '/x' }] })),
    } as unknown as Router;
    expect(resolveCreateRouteLocation(router, 'PartnerList')).toBeUndefined();
  });

  test('returns undefined when derive yields nothing', () => {
    const resolve = fnRecorder();
    const router = { resolve } as unknown as Router;
    expect(resolveCreateRouteLocation(router, 'login')).toBeUndefined();
    expect(resolve.calls).toEqual([]);
  });

  test('returns undefined when resolve throws', () => {
    const router = {
      resolve: fnRecorder(() => {
        throw new Error('No match');
      }),
    } as unknown as Router;
    expect(resolveCreateRouteLocation(router, 'PartnerList')).toBeUndefined();
  });
});

describe('useResolvedCreateAction', () => {
  function makeRouter(initialName = 'PartnerList') {
    const router = createRouter({
      history: createMemoryHistory(),
      routes: [
        { path: '/', name: 'home', component: EmptyRoute },
        { path: '/partners', name: 'PartnerList', component: EmptyRoute },
        { path: '/partners/new', name: 'PartnerCreate', component: EmptyRoute },
        { path: '/other', name: 'Other', component: EmptyRoute },
      ],
    });
    return { router, initialName };
  }

  async function mountHook(
    prop: unknown,
    enabled?: boolean | (() => boolean),
    initialName = 'PartnerList'
  ) {
    const { router } = makeRouter(initialName);
    await router.push({ name: initialName });
    await router.isReady();

    let result: ReturnType<typeof useResolvedCreateAction> | undefined;
    const Host = defineComponent({
      setup() {
        result = useResolvedCreateAction(
          () => prop as any,
          enabled === undefined
            ? undefined
            : { enabled: typeof enabled === 'function' ? enabled : () => enabled }
        );
        return () => h('div', String(result?.value ?? ''));
      },
    });
    const mounted = mountApp(Host, { plugins: [router] });
    return {
      ...mounted,
      get value() {
        return result!.value;
      },
    };
  }

  test('uses explicit prop when provided', async () => {
    const { value, unmount } = await mountHook('/explicit/new');
    expect(value).toBe('/explicit/new');
    unmount();
  });

  test('treats null prop as omitted and derives from route', async () => {
    const { value, unmount } = await mountHook(null);
    expect(value).toEqual({ name: 'PartnerCreate' });
    unmount();
  });

  test('treats empty string as disabled', async () => {
    const { value, unmount } = await mountHook('');
    expect(value).toBeUndefined();
    unmount();
  });

  test('derives from route name when prop is omitted', async () => {
    const { value, unmount } = await mountHook(undefined);
    expect(value).toEqual({ name: 'PartnerCreate' });
    unmount();
  });

  test('skips route fallback when enabled is false', async () => {
    const { value, unmount } = await mountHook(undefined, false);
    expect(value).toBeUndefined();
    unmount();
  });

  test('still honors explicit prop when enabled is false', async () => {
    const { value, unmount } = await mountHook({ name: 'ForcedCreate' }, false);
    expect(value).toEqual({ name: 'ForcedCreate' });
    unmount();
  });

  test('reacts when enabled getter flips on', async () => {
    const enabled = ref(false);
    const router = createRouter({
      history: createMemoryHistory(),
      routes: [
        { path: '/partners', name: 'PartnerList', component: EmptyRoute },
        { path: '/partners/new', name: 'PartnerCreate', component: EmptyRoute },
      ],
    });
    await router.push({ name: 'PartnerList' });
    await router.isReady();

    let result: ReturnType<typeof useResolvedCreateAction> | undefined;
    const Host = defineComponent({
      setup() {
        result = useResolvedCreateAction(() => undefined, { enabled: () => enabled.value });
        return () => h('div');
      },
    });
    const { unmount } = mountApp(Host, { plugins: [router] });
    expect(result!.value).toBeUndefined();
    enabled.value = true;
    await nextTick();
    expect(result!.value).toEqual({ name: 'PartnerCreate' });
    unmount();
  });
});
