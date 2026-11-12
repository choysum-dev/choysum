<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <Xpath expr="//*[@data-anchor='choy.shell.header-actions']" position="inside">
    <ChoyNotificationBell v-if="collaborationCaps.message" />
  </Xpath>
  <Xpath expr="//*[@data-anchor='choy.shell.sidebar-header']" position="inside">
    <SwitchCompany variant="sidebar" />
  </Xpath>
  <Xpath expr="//*[@data-anchor='choy.shell.sidebar-footer']" position="inside">
    <!-- Identity control only — not a SidebarMenu ul/li (footer is not a nav list). -->
    <ChoyCollapsedIdentityTipHost :label="userIdentityTooltip">
      <ChoyDropdownMenu v-model:open="userMenuOpen">
        <ChoyDropdownMenuTrigger as-child>
          <ChoySidebarMenuButton
            size="lg"
            type="button"
            class="w-full min-w-0 data-[state=open]:bg-sidebar-accent data-[state=open]:text-sidebar-accent-foreground"
            :aria-label="_t('User menu')"
            data-testid="auth-user-menu-trigger"
          >
            <ChoyAvatar class="size-8 shrink-0 rounded-full" data-testid="auth-user-menu-avatar">
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
            <span
              class="ms-auto inline-flex shrink-0 opacity-50"
              aria-hidden="true"
              data-testid="auth-user-menu-chevrons"
            >
              <ChevronsUpDown class="size-4" />
            </span>
          </ChoySidebarMenuButton>
        </ChoyDropdownMenuTrigger>
        <ChoyDropdownMenuContent side="right" align="end" class="min-w-56 rounded-lg">
          <!-- No identity heading: trigger already shows avatar / name / email. -->
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
  </Xpath>
</template>

<script lang="ts" _name="ChoyAppShell">
import { computed, defineComponent, onBeforeUnmount, onMounted, ref, watch } from 'vue';
import { useRouter } from 'vue-router';
import { ChevronsUpDown, LogOut, Settings } from 'lucide-vue-next';
import { Xpath } from '@/core/web';
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
  ChoyNotificationBell,
  ChoySidebarMenuButton,
  closeAllDialogs,
  onDialogInterrupt,
  openDialog,
} from '@/web';
import { firstGrapheme } from '@/web/web/components/layout/appRailGlyph';
import ChoyAppShell from '@/web/web/components/layout/ChoyAppShell.vue';
import { createTranslate } from '@/web/web/i18n';
import { useAuthStore } from '@/auth/web/stores/auth';
import { useI18nStore } from '@/web/web/stores/i18nStore';
import { useReactiveCollaborationCaps } from '@/web/web/platform/collaborationCaps';
import PreferencesDialog from '@/auth/web/components/preferences/PreferencesDialog.vue';
import { collapsedIdentityTooltip } from './collapsed_identity_tooltip';
import { reuseParentSetupState } from './reuse_parent_setup_state';
import { resolveUserMenuAvatarDisplaySrc } from './user_menu_avatar';
import { resolveUserMenuDisplayName, resolveUserMenuMeta } from './user_menu_display_name';
import SwitchCompany from './SwitchCompany.vue';

/**
 * Extends the app shell so session chrome merges into shell anchors at build time:
 * notifications → header-actions; company → sidebar-header; user → sidebar-footer.
 */
export default defineComponent({
  name: 'ChoyAppShell',
  extends: ChoyAppShell,
  components: {
    Xpath,
    ChevronsUpDown,
    LogOut,
    Settings,
    ChoyNotificationBell,
    SwitchCompany,
    ChoyCollapsedIdentityTipHost,
    ChoyAvatar,
    ChoyAvatarFallback,
    ChoyAvatarImage,
    ChoyDropdownMenu,
    ChoyDropdownMenuContent,
    ChoyDropdownMenuItem,
    ChoyDropdownMenuSeparator,
    ChoyDropdownMenuTrigger,
    ChoySidebarMenuButton,
  },
  setup(props, ctx) {
    const baseSetupFn = (ChoyAppShell as any)?.setup;
    if (typeof baseSetupFn !== 'function') {
      throw new Error('auth ChoyAppShell: base web ChoyAppShell exposes no setup() to merge');
    }
    const baseSetup = reuseParentSetupState(baseSetupFn(props, ctx));
    const { _t } = createTranslate('auth', { scope: 'web/components/layout/ChoyAppShell' });
    const collaborationCaps = useReactiveCollaborationCaps();
    const router = useRouter();
    const authStore = useAuthStore();
    const i18nStore = useI18nStore();
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

    /** Collapsed tip label; omitted while the user menu is open. */
    const userIdentityTooltip = computed(() =>
      collapsedIdentityTooltip(userMenuOpen.value, userMenuLabel.value),
    );

    /** Secondary line: username · email; drop segments that duplicate the primary label. */
    const userMenuMeta = computed(() => {
      const user = (authStore.currentUser as any) || {};
      const identity = (authStore.identity as any) || {};
      return resolveUserMenuMeta({
        email: user.Email || identity.email,
        username: user.Username || identity.username,
        primaryLabel: userMenuLabel.value,
      });
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
    watch(
      () => (authStore.currentUser as any)?.Avatar,
      avatar => {
        const gen = ++avatarResolveGen;
        userAvatarSrc.value = '';
        void resolveUserMenuAvatarDisplaySrc(avatar).then(src => {
          if (gen !== avatarResolveGen) return;
          userAvatarSrc.value = src || '';
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

    function onMenuPreferences() {
      closeUserMenu();
      openDialog({ component: PreferencesDialog, mode: 'replace' });
    }

    function onMenuLogout() {
      handleLogout();
    }

    return {
      ...baseSetup,
      _t,
      collaborationCaps,
      userMenuOpen,
      userMenuLabel,
      userIdentityTooltip,
      userMenuMeta,
      userInitials,
      userAvatarSrc,
      onMenuPreferences,
      onMenuLogout,
    };
  },
});
</script>
