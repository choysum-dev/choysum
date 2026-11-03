<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <ChoyPage :loading="loading" fill-height class="flex min-h-0 w-full flex-1 flex-col">
    <AuthPanel>
      <ChoyCard class="setup-card w-full">
        <template #header>
          <h3 class="font-semibold leading-none">{{ _t('Initial Setup') }}</h3>
          <div
            v-if="error"
            class="setup-error flex items-start justify-between gap-2 text-sm text-destructive"
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
          <p v-else class="text-sm text-foreground/60">
            {{ _t('Choose an administrator username and password to finish setup') }}
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
          :initial-values="setupInitialValues"
          :submit-handler="onSetupSubmit"
        >
          <template #default="{ formData }">
            <AuthPageErrorClear
              :username="formData.Username"
              :password="formData.Password"
              :confirm-password="formData.ConfirmPassword"
              :has-error="!!error"
              @clear="error = ''"
            />
            <ChoyFieldGroup class="setup-fields" :style="{ gap: '0' }">
              <div class="setup-username">
                <ChoyVarcharField
                  :store="formStore"
                  prop="Username"
                  :label="_t('Username')"
                  orientation="vertical"
                  :placeholder="_t('Enter username')"
                  autocomplete="username"
                  name="username"
                  id="setup-username"
                  buffer-strategy="live"
                  :nullable="false"
                  :show-word-limit="false"
                  show-inline-error
                  :rules="usernameRules"
                />
              </div>
              <div class="setup-password">
                <ChoyVarcharField
                  :store="formStore"
                  prop="Password"
                  :label="_t('Password')"
                  orientation="vertical"
                  widget="password"
                  :placeholder="_t('Enter password')"
                  autocomplete="new-password"
                  name="password"
                  id="setup-password"
                  buffer-strategy="live"
                  :nullable="false"
                  :show-word-limit="false"
                  show-inline-error
                  :rules="passwordRules"
                />
              </div>
              <div class="setup-confirm">
                <ChoyVarcharField
                  :store="formStore"
                  prop="ConfirmPassword"
                  :label="_t('Confirm Password')"
                  orientation="vertical"
                  widget="password"
                  :placeholder="_t('Re-enter password')"
                  autocomplete="new-password"
                  name="confirmPassword"
                  id="setup-confirm"
                  buffer-strategy="live"
                  :nullable="false"
                  :show-word-limit="false"
                  show-inline-error
                  :rules="confirmPasswordRules(formData.Password)"
                />
              </div>

              <ChoyField>
                <ChoyButton type="submit" class="submit-button w-full" :disabled="loading">
                  {{ loading ? _t('Complete Setup') + '…' : _t('Complete Setup') }}
                </ChoyButton>
              </ChoyField>
            </ChoyFieldGroup>
          </template>
        </ChoyFormView>
      </ChoyCard>
    </AuthPanel>
  </ChoyPage>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue';
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
  ChoyFieldGroup,
  ChoyFormView,
  ChoyVarcharField,
  createLocalFormStore,
} from '@/web';
import { createTranslate } from '@/web/web/i18n';
import { resolveLoginRedirect, runHandledAuthSubmit } from './login_form';
import {
  setupConfirmPasswordRules,
  setupPasswordRules,
  setupUsernameRules,
  runSetupSubmit,
} from './setup_form';

const { _t } = createTranslate('auth', { scope: 'web/pages/Setup' });

const router = useRouter();
const route = useRoute();
const authStore = useAuthStore();
const { loading } = storeToRefs(authStore);

const setupInitialValues = {
  Username: '',
  Password: '',
  ConfirmPassword: '',
};

const formStore = createLocalFormStore({
  fields: [
    { name: 'Username', label: 'Username', type: 'varchar' },
    { name: 'Password', label: 'Password', type: 'varchar' },
    { name: 'ConfirmPassword', label: 'Confirm Password', type: 'varchar' },
  ],
  initialValues: setupInitialValues,
  storeId: 'auth.setup',
});

const usernameRules = computed(() => setupUsernameRules(_t));
const passwordRules = computed(() => setupPasswordRules(_t));
const confirmPasswordRules = (password: unknown) =>
  setupConfirmPasswordRules(_t, () => password);

const error = ref('');

function setPageError(message: string) {
  error.value = message;
}

async function onSetupSubmit(ctx: { formData: Record<string, unknown> }) {
  const data = ctx.formData;
  const completeFn = authStore.completeInitialSetup.bind(authStore);
  const loginFn = authStore.login.bind(authStore);
  return runHandledAuthSubmit({
    submit: () =>
      runSetupSubmit({
        loading: !!loading.value,
        username: String(data.Username ?? ''),
        password: String(data.Password ?? ''),
        setupFailedMessage: _t('Initial setup failed. Please try again later.'),
        loginFailedMessage: _t('Setup succeeded, but signing in failed. Please log in.'),
        completeInitialSetup: (username, password) => completeFn(username, password),
        login: (username, password) => loginFn(username, password),
        setError: setPageError,
      }),
    fallbackMessage: _t('Initial setup failed. Please try again later.'),
    setError: setPageError,
    onSuccess: () => router.replace(resolveLoginRedirect(route.query.redirect?.toString())),
  });
}
</script>
