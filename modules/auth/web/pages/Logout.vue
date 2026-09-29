<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <ChoyPage :loading="loading" width="narrow" :padding="false" class="mx-auto flex min-h-full w-full max-w-lg items-center justify-center">
    <ChoyCard :title="_t('Sign Out')" class="logout-card w-full">
      <div class="logout-view flex flex-col items-center gap-4 py-4 text-center">
        <transition name="fade" mode="out-in">
          <div v-if="logoutSuccess" key="success" class="flex flex-col items-center gap-3">
            <CheckCircle2 class="size-12 text-emerald-600" aria-hidden="true" />
            <h4 class="text-lg font-semibold">{{ _t('Signed Out Successfully') }}</h4>
            <p class="max-w-md text-sm text-foreground/70">{{ redirectSubtitle }}</p>
            <div class="flex flex-wrap justify-center gap-2 pt-2">
              <ChoyButton @click="navigateToLogin">{{ _t('Log In Again') }}</ChoyButton>
              <ChoyButton variant="outline" @click="navigateToHome">{{ _t('Back to Home') }}</ChoyButton>
            </div>
          </div>

          <div v-else-if="error" key="error" class="flex flex-col items-center gap-3">
            <XCircle class="size-12 text-destructive" aria-hidden="true" />
            <h4 class="text-lg font-semibold">{{ _t('Sign-out Failed') }}</h4>
            <p class="max-w-md text-sm text-foreground/70">{{ error }}</p>
            <div class="flex flex-wrap justify-center gap-2 pt-2">
              <ChoyButton @click="retryLogout">{{ _t('Retry') }}</ChoyButton>
              <ChoyButton variant="outline" @click="navigateToHome">{{ _t('Back to Home') }}</ChoyButton>
              <ChoyButton variant="outline" @click="navigateToLogin">{{ _t('Back to Login') }}</ChoyButton>
            </div>
          </div>

          <div v-else key="loading" class="flex flex-col items-center gap-3">
            <Loader2 class="size-12 animate-spin text-foreground/50" aria-hidden="true" />
            <h4 class="text-lg font-semibold">{{ _t('Signing Out') }}</h4>
            <p class="max-w-md text-sm text-foreground/70">
              {{ _t('Please wait while your account is being signed out securely...') }}
            </p>
          </div>
        </transition>
      </div>
    </ChoyCard>
  </ChoyPage>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onBeforeUnmount } from 'vue';
import { useRouter } from 'vue-router';
import { storeToRefs } from 'pinia';
import { CheckCircle2, Loader2, XCircle } from 'lucide-vue-next';
import { useAuthStore } from '../stores/auth';
import { ChoysumError } from '../error';
import { ChoyPage, ChoyCard, ChoyButton } from '@/web';
import { createTranslate } from '@/web/web/i18n';

const { _t } = createTranslate('auth', { scope: 'web/pages/Logout' });

const router = useRouter();
const authStore = useAuthStore();
const { loading } = storeToRefs(authStore);

const logoutSuccess = ref(false);
const error = ref('');
const countdown = ref(5);
const redirectSubtitle = computed(() =>
  _t('Thank you for using our service. Redirecting to the login page in %s seconds...', countdown.value)
);
let autoRedirectTimer: ReturnType<typeof setInterval> | undefined;

onMounted(async () => {
  await performLogout();
});

onBeforeUnmount(() => {
  if (autoRedirectTimer) {
    clearInterval(autoRedirectTimer);
  }
});

/**
 * Perform logout and start the redirect countdown on success.
 */
async function performLogout() {
  try {
    await authStore.logout();
    logoutSuccess.value = true;
    autoRedirectTimer = setInterval(() => {
      countdown.value--;
      if (countdown.value <= 0) {
        clearInterval(autoRedirectTimer);
        navigateToLogin();
      }
    }, 1000);
  } catch (err) {
    error.value = err instanceof ChoysumError ? err.message : err instanceof Error ? err.message : _t('Unknown error occurred during logout');
    console.error('Logout failed:', err);
  }
}

/**
 * Navigate to the login page and stop the auto redirect timer.
 */
function navigateToLogin() {
  if (autoRedirectTimer) {
    clearInterval(autoRedirectTimer);
  }
  router.push('/login');
}

/**
 * Navigate to the home page and stop the auto redirect timer.
 */
function navigateToHome() {
  if (autoRedirectTimer) {
    clearInterval(autoRedirectTimer);
  }
  router.push('/');
}

/**
 * Reset the error state and retry the logout flow.
 */
function retryLogout() {
  error.value = '';
  performLogout();
}
</script>

