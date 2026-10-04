<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <Xpath expr="//*[@data-anchor='choy.shell.header-actions']" position="inside">
    <template v-if="!isAuthenticated">
      <ChoyButton
        v-if="!isLoginRoute"
        variant="default"
        size="sm"
        :aria-label="_t('Log in')"
        data-testid="choy-shell-login"
        @click="handleLogin"
      >
        {{ _t('Log In') }}
      </ChoyButton>
      <ChoyButton
        v-if="showRegister"
        variant="outline"
        size="sm"
        :aria-label="_t('Register')"
        data-testid="choy-shell-register"
        @click="handleRegister"
      >
        {{ _t('Register') }}
      </ChoyButton>
    </template>
    <ChoyNotificationBell v-if="isAuthenticated" />
    <SwitchCompany v-if="isAuthenticated" />
    <ChoyDropdownMenu v-if="isAuthenticated" v-model:open="userMenuOpen">
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
    <PreferencesDialog v-if="isAuthenticated" v-model="preferencesVisible" />
  </Xpath>
</template>

<script lang="ts" _name="ChoyWebShell">
import { computed, defineComponent, ref, watch } from 'vue';
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
import ChoyWebShell from '@/web/web/components/layout/ChoyWebShell.vue';
import { useAuthStore } from '@/auth/web/stores/auth';
import { createTranslate } from '@/web/web/i18n';
import { shouldResetAuthHeaderPopups } from './auth_header_popup_state';
import { reuseParentSetupState } from './reuse_parent_setup_state';
import SwitchCompany from './SwitchCompany.vue';
import PreferencesDialog from '../preferences/PreferencesDialog.vue';

/**
 * Extends the product shell so auth header actions merge into
 * data-anchor="choy.shell.header-actions" at web build time.
 */
export default defineComponent({
  name: 'ChoyWebShell',
  extends: ChoyWebShell,
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
    const baseSetupFn = (ChoyWebShell as any)?.setup;
    if (typeof baseSetupFn !== 'function') {
      throw new Error('auth ChoyWebShell: base web ChoyWebShell exposes no setup() to merge');
    }
    const baseSetup = reuseParentSetupState(baseSetupFn(props, ctx));
    const { _t } = createTranslate('auth', { scope: 'web/components/layout/ChoyWebShell' });
    const router = useRouter();
    const route = router?.currentRoute;
    const authStore = useAuthStore();
    const isAuthenticated = computed(() => authStore.isAuthenticated);
    const isLoginRoute = computed(() => route?.value?.name === 'login' || route?.value?.path === '/login');
    const isRegisterRoute = computed(
      () => route?.value?.name === 'register' || route?.value?.path === '/register',
    );
    const showRegister = computed(() => {
      if (isRegisterRoute.value) return false;
      try {
        return typeof router.hasRoute === 'function' ? router.hasRoute('register') : true;
      } catch {
        return true;
      }
    });
    const preferencesVisible = ref(false);
    const userMenuOpen = ref(false);

    function closeUserMenu() {
      userMenuOpen.value = false;
    }

    function resetHeaderPopups() {
      closeUserMenu();
      preferencesVisible.value = false;
    }

    watch(isAuthenticated, (authed, wasAuthed) => {
      if (shouldResetAuthHeaderPopups(Boolean(wasAuthed), Boolean(authed))) {
        resetHeaderPopups();
      }
    });

    function handleLogin() {
      router.push({ name: 'login' });
    }

    function handleRegister() {
      router.push({ name: 'register' });
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
      isLoginRoute,
      showRegister,
      preferencesVisible,
      userMenuOpen,
      handleLogin,
      handleRegister,
      onMenuProfile,
      onMenuSettings,
      onMenuLogout,
    };
  },
});
</script>
