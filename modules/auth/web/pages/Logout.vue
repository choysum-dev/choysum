<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <ChoyPage :loading="loading" :padding="false" fill-height class="flex min-h-0 w-full flex-1 flex-col">
    <AuthPanel>
      <ChoyCard class="logout-card w-full">
        <template #header>
          <h3 class="font-semibold leading-none tracking-tight">{{ headerTitle }}</h3>
          <div
            v-if="error"
            class="logout-error flex items-start gap-2 text-sm text-destructive"
            role="alert"
          >
            <span class="min-w-0 leading-5">{{ error }}</span>
          </div>
          <p v-else class="text-sm text-foreground/70 tabular-nums">
            {{ headerDescription }}
            <button
              v-if="logoutSuccess"
              type="button"
              class="logout-go-to-login ms-2 cursor-pointer border-0 bg-transparent p-0 text-primary appearance-none shadow-none hover:underline"
              data-testid="logout-login-again"
              @click="navigateToLogin"
            >
              {{ _t('Go now') }}
            </button>
          </p>
        </template>

        <div v-if="error" class="flex flex-col gap-4">
          <ChoyButton class="w-full" data-testid="logout-retry" @click="retryLogout">
            {{ _t('Retry') }}
          </ChoyButton>
          <p class="text-center text-sm">
            <button
              type="button"
              class="logout-back-to-login cursor-pointer border-0 bg-transparent p-0 text-primary appearance-none shadow-none hover:underline"
              @click="navigateToLogin"
            >
              {{ _t('Back to Login') }}
            </button>
          </p>
        </div>
      </ChoyCard>
    </AuthPanel>
  </ChoyPage>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onBeforeUnmount } from 'vue';
import { useRouter } from 'vue-router';
import { storeToRefs } from 'pinia';
import { useAuthStore } from '../stores/auth';
import AuthPanel from '../components/AuthPanel.vue';
import { ChoyPage, ChoyCard, ChoyButton } from '@/web';
import { createTranslate } from '@/web/web/i18n';
import { formatLogoutError, nextLogoutCountdown, stopLogoutRedirectTimer } from './logout_page';

const { _t } = createTranslate('auth', { scope: 'web/pages/Logout' });

const router = useRouter();
const authStore = useAuthStore();
const { loading } = storeToRefs(authStore);

const logoutSuccess = ref(false);
const error = ref('');
const countdown = ref(3);
const headerTitle = computed(() => {
  if (logoutSuccess.value) return _t('Signed Out Successfully');
  if (error.value) return _t('Sign-out Failed');
  return _t('Signing Out');
});
const headerDescription = computed(() => {
  if (logoutSuccess.value) {
    return _t('Redirecting to the login page in %s seconds', countdown.value);
  }
  return _t('Signing out this session');
});
let autoRedirectTimer: ReturnType<typeof setInterval> | undefined;

onMounted(async () => {
  await performLogout();
});

onBeforeUnmount(() => {
  autoRedirectTimer = stopLogoutRedirectTimer(autoRedirectTimer);
});

async function performLogout() {
  autoRedirectTimer = stopLogoutRedirectTimer(autoRedirectTimer);
  try {
    await authStore.logout();
    logoutSuccess.value = true;
    autoRedirectTimer = setInterval(() => {
      const step = nextLogoutCountdown(countdown.value);
      countdown.value = step.countdown;
      if (step.done) {
        autoRedirectTimer = stopLogoutRedirectTimer(autoRedirectTimer);
        navigateToLogin();
      }
    }, 1000);
  } catch (err) {
    error.value = formatLogoutError(err, _t('Unknown error occurred during logout'));
  }
}

function navigateToLogin() {
  autoRedirectTimer = stopLogoutRedirectTimer(autoRedirectTimer);
  router.push('/login');
}

function retryLogout() {
  error.value = '';
  countdown.value = 3;
  performLogout();
}
</script>
