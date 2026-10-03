<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <ChoyPage :loading="loading" width="narrow" :padding="false" class="w-full">
    <AuthPanel>
      <ChoyCard
        :title="_t('User Login')"
        :description="_t('Enter your username and password to continue')"
        class="login-card w-full"
      >
        <form class="flex flex-col gap-4" @submit.prevent="handleLogin">
          <ChoyFieldGroup class="gap-4">
            <ChoyField v-if="error">
              <div
                class="login-error flex items-start justify-between gap-2 rounded-md border border-destructive/40 bg-destructive/10 px-3 py-2 text-sm text-destructive"
                role="alert"
              >
                <span>{{ error }}</span>
                <button
                  type="button"
                  class="text-destructive/80 hover:text-destructive"
                  :aria-label="_t('Close')"
                  @click="error = ''"
                >
                  ×
                </button>
              </div>
            </ChoyField>

            <ChoyField :data-invalid="fieldErrors.username ? true : undefined">
              <ChoyFieldLabel for="login-username">{{ _t('Username') }}</ChoyFieldLabel>
              <ChoyInput
                id="login-username"
                v-model="form.username"
                name="username"
                type="text"
                autocomplete="username"
                :placeholder="_t('Enter username')"
                class="login-username"
                :aria-invalid="fieldErrors.username ? true : undefined"
              />
              <ChoyFieldError :errors="fieldErrors.username ? [fieldErrors.username] : []" />
            </ChoyField>

            <ChoyField :data-invalid="fieldErrors.password ? true : undefined">
              <ChoyFieldLabel for="login-password">{{ _t('Password') }}</ChoyFieldLabel>
              <ChoyInput
                id="login-password"
                v-model="form.password"
                name="password"
                type="password"
                autocomplete="current-password"
                :placeholder="_t('Enter password')"
                class="login-password"
                :aria-invalid="fieldErrors.password ? true : undefined"
              />
              <ChoyFieldError :errors="fieldErrors.password ? [fieldErrors.password] : []" />
            </ChoyField>

            <ChoyField orientation="horizontal">
              <ChoyCheckbox id="login-remember" v-model="form.rememberMe" class="login-options" />
              <ChoyFieldLabel for="login-remember" class="font-normal">
                {{ _t('Remember me') }}
              </ChoyFieldLabel>
            </ChoyField>

            <ChoyField>
              <ChoyButton type="submit" class="submit-button w-full" :disabled="loading">
                {{ loading ? _t('Log In') + '…' : _t('Log In') }}
              </ChoyButton>
              <ChoyFieldDescription v-if="showRegisterLink" class="text-center">
                {{ _t("Don't have an account?") }}
                <router-link to="/register" class="text-primary hover:underline">{{
                  _t('Register now')
                }}</router-link>
              </ChoyFieldDescription>
            </ChoyField>
          </ChoyFieldGroup>
        </form>
      </ChoyCard>
    </AuthPanel>
  </ChoyPage>
</template>

<script setup lang="ts">
import { ref, reactive, computed, onMounted } from 'vue';
import { useRouter, useRoute } from 'vue-router';
import { storeToRefs } from 'pinia';
import { useAuthStore } from '../stores/auth';
import AuthPanel from '../components/AuthPanel.vue';
import {
  ChoyPage,
  ChoyCard,
  ChoyButton,
  ChoyInput,
  ChoyCheckbox,
  ChoyField,
  ChoyFieldDescription,
  ChoyFieldError,
  ChoyFieldGroup,
  ChoyFieldLabel,
} from '@/web';
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
