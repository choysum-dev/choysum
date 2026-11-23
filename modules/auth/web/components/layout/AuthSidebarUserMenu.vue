<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <!--
    Footer identity row: avatar+name opens the user menu; ConnectivityPill (expanded
    + offline only) is a sibling one-click retry. Collapsed: badge + menu retry.
  -->
  <div
    class="auth-sidebar-user-menu flex w-full min-w-0 items-center gap-0.5"
    data-testid="auth-sidebar-user-menu"
  >
    <div class="auth-sidebar-user-menu__identity min-w-0 flex-1">
      <ChoyCollapsedIdentityTipHost :label="userIdentityTooltip">
        <ChoyDropdownMenu v-model:open="userMenuOpen">
          <ChoyDropdownMenuTrigger as-child>
            <ChoySidebarMenuButton
              size="lg"
              type="button"
              class="auth-sidebar-user-menu__trigger w-full min-w-0"
              :aria-label="userMenuAriaLabel"
              data-testid="auth-user-menu-trigger"
            >
              <span class="relative inline-flex size-8 shrink-0">
                <ChoyAvatar class="size-8 rounded-full" data-testid="auth-user-menu-avatar">
                  <ChoyAvatarImage
                    v-if="userAvatarSrc"
                    :src="userAvatarSrc"
                    :alt="userMenuLabel"
                  />
                  <ChoyAvatarFallback
                    class="rounded-full border border-sidebar-border bg-background text-xs font-medium text-sidebar-foreground"
                  >
                    {{ userInitials }}
                  </ChoyAvatarFallback>
                </ChoyAvatar>
                <span
                  v-if="showOfflineBadge"
                  class="auth-sidebar-user-menu__offline-badge pointer-events-none absolute end-0 bottom-0 size-2 rounded-full bg-warning ring-2 ring-[var(--choy-shell-chrome,var(--sidebar))]"
                  aria-hidden="true"
                  data-testid="auth-user-menu-offline-badge"
                />
              </span>
              <span
                class="grid min-w-0 flex-1 text-start text-sm leading-tight"
                data-testid="auth-user-menu-label"
              >
                <span class="truncate font-medium">{{ userMenuLabel }}</span>
                <span
                  v-if="userMenuMeta"
                  class="text-muted-foreground truncate text-xs"
                  data-testid="auth-user-menu-meta"
                >
                  {{ userMenuMeta }}
                </span>
              </span>
            </ChoySidebarMenuButton>
          </ChoyDropdownMenuTrigger>
          <ChoyDropdownMenuContent
            side="top"
            align="start"
            class="min-w-56 rounded-lg"
            data-testid="auth-user-menu-content"
            @close-auto-focus="onUserMenuCloseAutoFocus"
          >
            <template v-if="connectivity.isUnreachable">
              <ChoyDropdownMenuItem
                class="text-warning focus:text-warning data-[highlighted]:text-warning"
                data-testid="auth-user-menu-connectivity-retry"
                @select="onMenuConnectivityRetry"
              >
                <Link2Off class="size-4 text-warning" aria-hidden="true" />
                {{
                  connectivity.probing ? _tWeb('Checking…') : _tWeb('Retry connection')
                }}
              </ChoyDropdownMenuItem>
              <ChoyDropdownMenuSeparator />
            </template>
            <ChoyDropdownMenuItem
              data-testid="auth-user-menu-preferences"
              @select="onMenuPreferences"
            >
              <Settings class="size-4" aria-hidden="true" />
              {{ _t('Preferences') }}
            </ChoyDropdownMenuItem>
            <ChoyDropdownMenuSeparator />
            <ChoyDropdownMenuItem @select="onMenuLogout">
              <LogOut class="size-4" aria-hidden="true" />
              {{ _t('Log Out') }}
            </ChoyDropdownMenuItem>
          </ChoyDropdownMenuContent>
        </ChoyDropdownMenu>
      </ChoyCollapsedIdentityTipHost>
    </div>
    <ChoyConnectivityPill
      v-if="showFooterPill"
      class="auth-sidebar-user-menu__pill shrink-0"
    />
  </div>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue';
import { useRouter } from 'vue-router';
import { Link2Off, LogOut, Settings } from 'lucide-vue-next';
import {
  ChoyAvatar,
  ChoyAvatarFallback,
  ChoyAvatarImage,
  ChoyDropdownMenu,
  ChoyDropdownMenuContent,
  ChoyDropdownMenuItem,
  ChoyDropdownMenuSeparator,
  ChoyDropdownMenuTrigger,
  ChoyCollapsedIdentityTipHost,
  ChoyConnectivityPill,
  ChoySidebarMenuButton,
  closeAllDialogs,
  onDialogInterrupt,
  openDialog,
} from '@/web';
import { firstGrapheme } from '@/web/web/components/layout/appRailGlyph';
import { createTranslate } from '@/web/web/i18n';
import { useConnectivityStore } from '@/web/web/stores/connectivityStore';
import { useLayoutStore } from '@/web/web/stores/layoutStore';
import { useAuthStore } from '@/auth/web/stores/auth';
import { useI18nStore } from '@/web/web/stores/i18nStore';
import PreferencesDialog from '@/auth/web/dialogs/preferences/PreferencesDialog.vue';
import { collapsedIdentityTooltip } from './collapsed_identity_tooltip';
import { resolveUserMenuAvatarDisplaySrc } from './user_menu_avatar';
import { resolveUserMenuDisplayName, resolveUserMenuMeta } from './user_menu_display_name';

defineOptions({ name: 'AuthSidebarUserMenu' });

const { _t } = createTranslate('auth', { scope: 'web/components/layout/ChoyAppShell' });
const { _t: _tWeb } = createTranslate('web', {
  scope: 'web/components/layout/ChoyConnectivityPill',
});
const router = useRouter();
const authStore = useAuthStore();
const i18nStore = useI18nStore();
const connectivity = useConnectivityStore();
const layoutStore = useLayoutStore();

const userMenuOpen = ref(false);
let stopInterrupt: (() => void) | undefined;

onMounted(() => {
  stopInterrupt = onDialogInterrupt(() => {
    userMenuOpen.value = false;
  });
});
onBeforeUnmount(() => {
  stopInterrupt?.();
  stopInterrupt = undefined;
});

/** Icon-rail collapsed (not mobile sheet): badge instead of trailing pill. */
const sidebarCollapsedDesktop = computed(
  () => layoutStore.sidebarMode === 'collapsed' && !layoutStore.isMobile,
);

const showFooterPill = computed(
  () => !sidebarCollapsedDesktop.value && connectivity.isUnreachable,
);

const showOfflineBadge = computed(
  () => sidebarCollapsedDesktop.value && connectivity.isUnreachable,
);

const userMenuLabel = computed(() => {
  const user = (authStore.currentUser as any) || {};
  const fromProfile = resolveUserMenuDisplayName({
    firstName: user.FirstName,
    lastName: user.LastName,
    fullName: user.FullName,
    displayName: user.DisplayName,
    username: user.Username,
    email: user.Email,
    personNameFormat: i18nStore.getPersonNameFormat?.(),
  });
  if (fromProfile) return fromProfile;
  const identity = (authStore.identity as any) || {};
  const fromToken = String(identity.displayName || identity.username || '').trim();
  return fromToken || _t('Account');
});

const userIdentityTooltip = computed(() =>
  collapsedIdentityTooltip(userMenuOpen.value, userMenuLabel.value),
);

const userMenuMeta = computed(() => {
  const user = (authStore.currentUser as any) || {};
  const identity = (authStore.identity as any) || {};
  return resolveUserMenuMeta({
    email: user.Email || identity.email,
    username: user.Username || identity.username,
    primaryLabel: userMenuLabel.value,
  });
});

const userMenuAriaLabel = computed(() => {
  if (connectivity.isUnreachable) {
    return `${_t('User menu')}. ${_tWeb('Connection lost')}`;
  }
  return _t('User menu');
});

const userInitials = computed(() => {
  const label = userMenuLabel.value;
  const parts = label.split(/\s+/).filter(Boolean);
  if (parts.length >= 2 && /^[\x00-\x7F]+$/.test(parts[0] || '')) {
    const a = firstGrapheme(parts[0] || '');
    const b = firstGrapheme(parts[1] || '');
    return `${a}${b}`.toUpperCase() || firstGrapheme(label);
  }
  return firstGrapheme(label);
});

const userAvatarSrc = ref('');
let avatarResolveGen = 0;
/** Mutable so tests can inject a rejecting resolver for the catch path. */
let resolveAvatarSrc = resolveUserMenuAvatarDisplaySrc;
watch(
  () => (authStore.currentUser as any)?.Avatar,
  (avatar) => {
    const gen = ++avatarResolveGen;
    userAvatarSrc.value = '';
    void resolveAvatarSrc(avatar)
      .then((src) => {
        if (gen !== avatarResolveGen) return;
        userAvatarSrc.value = src || '';
      })
      .catch(() => {
        if (gen !== avatarResolveGen) return;
        userAvatarSrc.value = '';
      });
  },
  { immediate: true, deep: true },
);

function closeUserMenu() {
  userMenuOpen.value = false;
}

function resetHeaderPopups() {
  closeUserMenu();
  closeAllDialogs();
}

function handleLogout() {
  resetHeaderPopups();
  void router?.push?.({ name: 'logout' });
}

let openPreferencesOnMenuClose = false;

function onMenuPreferences(event?: Event) {
  event?.preventDefault?.();
  openPreferencesOnMenuClose = true;
  closeUserMenu();
}

function onUserMenuCloseAutoFocus(event: Event) {
  if (!openPreferencesOnMenuClose) return;
  openPreferencesOnMenuClose = false;
  event.preventDefault();
  openDialog({ component: PreferencesDialog, mode: 'replace' });
}

function onMenuLogout() {
  handleLogout();
}

async function onMenuConnectivityRetry(event?: Event) {
  // Keep the menu open while probing so Checking… / failure stay visible.
  event?.preventDefault?.();
  const ok = await connectivity.probeConnectivity();
  if (ok) closeUserMenu();
}
</script>
