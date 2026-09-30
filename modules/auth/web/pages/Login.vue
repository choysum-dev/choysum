<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <ChoyPage
    :loading="loading"
    width="narrow"
    :padding="false"
    class="mx-auto flex min-h-[calc(100vh_-_var(--choy-layout-header-height,3rem))] w-full max-w-md items-center justify-center px-4"
  >
    <ChoyCard :title="_t('User Login')" class="login-card w-full shadow-sm">
      <transition name="fade">
        <div
          v-if="error"
          class="login-error mb-4 flex items-start justify-between gap-2 rounded-md border border-destructive/40 bg-destructive/10 px-3 py-2 text-sm text-destructive"
          role="alert"
        >
          <span>{{ error }}</span>
          <button type="button" class="text-destructive/80 hover:text-destructive" :aria-label="_t('Close')" @click="error = ''">
            ×
          </button>
        </div>
      </transition>

      <form class="flex flex-col gap-3" @submit.prevent="handleLogin">
        <label class="flex flex-col gap-1 text-sm">
          <span class="text-foreground/80">{{ _t('Username') }}</span>
          <input
            v-model="form.username"
            name="username"
            type="text"
            autocomplete="username"
            :placeholder="_t('Enter username')"
            class="login-username choy-input"
            :class="{ 'border-destructive': fieldErrors.username }"
          />
          <span v-if="fieldErrors.username" class="text-xs text-destructive">{{ fieldErrors.username }}</span>
        </label>

        <label class="flex flex-col gap-1 text-sm">
          <span class="text-foreground/80">{{ _t('Password') }}</span>
          <input
            v-model="form.password"
            name="password"
            type="password"
            autocomplete="current-password"
            :placeholder="_t('Enter password')"
            class="login-password choy-input"
            :class="{ 'border-destructive': fieldErrors.password }"
          />
          <span v-if="fieldErrors.password" class="text-xs text-destructive">{{ fieldErrors.password }}</span>
        </label>

        <label class="login-options flex items-center gap-2 text-sm text-foreground/80">
          <input v-model="form.rememberMe" type="checkbox" class="size-4 rounded border-border" />
          <span>{{ _t('Remember me') }}</span>
        </label>

        <ChoyButton type="submit" class="submit-button w-full" :disabled="loading">
          {{ loading ? _t('Log In') + '…' : _t('Log In') }}
        </ChoyButton>

        <div v-if="showRegisterLink" class="text-center text-sm text-foreground/70">
          {{ _t("Don't have an account?") }}
          <router-link to="/register" class="ms-1 text-primary hover:underline">{{ _t('Register now') }}</router-link>
        </div>
      </form>
    </ChoyCard>
  </ChoyPage>
</template>

<script setup lang="ts">
import { ref, reactive, computed, onMounted } from 'vue';
import { useRouter, useRoute } from 'vue-router';
import { storeToRefs } from 'pinia';
import { useAuthStore } from '../stores/auth';
import { ChoyPage, ChoyCard, ChoyButton } from '@/web';
import { createTranslate } from '@/web/web/i18n';
import { runLoginAuthReady } from './login_auth_ready';
import { resolveLoginRedirect, runLoginSubmit } from './login_form';

const { _t } = createTranslate('auth', { scope: 'web/pages/Login' });

/**
 * Form model for the login page.
 */
interface LoginFormData {
  username: string;
  password: string;
  rememberMe: boolean;
}

const router = useRouter();
const route = useRoute();
const authStore = useAuthStore();
const { loading, isAuthenticated } = storeToRefs(authStore);

const form = reactive<LoginFormData>({
  username: '',
  password: '',
  rememberMe: true,
});

const fieldErrors = reactive({ username: '', password: '' });
const error = ref('');
const showRegisterLink = computed(() => import.meta.env.CHOYSUM_ENABLE_REGISTRATION !== false);

/**
 * Redirect the user to the requested destination after login.
 */
function handleRedirect() {
  router.replace(resolveLoginRedirect(route.query.redirect?.toString()));
}

onMounted(async () => {
  await runLoginAuthReady({
    ensureAuthReady: () => authStore.ensureAuthReady(),
    getRoutePath: () => route.path,
    isAuthenticated: () => !!isAuthenticated.value,
    redirect: handleRedirect,
  });
});

/**
 * Validate the form and start the login flow.
 */
async function handleLogin() {
  const ok = await runLoginSubmit({
    loading: !!loading.value,
    form,
    fieldErrors,
    t: _t,
    loginFailedMessage: _t('Login failed. Please try again later.'),
    login: (username, password, csrf, device, rememberMe) =>
      authStore.login(username, password, csrf, device, rememberMe),
    rememberMe: form.rememberMe,
    setError: message => {
      error.value = message;
    },
  });
  if (ok) handleRedirect();
}
</script>
