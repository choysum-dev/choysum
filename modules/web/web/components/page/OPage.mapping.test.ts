// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { defineComponent, h } from 'vue';

import { mountApp, stub, stubSfc, restoreSfc, type MountAppResult } from '@/web/web/__tests__/mountApp';
import OPage from './OPage.vue';
import OBreadcrumb from '@/web/web/components/view/OBreadcrumb.vue';
import OPageIoMenu from '@/web/web/components/page/OPageIoMenu.vue';

const ioMenuStub = defineComponent({
  name: 'OPageIoMenu',
  props: {
    actionImport: { type: Boolean, default: false },
    actionExport: { type: Boolean, default: false },
    actionImportUploadHint: { type: String, default: undefined },
    actionImportColumnMapping: { type: Object, default: undefined },
    actionListRef: { type: Object, default: undefined },
    actionCompanyId: { type: String, default: undefined },
    store: { type: Object, default: undefined },
  },
  setup(props) {
    return () =>
      h('div', {
        'data-test': 'page-io-menu-stub',
        'data-import': String(!!props.actionImport),
        'data-export': String(!!props.actionExport),
        'data-hint': props.actionImportUploadHint || '',
      });
  },
});

const pageStubs = {
  ElIcon: stub('ElIcon'),
};

function exists(m: MountAppResult, sel: string): boolean {
  return !!m.q(sel);
}

function attr(m: MountAppResult, sel: string, name: string): string | null {
  return m.q(sel)?.getAttribute(name) ?? null;
}

function textOf(m: MountAppResult, sel: string): string {
  return m.q(sel)?.textContent?.trim() ?? '';
}

function hasClass(m: MountAppResult, sel: string, cls: string): boolean {
  return m.q(sel)?.classList.contains(cls) ?? false;
}

describe('OPage component', () => {
  // Suite-scoped: root beforeEach is inherited by every later test in the QJS
  // bundle and would replace real OBreadcrumb mounts in other files.
  beforeEach(() => {
    stubSfc(OBreadcrumb, stub('OBreadcrumb'));
    stubSfc(OPageIoMenu, ioMenuStub);
  });

  afterEach(() => {
    restoreSfc(OBreadcrumb);
    restoreSfc(OPageIoMenu);
  });

  test('renders title with useId-based id and binds aria-labelledby', () => {
    const mounted = mountApp(OPage as any, {
      props: { title: 'Test Title', showBreadcrumb: false },
      stubs: pageStubs,
    });
    expect(exists(mounted, 'h1.o-page__title')).toBe(true);
    expect(textOf(mounted, 'h1.o-page__title')).toBe('Test Title');
    const titleId = attr(mounted, 'h1.o-page__title', 'id');
    expect(titleId).toBeTruthy();
    expect(attr(mounted, '.o-page', 'aria-labelledby')).toBe(titleId);
    expect(attr(mounted, '.o-page', 'aria-label')).toBeNull();
    expect(attr(mounted, '.o-page', 'role')).toBe('region');
    mounted.unmount();
  });

  test('does not bind aria-labelledby when header slot is provided', () => {
    const mounted = mountApp(OPage as any, {
      props: { title: 'Test Title', showBreadcrumb: false },
      slots: {
        header: () => h('div', { class: 'custom-header' }, 'Custom'),
      },
      stubs: pageStubs,
    });
    expect(attr(mounted, '.o-page', 'aria-labelledby')).toBeNull();
    expect(attr(mounted, '.o-page', 'aria-label')).toBe('Test Title');
    expect(attr(mounted, '.o-page', 'role')).toBe('region');
    mounted.unmount();
  });

  test('sets aria-busy when loading is true', () => {
    const mounted = mountApp(OPage as any, {
      props: { title: 'Loading Page', loading: true, showBreadcrumb: false },
      stubs: pageStubs,
    });
    expect(attr(mounted, '.o-page', 'aria-busy')).toBe('true');
    mounted.unmount();
  });

  test('omits region role when title is empty', () => {
    const mounted = mountApp(OPage as any, {
      props: { showBreadcrumb: false },
      stubs: pageStubs,
    });
    expect(attr(mounted, '.o-page', 'role')).toBeNull();
    expect(exists(mounted, '.o-page__header')).toBe(false);
    mounted.unmount();
  });

  test('renders title-actions beside the title', () => {
    const mounted = mountApp(OPage as any, {
      props: { title: 'Partners', showBreadcrumb: false },
      slots: {
        'title-actions': () => h('button', { 'data-test': 'io-action' }, 'IO'),
      },
      stubs: pageStubs,
    });
    expect(exists(mounted, '.o-page__title-row')).toBe(true);
    expect(exists(mounted, '[data-test="io-action"]')).toBe(true);
    expect(textOf(mounted, 'h1.o-page__title')).toBe('Partners');
    mounted.unmount();
  });

  test('renders title-actions without a title', () => {
    const mounted = mountApp(OPage as any, {
      props: { showBreadcrumb: false },
      slots: {
        'title-actions': () => h('button', { 'data-test': 'io-only' }, 'IO'),
      },
      stubs: pageStubs,
    });
    expect(exists(mounted, '.o-page__header')).toBe(true);
    expect(exists(mounted, 'h1.o-page__title')).toBe(false);
    expect(exists(mounted, '[data-test="io-only"]')).toBe(true);
    mounted.unmount();
  });

  test('keeps title-actions when a custom header slot is provided', () => {
    const mounted = mountApp(OPage as any, {
      props: { title: 'Partners', showBreadcrumb: false },
      slots: {
        header: () => h('div', { class: 'custom-header' }, 'Custom'),
        'title-actions': () => h('button', { 'data-test': 'io-with-header' }, 'IO'),
      },
      stubs: pageStubs,
    });
    expect(exists(mounted, '.custom-header')).toBe(true);
    expect(exists(mounted, '[data-test="io-with-header"]')).toBe(true);
    expect(exists(mounted, 'h1.o-page__title')).toBe(false);
    mounted.unmount();
  });

  test('renders custom header alone without title-actions row', () => {
    const mounted = mountApp(OPage as any, {
      props: { title: 'Partners', showBreadcrumb: false },
      slots: {
        header: () => h('div', { class: 'custom-header-only' }, 'Custom'),
      },
      stubs: pageStubs,
    });
    expect(exists(mounted, '.custom-header-only')).toBe(true);
    expect(exists(mounted, '.o-page__title-row')).toBe(false);
    mounted.unmount();
  });

  test('renders breadcrumb slot in the default header', () => {
    const mounted = mountApp(OPage as any, {
      props: { title: 'Partners', showBreadcrumb: true },
      slots: {
        breadcrumb: () => h('nav', { 'data-test': 'crumb' }, 'Crumb'),
      },
      stubs: pageStubs,
    });
    expect(exists(mounted, '[data-test="crumb"]')).toBe(true);
    expect(textOf(mounted, 'h1.o-page__title')).toBe('Partners');
    mounted.unmount();
  });

  test('renders default breadcrumb when no breadcrumb slot is provided', () => {
    restoreSfc(OBreadcrumb);
    stubSfc(OBreadcrumb, {
      name: 'OBreadcrumb',
      setup() {
        return () => h('nav', { 'data-test': 'default-crumb' });
      },
    });
    const mounted = mountApp(OPage as any, {
      props: { title: 'Partners', showBreadcrumb: true },
      stubs: pageStubs,
    });
    expect(exists(mounted, '[data-test="default-crumb"]')).toBe(true);
    mounted.unmount();
  });

  test('renders breadcrumb slot even when showBreadcrumb is false', () => {
    const mounted = mountApp(OPage as any, {
      props: { title: 'Partners', showBreadcrumb: false },
      slots: {
        breadcrumb: () => h('nav', { 'data-test': 'forced-crumb' }, 'Crumb'),
      },
      stubs: pageStubs,
    });
    expect(exists(mounted, '[data-test="forced-crumb"]')).toBe(true);
    mounted.unmount();
  });

  test('renders breadcrumb alone without a title row', () => {
    restoreSfc(OBreadcrumb);
    stubSfc(OBreadcrumb, {
      name: 'OBreadcrumb',
      setup() {
        return () => h('nav', { 'data-test': 'only-crumb' });
      },
    });
    const mounted = mountApp(OPage as any, {
      props: { showBreadcrumb: true },
      stubs: pageStubs,
    });
    expect(exists(mounted, '[data-test="only-crumb"]')).toBe(true);
    expect(exists(mounted, '.o-page__title-row')).toBe(false);
    mounted.unmount();
  });

  test('renders toolbar and footer slots', () => {
    const mounted = mountApp(OPage as any, {
      props: { title: 'Partners', showBreadcrumb: false },
      slots: {
        toolbar: () => h('div', { 'data-test': 'toolbar' }, 'Tools'),
        footer: () => h('div', { 'data-test': 'footer' }, 'Foot'),
      },
      stubs: pageStubs,
    });
    expect(exists(mounted, '[data-test="toolbar"]')).toBe(true);
    expect(exists(mounted, '[data-test="footer"]')).toBe(true);
    expect(exists(mounted, '.o-page__body--with-footer')).toBe(true);
    mounted.unmount();
  });

  test('applies layout modifiers and shows the loading overlay', () => {
    const mounted = mountApp(OPage as any, {
      props: {
        title: 'Partners',
        showBreadcrumb: false,
        padding: false,
        width: 'wide',
        elevated: true,
        loading: true,
      },
      stubs: {
        ...pageStubs,
        ElIcon: {
          name: 'ElIcon',
          setup(_p: any, { slots }: any) {
            return () => h('span', { class: 'el-icon-stub' }, slots.default?.());
          },
        },
        Loading: stub('Loading'),
      },
    });
    expect(hasClass(mounted, '.o-page', 'o-page--without-padding')).toBe(true);
    expect(hasClass(mounted, '.o-page', 'o-page--wide')).toBe(true);
    expect(hasClass(mounted, '.o-page', 'o-page--elevated')).toBe(true);
    expect(hasClass(mounted, '.o-page', 'o-page--loading')).toBe(true);
    expect(exists(mounted, '.o-page__loading-mask')).toBe(true);
    mounted.unmount();
  });

  test('accepts an optional store prop without changing chrome', () => {
    const store = { storeId: 's1' };
    const mounted = mountApp(OPage as any, {
      props: { title: 'With Store', showBreadcrumb: false, store },
      stubs: pageStubs,
    });
    expect(textOf(mounted, 'h1.o-page__title')).toBe('With Store');
    expect(mounted.props.store).toEqual(store);
    mounted.unmount();
  });

  test('mounts default OPageIoMenu from action-import/export props', () => {
    const listRef = { refresh: () => undefined };
    const mounted = mountApp(OPage as any, {
      props: {
        title: 'Partners',
        showBreadcrumb: false,
        actionImport: true,
        actionExport: true,
        actionImportUploadHint: 'hint',
        actionListRef: listRef,
      },
      stubs: pageStubs,
    });
    expect(exists(mounted, '[data-test="page-io-menu-stub"]')).toBe(true);
    expect(attr(mounted, '[data-test="page-io-menu-stub"]', 'data-import')).toBe('true');
    expect(attr(mounted, '[data-test="page-io-menu-stub"]', 'data-export')).toBe('true');
    expect(attr(mounted, '[data-test="page-io-menu-stub"]', 'data-hint')).toBe('hint');
    mounted.unmount();
  });

  test('keeps title-actions slot additive beside the default IO menu', () => {
    const mounted = mountApp(OPage as any, {
      props: { title: 'Partners', showBreadcrumb: false, actionImport: true },
      slots: {
        'title-actions': () => h('button', { 'data-test': 'extra-action' }, 'Extra'),
      },
      stubs: pageStubs,
    });
    expect(exists(mounted, '[data-test="page-io-menu-stub"]')).toBe(true);
    expect(exists(mounted, '[data-test="extra-action"]')).toBe(true);
    mounted.unmount();
  });

  test('mounts OPageIoMenu beside a custom header when action-import is set', () => {
    const mounted = mountApp(OPage as any, {
      props: {
        title: 'Partners',
        showBreadcrumb: false,
        actionImport: true,
        actionExport: true,
      },
      slots: {
        header: () => h('div', { class: 'custom-header-io' }, 'Custom'),
      },
      stubs: pageStubs,
    });
    expect(exists(mounted, '.custom-header-io')).toBe(true);
    expect(exists(mounted, '[data-test="page-io-menu-stub"]')).toBe(true);
    expect(attr(mounted, '[data-test="page-io-menu-stub"]', 'data-import')).toBe('true');
    expect(attr(mounted, '[data-test="page-io-menu-stub"]', 'data-export')).toBe('true');
    mounted.unmount();
  });

  test('does not mount OPageIoMenu when action-import/export are unset', () => {
    const mounted = mountApp(OPage as any, {
      props: { title: 'Partners', showBreadcrumb: false },
      stubs: pageStubs,
    });
    expect(exists(mounted, '[data-test="page-io-menu-stub"]')).toBe(false);
    mounted.unmount();
  });
});
