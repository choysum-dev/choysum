<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <ChoyPage :loading="loading" width="narrow" :padding="false" class="flex w-full flex-1 flex-col">
    <AuthPanel>
      <ChoyCard class="register-card w-full">
        <template #header>
          <h3 class="font-semibold leading-none tracking-tight">{{ _t('Create Account') }}</h3>
          <div
            v-if="error"
            class="register-error flex items-start justify-between gap-2 text-sm text-destructive"
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
            {{ _t('Fill in the fields below to register') }}
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
          :initial-values="registerInitialValues"
          :submit-handler="onRegisterSubmit"
        >
          <template #default="{ formData }">
            <AuthPageErrorClear
              :username="formData.Username"
              :password="formData.Password"
              :email="formData.Email"
              :confirm-password="formData.ConfirmPassword"
              :has-error="!!error"
              @clear="error = ''"
            />
            <ChoyFieldGroup class="register-fields" :style="{ gap: '0' }">
              <div class="register-username">
                <ChoyVarcharField
                  :store="formStore"
                  prop="Username"
                  :label="_t('Username')"
                  :placeholder="_t('Enter username')"
                  autocomplete="username"
                  name="username"
                  id="register-username"
                  buffer-strategy="live"
                  :nullable="false"
                  :show-word-limit="false"
                  show-inline-error
                  :rules="usernameRules"
                />
              </div>
              <div class="register-email">
                <ChoyVarcharField
                  :store="formStore"
                  prop="Email"
                  :label="_t('Email')"
                  type="email"
                  :placeholder="_t('Enter email address')"
                  autocomplete="email"
                  name="email"
                  id="register-email"
                  buffer-strategy="live"
                  :nullable="false"
                  :show-word-limit="false"
                  show-inline-error
                  :rules="emailRules"
                />
              </div>
              <div class="register-password">
                <ChoyVarcharField
                  :store="formStore"
                  prop="Password"
                  :label="_t('Password')"
                  type="password"
                  :placeholder="_t('Enter password')"
                  autocomplete="new-password"
                  name="password"
                  id="register-password"
                  buffer-strategy="live"
                  :nullable="false"
                  :show-word-limit="false"
                  show-inline-error
                  :rules="passwordRules"
                />
              </div>
              <div class="register-confirm">
                <ChoyVarcharField
                  :store="formStore"
                  prop="ConfirmPassword"
                  :label="_t('Confirm Password')"
                  type="password"
                  :placeholder="_t('Re-enter password')"
                  autocomplete="new-password"
                  name="confirmPassword"
                  id="register-confirm"
                  buffer-strategy="live"
                  :nullable="false"
                  :show-word-limit="false"
                  show-inline-error
                  :rules="confirmPasswordRules(formData.Password)"
                />
              </div>

              <div class="w-full" data-testid="register-terms">
                <ChoyBooleanField
                  v-slot:side
                  class="w-full"
                  :store="formStore"
                  prop="AgreeTerms"
                  widget="checkbox"
                  :label="''"
                  render-mode="inline"
                  buffer-strategy="live"
                  show-inline-error
                  :rules="agreeTermsRules"
                >
                    <label for="fld-AgreeTerms" class="text-sm leading-5">
                      {{ _t('I have read and agree to') }}
                      <a
                        href="#"
                        target="_blank"
                        class="text-primary underline-offset-4 hover:underline"
                        @click.stop
                      >
                        {{ _t('Terms of Service') }}
                      </a>
                      {{ _t('and') }}
                      <a
                        href="#"
                        target="_blank"
                        class="text-primary underline-offset-4 hover:underline"
                        @click.stop
                      >
                        {{ _t('Privacy Policy') }}
                      </a>
                    </label>
                </ChoyBooleanField>
              </div>

              <ChoyField>
                <ChoyButton
                  type="submit"
                  class="submit-button w-full"
                  :disabled="loading || !formData.AgreeTerms"
                >
                  {{ _t('Create Account') }}
                </ChoyButton>
                <ChoyFieldDescription class="text-center">
                  {{ _t('Already have an account?') }}
                  <router-link to="/login" class="text-primary hover:underline">{{
                    _t('Log in now')
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
  ChoyFieldDescription,
  ChoyFieldGroup,
  ChoyFormView,
  ChoyVarcharField,
  ChoyBooleanField,
  createLocalFormStore,
} from '@/web';
import { createTranslate } from '@/web/web/i18n';
import { resolveLoginRedirect, runHandledAuthSubmit } from './login_form';
import {
  registerAgreeTermsRules,
  registerConfirmPasswordRules,
  registerEmailRules,
  registerPasswordRules,
  registerUsernameRules,
  runRegisterSubmit,
} from './register_form';

const { _t } = createTranslate('auth', { scope: 'web/pages/Register' });

const router = useRouter();
const route = useRoute();
const authStore = useAuthStore();
const { loading } = storeToRefs(authStore);

const registerInitialValues = {
  Username: '',
  Email: '',
  Password: '',
  ConfirmPassword: '',
  AgreeTerms: false,
};

const formStore = createLocalFormStore({
  fields: [
    { name: 'Username', label: 'Username', type: 'varchar' },
    { name: 'Email', label: 'Email', type: 'varchar' },
    { name: 'Password', label: 'Password', type: 'varchar' },
    { name: 'ConfirmPassword', label: 'Confirm Password', type: 'varchar' },
    { name: 'AgreeTerms', label: 'I have read and agree to', type: 'boolean' },
  ],
  initialValues: registerInitialValues,
  storeId: 'auth.register',
});

// Rebuild rules when locale changes so inline errors stay translated.
const usernameRules = computed(() => registerUsernameRules(_t));
const emailRules = computed(() => registerEmailRules(_t));
const passwordRules = computed(() => registerPasswordRules(_t));
const agreeTermsRules = computed(() => registerAgreeTermsRules(_t));
const confirmPasswordRules = (password: unknown) =>
  registerConfirmPasswordRules(_t, () => password);

const error = ref('');

function setPageError(message: string) {
  error.value = message;
}

async function onRegisterSubmit(ctx: { formData: Record<string, unknown> }) {
  const data = ctx.formData;
  const registerFn = authStore.register.bind(authStore);
  const loginFn = authStore.login.bind(authStore);
  return runHandledAuthSubmit({
    submit: () =>
      runRegisterSubmit({
        loading: !!loading.value,
        username: String(data.Username ?? ''),
        email: String(data.Email ?? ''),
        password: String(data.Password ?? ''),
        registerFailedMessage: _t('Registration failed. Please try again later.'),
        loginFailedMessage: _t('Your account was created, but signing in failed. Please log in.'),
        register: (username, email, password) => registerFn(username, email, password),
        login: (username, password) => loginFn(username, password),
        setError: setPageError,
      }),
    fallbackMessage: _t('Registration failed. Please try again later.'),
    setError: setPageError,
    onSuccess: () => router.replace(resolveLoginRedirect(route.query.redirect?.toString())),
  });
}
</script>
