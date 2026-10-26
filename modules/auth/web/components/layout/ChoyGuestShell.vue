<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <Xpath expr="//*[@data-anchor='choy.shell.header-actions']" position="inside">
    <template v-if="!isAuthenticated">
      <ChoyButton
        v-if="!isLoginRoute"
        variant="ghost"
        size="sm"
        :aria-label="_t('Log in')"
        data-testid="choy-shell-login"
        @click="handleLogin"
      >
        {{ _t('Log In') }}
      </ChoyButton>
      <ChoyButton
        v-if="showRegister"
        variant="default"
        size="sm"
        class="h-[31px] rounded-lg"
        :aria-label="_t('Sign up')"
        data-testid="choy-shell-register"
        @click="handleRegister"
      >
        {{ _t('Sign Up') }}
      </ChoyButton>
    </template>
  </Xpath>
</template>

<script lang="ts" _name="ChoyGuestShell">
import { computed, defineComponent } from 'vue';
import { useRouter } from 'vue-router';
import { Xpath } from '@/core/web';
import { ChoyButton } from '@/web';
import ChoyGuestShell from '@/web/web/components/layout/ChoyGuestShell.vue';
import { useAuthStore } from '@/auth/web/stores/auth';
import { createTranslate } from '@/web/web/i18n';
import { reuseParentSetupState } from './reuse_parent_setup_state';

/**
 * Extends the guest shell so login/register actions merge into
 * data-anchor="choy.shell.header-actions" at web build time.
 * Authenticated sessions on guest routes (e.g. /error/403) get no guest actions.
 */
export default defineComponent({
  name: 'ChoyGuestShell',
  extends: ChoyGuestShell,
  components: {
    Xpath,
    ChoyButton,
  },
  setup(props, ctx) {
    const baseSetupFn = (ChoyGuestShell as any)?.setup;
    if (typeof baseSetupFn !== 'function') {
      throw new Error('auth ChoyGuestShell: base web ChoyGuestShell exposes no setup() to merge');
    }
    const baseSetup = reuseParentSetupState(baseSetupFn(props, ctx));
    const { _t } = createTranslate('auth', { scope: 'web/components/layout/ChoyGuestShell' });
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

    function handleLogin() {
      void router?.push?.({ name: 'login' });
    }

    function handleRegister() {
      void router?.push?.({ name: 'register' });
    }

    return {
      ...baseSetup,
      _t,
      isAuthenticated,
      isLoginRoute,
      showRegister,
      handleLogin,
      handleRegister,
    };
  },
});
</script>
