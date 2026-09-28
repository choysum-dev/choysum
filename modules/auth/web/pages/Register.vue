<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <ChoyPage :loading="loading" width="narrow" :padding="false" class="register-page-container mx-auto w-full max-w-md py-6">
    <ChoyCard :title="_t('Create Account')" class="register-card w-full">
      <transition name="fade">
        <div
          v-if="error"
          class="mb-4 flex items-start justify-between gap-2 rounded-md border border-destructive/40 bg-destructive/10 px-3 py-2 text-sm text-destructive"
          role="alert"
        >
          <span>{{ error }}</span>
          <button type="button" class="text-destructive/80 hover:text-destructive" :aria-label="_t('Close')" @click="error = ''">
            ×
          </button>
        </div>
      </transition>

      <form class="flex flex-col gap-3" @submit.prevent="handleRegister">
        <label class="flex flex-col gap-1 text-sm">
          <span>{{ _t('Username') }}</span>
          <input
            v-model="form.username"
            name="username"
            type="text"
            autocomplete="username"
            :placeholder="_t('Enter username')"
            class="rounded-md border border-border bg-background px-3 py-2 text-sm"
            :class="{ 'border-destructive': fieldErrors.username }"
            @blur="validateUsernameField"
          />
          <span v-if="fieldErrors.username" class="text-xs text-destructive">{{ fieldErrors.username }}</span>
        </label>

        <label class="flex flex-col gap-1 text-sm">
          <span>{{ _t('Email') }}</span>
          <input
            v-model="form.email"
            name="email"
            type="email"
            autocomplete="email"
            :placeholder="_t('Enter email address')"
            class="rounded-md border border-border bg-background px-3 py-2 text-sm"
            :class="{ 'border-destructive': fieldErrors.email }"
            @blur="validateEmailField"
          />
          <span v-if="fieldErrors.email" class="text-xs text-destructive">{{ fieldErrors.email }}</span>
        </label>

        <label class="flex flex-col gap-1 text-sm">
          <span>{{ _t('Password') }}</span>
          <input
            v-model="form.password"
            name="password"
            type="password"
            autocomplete="new-password"
            :placeholder="_t('Enter password')"
            class="rounded-md border border-border bg-background px-3 py-2 text-sm"
            :class="{ 'border-destructive': fieldErrors.password }"
            @blur="validatePasswordField"
          />
          <span v-if="fieldErrors.password" class="text-xs text-destructive">{{ fieldErrors.password }}</span>
        </label>

        <label class="flex flex-col gap-1 text-sm">
          <span>{{ _t('Confirm Password') }}</span>
          <input
            v-model="form.confirmPassword"
            name="confirmPassword"
            type="password"
            autocomplete="new-password"
            :placeholder="_t('Re-enter password')"
            class="rounded-md border border-border bg-background px-3 py-2 text-sm"
            :class="{ 'border-destructive': fieldErrors.confirmPassword }"
            @blur="validateConfirmPasswordField"
          />
          <span v-if="fieldErrors.confirmPassword" class="text-xs text-destructive">{{ fieldErrors.confirmPassword }}</span>
        </label>

        <label class="flex items-start gap-2 text-sm" data-testid="register-terms">
          <input v-model="form.agreeTerms" type="checkbox" class="mt-0.5 size-4 rounded border-border" @change="validateAgreeTermsField" />
          <span>
            {{ _t('I have read and agree to') }}
            <a href="#" target="_blank" class="text-primary hover:underline">{{ _t('Terms of Service') }}</a>
            {{ _t('and') }}
            <a href="#" target="_blank" class="text-primary hover:underline">{{ _t('Privacy Policy') }}</a>
          </span>
        </label>
        <span v-if="fieldErrors.agreeTerms" class="-mt-2 text-xs text-destructive">{{ fieldErrors.agreeTerms }}</span>

        <ChoyButton
          type="submit"
          class="submit-button w-full"
          :disabled="loading || !form.agreeTerms"
        >
          {{ _t('Create Account') }}
        </ChoyButton>

        <div class="login-link text-center text-sm text-foreground/70">
          {{ _t('Already have an account?') }}
          <router-link to="/login" class="text-primary hover:underline">{{ _t('Log in now') }}</router-link>
        </div>
      </form>
    </ChoyCard>
  </ChoyPage>
</template>

<script setup lang="ts">
import { ref, reactive } from 'vue';
import { useRouter, useRoute } from 'vue-router';
import { storeToRefs } from 'pinia';
import { useAuthStore } from '../stores/auth';
import { ChoysumError } from '../error';
import { ChoyPage, ChoyCard, ChoyButton } from '@/web';
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

<style lang="scss" scoped>
.register-page-container {
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 100%;
}

.login-link :deep(a) {
  margin-inline-start: 0.25rem;
}
</style>
