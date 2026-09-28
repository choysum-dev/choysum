<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <Xpath expr="//*[@data-anchor='choy.shell.header-actions']" position="inside">
    <span class="mx-1 inline-block h-5 w-px bg-border" role="separator" />
    <ChoyButton
      v-if="!isAuthenticated"
      variant="ghost"
      size="sm"
      :aria-label="_t('Log in')"
      @click="handleLogin"
    >
      {{ _t('Log In') }}
    </ChoyButton>
    <ChoyNotificationBell v-if="isAuthenticated" />
    <SwitchCompany v-if="isAuthenticated" />
    <div v-if="isAuthenticated" ref="userMenuRoot" class="relative">
      <ChoyButton
        variant="ghost"
        size="sm"
        :aria-expanded="userMenuOpen"
        :aria-label="_t('User menu')"
        data-testid="auth-user-menu-trigger"
        @click.stop="userMenuOpen = !userMenuOpen"
      >
        <User class="size-5" aria-hidden="true" />
      </ChoyButton>
      <div
        v-if="userMenuOpen"
        class="absolute end-0 top-full z-50 mt-1 min-w-[10rem] rounded-md border border-border bg-background py-1 shadow-md"
        role="menu"
        @click.stop
      >
        <button type="button" class="block w-full px-3 py-1.5 text-start text-sm hover:bg-muted" role="menuitem" @click="onMenuProfile">
          {{ _t('Profile') }}
        </button>
        <button type="button" class="block w-full px-3 py-1.5 text-start text-sm hover:bg-muted" role="menuitem" @click="onMenuSettings">
          {{ _t('Settings') }}
        </button>
        <button type="button" class="block w-full px-3 py-1.5 text-start text-sm hover:bg-muted" role="menuitem" @click="onMenuLogout">
          {{ _t('Log Out') }}
        </button>
      </div>
    </div>
    <PreferencesDialog v-if="isAuthenticated" v-model="preferencesVisible" />
  </Xpath>
</template>

<script lang="ts" _name="ChoyShellLayout">
import { computed, defineComponent, onBeforeUnmount, onMounted, ref, watch } from 'vue';
import { useRouter } from 'vue-router';
import { User } from 'lucide-vue-next';
import { Xpath } from '@/core/web';
import { ChoyButton, ChoyNotificationBell } from '@/web';
import ChoyShellLayout from '@/web/web/components/layout/ChoyShellLayout.vue';
import { useAuthStore } from '@/auth/web/stores/auth';
import { createTranslate } from '@/web/web/i18n';
import { shouldResetAuthHeaderPopups } from './auth_header_popup_state';
import SwitchCompany from './SwitchCompany.vue';
import { dismissPopupOnEscape } from './popup_escape_focus';
import PreferencesDialog from '../preferences/PreferencesDialog.vue';

/**
 * Extends the product shell so auth header actions merge into
 * data-anchor="choy.shell.header-actions" at web build time.
 */
export default defineComponent({
  name: 'ChoyShellLayout',
  extends: ChoyShellLayout,
  components: {
    Xpath,
    User,
    ChoyButton,
    ChoyNotificationBell,
    SwitchCompany,
    PreferencesDialog,
  },
  setup(props, ctx) {
    const baseSetup = (ChoyShellLayout as any)?.setup?.(props, ctx) || {};
    const { _t } = createTranslate('auth', { scope: 'web/components/layout/AuthHeader' });
    const router = useRouter();
    const authStore = useAuthStore();
    const isAuthenticated = computed(() => authStore.isAuthenticated);
    const preferencesVisible = ref(false);
    const userMenuOpen = ref(false);
    const userMenuRoot = ref<HTMLElement | null>(null);

    function closeUserMenu() {
      userMenuOpen.value = false;
    }

    function resetHeaderPopups() {
      closeUserMenu();
      preferencesVisible.value = false;
    }

    // Shell stays mounted; clear popup refs on logout / token expiry.
    watch(isAuthenticated, (authed, wasAuthed) => {
      if (shouldResetAuthHeaderPopups(Boolean(wasAuthed), Boolean(authed))) {
        resetHeaderPopups();
      }
    });

    function onDocumentClick(event: MouseEvent) {
      if (!userMenuOpen.value) return;
      const root = userMenuRoot.value;
      if (root && !root.contains(event.target as Node)) {
        closeUserMenu();
      }
    }

    function onDocumentKeydown(event: KeyboardEvent) {
      if (event.key !== 'Escape') return;
      const trigger = userMenuRoot.value?.querySelector<HTMLElement>('[data-testid="auth-user-menu-trigger"]');
      dismissPopupOnEscape(userMenuOpen.value, closeUserMenu, trigger);
    }

    onMounted(() => {
      document.addEventListener('click', onDocumentClick);
      document.addEventListener('keydown', onDocumentKeydown);
    });

    onBeforeUnmount(() => {
      document.removeEventListener('click', onDocumentClick);
      document.removeEventListener('keydown', onDocumentKeydown);
    });

    function handleLogin() {
      router.push({ name: 'login' });
    }

    function openPreferences() {
      preferencesVisible.value = true;
      closeUserMenu();
    }

    function handleLogout() {
      resetHeaderPopups();
      router.push({ name: 'logout' });
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
      isAuthenticated,
      preferencesVisible,
      userMenuOpen,
      userMenuRoot,
      handleLogin,
      onMenuProfile,
      onMenuSettings,
      onMenuLogout,
    };
  },
});
</script>
