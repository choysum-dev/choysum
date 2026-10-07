<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <Xpath expr="//*[@data-anchor='choy.shell.header-actions']" position="inside">
    <ChoyNotificationBell />
    <SwitchCompany />
    <ChoyDropdownMenu v-model:open="userMenuOpen">
      <ChoyDropdownMenuTrigger as-child>
        <ChoyButton
          variant="ghost"
          size="icon"
          type="button"
          :aria-label="_t('User menu')"
          data-testid="auth-user-menu-trigger"
        >
          <ChoyAvatar class="size-6">
            <ChoyAvatarFallback class="bg-muted text-muted-foreground text-xs">
              <User class="size-3.5" aria-hidden="true" />
            </ChoyAvatarFallback>
          </ChoyAvatar>
        </ChoyButton>
      </ChoyDropdownMenuTrigger>
      <ChoyDropdownMenuContent align="end" class="min-w-[10rem]">
        <ChoyDropdownMenuItem @select="onMenuProfile">
          {{ _t('Profile') }}
        </ChoyDropdownMenuItem>
        <ChoyDropdownMenuItem @select="onMenuSettings">
          {{ _t('Settings') }}
        </ChoyDropdownMenuItem>
        <ChoyDropdownMenuItem @select="onMenuLogout">
          {{ _t('Log Out') }}
        </ChoyDropdownMenuItem>
      </ChoyDropdownMenuContent>
    </ChoyDropdownMenu>
    <PreferencesDialog v-model="preferencesVisible" />
  </Xpath>
</template>

<script lang="ts" _name="ChoyAppShell">
import { defineComponent, ref } from 'vue';
import { useRouter } from 'vue-router';
import { User } from 'lucide-vue-next';
import { Xpath } from '@/core/web';
import {
  ChoyAvatar,
  ChoyAvatarFallback,
  ChoyButton,
  ChoyDropdownMenu,
  ChoyDropdownMenuContent,
  ChoyDropdownMenuItem,
  ChoyDropdownMenuTrigger,
  ChoyNotificationBell,
} from '@/web';
import ChoyAppShell from '@/web/web/components/layout/ChoyAppShell.vue';
import { createTranslate } from '@/web/web/i18n';
import { reuseParentSetupState } from './reuse_parent_setup_state';
import SwitchCompany from './SwitchCompany.vue';
import PreferencesDialog from '../preferences/PreferencesDialog.vue';

/**
 * Extends the app shell so session header actions merge into
 * data-anchor="choy.shell.header-actions" at web build time.
 */
export default defineComponent({
  name: 'ChoyAppShell',
  extends: ChoyAppShell,
  components: {
    Xpath,
    User,
    ChoyButton,
    ChoyNotificationBell,
    SwitchCompany,
    PreferencesDialog,
    ChoyAvatar,
    ChoyAvatarFallback,
    ChoyDropdownMenu,
    ChoyDropdownMenuContent,
    ChoyDropdownMenuItem,
    ChoyDropdownMenuTrigger,
  },
  setup(props, ctx) {
    const baseSetupFn = (ChoyAppShell as any)?.setup;
    if (typeof baseSetupFn !== 'function') {
      throw new Error('auth ChoyAppShell: base web ChoyAppShell exposes no setup() to merge');
    }
    const baseSetup = reuseParentSetupState(baseSetupFn(props, ctx));
    const { _t } = createTranslate('auth', { scope: 'web/components/layout/ChoyAppShell' });
    const router = useRouter();
    const preferencesVisible = ref(false);
    const userMenuOpen = ref(false);

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
      onMenuProfile,
      onMenuSettings,
      onMenuLogout,
    };
  },
});
</script>
