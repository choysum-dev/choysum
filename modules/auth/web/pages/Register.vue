<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <ChoyPage :loading="loading" width="narrow" :padding="false" class="w-full">
    <AuthPanel>
      <ChoyCard
        :title="_t('Create Account')"
        :description="_t('Fill in the fields below to register')"
        class="register-card w-full"
      >
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
            <ChoyFieldGroup class="gap-4">
              <ChoyField v-if="error">
                <div
                  class="flex items-center justify-between gap-2 rounded-md border border-destructive/40 bg-destructive/10 px-3 py-2 text-sm text-destructive"
                  role="alert"
                >
                  <span class="min-w-0 leading-5">{{ error }}</span>
                  <button
                    type="button"
                    class="inline-flex size-6 shrink-0 items-center justify-center rounded-sm border-0 bg-transparent p-0 text-destructive/70 appearance-none shadow-none outline-none hover:bg-destructive/15 hover:text-destructive focus-visible:ring-2 focus-visible:ring-destructive/40"
                    :aria-label="_t('Close')"
                    @click="error = ''"
                  >
                    <X class="size-3.5" />
                  </button>
                </div>
              </ChoyField>

              <div class="register-username">
              <ChoyVarcharField
                :store="formStore"
                prop="Username"
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
                type="password"
                :placeholder="_t('Re-enter password')"
                autocomplete="new-password"
                name="confirmPassword"
                id="register-confirm"
                buffer-strategy="live"
                :nullable="false"
                :show-word-limit="false"
                show-inline-error
                :rules="registerConfirmPasswordRules(_t, () => formData.Password)"
              />
              </div>

              <div class="flex items-start gap-2" data-testid="register-terms">
                <ChoyBooleanField
                  :store="formStore"
                  prop="AgreeTerms"
                  widget="checkbox"
                  :label="''"
                  render-mode="inline"
                  buffer-strategy="live"
                  show-inline-error
                  :rules="agreeTermsRules"
                />
                <label for="fld-AgreeTerms" class="mt-1 text-sm leading-5">
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
import { ref } from 'vue';
import { useRouter, useRoute } from 'vue-router';
import { storeToRefs } from 'pinia';
import { X } from 'lucide-vue-next';
import { useAuthStore } from '../stores/auth';
import AuthPanel from '../components/AuthPanel.vue';
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
import { resolveLoginRedirect } from './login_form';
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
    { name: 'Username', label: _t('Username'), type: 'varchar' },
    { name: 'Email', label: _t('Email'), type: 'varchar' },
    { name: 'Password', label: _t('Password'), type: 'varchar' },
    { name: 'ConfirmPassword', label: _t('Confirm Password'), type: 'varchar' },
    { name: 'AgreeTerms', label: _t('I have read and agree to'), type: 'boolean' },
  ],
  initialValues: registerInitialValues,
  storeId: 'auth.register',
});

const usernameRules = registerUsernameRules(_t);
const emailRules = registerEmailRules(_t);
const passwordRules = registerPasswordRules(_t);
const agreeTermsRules = registerAgreeTermsRules(_t);

const error = ref('');

async function onRegisterSubmit(ctx: { formData: Record<string, unknown> }) {
  const data = ctx.formData || {};
  try {
    const registerFn = authStore.register.bind(authStore);
    const loginFn = authStore.login.bind(authStore);
    const ok = await runRegisterSubmit({
      loading: !!loading.value,
      username: String(data.Username ?? ''),
      email: String(data.Email ?? ''),
      password: String(data.Password ?? ''),
      registerFailedMessage: _t('Registration failed. Please try again later.'),
      register: (username, email, password) => registerFn(username, email, password),
      login: (username, password) => loginFn(username, password),
      setError: message => {
        error.value = message;
      },
    });
    if (ok) router.replace(resolveLoginRedirect(route.query.redirect?.toString()));
  } catch (err) {
    error.value = err instanceof Error ? err.message : _t('Registration failed. Please try again later.');
  }
  return { handled: true, skipSuccessMessage: true };
}
</script>
