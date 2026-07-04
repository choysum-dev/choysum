// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * @vitest-environment happy-dom
 */

import { describe, expect, test } from 'vitest';
import { mount } from '@vue/test-utils';
import { createPinia } from 'pinia';
import OLayout from './OLayout.vue';
import OContent from './OContent.vue';

describe('OLayout component', () => {
  test('passes padding=false to OContent when spacing is none', () => {
    const wrapper = mount(OLayout, {
      props: {
        spacing: 'none',
        showHeader: false,
        showSidebar: false,
      },
      global: {
        plugins: [createPinia()],
        stubs: {
          'router-view': true,
          OHeader: true,
          OSidebar: true,
          OFooter: true,
        },
      },
    });

    const content = wrapper.findComponent(OContent);
    expect(content.props('padding')).toBe(false);
  });

  test('passes padding=true and paddingSize to OContent when spacing is medium', () => {
    const wrapper = mount(OLayout, {
      props: {
        spacing: 'medium',
        showHeader: false,
        showSidebar: false,
      },
      global: {
        plugins: [createPinia()],
        stubs: {
          'router-view': true,
          OHeader: true,
          OSidebar: true,
          OFooter: true,
        },
      },
    });

    const content = wrapper.findComponent(OContent);
    expect(content.props('padding')).toBe(true);
    expect(content.props('paddingSize')).toBe('medium');
  });

  test('renders with fixed-header class when fixedHeader is true', () => {
    const wrapper = mount(OLayout, {
      props: {
        fixedHeader: true,
        showHeader: true,
        showSidebar: false,
      },
      global: {
        plugins: [createPinia()],
        stubs: {
          'router-view': true,
          OHeader: true,
          OSidebar: true,
          OFooter: true,
        },
      },
    });

    expect(wrapper.find('.o-layout--fixed-header').exists()).toBe(true);
  });
});
