// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * @vitest-environment happy-dom
 */

import { describe, expect, test } from 'vitest';
import { mount } from '@vue/test-utils';
import OPage from './OPage.vue';

const pageStubs = {
  OBreadcrumb: true,
  // Loading spinner; Element Plus is not registered in this unit suite.
  'el-icon': true,
};

describe('OPage component', () => {
  test('renders title with useId-based id and binds aria-labelledby', () => {
    const wrapper = mount(OPage, {
      props: {
        title: 'Test Title',
        showBreadcrumb: false,
      },
      global: { stubs: pageStubs },
    });

    const titleEl = wrapper.find('h1.o-page__title');
    expect(titleEl.exists()).toBe(true);
    expect(titleEl.text()).toBe('Test Title');

    const titleId = titleEl.attributes('id');
    expect(titleId).toBeDefined();
    expect(titleId).not.toBe('');

    const region = wrapper.find('.o-page');
    expect(region.attributes('aria-labelledby')).toBe(titleId);
    expect(region.attributes('aria-label')).toBeUndefined();
    expect(region.attributes('role')).toBe('region');
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
      global: { stubs: pageStubs },
    });

    const region = wrapper.find('.o-page');
    expect(region.attributes('aria-labelledby')).toBeUndefined();
    expect(region.attributes('aria-label')).toBe('Test Title');
    expect(region.attributes('role')).toBe('region');
  });

  test('sets aria-busy when loading is true', () => {
    const wrapper = mount(OPage, {
      props: {
        title: 'Loading Page',
        loading: true,
        showBreadcrumb: false,
      },
      global: { stubs: pageStubs },
    });

    const region = wrapper.find('.o-page');
    expect(region.attributes('aria-busy')).toBe('true');
  });

  test('omits region role when title is empty', () => {
    const wrapper = mount(OPage, {
      props: {
        showBreadcrumb: false,
      },
      global: { stubs: pageStubs },
    });

    expect(wrapper.find('.o-page').attributes('role')).toBeUndefined();
    expect(wrapper.find('.o-page__header').exists()).toBe(false);
  });

  test('renders title-actions beside the title', () => {
    const wrapper = mount(OPage, {
      props: {
        title: 'Partners',
        showBreadcrumb: false,
      },
      slots: {
        'title-actions': '<button data-test="io-action">IO</button>',
      },
      global: { stubs: pageStubs },
    });

    expect(wrapper.find('.o-page__title-row').exists()).toBe(true);
    expect(wrapper.find('.o-page__title-actions [data-test="io-action"]').exists()).toBe(true);
    expect(wrapper.find('h1.o-page__title').text()).toBe('Partners');
  });

  test('renders title-actions without a title', () => {
    const wrapper = mount(OPage, {
      props: {
        showBreadcrumb: false,
      },
      slots: {
        'title-actions': '<button data-test="io-only">IO</button>',
      },
      global: { stubs: pageStubs },
    });

    expect(wrapper.find('.o-page__header').exists()).toBe(true);
    expect(wrapper.find('h1.o-page__title').exists()).toBe(false);
    expect(wrapper.find('[data-test="io-only"]').exists()).toBe(true);
  });

  test('keeps title-actions when a custom header slot is provided', () => {
    const wrapper = mount(OPage, {
      props: {
        title: 'Partners',
        showBreadcrumb: false,
      },
      slots: {
        header: '<div class="custom-header">Custom</div>',
        'title-actions': '<button data-test="io-with-header">IO</button>',
      },
      global: { stubs: pageStubs },
    });

    expect(wrapper.find('.custom-header').exists()).toBe(true);
    expect(wrapper.find('.o-page__title-actions [data-test="io-with-header"]').exists()).toBe(true);
    expect(wrapper.find('h1.o-page__title').exists()).toBe(false);
  });

  test('renders custom header alone without title-actions row', () => {
    const wrapper = mount(OPage, {
      props: {
        title: 'Partners',
        showBreadcrumb: false,
      },
      slots: {
        header: '<div class="custom-header-only">Custom</div>',
      },
      global: { stubs: pageStubs },
    });

    expect(wrapper.find('.custom-header-only').exists()).toBe(true);
    expect(wrapper.find('.o-page__title-row').exists()).toBe(false);
  });

  test('renders breadcrumb slot in the default header', () => {
    const wrapper = mount(OPage, {
      props: {
        title: 'Partners',
        showBreadcrumb: true,
      },
      slots: {
        breadcrumb: '<nav data-test="crumb">Crumb</nav>',
      },
      global: { stubs: pageStubs },
    });

    expect(wrapper.find('[data-test="crumb"]').exists()).toBe(true);
    expect(wrapper.find('h1.o-page__title').text()).toBe('Partners');
  });

  test('renders default breadcrumb when no breadcrumb slot is provided', () => {
    const wrapper = mount(OPage, {
      props: {
        title: 'Partners',
        showBreadcrumb: true,
      },
      global: { stubs: { ...pageStubs, OBreadcrumb: { template: '<nav data-test="default-crumb" />' } } },
    });

    expect(wrapper.find('[data-test="default-crumb"]').exists()).toBe(true);
  });

  test('renders breadcrumb slot even when showBreadcrumb is false', () => {
    const wrapper = mount(OPage, {
      props: {
        title: 'Partners',
        showBreadcrumb: false,
      },
      slots: {
        breadcrumb: '<nav data-test="forced-crumb">Crumb</nav>',
      },
      global: { stubs: pageStubs },
    });

    expect(wrapper.find('[data-test="forced-crumb"]').exists()).toBe(true);
  });

  test('renders breadcrumb alone without a title row', () => {
    const wrapper = mount(OPage, {
      props: {
        showBreadcrumb: true,
      },
      global: { stubs: { ...pageStubs, OBreadcrumb: { template: '<nav data-test="only-crumb" />' } } },
    });

    expect(wrapper.find('[data-test="only-crumb"]').exists()).toBe(true);
    expect(wrapper.find('.o-page__title-row').exists()).toBe(false);
  });

  test('renders toolbar and footer slots', () => {
    const wrapper = mount(OPage, {
      props: {
        title: 'Partners',
        showBreadcrumb: false,
      },
      slots: {
        toolbar: '<div data-test="toolbar">Tools</div>',
        footer: '<div data-test="footer">Foot</div>',
      },
      global: { stubs: pageStubs },
    });

    expect(wrapper.find('[data-test="toolbar"]').exists()).toBe(true);
    expect(wrapper.find('[data-test="footer"]').exists()).toBe(true);
    expect(wrapper.find('.o-page__body--with-footer').exists()).toBe(true);
  });

  test('applies layout modifiers and shows the loading overlay', () => {
    const wrapper = mount(OPage, {
      props: {
        title: 'Partners',
        showBreadcrumb: false,
        padding: false,
        width: 'wide',
        elevated: true,
        loading: true,
      },
      global: {
        stubs: {
          ...pageStubs,
          'el-icon': { template: '<span class="el-icon-stub"><slot /></span>' },
          Loading: true,
        },
      },
    });

    const root = wrapper.find('.o-page');
    expect(root.classes()).toContain('o-page--without-padding');
    expect(root.classes()).toContain('o-page--wide');
    expect(root.classes()).toContain('o-page--elevated');
    expect(root.classes()).toContain('o-page--loading');
    expect(wrapper.find('.o-page__loading-mask').exists()).toBe(true);
  });
});
