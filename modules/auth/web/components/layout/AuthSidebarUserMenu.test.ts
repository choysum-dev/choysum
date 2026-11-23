// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { defineComponent, h, nextTick } from 'vue';
import { createPinia, setActivePinia } from 'pinia';
import { buildPageMountGlobal } from '@choysum/page-mount';
import { flushPromises, mountApp, restoreSfc, stubSfc } from '@/web/web/__tests__/mountApp';
import { ChoySidebarProvider, closeAllDialogs, openDialog, useDialogStack } from '@/web';
import { resetDialogServiceForTests } from '@/web/web/composables/dialogService';
import { useConnectivityStore } from '@/web/web/stores/connectivityStore';
import { useLayoutStore } from '@/web/web/stores/layoutStore';
import { pinViewportWidth } from '@/web/web/stores/layoutStore/pinViewport';
import ChoyButton from '@/web/web/components/layout/ChoyButton.vue';
import { useAuthStore } from '@/auth/web/stores/auth';
import AuthSidebarUserMenu from './AuthSidebarUserMenu.vue';

/**
 * FE QuickJS host unwraps refs/computeds on setupState; assign/read plain values.
 * Functions and let-bindings (e.g. openPreferencesOnMenuClose) stay mutable.
 */
type MenuSetup = {
  userMenuOpen: boolean;
  userMenuLabel: string;
  userMenuAriaLabel: string;
  userInitials: string;
  userAvatarSrc: string;
  showFooterPill: boolean;
  showOfflineBadge: boolean;
  openPreferencesOnMenuClose: boolean;
  onMenuPreferences: (event?: Event) => void;
  onUserMenuCloseAutoFocus: (event: Event) => void;
  onMenuLogout: () => void;
  onMenuConnectivityRetry: (event?: Event) => Promise<void>;
  resolveAvatarSrc: (avatar: unknown) => Promise<string | undefined>;
};

/** Walk MountAppHost → Host → … to the AuthSidebarUserMenu setupState. */
function findMenuSetup(root: any): MenuSetup {
  const walk = (vnode: any): MenuSetup | null => {
    if (!vnode) return null;
    const comp = vnode.component;
    if (comp) {
      const type = comp.type || {};
      const name = type.name || type.__name || '';
      if (name === 'AuthSidebarUserMenu' && comp.setupState) {
        return comp.setupState as MenuSetup;
      }
      const nested = walk(comp.subTree);
      if (nested) return nested;
    }
    const children = vnode.children;
    if (Array.isArray(children)) {
      for (const child of children) {
        const nested = walk(child);
        if (nested) return nested;
      }
    } else if (children && typeof children === 'object') {
      for (const child of Object.values(children)) {
        if (Array.isArray(child)) {
          for (const item of child) {
            const nested = walk(item);
            if (nested) return nested;
          }
        } else if (child && typeof child === 'object') {
          const nested = walk(child);
          if (nested) return nested;
        }
      }
    }
    return null;
  };
  const fromRoot = root?.$?.subTree ? walk(root.$.subTree) : null;
  if (fromRoot) return fromRoot;
  throw new Error('AuthSidebarUserMenu setupState not found');
}

function seedAuthUser(overrides?: Record<string, unknown>) {
  const auth = useAuthStore() as any;
  const { identity: identityOverride, ...userFields } = overrides || {};
  auth.currentUser = {
    Username: 'admin',
    Email: 'admin@example.com',
    DisplayName: 'Admin',
    ...userFields,
  };
  auth.identity = {
    username: 'admin',
    displayName: 'Admin',
    email: 'admin@example.com',
    ...(typeof identityOverride === 'object' && identityOverride ? identityOverride : {}),
  };
}

async function mountUserMenu(opts: {
  sidebarMode: 'expanded' | 'collapsed';
  userOverrides?: Record<string, unknown>;
}) {
  pinViewportWidth(1280);
  const pinia = createPinia();
  setActivePinia(pinia);
  useConnectivityStore().resetForTests();
  resetDialogServiceForTests();
  seedAuthUser(opts.userOverrides);
  const pushCalls: unknown[] = [];
  const { plugins } = buildPageMountGlobal({
    route: { path: '/web', fullPath: '/web' },
    router: {
      push: async (loc: unknown) => {
        pushCalls.push(loc);
        return loc;
      },
    } as any,
  });
  const Host = defineComponent({
    name: 'AuthSidebarUserMenuTestHost',
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
  useLayoutStore().sidebarMode = opts.sidebarMode;
  await flushPromises();
  await nextTick();
  const ss = findMenuSetup(mounted.root);
  return { mounted, pushCalls, ss };
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
  resetDialogServiceForTests();
});

test('AuthSidebarUserMenu: expanded offline shows footer pill and no chevrons', async () => {
  const { mounted, ss } = await mountUserMenu({ sidebarMode: 'expanded' });
  useConnectivityStore().markUnreachable();
  await flushPromises();
  await nextTick();
  expect(mounted.q('[data-testid=auth-user-menu-trigger]')).not.toBeNull();
  expect(mounted.q('[data-testid=choy-connectivity-pill]')).not.toBeNull();
  expect(ss.showFooterPill).toBe(true);
  expect(ss.showOfflineBadge).toBe(false);
  expect(mounted.q('[data-testid=auth-user-menu-chevrons]')).toBeNull();
  expect(mounted.q('[data-testid=auth-user-menu-offline-badge]')).toBeNull();
  expect(ss.userMenuAriaLabel).toContain('Connection lost');
  mounted.unmount();
});

test('AuthSidebarUserMenu: expanded online hides pill and connectivity menu item', async () => {
  const { mounted, ss } = await mountUserMenu({ sidebarMode: 'expanded' });
  await flushPromises();
  await nextTick();
  expect(mounted.q('[data-testid=auth-user-menu-trigger]')).not.toBeNull();
  expect(mounted.q('[data-testid=choy-connectivity-pill]')).toBeNull();
  expect(ss.showFooterPill).toBe(false);
  expect(mounted.q('[data-testid=auth-user-menu-connectivity-retry]')).toBeNull();
  expect(ss.userMenuAriaLabel).toBe('User menu');
  mounted.unmount();
});

test('AuthSidebarUserMenu: collapsed offline shows badge, hides pill', async () => {
  const { mounted, ss } = await mountUserMenu({ sidebarMode: 'collapsed' });
  useConnectivityStore().markUnreachable();
  await flushPromises();
  await nextTick();
  expect(mounted.q('[data-testid=auth-user-menu-trigger]')).not.toBeNull();
  expect(mounted.q('[data-testid=choy-connectivity-pill]')).toBeNull();
  expect(ss.showOfflineBadge).toBe(true);
  expect(ss.showFooterPill).toBe(false);
  expect(mounted.q('[data-testid=auth-user-menu-offline-badge]')).not.toBeNull();
  mounted.unmount();
});

test('AuthSidebarUserMenu: collapsed offline menu lists connectivity retry action', async () => {
  const { mounted } = await mountUserMenu({ sidebarMode: 'collapsed' });
  useConnectivityStore().markUnreachable();
  await flushPromises();
  await nextTick();

  mounted.click('[data-testid=auth-user-menu-trigger]');
  await flushPromises();
  await nextTick();

  expect(mounted.q('[data-testid=auth-user-menu-content]')).not.toBeNull();
  const retry = mounted.q('[data-testid=auth-user-menu-connectivity-retry]');
  expect(retry).not.toBeNull();
  expect(retry?.textContent || '').toContain('Retry connection');
  expect(retry?.className || '').toContain('text-warning');
  const prefs = mounted.q('[data-testid=auth-user-menu-preferences]');
  expect(prefs).not.toBeNull();
  const content = mounted.q('[data-testid=auth-user-menu-content]') as HTMLElement;
  const items = Array.from(content.querySelectorAll('[data-slot="dropdown-menu-item"]'));
  expect(items[0]?.getAttribute('data-testid')).toBe('auth-user-menu-connectivity-retry');
  expect(items[1]?.getAttribute('data-testid')).toBe('auth-user-menu-preferences');
  mounted.unmount();
});

test('AuthSidebarUserMenu: connectivity retry keeps menu open until probe succeeds', async () => {
  const { mounted, ss } = await mountUserMenu({ sidebarMode: 'collapsed' });
  const store = useConnectivityStore();
  store.markUnreachable();
  ss.userMenuOpen = true;
  await flushPromises();

  let probed = 0;
  const orig = store.probeConnectivity.bind(store);
  store.probeConnectivity = (async () => {
    probed += 1;
    return false;
  }) as typeof store.probeConnectivity;

  try {
    const selectEv = new Event('select', { cancelable: true });
    await ss.onMenuConnectivityRetry(selectEv);
    expect(selectEv.defaultPrevented).toBe(true);
    expect(probed).toBe(1);
    expect(ss.userMenuOpen).toBe(true);
  } finally {
    store.probeConnectivity = orig;
  }

  store.probeConnectivity = (async () => {
    probed += 1;
    store.markReachable();
    return true;
  }) as typeof store.probeConnectivity;

  try {
    await ss.onMenuConnectivityRetry(new Event('select', { cancelable: true }));
    expect(probed).toBe(2);
    expect(ss.userMenuOpen).toBe(false);
    expect(store.status).toBe('reachable');
  } finally {
    store.probeConnectivity = orig;
  }
  mounted.unmount();
});

test('AuthSidebarUserMenu: preferences select opens dialog on close-auto-focus', async () => {
  const { mounted, ss } = await mountUserMenu({ sidebarMode: 'expanded' });
  ss.userMenuOpen = true;
  await flushPromises();

  const selectEv = new Event('select', { cancelable: true });
  ss.onMenuPreferences(selectEv);
  expect(selectEv.defaultPrevented).toBe(true);
  expect(ss.userMenuOpen).toBe(false);
  expect(ss.openPreferencesOnMenuClose).toBe(true);

  const focusEv = new Event('close-auto-focus', { cancelable: true });
  ss.onUserMenuCloseAutoFocus(focusEv);
  expect(focusEv.defaultPrevented).toBe(true);
  expect(ss.openPreferencesOnMenuClose).toBe(false);
  const stack = useDialogStack();
  expect(stack.value.length).toBe(1);
  expect((stack.value[0]?.component as any)?.name || (stack.value[0]?.component as any)?.__name || '').toMatch(
    /PreferencesDialog/,
  );

  // Idle close-auto-focus must not reopen.
  const idle = new Event('close-auto-focus', { cancelable: true });
  ss.onUserMenuCloseAutoFocus(idle);
  expect(idle.defaultPrevented).toBe(false);
  expect(stack.value.length).toBe(1);

  closeAllDialogs();
  mounted.unmount();
});

test('AuthSidebarUserMenu: logout navigates to logout route', async () => {
  const { mounted, pushCalls, ss } = await mountUserMenu({ sidebarMode: 'expanded' });
  ss.userMenuOpen = true;
  openDialog({
    component: defineComponent({ name: 'AuthMenuLogoutStub', setup: () => () => h('div') }),
  });
  expect(useDialogStack().value.length).toBe(1);

  ss.onMenuLogout();
  await flushPromises();
  expect(ss.userMenuOpen).toBe(false);
  expect(useDialogStack().value.length).toBe(0);
  expect(pushCalls.some(loc => (loc as any)?.name === 'logout')).toBe(true);
  mounted.unmount();
});

test('AuthSidebarUserMenu: latin two-word label yields two-letter initials', async () => {
  const { mounted, ss } = await mountUserMenu({ sidebarMode: 'expanded' });
  const auth = useAuthStore() as any;
  auth.currentUser = {
    Username: 'jane',
    Email: 'jane@example.com',
    DisplayName: 'Jane Doe',
    FirstName: 'Jane',
    LastName: 'Doe',
  };
  await flushPromises();
  await nextTick();
  expect(ss.userMenuLabel).toContain('Jane');
  expect(ss.userInitials).toBe('JD');
  mounted.unmount();
});

test('AuthSidebarUserMenu: mixed-script label uses first grapheme only', async () => {
  const { mounted, ss } = await mountUserMenu({ sidebarMode: 'expanded' });
  const auth = useAuthStore() as any;
  auth.currentUser = {
    Username: 'jane',
    Email: 'jane@example.com',
    DisplayName: 'Jane 王',
    FirstName: 'Jane',
    LastName: '王',
  };
  await flushPromises();
  await nextTick();
  expect(ss.userMenuLabel).toContain('Jane');
  // Not "J王" — second token is non-ASCII, so fall back to first grapheme.
  expect(ss.userInitials).toBe('J');
  mounted.unmount();
});

test('AuthSidebarUserMenu: falls back to token identity then Account', async () => {
  const { mounted, ss } = await mountUserMenu({
    sidebarMode: 'expanded',
    userOverrides: {
      DisplayName: '',
      Username: '',
      Email: '',
      FirstName: '',
      LastName: '',
      FullName: '',
    },
  });
  const auth = useAuthStore() as any;
  auth.currentUser = {};
  auth.identity = { username: 'token-user', displayName: '' };
  await flushPromises();
  await nextTick();
  expect(ss.userMenuLabel).toBe('token-user');

  auth.identity = {};
  await flushPromises();
  await nextTick();
  expect(ss.userMenuLabel).toBe('Account');
  mounted.unmount();
});

test('AuthSidebarUserMenu: resolves public avatar url onto the image', async () => {
  const { mounted, ss } = await mountUserMenu({ sidebarMode: 'expanded' });
  const auth = useAuthStore() as any;
  auth.currentUser = { ...auth.currentUser, Avatar: 'https://example.com/a.png' };
  await flushPromises();
  await nextTick();
  expect(ss.userAvatarSrc).toBe('https://example.com/a.png');
  mounted.unmount();
});


test('AuthSidebarUserMenu: avatar resolve rejection clears src (and ignores stale)', async () => {
  const { mounted, ss } = await mountUserMenu({ sidebarMode: 'expanded' }) as any;
  const auth = useAuthStore() as any;
  const menu = ss as MenuSetup & {
    resolveAvatarSrc: (avatar: unknown) => Promise<string | undefined>;
    avatarResolveGen: number;
  };

  let rejectSlow: (err: Error) => void = () => {};
  menu.resolveAvatarSrc = () =>
    new Promise<string | undefined>((_resolve, reject) => {
      rejectSlow = reject;
    });

  auth.currentUser = { ...auth.currentUser, Avatar: 'pending-reject' };
  await nextTick();
  // Supersede generation while the first resolve is still pending.
  menu.resolveAvatarSrc = async () => {
    throw new Error('avatar-resolve-failed');
  };
  auth.currentUser = { ...auth.currentUser, Avatar: 'will-reject' };
  await flushPromises();
  await nextTick();
  expect(menu.userAvatarSrc).toBe('');

  // Stale rejection must not clobber a later success.
  menu.resolveAvatarSrc = async () => 'https://example.com/ok.png';
  auth.currentUser = { ...auth.currentUser, Avatar: 'ok' };
  await flushPromises();
  await nextTick();
  expect(menu.userAvatarSrc).toBe('https://example.com/ok.png');
  rejectSlow(new Error('stale'));
  await flushPromises();
  await nextTick();
  expect(menu.userAvatarSrc).toBe('https://example.com/ok.png');
  mounted.unmount();
});

test('AuthSidebarUserMenu: avatar resolve race keeps the latest generation', async () => {
  const { mounted, ss } = await mountUserMenu({ sidebarMode: 'expanded' });
  const auth = useAuthStore() as any;

  auth.currentUser = { ...auth.currentUser, Avatar: 'https://example.com/slow.png' };
  await nextTick();
  auth.currentUser = { ...auth.currentUser, Avatar: 'https://example.com/fast.png' };
  await flushPromises();
  await nextTick();
  expect(ss.userAvatarSrc).toBe('https://example.com/fast.png');

  auth.currentUser = { ...auth.currentUser, Avatar: '' };
  await flushPromises();
  await nextTick();
  expect(ss.userAvatarSrc).toBe('');
  mounted.unmount();
});

test('AuthSidebarUserMenu: dialog interrupt closes the open menu', async () => {
  const { mounted, ss } = await mountUserMenu({ sidebarMode: 'expanded' });
  ss.userMenuOpen = true;
  await flushPromises();
  expect(ss.userMenuOpen).toBe(true);

  openDialog({
    component: defineComponent({ name: 'AuthMenuInterruptStub', setup: () => () => h('div') }),
    mode: 'interrupt',
  });
  await flushPromises();
  await nextTick();
  expect(ss.userMenuOpen).toBe(false);
  mounted.unmount();
});

test('AuthSidebarUserMenu: unmount unsubscribes dialog interrupt handler', async () => {
  const { mounted, ss } = await mountUserMenu({ sidebarMode: 'expanded' });
  ss.userMenuOpen = true;
  mounted.unmount();

  openDialog({
    component: defineComponent({ name: 'AuthMenuPostUnmountStub', setup: () => () => h('div') }),
    mode: 'interrupt',
  });
  await flushPromises();
});
