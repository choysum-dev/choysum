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
      setup: ((_props: any, { slots, emit }: any) => {
        return () =>
          h(
            'button',
            {
              'data-test': 'refresh',
              type: 'button',
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
