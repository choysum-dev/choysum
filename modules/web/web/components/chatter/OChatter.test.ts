// @vitest-environment happy-dom
// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { flushPromises, mount } from '@vue/test-utils';
import { defineComponent, h, ref } from 'vue';
import { beforeEach, describe, expect, it, vi } from 'vitest';

const refresh = vi.fn(async () => undefined);
const entries = ref([]);
const loading = ref(false);
const error = ref<string | null>(null);

vi.mock('@/web/web/composables/chatter/useChatterTimeline', () => ({
  useChatterTimeline: () => ({ entries, loading, error, refresh }),
}));

vi.mock('@/web/web/composables/chatter/useChatterThreadTips', () => ({
  useChatterThreadTips: vi.fn(),
}));

vi.mock('@/auth/web/stores/auth', () => ({
  useAuthStore: () => ({
    currentUser: ref({ Id: 'usr_test', Name: 'Tester' }),
  }),
}));

vi.mock('@/web/web/stores/registry', () => ({
  createStoreByModel: () => ({
    Post: vi.fn(async () => ({ Id: 'msg1' })),
    SearchByRecord: vi.fn(async () => []),
    Follow: vi.fn(async () => ({ Id: 'f1' })),
    Unfollow: vi.fn(async () => 1),
  }),
}));

import OChatter from './OChatter.vue';

describe('OChatter', () => {
  beforeEach(() => {
    refresh.mockClear();
    entries.value = [];
    loading.value = false;
    error.value = null;
  });

  it('renders empty timeline state and refreshes after post', async () => {
    const wrapper = mount(OChatter, {
      props: {
        model: 'partner.Partner',
        resId: 'res_partner_1',
      },
      global: {
        stubs: {
          ElCard: defineComponent({
            setup(_, { slots }) {
              return () => h('div', [slots.header?.(), slots.default?.()]);
            },
          }),
        },
      },
    });

    expect(wrapper.text()).toContain('No activity yet');
    await wrapper.findComponent({ name: 'OChatterComposer' }).vm.$emit('posted');
    await flushPromises();
    expect(refresh).toHaveBeenCalled();
  });
});
