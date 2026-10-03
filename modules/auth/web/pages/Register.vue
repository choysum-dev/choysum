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
        <form class="flex flex-col gap-4" @submit.prevent="handleRegister">
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

            <ChoyField :data-invalid="fieldErrors.username ? true : undefined">
              <ChoyFieldLabel for="register-username">{{ _t('Username') }}</ChoyFieldLabel>
              <ChoyInput
                id="register-username"
                v-model="form.username"
                name="username"
                type="text"
                autocomplete="username"
                :placeholder="_t('Enter username')"
                :aria-invalid="fieldErrors.username ? true : undefined"
                @blur="validateUsernameField"
              />
              <ChoyFieldError :errors="fieldErrors.username ? [fieldErrors.username] : []" />
            </ChoyField>

            <ChoyField :data-invalid="fieldErrors.email ? true : undefined">
              <ChoyFieldLabel for="register-email">{{ _t('Email') }}</ChoyFieldLabel>
              <ChoyInput
                id="register-email"
                v-model="form.email"
                name="email"
                type="email"
                autocomplete="email"
                :placeholder="_t('Enter email address')"
                :aria-invalid="fieldErrors.email ? true : undefined"
                @blur="validateEmailField"
              />
              <ChoyFieldError :errors="fieldErrors.email ? [fieldErrors.email] : []" />
            </ChoyField>

            <ChoyField :data-invalid="fieldErrors.password ? true : undefined">
              <ChoyFieldLabel for="register-password">{{ _t('Password') }}</ChoyFieldLabel>
              <ChoyInput
                id="register-password"
                v-model="form.password"
                name="password"
                type="password"
                autocomplete="new-password"
                :placeholder="_t('Enter password')"
                :aria-invalid="fieldErrors.password ? true : undefined"
                @blur="validatePasswordField"
              />
              <ChoyFieldError :errors="fieldErrors.password ? [fieldErrors.password] : []" />
            </ChoyField>

            <ChoyField :data-invalid="fieldErrors.confirmPassword ? true : undefined">
              <ChoyFieldLabel for="register-confirm">{{ _t('Confirm Password') }}</ChoyFieldLabel>
              <ChoyInput
                id="register-confirm"
                v-model="form.confirmPassword"
                name="confirmPassword"
                type="password"
                autocomplete="new-password"
                :placeholder="_t('Re-enter password')"
                :aria-invalid="fieldErrors.confirmPassword ? true : undefined"
                @blur="validateConfirmPasswordField"
              />
              <ChoyFieldError :errors="fieldErrors.confirmPassword ? [fieldErrors.confirmPassword] : []" />
            </ChoyField>

            <ChoyField orientation="horizontal" data-testid="register-terms">
              <ChoyCheckbox
                id="register-terms"
                v-model="form.agreeTerms"
                @update:model-value="validateAgreeTermsField"
              />
              <ChoyFieldLabel for="register-terms" class="font-normal">
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
              </ChoyFieldLabel>
            </ChoyField>
            <ChoyFieldError :errors="fieldErrors.agreeTerms ? [fieldErrors.agreeTerms] : []" />

            <ChoyField>
              <ChoyButton
                type="submit"
                class="submit-button w-full"
                :disabled="loading || !form.agreeTerms"
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
        </form>
      </ChoyCard>
    </AuthPanel>
  </ChoyPage>
</template>

<script setup lang="ts">
import { ref, reactive } from 'vue';
import { useRouter, useRoute } from 'vue-router';
import { storeToRefs } from 'pinia';
import { X } from 'lucide-vue-next';
import { useAuthStore } from '../stores/auth';
import { ChoysumError } from '../error';
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
import { resolveLoginRedirect } from './login_form';

const { _t } = createTranslate('auth', { scope: 'web/pages/Register' });

const router = useRouter();
const route = useRoute();
const authStore = useAuthStore();
const { loading } = storeToRefs(authStore);

const form = reactive({
  username: '',
  email: '',
  password: '',
  confirmPassword: '',
  fullName: '',
  agreeTerms: false,
});

const fieldErrors = reactive({
  username: '',
  email: '',
  password: '',
  confirmPassword: '',
  agreeTerms: '',
});

const error = ref('');

function validateUsernameField(): boolean {
  const value = form.username;
  if (!value) {
    fieldErrors.username = _t('Enter username');
    return false;
  }
  if (value.length < 3) {
    fieldErrors.username = _t('Username must be at least 3 characters');
    return false;
  }
  if (!/^[a-zA-Z0-9_\-\.]+$/.test(value)) {
    fieldErrors.username = _t('Username can only contain letters, numbers, underscores, hyphens, and dots');
    return false;
  }
  fieldErrors.username = '';
  return true;
}

function validateEmailField(): boolean {
  const value = form.email;
  if (!value) {
    fieldErrors.email = _t('Enter email address');
    return false;
  }
  if (!/^[\w-]+(\.[\w-]+)*@[\w-]+(\.[\w-]+)+$/.test(value)) {
    fieldErrors.email = _t('Enter a valid email address');
    return false;
  }
  fieldErrors.email = '';
  return true;
}

function validatePasswordField(): boolean {
  const value = form.password;
  if (!value) {
    fieldErrors.password = _t('Enter password');
    return false;
  }
  if (value.length < 6) {
    fieldErrors.password = _t('Password must be at least 6 characters');
    return false;
  }
  fieldErrors.password = '';
  if (form.confirmPassword) {
    validateConfirmPasswordField();
  }
  return true;
}

function validateConfirmPasswordField(): boolean {
  const value = form.confirmPassword;
  if (!value) {
    fieldErrors.confirmPassword = _t('Re-enter password');
    return false;
  }
  if (value !== form.password) {
    fieldErrors.confirmPassword = _t('Passwords do not match');
    return false;
  }
  fieldErrors.confirmPassword = '';
  return true;
}

function validateAgreeTermsField(): boolean {
  if (!form.agreeTerms) {
    fieldErrors.agreeTerms = _t('You must agree to the Terms of Service and Privacy Policy');
    return false;
  }
  fieldErrors.agreeTerms = '';
  return true;
}

function validateForm(): boolean {
  const results = [
    validateUsernameField(),
    validateEmailField(),
    validatePasswordField(),
    validateConfirmPasswordField(),
    validateAgreeTermsField(),
  ];
  return results.every(Boolean);
}

/**
 * Validate the registration form and create a new user session.
 */
async function handleRegister() {
  if (loading.value || !validateForm()) return;

  try {
    error.value = '';
    await authStore.register(form.username, form.email, form.password, form.fullName ? { fullName: form.fullName } : {});
    await authStore.login(form.username, form.password);
    router.replace(resolveLoginRedirect(route.query.redirect?.toString()));
  } catch (err) {
    if (err instanceof ChoysumError) {
      error.value = err.message;
    } else {
      error.value = _t('Registration failed. Please try again later.');
      console.error('Registration flow failed:', err);
    }
  }
}
</script>
