// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { defineComponent, h, nextTick } from 'vue';
import { createPinia, setActivePinia } from 'pinia';
import { buildPageMountGlobal } from '@choysum/page-mount';
import { flushPromises, mountApp, restoreSfc, stubSfc } from '@/web/web/__tests__/mountApp';
import { ChoySidebarProvider } from '@/web';
import { useConnectivityStore } from '@/web/web/stores/connectivityStore';
import { useLayoutStore } from '@/web/web/stores/layoutStore';
import { pinViewportWidth } from '@/web/web/stores/layoutStore/pinViewport';
import ChoyButton from '@/web/web/components/layout/ChoyButton.vue';
import { useAuthStore } from '@/auth/web/stores/auth';
import AuthSidebarUserMenu from './AuthSidebarUserMenu.vue';

function seedAuthUser() {
  const auth = useAuthStore() as any;
  auth.currentUser = {
    Username: 'admin',
    Email: 'admin@example.com',
    DisplayName: 'Admin',
  };
  auth.identity = { username: 'admin', displayName: 'Admin' };
}

async function mountUserMenu(opts: { sidebarMode: 'expanded' | 'collapsed' }) {
  pinViewportWidth(1280);
  const pinia = createPinia();
  setActivePinia(pinia);
  useConnectivityStore().resetForTests();
  seedAuthUser();
  const { plugins } = buildPageMountGlobal({
    route: { path: '/web', fullPath: '/web' },
  });
  const Host = defineComponent({
    setup() {
      return () =>
        h(
          ChoySidebarProvider as any,
          { defaultOpen: opts.sidebarMode === 'expanded' },
          { default: () => h(AuthSidebarUserMenu as any) },
        );
    },
  });
  const mounted = mountApp(Host as any, { plugins: [pinia, ...plugins] });
  // Apply after mount so persist rehydration cannot clobber the mode.
  useLayoutStore().sidebarMode = opts.sidebarMode;
  await flushPromises();
  await nextTick();
  return mounted;
}

beforeEach(() => {
  stubSfc(ChoyButton as any, {
    name: 'ChoyButton',
    inheritAttrs: false,
    props: {
      type: { type: String, default: 'button' },
      variant: { type: String, default: 'default' },
      size: { type: String, default: 'default' },
      disabled: { type: Boolean, default: false },
    },
    emits: ['click'],
    setup(props: any, { slots, emit, attrs }: any) {
      return () =>
        h(
          'button',
          {
            type: props.type || 'button',
            ...attrs,
            disabled: props.disabled || undefined,
            onClick: (e: Event) => emit('click', e),
          },
          slots.default?.(),
        );
    },
  });
});

afterEach(() => {
  restoreSfc(ChoyButton as any);
  try {
    useConnectivityStore().resetForTests();
  } catch {
    // pinia may already be torn down
  }
});

test('AuthSidebarUserMenu: expanded offline shows footer pill and no chevrons', async () => {
  const mounted = await mountUserMenu({ sidebarMode: 'expanded' });
  useConnectivityStore().markUnreachable();
  await flushPromises();
  await nextTick();
  expect(mounted.q('[data-testid=auth-user-menu-trigger]')).not.toBeNull();
  expect(mounted.q('[data-testid=choy-connectivity-pill]')).not.toBeNull();
  expect(mounted.q('[data-testid=auth-user-menu-chevrons]')).toBeNull();
  expect(mounted.q('[data-testid=auth-user-menu-offline-badge]')).toBeNull();
  mounted.unmount();
});

test('AuthSidebarUserMenu: expanded online hides pill and connectivity menu item', async () => {
  const mounted = await mountUserMenu({ sidebarMode: 'expanded' });
  await flushPromises();
  await nextTick();
  expect(mounted.q('[data-testid=auth-user-menu-trigger]')).not.toBeNull();
  expect(mounted.q('[data-testid=choy-connectivity-pill]')).toBeNull();
  expect(mounted.q('[data-testid=auth-user-menu-connectivity-retry]')).toBeNull();
  mounted.unmount();
});

test('AuthSidebarUserMenu: collapsed offline shows badge, hides pill', async () => {
  const mounted = await mountUserMenu({ sidebarMode: 'collapsed' });
  useConnectivityStore().markUnreachable();
  await flushPromises();
  await nextTick();
  expect(mounted.q('[data-testid=auth-user-menu-trigger]')).not.toBeNull();
  expect(mounted.q('[data-testid=choy-connectivity-pill]')).toBeNull();
  expect(mounted.q('[data-testid=auth-user-menu-offline-badge]')).not.toBeNull();
  mounted.unmount();
});

test('AuthSidebarUserMenu: collapsed offline menu lists connectivity retry action', async () => {
  const mounted = await mountUserMenu({ sidebarMode: 'collapsed' });
  useConnectivityStore().markUnreachable();
  await flushPromises();
  await nextTick();

  mounted.click('[data-testid=auth-user-menu-trigger]');
  await flushPromises();
  await nextTick();

  expect(mounted.q('[data-testid=auth-user-menu-content]')).not.toBeNull();
  const retry = mounted.q('[data-testid=auth-user-menu-connectivity-retry]');
  expect(retry).not.toBeNull();
  expect((retry?.textContent || '')).toContain('Retry connection');
  expect(retry?.className || '').toContain('text-warning');
  const prefs = mounted.q('[data-testid=auth-user-menu-preferences]');
  expect(prefs).not.toBeNull();
  // Retry is first; a separator then Preferences / Log Out.
  const content = mounted.q('[data-testid=auth-user-menu-content]') as HTMLElement;
  const items = Array.from(content.querySelectorAll('[data-slot="dropdown-menu-item"]'));
  expect(items[0]?.getAttribute('data-testid')).toBe('auth-user-menu-connectivity-retry');
  expect(items[1]?.getAttribute('data-testid')).toBe('auth-user-menu-preferences');
  mounted.unmount();
});
