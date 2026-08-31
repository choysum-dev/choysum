// @vitest-environment happy-dom
// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it, vi, beforeEach } from 'vitest';
import { defineComponent, h, nextTick, ref } from 'vue';
import { mount } from '@vue/test-utils';
import {
  deriveCreateRouteName,
  resolveCreateRouteLocation,
  useResolvedCreateAction,
} from './resolveCreateRoute';
import type { Router } from 'vue-router';

const routeState = { name: 'PartnerList' as string | undefined };
const resolveMock = vi.fn((loc: { name?: string }) => ({
  name: loc?.name,
  matched: loc?.name === 'PartnerCreate' ? [{ path: '/partner/partners/new' }] : [],
}));

vi.mock('vue-router', () => ({
  useRouter: () => ({ resolve: resolveMock }),
  useRoute: () => routeState,
}));

describe('deriveCreateRouteName', () => {
  it('maps List/Detail/Kanban stems to Create', () => {
    expect(deriveCreateRouteName('PartnerList')).toBe('PartnerCreate');
    expect(deriveCreateRouteName('PartnerDetail')).toBe('PartnerCreate');
    expect(deriveCreateRouteName('TokenKanban')).toBe('TokenCreate');
    expect(deriveCreateRouteName('FieldRuleList')).toBe('FieldRuleCreate');
  });

  it('keeps Create names for form New on the create screen', () => {
    expect(deriveCreateRouteName('PartnerCreate')).toBe('PartnerCreate');
  });

  it('returns undefined for non-surface names', () => {
    expect(deriveCreateRouteName('MetaModuleListTable')).toBeUndefined();
    expect(deriveCreateRouteName('MetaModuleHistory')).toBeUndefined();
    expect(deriveCreateRouteName('login')).toBeUndefined();
    expect(deriveCreateRouteName('')).toBeUndefined();
    expect(deriveCreateRouteName(undefined)).toBeUndefined();
    expect(deriveCreateRouteName(null)).toBeUndefined();
  });
});

describe('resolveCreateRouteLocation', () => {
  it('returns a named location when the Create route matches', () => {
    const router = {
      resolve: vi.fn(() => ({ name: 'PartnerCreate', matched: [{ path: '/partner/partners/new' }] })),
    } as unknown as Router;
    expect(resolveCreateRouteLocation(router, 'PartnerList')).toEqual({ name: 'PartnerCreate' });
    expect(router.resolve).toHaveBeenCalledWith({ name: 'PartnerCreate' });
  });

  it('returns undefined when resolve has no match', () => {
    const router = {
      resolve: vi.fn(() => ({ name: 'MetaModuleCreate', matched: [] })),
    } as unknown as Router;
    expect(resolveCreateRouteLocation(router, 'MetaModuleList')).toBeUndefined();
  });

  it('returns undefined when resolved name mismatches', () => {
    const router = {
      resolve: vi.fn(() => ({ name: 'Other', matched: [{ path: '/x' }] })),
    } as unknown as Router;
    expect(resolveCreateRouteLocation(router, 'PartnerList')).toBeUndefined();
  });

  it('returns undefined when derive yields nothing', () => {
    const router = { resolve: vi.fn() } as unknown as Router;
    expect(resolveCreateRouteLocation(router, 'login')).toBeUndefined();
    expect(router.resolve).not.toHaveBeenCalled();
  });

  it('returns undefined when resolve throws', () => {
    const router = {
      resolve: vi.fn(() => {
        throw new Error('No match');
      }),
    } as unknown as Router;
    expect(resolveCreateRouteLocation(router, 'PartnerList')).toBeUndefined();
  });
});

describe('useResolvedCreateAction', () => {
  beforeEach(() => {
    routeState.name = 'PartnerList';
    resolveMock.mockClear();
    resolveMock.mockImplementation((loc: { name?: string }) => ({
      name: loc?.name,
      matched: loc?.name === 'PartnerCreate' ? [{ path: '/partner/partners/new' }] : [],
    }));
  });

  function mountHook(
    prop: unknown,
    enabled?: boolean | (() => boolean)
  ) {
    let result: ReturnType<typeof useResolvedCreateAction> | undefined;
    const Host = defineComponent({
      setup() {
        result = useResolvedCreateAction(
          () => prop as any,
          enabled === undefined ? undefined : { enabled: typeof enabled === 'function' ? enabled : () => enabled }
        );
        return () => h('div', String(result?.value ?? ''));
      },
    });
    const wrapper = mount(Host);
    return { wrapper, get value() { return result!.value; } };
  }

  it('uses explicit prop when provided', () => {
    const { value, wrapper } = mountHook('/explicit/new');
    expect(value).toBe('/explicit/new');
    expect(resolveMock).not.toHaveBeenCalled();
    wrapper.unmount();
  });

  it('treats null prop as omitted and derives from route', () => {
    const { value, wrapper } = mountHook(null);
    expect(value).toEqual({ name: 'PartnerCreate' });
    wrapper.unmount();
  });

  it('treats empty string as disabled', () => {
    const { value, wrapper } = mountHook('');
    expect(value).toBeUndefined();
    wrapper.unmount();
  });

  it('derives from route name when prop is omitted', () => {
    const { value, wrapper } = mountHook(undefined);
    expect(value).toEqual({ name: 'PartnerCreate' });
    wrapper.unmount();
  });

  it('skips route fallback when enabled is false', () => {
    const { value, wrapper } = mountHook(undefined, false);
    expect(value).toBeUndefined();
    wrapper.unmount();
  });

  it('still honors explicit prop when enabled is false', () => {
    const { value, wrapper } = mountHook({ name: 'ForcedCreate' }, false);
    expect(value).toEqual({ name: 'ForcedCreate' });
    wrapper.unmount();
  });

  it('reacts when enabled getter flips on', async () => {
    const enabled = ref(false);
    let result: ReturnType<typeof useResolvedCreateAction> | undefined;
    const Host = defineComponent({
      setup() {
        result = useResolvedCreateAction(() => undefined, { enabled: () => enabled.value });
        return () => h('div');
      },
    });
    const wrapper = mount(Host);
    expect(result!.value).toBeUndefined();
    enabled.value = true;
    await nextTick();
    expect(result!.value).toEqual({ name: 'PartnerCreate' });
    wrapper.unmount();
  });
});
