<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <Xpath expr="//*[@data-anchor='choy.shell.header-actions']" position="inside">
    <ChoyNotificationBell />
  </Xpath>
  <Xpath expr="//*[@data-anchor='choy.shell.sidebar-header']" position="inside">
    <SwitchCompany variant="sidebar" />
  </Xpath>
  <Xpath expr="//*[@data-anchor='choy.shell.sidebar-footer']" position="inside">
    <ChoySidebarMenu class="w-full min-w-0">
      <ChoySidebarMenuItem>
        <ChoyCollapsedIdentityTipHost :label="userIdentityTooltip">
          <ChoyDropdownMenu v-model:open="userMenuOpen">
            <ChoyDropdownMenuTrigger as-child>
              <ChoySidebarMenuButton
                size="lg"
                type="button"
                class="data-[state=open]:bg-sidebar-accent data-[state=open]:text-sidebar-accent-foreground"
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
              <ChoyDropdownMenuLabel class="p-0 font-normal text-foreground">
                <div
                  class="flex items-center gap-2 px-1 py-1.5 text-start text-sm"
                  data-testid="auth-user-menu-heading"
                >
                  <ChoyAvatar class="size-8 shrink-0 rounded-full">
                    <ChoyAvatarFallback
                      class="rounded-full border border-sidebar-border bg-background text-xs font-medium text-sidebar-foreground"
                    >
                      {{ userInitials }}
                    </ChoyAvatarFallback>
                  </ChoyAvatar>
                  <div class="grid min-w-0 flex-1 text-start text-sm leading-tight">
                    <span class="truncate font-medium">{{ userMenuLabel }}</span>
                    <span class="text-muted-foreground truncate text-xs">{{ userMenuMeta }}</span>
                  </div>
                </div>
              </ChoyDropdownMenuLabel>
              <ChoyDropdownMenuSeparator />
              <ChoyDropdownMenuItem @select="onMenuProfile">
                {{ _t('Profile') }}
              </ChoyDropdownMenuItem>
              <ChoyDropdownMenuItem @select="onMenuSettings">
                {{ _t('Settings') }}
              </ChoyDropdownMenuItem>
              <ChoyDropdownMenuSeparator />
              <ChoyDropdownMenuLabel>{{ _t('Theme') }}</ChoyDropdownMenuLabel>
              <ChoyDropdownMenuItem
                v-for="item in themeItems"
                :key="item.mode"
                role="menuitemradio"
                :aria-checked="themeMode === item.mode"
                :data-testid="`auth-user-menu-theme-${item.mode}`"
                class="gap-2"
                @select="applyTheme(item.mode)"
              >
                <Check
                  class="size-4"
                  :class="themeMode === item.mode ? 'opacity-100' : 'opacity-0'"
                  aria-hidden="true"
                />
                {{ item.label }}
              </ChoyDropdownMenuItem>
              <ChoyDropdownMenuSeparator />
              <ChoyDropdownMenuLabel>{{ _t('Language') }}</ChoyDropdownMenuLabel>
              <ChoyDropdownMenuItem
                v-for="code in localeCodes"
                :key="code"
                role="menuitemradio"
                :aria-checked="code === currentLocaleCode"
                :data-testid="`auth-user-menu-locale-${code}`"
                class="gap-2"
                @select="applyLocale(code)"
              >
                <Check
                  class="size-4"
                  :class="code === currentLocaleCode ? 'opacity-100' : 'opacity-0'"
                  aria-hidden="true"
                />
                {{ localeName(code) }}
              </ChoyDropdownMenuItem>
              <ChoyDropdownMenuSeparator />
              <ChoyDropdownMenuItem @select="onMenuLogout">
                {{ _t('Log Out') }}
              </ChoyDropdownMenuItem>
            </ChoyDropdownMenuContent>
          </ChoyDropdownMenu>
        </ChoyCollapsedIdentityTipHost>
      </ChoySidebarMenuItem>
    </ChoySidebarMenu>
    <PreferencesDialog v-model="preferencesVisible" />
  </Xpath>
</template>

<script lang="ts" _name="ChoyAppShell">
import { computed, defineComponent, ref, watch } from 'vue';
import { useRouter } from 'vue-router';
import { Check, ChevronsUpDown } from 'lucide-vue-next';
import { Xpath } from '@/core/web';
import {
  ChoyAvatar,
  ChoyAvatarFallback,
  ChoyDropdownMenu,
  ChoyDropdownMenuContent,
  ChoyDropdownMenuItem,
  ChoyDropdownMenuLabel,
  ChoyDropdownMenuSeparator,
  ChoyDropdownMenuTrigger,
  ChoyCollapsedIdentityTipHost,
  ChoyNotificationBell,
  ChoySidebarMenu,
  ChoySidebarMenuButton,
  ChoySidebarMenuItem,
} from '@/web';
import { firstGrapheme } from '@/web/web/components/layout/appRailGlyph';
import ChoyAppShell from '@/web/web/components/layout/ChoyAppShell.vue';
import { createTranslate } from '@/web/web/i18n';
import { useAuthStore } from '@/auth/web/stores/auth';
import {
  applyChoyThemePreference,
  readChoyThemePreference,
  type ChoyThemeMode,
} from '@/web/web/composables/applyChoyThemePreference';
import { useI18nStore } from '@/web/web/stores/i18nStore';
import { DEFAULT_ACTIVE_UI_KEYS } from '@/web/web/stores/i18nStore/activeUiKeys';
import { SUPPORTED_LOCALES } from '@/web/web/stores/i18nStore/locales';
import type { SupportedLocale } from '@/web/web/stores/i18nStore/types';
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
    Check,
    ChevronsUpDown,
    ChoyNotificationBell,
    SwitchCompany,
    PreferencesDialog,
    ChoyCollapsedIdentityTipHost,
    ChoyAvatar,
    ChoyAvatarFallback,
    ChoyDropdownMenu,
    ChoyDropdownMenuContent,
    ChoyDropdownMenuItem,
    ChoyDropdownMenuLabel,
    ChoyDropdownMenuSeparator,
    ChoyDropdownMenuTrigger,
    ChoySidebarMenu,
    ChoySidebarMenuButton,
    ChoySidebarMenuItem,
  },
  setup(props, ctx) {
    const baseSetupFn = (ChoyAppShell as any)?.setup;
    if (typeof baseSetupFn !== 'function') {
      throw new Error('auth ChoyAppShell: base web ChoyAppShell exposes no setup() to merge');
    }
    const baseSetup = reuseParentSetupState(baseSetupFn(props, ctx));
    const { _t } = createTranslate('auth', { scope: 'web/components/layout/ChoyAppShell' });
    const router = useRouter();
    const authStore = useAuthStore();
    const preferencesVisible = ref(false);
    const userMenuOpen = ref(false);
    const themeMode = ref<ChoyThemeMode>(readChoyThemePreference().theme || 'light');
    watch(userMenuOpen, (open) => {
      if (open) themeMode.value = readChoyThemePreference().theme || 'light';
    });

    let i18nStore: ReturnType<typeof useI18nStore> | null = null;
    try {
      i18nStore = useI18nStore();
    } catch {
      i18nStore = null;
    }

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

    const userMenuMeta = computed(() => {
      const user = (authStore.currentUser as any) || {};
      const email = String(user.Email || '').trim();
      if (email) return email;
      const identity = (authStore.identity as any) || {};
      const fromToken = String(identity.email || identity.username || '').trim();
      return fromToken || _t('Account');
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

    const themeItems = computed(() => [
      { mode: 'light' as const, label: _t('Light') },
      { mode: 'dark' as const, label: _t('Dark') },
      { mode: 'auto' as const, label: _t('System') },
    ]);

    const localeCodes = computed(() => {
      const fromStore = i18nStore?.activeUiKeys;
      if (Array.isArray(fromStore) && fromStore.length) return fromStore as SupportedLocale[];
      return [...DEFAULT_ACTIVE_UI_KEYS] as SupportedLocale[];
    });

    const currentLocaleCode = computed(
      () => (i18nStore?.localeCode as SupportedLocale | null) || 'en',
    );

    function localeName(code: string): string {
      return SUPPORTED_LOCALES[code as SupportedLocale]?.name || code;
    }

    function applyTheme(next: ChoyThemeMode) {
      themeMode.value = next;
      applyChoyThemePreference({ theme: next });
    }

    function applyLocale(code: SupportedLocale) {
      void i18nStore?.setUiKey(code);
    }

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

    function onMenuProfile() {
      openPreferences();
    }

    function onMenuSettings() {
      openPreferences();
    }

    function onMenuLogout() {
      handleLogout();
    }

    return {
      ...baseSetup,
      _t,
      preferencesVisible,
      userMenuOpen,
      userMenuLabel,
      userIdentityTooltip,
      userMenuMeta,
      userInitials,
      themeMode,
      themeItems,
      localeCodes,
      currentLocaleCode,
      localeName,
      applyTheme,
      applyLocale,
      onMenuProfile,
      onMenuSettings,
      onMenuLogout,
    };
  },
});
</script>
