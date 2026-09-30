// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { KeepAlive, defineComponent, h, ref } from 'vue';
import { flushPromises, mountApp, restoreSfc, stubSfc } from '@/web/web/__tests__/mountApp';
import {
  applyChoyThemePreference,
  CHOY_THEME_STORAGE_KEY,
} from '@/web/web/composables/applyChoyThemePreference';
import HomeView from './HomeView.vue';
import ChoyPage from '@/web/web/components/layout/ChoyPage.vue';
import ChoyCard from '@/web/web/components/layout/ChoyCard.vue';
import ChoyButton from '@/web/web/components/layout/ChoyButton.vue';

describe('HomeView', () => {
  beforeEach(() => {
    localStorage.removeItem(CHOY_THEME_STORAGE_KEY);
    stubSfc(ChoyPage, {
      props: { title: String, padding: Boolean, width: String },
      setup: ((_props: any, { slots }: any) => {
        return () =>
          h('div', { 'data-test': 'choy-page' }, [slots['title-actions']?.(), slots.default?.()]);
      }) as any,
    });
    stubSfc(ChoyCard, {
      props: { title: String },
      setup: ((props: any, { slots }: any) => {
        return () =>
          h('section', { 'data-test': 'choy-card', 'data-title': props.title || '' }, slots.default?.());
      }) as any,
    });
    stubSfc(ChoyButton, {
      props: { type: String, variant: String },
      emits: ['click'],
      setup: ((_props: any, { slots, attrs, emit }: any) => {
        return () =>
          h(
            'button',
            {
              'data-test': 'refresh',
              type: 'button',
              ...attrs,
              onClick: (e: MouseEvent) => emit('click', e),
            },
            slots.default?.(),
          );
      }) as any,
    });
  });

  afterEach(() => {
    restoreSfc(ChoyPage);
    restoreSfc(ChoyCard);
    restoreSfc(ChoyButton);
    localStorage.removeItem(CHOY_THEME_STORAGE_KEY);
  });

  test('renders default theme/density labels and refreshes from storage', async () => {
    const mounted = mountApp(HomeView as any);
    await flushPromises();

    expect(mounted.text()).toContain('light');
    expect(mounted.text()).toContain('comfortable');

    applyChoyThemePreference({ theme: 'dark', density: 'compact' });
    mounted.click('[data-test=refresh]');
    await flushPromises();

    expect(mounted.text()).toContain('dark');
    expect(mounted.text()).toContain('compact');
    mounted.unmount();
  });

  test('lists menu shortcuts and navigates on click when router/pinia are installed', async () => {
    const VueRouter = await import('vue-router');
    const createFeStubRouter = (VueRouter as any).createFeStubRouter;
    const { createPinia, setActivePinia } = await import('pinia');
    const { createMenuPlugin } = await import('@/core/web/menu');
    const { createI18n } = await import('vue-i18n');

    const menuPlugin = createMenuPlugin();
    menuPlugin.manager.addMenu({
      id: 'partners',
      title: 'Partners',
      path: '/partners',
    } as any);

    const pinia = createPinia();
    setActivePinia(pinia);
    const { router } = createFeStubRouter({
      route: { path: '/home', fullPath: '/home', meta: {} },
    });
    const pushes: unknown[] = [];
    const originalPush = router.push?.bind(router);
    router.push = (to: unknown) => {
      pushes.push(to);
      return originalPush ? originalPush(to) : Promise.resolve();
    };
    const i18n = createI18n({ legacy: false, locale: 'en', messages: { en: {} } });

    const mounted = mountApp(HomeView as any, {
      plugins: [menuPlugin, pinia, router, i18n],
    });
    await flushPromises();

    expect(mounted.text()).toContain('Partners');
    expect(mounted.text()).toContain('/partners');
    const btn = Array.from(mounted.el.querySelectorAll('button')).find(b =>
      (b.textContent || '').includes('Partners'),
    ) as HTMLElement | undefined;
    expect(btn).toBeTruthy();
    btn!.click();
    await flushPromises();
    expect(pushes).toContain('/partners');
    mounted.unmount();
  });

  test('resyncs theme labels when a keepAlive view is re-activated', async () => {
    const visible = ref(true);
    const Parked = defineComponent({
      name: 'ParkedView',
      setup: () => () => h('div', { 'data-test': 'parked' }),
    });
    const Host = defineComponent({
      setup() {
        return () =>
          h(KeepAlive, null, {
            default: () => (visible.value ? h(HomeView as any) : h(Parked)),
          });
      },
    });

    const mounted = mountApp(Host as any);
    await flushPromises();
    expect(mounted.text()).toContain('light');

    applyChoyThemePreference({ theme: 'dark', density: 'compact' });
    visible.value = false;
    await flushPromises();
    expect(mounted.q('[data-test=parked]')).not.toBeNull();

    visible.value = true;
    await flushPromises();
    expect(mounted.text()).toContain('dark');
    expect(mounted.text()).toContain('compact');
    mounted.unmount();
  });
});
