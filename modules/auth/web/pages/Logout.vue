<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <OPage :loading="loading" class="logout-page-container">
    <el-card class="logout-card" shadow="hover">
      <template #header>
        <div class="card-header">
          <h3>Sign Out</h3>
        </div>
      </template>
      <div class="logout-view">
        <transition name="fade" mode="out-in">
          <el-result
            v-if="logoutSuccess"
            key="success"
            icon="success"
            title="Signed Out Successfully"
            :sub-title="`Thank you for using our service. Redirecting to the login page in ${countdown} seconds...`"
          >
            <template #extra>
              <el-button type="primary" @click="navigateToLogin">Log In Again</el-button>
              <el-button @click="navigateToHome">Back to Home</el-button>
            </template>
          </el-result>

          <el-result v-else-if="error" key="error" icon="error" title="Sign-out Failed" :sub-title="error">
            <template #extra>
              <el-button type="primary" @click="retryLogout">Retry</el-button>
              <el-button @click="navigateToHome">Back to Home</el-button>
              <el-button @click="navigateToLogin">Back to Login</el-button>
            </template>
          </el-result>

          <el-result v-else key="loading" icon="info" title="Signing Out" sub-title="Please wait while your account is being signed out securely..." />
        </transition>
      </div>
    </el-card>
  </OPage>
</template>

<script setup lang="ts">
import { ref, onMounted, onBeforeUnmount } from 'vue';
import { useRouter } from 'vue-router';
import { storeToRefs } from 'pinia';
import { useAuthStore } from '../stores/auth';
import { ChoysumError } from '../error';
import OPage from '@/web/web/components/page/OPage.vue';
import { ElResult, ElButton, ElCard } from 'element-plus';

const router = useRouter();
const authStore = useAuthStore();
const { loading } = storeToRefs(authStore);

const logoutSuccess = ref(false);
const error = ref('');
const countdown = ref(5);
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
    error.value = err instanceof ChoysumError ? err.message : err instanceof Error ? err.message : 'Unknown error occurred during logout';
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

<style lang="scss" scoped>
.logout-page-container {
  &.o-page--with-padding {
    padding: 0;
  }

  :deep(.o-page__body) {
    display: flex;
    align-items: center;
    justify-content: center;
    height: 100%;
    background-color: var(--el-bg-color-page);
  }
}

.logout-card {
  width: 100%;
  max-width: 500px;
}

.card-header {
  text-align: center;
  h3 {
    margin: 0;
    font-size: var(--el-font-size-large);
    font-weight: var(--el-font-weight-bold);
  }
}

.logout-view {
  text-align: center;

  :deep(.el-result__icon) {
    margin-block-end: var(--el-margin-large, 20px);
  }

  :deep(.el-result__title) {
    margin-block-start: 0;
  }

  :deep(.el-result__subtitle) {
    margin-block-start: var(--el-margin-small, 10px);
    max-width: 500px;
    margin-inline: auto;
  }

  :deep(.el-button) {
    margin: var(--el-margin-extra-small, 5px);
  }
}
</style>
