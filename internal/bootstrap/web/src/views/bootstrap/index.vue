<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: LGPL-3.0-or-later
-->

<template>
  <main class="bootstrap-page">
    <div class="bg-shape bg-shape-a"></div>
    <div class="bg-shape bg-shape-b"></div>

    <el-card class="panel" shadow="hover">
      <template #header>
        <div class="panel-header">
          <h1>Initial Setup</h1>
        </div>
      </template>

      <section v-if="showFailureScreen" class="failure-shell">
        <div class="failure-card">
          <h2 class="failure-title">{{ failureHeadline }}</h2>
          <p class="failure-copy">{{ failurePrimaryMessage }}</p>
          <p class="failure-note">{{ failureRecoveryMessage }}</p>
        </div>

        <div class="failure-actions">
          <el-button size="large" @click="onBackToSetup">Return to setup</el-button>
        </div>

        <el-collapse v-if="showFailureDetails" v-model="failureDetailPanels" class="status-details-collapse">
          <el-collapse-item name="technical">
            <template #title>
              <div class="technical-trigger">
                <span class="technical-trigger-title">Technical details</span>
              </div>
            </template>

            <el-descriptions :column="1" class="status-table-compact">
              <el-descriptions-item label="Setup ID">{{ operationId || latestStatus?.operationId || 'Not available' }}</el-descriptions-item>
              <el-descriptions-item label="Current step">{{ latestStatus ? formatFriendlyStage(latestStatus.stage) : 'Not available' }}</el-descriptions-item>
              <el-descriptions-item v-if="failureCode" label="Error code">{{ failureCode }}</el-descriptions-item>
              <el-descriptions-item v-if="failureDetails" label="Error">{{ failureDetails }}</el-descriptions-item>
            </el-descriptions>
          </el-collapse-item>
        </el-collapse>
      </section>

      <section v-else class="setup-shell">
        <div class="credentials-heading">
          <h2 class="credentials-title">Create your administrator account</h2>
          <p class="credentials-description">Choose the username and password for the first sign-in.</p>
        </div>

        <el-alert v-if="statusText" :title="statusText" :type="statusAlertType" :closable="false" show-icon class="status-alert" />

        <el-form class="bootstrap-form" label-position="top" :disabled="submitting">
          <el-form-item label="Administrator username" required>
            <el-input v-model.trim="adminUsername" autocomplete="username" placeholder="Enter administrator username" clearable size="large" />
          </el-form-item>

          <el-form-item label="Password" required>
            <el-input v-model="password" type="password" autocomplete="new-password" placeholder="Create a secure password" show-password size="large" />
          </el-form-item>

          <el-form-item class="actions">
            <el-button type="primary" native-type="button" :loading="submitting" :disabled="!canSubmit || submitting" size="large" @click="onSubmit">
              {{ submitting ? 'Starting setup...' : 'Start setup' }}
            </el-button>
          </el-form-item>
        </el-form>
      </section>
    </el-card>

    <el-overlay v-if="submitting" :show="true" class="bootstrap-overlay">
      <div class="overlay-content">
        <el-text class="overlay-kicker" size="small">{{ overlayKicker }}</el-text>
        <el-icon v-if="isRedirectingAfterSuccess" class="overlay-success-icon">
          <CircleCheckFilled />
        </el-icon>
        <el-icon v-else class="overlay-spinner is-loading">
          <Loading />
        </el-icon>
        <el-text class="overlay-title" size="large">{{ overlayTitle }}</el-text>
        <el-text class="overlay-subtitle" size="small">{{ overlayStatusText }}</el-text>
      </div>
    </el-overlay>
  </main>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue';
import { CircleCheckFilled, Loading } from '@element-plus/icons-vue';
import {
  ElAlert,
  ElButton,
  ElCard,
  ElCollapse,
  ElCollapseItem,
  ElDescriptions,
  ElDescriptionsItem,
  ElForm,
  ElFormItem,
  ElIcon,
  ElInput,
  ElOverlay,
  ElText,
} from 'element-plus';

import { InitializationStage, InitializationState } from '../../gen/bootstrap/internal/bootstrap/proto/bootstrap_pb';
import { getInitializationStatus, initializeWorkspace, type BootstrapAPIError, type InitializationStatus } from '../../api/bootstrap';

const adminUsername = ref('admin');
const password = ref('');
const submitting = ref(false);
const statusText = ref('');
const latestStatus = ref<InitializationStatus | null>(null);
const failureDetailPanels = ref<string[]>([]);
const operationId = ref('');
const redirectCountdown = ref<number | null>(null);
const failureCode = ref('');
const failureDetails = ref('');

const canSubmit = computed(() => adminUsername.value.trim().length > 0 && password.value.length > 0);
const isRedirectingAfterSuccess = computed(() => redirectCountdown.value !== null);
const hasFailureState = computed(() => latestStatus.value?.state === InitializationState.FAILED || failureCode.value !== '' || failureDetails.value !== '');
const showFailureScreen = computed(() => hasFailureState.value);
const showFailureDetails = computed(() => failureCode.value !== '' || failureDetails.value !== '' || operationId.value !== '');

const statusAlertType = computed(() => {
  if (statusText.value === '') {
    return 'info' as const;
  }
  if (statusText.value.toLowerCase().includes('complete')) {
    return 'success' as const;
  }
  return 'info' as const;
});

const failureHeadline = computed(() => {
  switch (failureCode.value) {
    case 'BOOTSTRAP_WORKSPACE_NOT_FRESH':
      return 'This instance already contains setup data';
    case 'BOOTSTRAP_CONFLICT':
      return 'Setup is already in progress';
    case 'BOOTSTRAP_INPUT_INVALID':
      return 'Review the setup details';
    default:
      return 'Setup needs your attention';
  }
});

const failurePrimaryMessage = computed(() => {
  return formatFriendlyFailurePrimaryMessage(failureCode.value, failureDetails.value);
});

const failureRecoveryMessage = computed(() => {
  switch (failureCode.value) {
    case 'BOOTSTRAP_WORKSPACE_NOT_FRESH':
      return 'Clear the existing setup data, then return to setup and try again.';
    case 'BOOTSTRAP_CONFLICT':
      return 'Wait for the active setup request to finish, then return to setup and try again.';
    case 'BOOTSTRAP_INPUT_INVALID':
      return 'Check the administrator username and password, then return to setup and try again.';
    default:
      return 'Resolve the issue, then return to setup and try again.';
  }
});

const overlayKicker = computed(() => (isRedirectingAfterSuccess.value ? 'Setup complete' : 'Initial setup'));

const overlayTitle = computed(() => {
  if (isRedirectingAfterSuccess.value) {
    return 'Redirecting to sign in';
  }
  return 'Preparing your system';
});

const overlayStatusText = computed(() => {
  if (redirectCountdown.value !== null) {
    return `Sign-in will open in ${redirectCountdown.value} seconds.`;
  }
  if (latestStatus.value) {
    return formatFriendlyStage(latestStatus.value.stage);
  }
  if (statusText.value !== '') {
    return statusText.value;
  }
  return 'Starting setup...';
});

function createIdempotencyKey(): string {
  if (typeof crypto !== 'undefined' && typeof crypto.randomUUID === 'function') {
    return crypto.randomUUID();
  }
  return `bootstrap-${Date.now()}-${Math.random().toString(16).slice(2)}`;
}

function delay(ms: number): Promise<void> {
  return new Promise(resolve => {
    setTimeout(resolve, ms);
  });
}

function toPollMs(v: bigint): number {
  const n = Number(v);
  if (!Number.isFinite(n) || n <= 0) {
    return 1000;
  }
  return Math.max(200, Math.min(5000, Math.floor(n)));
}

function clearFailureState(): void {
  failureCode.value = '';
  failureDetails.value = '';
  failureDetailPanels.value = [];
}

function resetSetupScreen(): void {
  latestStatus.value = null;
  operationId.value = '';
  redirectCountdown.value = null;
  statusText.value = '';
  clearFailureState();
}

function onBackToSetup(): void {
  submitting.value = false;
  resetSetupScreen();
}

function setFailureState(message: string, options?: { code?: string; details?: string }): void {
  statusText.value = message;
  failureCode.value = options?.code ?? '';
  failureDetails.value = options?.details ?? '';
  failureDetailPanels.value = [];
}

function formatFriendlyStage(stage: InitializationStage): string {
  switch (stage) {
    case InitializationStage.ACQUIRE_LOCK:
      return 'Securing the setup session';
    case InitializationStage.CHECK_WORKSPACE_FRESHNESS:
      return 'Checking whether setup can continue';
    case InitializationStage.ENSURE_MINIMAL_RUNTIME:
      return 'Preparing required system components';
    case InitializationStage.VALIDATE_RUNTIME_READY:
      return 'Checking required system files';
    case InitializationStage.UPDATE_ADMIN:
      return 'Saving your administrator account';
    case InitializationStage.SWITCH_MODE:
      return 'Finishing setup';
    case InitializationStage.DONE:
      return 'Setup is complete';
    default:
      return 'Continuing setup';
  }
}

function formatFriendlyFailureSummary(code: string): string {
  switch (code) {
    case 'BOOTSTRAP_WORKSPACE_NOT_FRESH':
      return 'This instance already contains setup data.';
    case 'BOOTSTRAP_CONFLICT':
      return 'Another setup request is already running.';
    case 'BOOTSTRAP_INPUT_INVALID':
      return 'The submitted setup details are invalid.';
    default:
      return 'Setup could not continue.';
  }
}

function formatFriendlyFailurePrimaryMessage(code: string, details: string): string {
  switch (code) {
    case 'BOOTSTRAP_WORKSPACE_NOT_FRESH':
      return 'Setup cannot continue because this instance already has setup information.';
    default: {
      const normalized = details.trim();
      if (normalized !== '') {
        return normalized;
      }
      return 'Setup could not continue with the current system state.';
    }
  }
}

function formatFriendlyFailureDetails(code: string, details: string): string {
  const normalized = details.trim();
  if (normalized !== '') {
    return normalized;
  }

  switch (code) {
    case 'BOOTSTRAP_WORKSPACE_NOT_FRESH':
      return 'Existing setup data was found.';
    case 'BOOTSTRAP_RUNTIME_PREPARE_FAILED':
      return 'Required system components could not be prepared.';
    case 'BOOTSTRAP_RUNTIME_NOT_READY':
      return 'Required system files are not ready.';
    case 'BOOTSTRAP_ADMIN_UPDATE_FAILED':
      return 'Administrator setup could not be saved.';
    case 'BOOTSTRAP_SWITCH_FAILED':
      return 'Setup completed but sign-in could not be activated.';
    case 'BOOTSTRAP_CONFLICT':
      return 'Another setup request is already running.';
    case 'BOOTSTRAP_INPUT_INVALID':
      return 'The submitted setup details are invalid.';
    default:
      return 'An unexpected setup error occurred.';
  }
}

function isTransientPollingError(error: unknown): boolean {
  const message = error instanceof Error ? error.message.toLowerCase() : '';
  if (message === '') {
    return false;
  }

  return (
    message.includes('rpc 14') ||
    message.includes('unavailable') ||
    message.includes('no children to pick from') ||
    message.includes('failed to fetch') ||
    message.includes('networkerror')
  );
}

async function isWebLoginReady(): Promise<boolean> {
  try {
    const resp = await fetch('/web/login', {
      method: 'GET',
      cache: 'no-store',
      redirect: 'manual',
    });

    if (resp.type === 'opaqueredirect') {
      return true;
    }
    return resp.status >= 200 && resp.status < 400;
  } catch {
    return false;
  }
}

async function redirectToLoginWithCountdown(targetUrl: string, seconds = 3): Promise<void> {
  for (let remaining = seconds; remaining > 0; remaining--) {
    redirectCountdown.value = remaining;
    statusText.value = `Setup complete. Redirecting to sign in in ${remaining} seconds...`;
    await delay(1000);
  }

  statusText.value = 'Setup complete. Redirecting to sign in...';
  window.location.assign(targetUrl);
}

async function pollInitialization(currentOperationId: string, nextPollAfterMs: bigint): Promise<void> {
  let pollMs = toPollMs(nextPollAfterMs);
  let consecutiveTransientErrors = 0;

  for (let attempt = 0; attempt < 120; attempt++) {
    await delay(pollMs);

    let status: InitializationStatus;
    try {
      status = await getInitializationStatus(currentOperationId);
      latestStatus.value = status;
      consecutiveTransientErrors = 0;
    } catch (error) {
      if (isTransientPollingError(error)) {
        consecutiveTransientErrors += 1;
        statusText.value = 'Finalizing setup...';

        if (await isWebLoginReady()) {
          await redirectToLoginWithCountdown('/web/login');
          return;
        }

        if (consecutiveTransientErrors < 20) {
          pollMs = Math.max(400, Math.min(2000, pollMs));
          continue;
        }
      }
      throw error;
    }

    if (status.state === InitializationState.SUCCEEDED || status.readyForLogin) {
      statusText.value = 'Setup complete.';
      await redirectToLoginWithCountdown(status.redirectUrl || '/web/login');
      return;
    }

    if (status.state === InitializationState.FAILED) {
      const code = status.errorCode;
      const details = formatFriendlyFailureDetails(code, status.errorMessage || '');
      setFailureState(formatFriendlyFailureSummary(code), {
        code,
        details,
      });
      return;
    }

    statusText.value = formatFriendlyStage(status.stage);
    pollMs = toPollMs(status.nextPollAfterMs);
  }

  setFailureState('Setup is taking longer than expected.', {
    details: 'Refresh the page and check the latest setup status before trying again.',
  });
}

async function onSubmit(): Promise<void> {
  if (submitting.value) {
    return;
  }

  if (!canSubmit.value) {
    statusText.value = 'Enter both the administrator username and password.';
    return;
  }

  resetSetupScreen();
  statusText.value = 'Starting setup...';
  submitting.value = true;

  try {
    const result = await initializeWorkspace({
      adminUsername: adminUsername.value,
      password: password.value,
      clientHashingEnabled: true,
      idempotencyKey: createIdempotencyKey(),
    });

    operationId.value = result.operationId;
    statusText.value = 'Preparing your system...';
    await pollInitialization(result.operationId, result.nextPollAfterMs);
  } catch (error) {
    const apiError = error as BootstrapAPIError;
    const code = typeof apiError.bootstrapCode === 'string' ? apiError.bootstrapCode : '';
    const details =
      typeof apiError.bootstrapDetails === 'string' && apiError.bootstrapDetails.trim() !== ''
        ? apiError.bootstrapDetails
        : error instanceof Error
          ? error.message
          : 'Unknown setup error';

    setFailureState(formatFriendlyFailureSummary(code), {
      code,
      details: formatFriendlyFailureDetails(code, details),
    });
  } finally {
    submitting.value = false;
  }
}
</script>

<style scoped>
.bootstrap-page {
  --bootstrap-bg-1: #f4f6f4;
  --bootstrap-bg-2: #dcece2;
  --bootstrap-ink: #1d2f24;
  --bootstrap-muted: #5e7467;

  box-sizing: border-box;
  position: relative;
  overflow: hidden;
  width: 100%;
  min-height: 100dvh;
  display: grid;
  place-items: center;
  padding: 0;
  background:
    radial-gradient(circle at 20% 20%, #eef5da 0%, transparent 30%), radial-gradient(circle at 80% 10%, #d8efe6 0%, transparent 34%),
    linear-gradient(150deg, var(--bootstrap-bg-1) 0%, var(--bootstrap-bg-2) 100%);
}

.bg-shape {
  position: absolute;
  z-index: 0;
  border-radius: 999px;
  filter: blur(16px);
  opacity: 0.65;
  pointer-events: none;
}

.bg-shape-a {
  width: 260px;
  height: 260px;
  background: #bfd9b0;
  top: -80px;
  left: -70px;
}

.bg-shape-b {
  width: 220px;
  height: 220px;
  background: #afcbd8;
  bottom: -70px;
  right: -60px;
}

.panel {
  position: relative;
  z-index: 1;
  width: min(568px, 100%);
  border-radius: 16px;
  border: 1px solid #dae5dc;
  backdrop-filter: blur(2px);
}

.panel-header {
  display: flex;
  align-items: flex-start;
  gap: 12px;
}

h1 {
  margin: 0;
  font-size: 18px;
  line-height: 1.15;
  font-weight: 600;
  letter-spacing: -0.01em;
  color: #3d5648;
  font-family:
    Avenir Next,
    Avenir,
    Segoe UI,
    Helvetica Neue,
    sans-serif;
}

.setup-shell,
.failure-shell {
  display: grid;
  gap: 10px;
}

.credentials-heading {
  display: grid;
  gap: 7px;
  max-width: 42rem;
}

.credentials-title {
  margin: 0;
  color: var(--bootstrap-ink);
  font-size: 23px;
  line-height: 1.1;
  font-weight: 700;
  letter-spacing: -0.02em;
}

.credentials-description,
.failure-copy,
.failure-note {
  margin: 0;
  color: var(--bootstrap-muted);
  line-height: 1.5;
}

.failure-title {
  margin: 0;
  color: var(--bootstrap-ink);
  font-size: 20px;
  line-height: 1.15;
}

.failure-card {
  display: grid;
  gap: 10px;
  max-width: 52rem;
}

.failure-actions {
  display: flex;
}

.failure-actions :deep(.el-button) {
  width: 100%;
}

.status-alert {
  margin-bottom: 6px;
}

.status-details-collapse {
  margin-bottom: 0;
  border: 0;
  border-radius: 0;
  background: transparent;
}

.status-details-collapse :deep(.el-collapse) {
  border-top: 0;
  border-bottom: 0;
}

.status-details-collapse :deep(.el-collapse-item__header) {
  min-height: 34px;
  padding: 0 2px;
  background: transparent;
  color: var(--bootstrap-muted);
  border-bottom: 0;
}

.status-details-collapse :deep(.el-collapse-item__wrap) {
  background: transparent;
  border-bottom: 0;
}

.status-details-collapse :deep(.el-collapse-item__content) {
  padding: 6px 0 0;
}

.technical-trigger {
  display: flex;
  align-items: center;
}

.technical-trigger-title {
  color: var(--bootstrap-muted);
  font-size: 11px;
  font-weight: 500;
}

.status-table-compact :deep(.el-descriptions__table) {
  background: rgba(255, 255, 255, 0.58);
  border-radius: 8px;
  border: 1px solid rgba(217, 228, 221, 0.7);
  overflow: hidden;
}

.status-table-compact :deep(.el-descriptions__body) {
  background: transparent;
}

.status-table-compact :deep(.el-descriptions__cell) {
  padding: 8px 10px;
}

.status-table-compact :deep(.el-descriptions__label:not(.is-bordered-label)) {
  display: inline-block;
  width: 104px;
  margin-right: 0;
  white-space: nowrap;
  vertical-align: top;
  color: var(--bootstrap-ink);
}

.status-table-compact :deep(.el-descriptions__content:not(.is-bordered-label)) {
  display: inline-block;
  width: calc(100% - 104px);
  vertical-align: top;
  line-height: 1.5;
}

.bootstrap-form {
  margin-top: 10px;
}

.bootstrap-form :deep(.el-form-item) {
  margin-bottom: 14px;
}

.bootstrap-form :deep(.el-form-item__label) {
  line-height: 1.35;
  margin-bottom: 5px;
}

.actions {
  margin-bottom: 0;
}

.actions :deep(.el-form-item__content) {
  display: flex;
}

.actions :deep(.el-button) {
  width: 100%;
}

:deep(.el-card__body) {
  display: grid;
  gap: 10px;
}

.bootstrap-overlay {
  z-index: 30;
  display: grid;
  place-items: center;
  background: rgba(23, 46, 34, 0.38);
  backdrop-filter: blur(2px);
}

.overlay-content {
  width: min(340px, calc(100vw - 40px));
  padding: 18px 18px 16px;
  border-radius: 14px;
  border: 1px solid #d2dfd4;
  background: rgba(255, 255, 255, 0.95);
  display: grid;
  justify-items: center;
  gap: 5px;
  box-shadow: 0 14px 34px rgba(16, 40, 28, 0.18);
}

.overlay-spinner {
  font-size: 24px;
  color: #2c8360;
}

.overlay-success-icon {
  font-size: 32px;
  color: #66bd42;
}

.overlay-kicker {
  color: #2c8360;
  font-size: 11px;
  font-weight: 700;
  letter-spacing: 0.08em;
  text-transform: uppercase;
}

.overlay-title {
  color: var(--bootstrap-ink);
  font-weight: 600;
  text-align: center;
}

.overlay-subtitle {
  color: var(--bootstrap-muted);
  text-align: center;
  line-height: 1.45;
}

@media (max-width: 640px) {
  .bootstrap-page {
    padding: 12px;
  }

  .panel {
    width: 100%;
  }

  .credentials-title,
  .failure-title {
    font-size: 18px;
  }

  .credentials-description {
    font-size: 14px;
    line-height: 1.45;
  }

  .overlay-content {
    width: min(320px, calc(100vw - 28px));
    padding: 16px 14px 14px;
  }

  h1 {
    font-size: 16px;
  }
}
</style>
