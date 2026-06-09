// @vitest-environment happy-dom
// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { describe, it, expect, vi } from 'vitest';
import { shallowMount } from '@vue/test-utils';

import ModuleListView from '../views/ModuleListView.vue';
import ModuleLogListView from '../views/ModuleLogListView.vue';

vi.mock('vue-router', () => ({
  useRouter: () => ({ push: vi.fn() }),
}));

vi.mock('@/auth/web/composables/usePermission', () => ({
  usePermission: () => ({
    canRoute: () => true,
    hasAction: () => true,
  }),
}));

const fakeStore = {
  state: { queryState: {}, result: undefined, selection: [], planCache: new Map() },
  setContext: vi.fn(),
  getContext: vi.fn(),
  withContext: vi.fn(),
};

const stubs = {
  OListView: { template: '<div><slot /></div>' },
  OVColumn: { template: '<div />' },
  OVarCharField: { template: '<div />' },
  ODateTimeField: { template: '<div />' },
  OJsonobjectField: { template: '<div />' },
  OIntField: { template: '<div />' },
  OSearchView: { template: '<div />' },
  ElButton: { template: '<button />' },
  ElTooltip: { template: '<div><slot /></div>' },
  ElButtonGroup: { template: '<div><slot /></div>' },
};

describe('Meta module views', () => {
  it('mounts ModuleListView', () => {
    const wrapper = shallowMount(ModuleListView, {
      props: { store: fakeStore as any },
      global: { stubs },
    });
    expect(wrapper.exists()).toBe(true);
  });

  it('mounts ModuleLogListView', () => {
    const wrapper = shallowMount(ModuleLogListView, {
      props: { store: fakeStore as any },
      global: { stubs },
    });
    expect(wrapper.exists()).toBe(true);
  });
});
