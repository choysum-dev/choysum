// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * @vitest-environment happy-dom
 */

import { mount } from '@vue/test-utils';
import { createI18n } from 'vue-i18n';
import { describe, expect, it, vi } from 'vitest';

import { createTextDescriptor } from '@/core/service/i18n';
import { projectTerminologyMessages } from '../../i18n/terminology';

const mocks = vi.hoisted(() => ({
  breadcrumbs: [] as any[],
  navigateTo: vi.fn(),
}));

vi.mock('../../composables/useBreadcrumb', () => ({
  useBreadcrumb: () => mocks,
}));

import OBreadcrumb from './OBreadcrumb.vue';

describe('OBreadcrumb terminology display', () => {
  it('translates descriptors directly in the template and preserves plain strings', async () => {
    mocks.breadcrumbs = [
      {
        title: 'Settings',
        titleText: createTextDescriptor('base', 'Settings', { scope: 'base.route.settings' }),
        path: '/settings',
        clickable: false,
      },
      {
        title: 'Legacy page',
        path: '/legacy',
        clickable: false,
      },
    ];
    const i18n = createI18n({
      legacy: false,
      locale: 'en',
      missingWarn: false,
      fallbackWarn: false,
      messages: {
        en: {},
        'zh-CN': projectTerminologyMessages({
          base: { 'base.route.settings': { Settings: '设置' } },
        }),
      },
    });
    const wrapper = mount(OBreadcrumb, {
      global: {
        plugins: [i18n],
        stubs: {
          ElBreadcrumb: { template: '<nav><slot /></nav>' },
          ElBreadcrumbItem: { template: '<span><slot /></span>' },
        },
      },
    });

    expect(wrapper.text()).toContain('Settings');
    expect(wrapper.text()).toContain('Legacy page');

    i18n.global.locale.value = 'zh-CN';
    await wrapper.vm.$nextTick();
    expect(wrapper.text()).toContain('设置');
    expect(wrapper.text()).toContain('Legacy page');
  });
});
