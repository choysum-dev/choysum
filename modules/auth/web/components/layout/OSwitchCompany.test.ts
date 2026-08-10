// @vitest-environment happy-dom
// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { mount, flushPromises } from '@vue/test-utils';
import { nextTick } from 'vue';
import { beforeEach, describe, expect, it, vi } from 'vitest';

const { authBag } = vi.hoisted(() => ({
  authBag: { state: null as null | Record<string, any> },
}));

vi.mock('@/auth/web/stores/auth', async () => {
  const { reactive } = await import('vue');
  authBag.state = reactive({
    identity: {
      metadata: {
        activeCompanyId: 'c1',
        enabledCompanyIds: ['c1'],
        allowedCompanyIds: ['c1', 'c2'],
      },
    },
    refreshToken: vi.fn(async () => {}),
    switchCompanyScope: vi.fn(async () => {}),
  });
  return {
    useAuthStore: () => authBag.state!,
  };
});

vi.mock('@/web/web/stores/registry', () => ({
  createStoreByModel: () => ({
    Search: vi.fn(async () => [
      { Id: 'c1', DisplayName: 'One' },
      { Id: 'c2', DisplayName: 'Two' },
    ]),
  }),
}));

vi.mock('@/web/web/i18n', () => ({
  createTranslate: () => ({ _t: (msg: string) => msg }),
}));

import OSwitchCompany from './OSwitchCompany.vue';

describe('OSwitchCompany open-panel draft guard', () => {
  beforeEach(() => {
    authBag.state!.identity = {
      metadata: {
        activeCompanyId: 'c1',
        enabledCompanyIds: ['c1'],
        allowedCompanyIds: ['c1', 'c2'],
      },
    };
    authBag.state!.refreshToken.mockClear();
    authBag.state!.switchCompanyScope.mockClear();
  });

  it('does not reset drafts from JWT watch while the popover is open', async () => {
    const wrapper = mount(OSwitchCompany as any, {
      global: {
        stubs: {
          'el-popover': {
            props: ['visible'],
            emits: ['update:visible'],
            template: `<div class="pop">
              <button class="open" @click="$emit('update:visible', true)" />
              <div v-if="visible"><slot /></div>
              <slot name="reference" />
            </div>`,
          },
          'el-button': {
            template: `<button class="el-btn" @click="$emit('click')"><slot /></button>`,
          },
          'el-form': { template: `<form><slot /></form>` },
          'el-form-item': { template: `<div><slot /></div>` },
          'el-select': {
            props: ['modelValue'],
            emits: ['update:modelValue', 'change', 'remove-tag'],
            template: `<div class="sel" :data-value="JSON.stringify(modelValue)"><slot /></div>`,
          },
          'el-option': true,
        },
      },
    });
    await flushPromises();

    await wrapper.find('button.open').trigger('click');
    await flushPromises();
    await nextTick();

    // Token refresh updates JWT metadata while the panel stays open.
    authBag.state!.identity = {
      metadata: {
        activeCompanyId: 'c2',
        enabledCompanyIds: ['c2'],
        allowedCompanyIds: ['c1', 'c2'],
      },
    };
    await nextTick();
    await flushPromises();

    // Drafts remain on the open-panel seed (c1) → dirty vs JWT c2 → Apply enabled.
    const apply = wrapper.find('[data-testid="company-switch-apply"]');
    expect(apply.exists()).toBe(true);
    expect((apply.element as HTMLButtonElement).disabled).toBe(false);
  });
});
