// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * @vitest-environment happy-dom
 */

import { describe, expect, test } from 'vitest';
import { mount } from '@vue/test-utils';
import OPage from './OPage.vue';

describe('OPage component', () => {
  test('renders title with useId-based id and binds aria-labelledby', () => {
    const wrapper = mount(OPage, {
      props: {
        title: 'Test Title',
        showBreadcrumb: false,
      },
      global: {
        stubs: {
          OBreadcrumb: true,
        },
      },
    });

    const titleEl = wrapper.find('h1.o-page__title');
    expect(titleEl.exists()).toBe(true);
    expect(titleEl.text()).toBe('Test Title');

    const titleId = titleEl.attributes('id');
    expect(titleId).toBeDefined();
    expect(titleId).not.toBe('');

    const region = wrapper.find('.o-page');
    expect(region.attributes('aria-labelledby')).toBe(titleId);
  });

  test('does not bind aria-labelledby when header slot is provided', () => {
    const wrapper = mount(OPage, {
      props: {
        title: 'Test Title',
        showBreadcrumb: false,
      },
      slots: {
        header: '<div class="custom-header">Custom</div>',
      },
      global: {
        stubs: {
          OBreadcrumb: true,
        },
      },
    });

    const region = wrapper.find('.o-page');
    expect(region.attributes('aria-labelledby')).toBeUndefined();
  });

  test('sets aria-busy when loading is true', () => {
    const wrapper = mount(OPage, {
      props: {
        title: 'Loading Page',
        loading: true,
        showBreadcrumb: false,
      },
      global: {
        stubs: {
          OBreadcrumb: true,
        },
      },
    });

    const region = wrapper.find('.o-page');
    expect(region.attributes('aria-busy')).toBe('true');
  });
});
