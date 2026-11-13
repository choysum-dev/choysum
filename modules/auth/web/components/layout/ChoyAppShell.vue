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
    <PreferencesDialog v-model="preferencesVisible" />
  </Xpath>
</template>

<script lang="ts" _name="ChoyAppShell">
import { computed, defineComponent, ref } from 'vue';
import { useRouter } from 'vue-router';
import { ChevronsUpDown, LogOut, Settings } from 'lucide-vue-next';
import { Xpath } from '@/core/web';
import {
  ChoyAvatar,
  ChoyAvatarFallback,
  ChoyDropdownMenu,
  ChoyDropdownMenuContent,
  ChoyDropdownMenuItem,
  ChoyDropdownMenuSeparator,
  ChoyDropdownMenuTrigger,
  ChoyCollapsedIdentityTipHost,
  ChoyNotificationBell,
  ChoySidebarMenuButton,
} from '@/web';
import { firstGrapheme } from '@/web/web/components/layout/appRailGlyph';
import ChoyAppShell from '@/web/web/components/layout/ChoyAppShell.vue';
import { createTranslate } from '@/web/web/i18n';
import { useAuthStore } from '@/auth/web/stores/auth';
import { useReactiveCollaborationCaps } from '@/web/web/platform/collaborationCaps';
import { collapsedIdentityTooltip } from './collapsed_identity_tooltip';
import { reuseParentSetupState } from './reuse_parent_setup_state';
import SwitchCompany from './SwitchCompany.vue';
import PreferencesDialog from '../preferences/PreferencesDialog.vue';

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
    PreferencesDialog,
    ChoyCollapsedIdentityTipHost,
    ChoyAvatar,
    ChoyAvatarFallback,
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
    const preferencesVisible = ref(false);
    const userMenuOpen = ref(false);

    const userMenuLabel = computed(() => {
      const user = (authStore.currentUser as any) || {};
      const fromProfile = String(
        user.FullName || user.DisplayName || user.Username || user.Email || '',
      ).trim();
      if (fromProfile) return fromProfile;
      const identity = (authStore.identity as any) || {};
      const fromToken = String(identity.displayName || identity.username || '').trim();
      return fromToken || _t('Account');
    });

    /** Collapsed tip label; omitted while the user menu is open. */
    const userIdentityTooltip = computed(() =>
      collapsedIdentityTooltip(userMenuOpen.value, userMenuLabel.value),
    );

    /** Secondary line: email only. Avoid repeating the display name or a generic “Account”. */
    const userMenuMeta = computed(() => {
      const user = (authStore.currentUser as any) || {};
      const email = String(user.Email || '').trim();
      if (email) return email;
      const identity = (authStore.identity as any) || {};
      return String(identity.email || '').trim();
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

    function closeUserMenu() {
      userMenuOpen.value = false;
    }

    function resetHeaderPopups() {
      closeUserMenu();
      preferencesVisible.value = false;
    }

    function openPreferences() {
      preferencesVisible.value = true;
      closeUserMenu();
    }

    function handleLogout() {
      resetHeaderPopups();
      void router?.push?.({ name: 'logout' });
    }

    function onMenuPreferences() {
      openPreferences();
    }

    function onMenuLogout() {
      handleLogout();
    }

    return {
      ...baseSetup,
      _t,
      collaborationCaps,
      preferencesVisible,
      userMenuOpen,
      userMenuLabel,
      userIdentityTooltip,
      userMenuMeta,
      userInitials,
      onMenuPreferences,
      onMenuLogout,
    };
  },
});
</script>
