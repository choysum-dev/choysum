<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <OPage :loading="loading" class="login-page-container">
    <el-card class="login-card" shadow="hover">
      <template #header>
        <div class="card-header">
          <h3>{{ _t('User Login') }}</h3>
        </div>
      </template>

      <transition name="fade">
        <el-alert v-if="error" :title="error" type="error" :closable="true" @close="error = ''" />
      </transition>

      <el-form ref="loginForm" :model="form" :rules="rules" label-position="top" @keydown="handleKeyDown" @submit.prevent="handleLogin">
        <el-form-item prop="username" :label="_t('Username')">
          <el-input v-model="form.username" :placeholder="_t('Enter username')" :prefix-icon="User" autocomplete="username" />
        </el-form-item>

        <el-form-item prop="password" :label="_t('Password')">
          <el-input v-model="form.password" :placeholder="_t('Enter password')" :prefix-icon="Lock" type="password" autocomplete="current-password" show-password />
        </el-form-item>

        <div class="login-options">
          <el-checkbox v-model="form.rememberMe">{{ _t('Remember me') }}</el-checkbox>
        </div>

        <el-form-item>
          <el-button type="primary" native-type="submit" :loading="loading" class="submit-button">{{ _t('Log In') }}</el-button>
        </el-form-item>

        <div v-if="showRegisterLink" class="register-link">
          {{ _t("Don't have an account?") }}
          <router-link to="/register">{{ _t('Register now') }}</router-link>
        </div>
      </el-form>
    </el-card>
  </OPage>
</template>

<script setup lang="ts">
import { ref, reactive, computed, onMounted } from 'vue';
import { useRouter, useRoute } from 'vue-router';
import { storeToRefs } from 'pinia';
import { useAuthStore } from '../stores/auth';
import { ChoysumError } from '../error';
import OPage from '@/web/web/components/page/OPage.vue';
import { ElForm, ElFormItem, ElInput, ElButton, ElCheckbox, ElAlert, ElCard } from 'element-plus';
import { User, Lock } from '@element-plus/icons-vue';
import type { FormRules } from 'element-plus';
import { createTranslate } from '@/web/web/i18n';
import { useI18nStore, langToUiKey } from '@/web/web/stores/i18nStore';

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

const loginForm = ref();
const form = reactive<LoginFormData>({
  username: '',
  password: '',
  rememberMe: true,
});

const rules = computed<FormRules<LoginFormData>>(() => ({
  username: [
    {
      required: true,
      message: _t('Enter username'),
      trigger: 'blur',
    },
  ],
  password: [
    {
      required: true,
      message: _t('Enter password'),
      trigger: 'blur',
    },
  ],
}));

const error = ref('');
const showRegisterLink = computed(() => import.meta.env.CHOYSUM_ENABLE_REGISTRATION !== false);

/**
 * Redirect the user to the requested destination after login.
 */
function handleRedirect() {
  const redirect = route.query.redirect?.toString() || '/';
  router.replace(redirect);
}

onMounted(async () => {
  // Capture the current route so we don't redirect if the user already
  // navigated away (e.g. clicked "Register now") while ensureAuthReady runs.
  const currentPath = route.path;

  // Ensure auth initialization runs so stale tokens (e.g. from a previous
  // database reset) are cleared before the user submits the login form.
  // Without this the auth interceptor may try to refresh an invalid token
  // during the Login RPC and produce noisy console errors.
  try {
    await authStore.ensureAuthReady();
  } catch {
    // Stale tokens are cleared by initAuth internally; continue to login.
  }

  if (route.path === currentPath && isAuthenticated.value) {
    handleRedirect();
  }
});

/**
 * Validate the form and start the login flow.
 */
async function handleLogin() {
  if (!loginForm.value) return;

  try {
    await loginForm.value.validate();
    error.value = '';

    await authStore.login(form.username, form.password, '', '', form.rememberMe);

    // Apply persisted User.Language when present (terminology lang → UI locale).
    try {
      await authStore.loadUser(true);
      const preferredLang = String((authStore.currentUser as any)?.Language || '').trim();
      const i18nStore = useI18nStore();
      if (preferredLang) {
        await i18nStore.setUiKey(langToUiKey(preferredLang));
      }
      i18nStore.setDisplayOverrides((authStore.currentUser as any)?.Preferences?.display ?? null);
    } catch {
      // Preference apply is best-effort; login already succeeded.
    }

    handleRedirect();
  } catch (err) {
    if (err instanceof ChoysumError) {
      // Show translated message; keep machine code on the error object for FE routing.
      error.value = err.message;
    } else {
      error.value = _t('Login failed. Please try again later.');
      console.error('Login flow failed:', err);
    }
  }
}

/**
 * Submit the form when the user presses Enter.
 */
function handleKeyDown(event: KeyboardEvent) {
  if (event.key === 'Enter') {
    handleLogin();
  }
}
</script>

<style lang="scss" scoped>
.login-page-container {
  &.o-page--with-padding {
    padding: 0;
  }

  :deep(.o-page__body) {
    display: flex;
    align-items: center;
    justify-content: center;
    height: 100%;
  }
}

.login-card {
  width: 100%;
  max-width: 400px;
}

.card-header {
  text-align: center;
  h3 {
    margin: 0;
    font-size: var(--el-font-size-large);
    font-weight: var(--el-font-weight-bold);
  }
}

.login-options {
  display: flex;
  justify-content: space-between;
  margin-block-end: var(--el-margin-base, 16px);
}

.submit-button {
  width: 100%;
}

.register-link {
  text-align: center;
  margin-block-start: var(--el-margin-base, 16px);
  font-size: var(--el-font-size-small, 14px);
  color: var(--el-text-color-secondary);

  a {
    color: var(--el-color-primary);
    text-decoration: none;
    margin-inline-start: var(--el-margin-small, 4px);

    &:hover {
      text-decoration: underline;
    }
  }
}

:deep(.el-alert) {
  margin-block-end: var(--el-margin-medium, 20px);
}
</style>
