<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <ChoyPage :loading="loading" width="narrow" :padding="false" class="w-full">
    <AuthPanel>
      <ChoyCard class="login-card w-full">
        <template #header>
          <h3 class="font-semibold leading-none tracking-tight">{{ _t('User Login') }}</h3>
          <div
            v-if="error"
            class="login-error flex items-start justify-between gap-2 text-sm text-destructive"
            role="alert"
          >
            <span class="min-w-0 leading-5">{{ error }}</span>
            <button
              type="button"
              class="inline-flex size-5 shrink-0 items-center justify-center rounded-sm border-0 bg-transparent p-0 text-destructive/70 appearance-none shadow-none outline-none hover:text-destructive focus-visible:ring-2 focus-visible:ring-destructive/40"
              :aria-label="_t('Close')"
              @click="error = ''"
            >
              <X class="size-3.5" />
            </button>
          </div>
          <p v-else class="text-sm text-foreground/70">
            {{ _t('Enter your username and password to continue') }}
          </p>
        </template>
        <ChoyFormView
          :store="formStore"
          view-mode="create"
          embedded
          :show-header="false"
          :show-actions="false"
          :show-messages="false"
          :resolve-record-id-from-route="false"
          :initial-values="loginInitialValues"
          :submit-handler="onLoginSubmit"
        >
          <template #default="{ formData }">
            <AuthPageErrorClear
              :username="formData.Username"
              :password="formData.Password"
              :has-error="!!error"
              @clear="error = ''"
            />
            <ChoyFieldGroup>
              <div class="login-username">
                <ChoyVarcharField
                  :store="formStore"
                  prop="Username"
                  :label="_t('Username')"
                  :placeholder="_t('Enter username')"
                  autocomplete="username"
                  name="username"
                  id="login-username"
                  buffer-strategy="live"
                  :nullable="false"
                  :show-word-limit="false"
                  show-inline-error
                  :rules="usernameRules"
                />
              </div>
              <div class="login-password">
                <ChoyVarcharField
                  :store="formStore"
                  prop="Password"
                  :label="_t('Password')"
                  type="password"
                  :placeholder="_t('Enter password')"
                  autocomplete="current-password"
                  name="password"
                  id="login-password"
                  buffer-strategy="live"
                  :nullable="false"
                  :show-word-limit="false"
                  show-inline-error
                  :rules="passwordRules"
                />
              </div>
              <ChoyBooleanField
                class="login-options"
                :store="formStore"
                prop="RememberMe"
                widget="checkbox"
                :label="''"
                :checkbox-label="_t('Remember me')"
                render-mode="inline"
                buffer-strategy="live"
              />

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
          </template>
        </ChoyFormView>
      </ChoyCard>
    </AuthPanel>
  </ChoyPage>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue';
import { useRouter, useRoute } from 'vue-router';
import { storeToRefs } from 'pinia';
import { X } from 'lucide-vue-next';
import { useAuthStore } from '../stores/auth';
import AuthPanel from '../components/AuthPanel.vue';
import AuthPageErrorClear from './AuthPageErrorClear.vue';
import {
  ChoyPage,
  ChoyCard,
  ChoyButton,
  ChoyField,
  ChoyFieldDescription,
  ChoyFieldGroup,
  ChoyFormView,
  ChoyVarcharField,
  ChoyBooleanField,
  createLocalFormStore,
} from '@/web';
import { createTranslate } from '@/web/web/i18n';
import { runLoginAuthReady } from './login_auth_ready';
import {
  loginPasswordRules,
  loginUsernameRules,
  resolveLoginRedirect,
  runHandledAuthSubmit,
  runLoginSubmit,
} from './login_form';

const { _t } = createTranslate('auth', { scope: 'web/pages/Login' });

const router = useRouter();
const route = useRoute();
const authStore = useAuthStore();
const { loading, isAuthenticated } = storeToRefs(authStore);

const loginInitialValues = {
  Username: '',
  Password: '',
  RememberMe: true,
};

const formStore = createLocalFormStore({
  fields: [
    { name: 'Username', label: 'Username', type: 'varchar' },
    { name: 'Password', label: 'Password', type: 'varchar' },
    { name: 'RememberMe', label: 'Remember me', type: 'boolean' },
  ],
  initialValues: loginInitialValues,
  storeId: 'auth.login',
});

// Rebuild rules when locale changes so inline errors stay translated.
const usernameRules = computed(() => loginUsernameRules(_t));
const passwordRules = computed(() => loginPasswordRules(_t));

const error = ref('');
const showRegisterLink = computed(() => import.meta.env.CHOYSUM_ENABLE_REGISTRATION !== false);

/**
 * Redirect the user to the requested destination after login.
 */
function handleRedirect() {
  router.replace(resolveLoginRedirect(route.query.redirect?.toString()));
}

function setPageError(message: string) {
  error.value = message;
}

onMounted(async () => {
  await runLoginAuthReady({
    ensureAuthReady: () => authStore.ensureAuthReady(),
    getRoutePath: () => route.path,
    isAuthenticated: () => !!isAuthenticated.value,
    redirect: handleRedirect,
  });
});

async function onLoginSubmit(ctx: { formData: Record<string, unknown> }) {
  const fallback = _t('Login failed. Please try again later.');
  const data = ctx.formData;
  return runHandledAuthSubmit({
    submit: () =>
      runLoginSubmit({
        loading: !!loading.value,
        username: String(data.Username ?? ''),
        password: String(data.Password ?? ''),
        rememberMe: data.RememberMe === true,
        loginFailedMessage: fallback,
        login: (username, password, csrf, device, rememberMe) =>
          authStore.login(username, password, csrf, device, rememberMe),
        setError: setPageError,
      }),
    fallbackMessage: fallback,
    setError: setPageError,
    onSuccess: handleRedirect,
  });
}
</script>
