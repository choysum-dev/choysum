<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <ChoySheet v-model:open="open">
    <ChoySheetContent
      side="right"
      class="module-op-drawer module-op-dialog flex w-full flex-col sm:max-w-lg"
      data-testid="module-op-drawer"
    >
      <ChoySheetHeader>
        <ChoySheetTitle>{{ titleText }}</ChoySheetTitle>
        <ChoySheetDescription v-if="sessionOperationId">
          {{ _t('Operation %s', sessionOperationId) }}
        </ChoySheetDescription>
      </ChoySheetHeader>

      <div class="flex min-h-0 flex-1 flex-col gap-3 overflow-y-auto px-4 pb-2 text-sm">
        <div v-if="session.step.value === 'plan'" class="flex flex-col gap-3">
          <div v-if="session.planLoading.value" class="flex flex-col gap-2" data-testid="module-op-plan-loading">
            <div v-for="n in 5" :key="n" class="h-4 animate-pulse rounded bg-muted" />
          </div>
          <template v-else-if="session.plan.value">
            <div
              v-if="session.plan.value.blockers?.length"
              class="rounded-md border border-destructive/40 bg-destructive/10 px-3 py-2 text-destructive"
            >
              {{ _t('Resolve blockers before continuing') }}
            </div>
            <div v-else class="rounded-md border border-border bg-muted/40 px-3 py-2 text-foreground">
              {{ _t('Confirm the impact of this operation') }}
            </div>

            <div class="flex flex-col gap-2">
              <div class="font-semibold text-foreground">{{ _t('Affected Modules') }}</div>
              <ul class="m-0 flex list-none flex-col gap-1.5 p-0 text-xs text-foreground/80">
                <li
                  v-for="item in session.plan.value.affectedModules"
                  :key="item.moduleName"
                  class="flex flex-wrap items-center gap-2"
                >
                  <span class="font-semibold">{{ item.moduleName }}</span>
                  <span class="text-foreground/60">{{ item.currentVersion || '—' }}</span>
                  <span v-if="item.targetVersion" class="text-foreground/60">→ {{ item.targetVersion }}</span>
                  <span v-if="item.origin" class="text-muted-foreground">({{ item.origin }})</span>
                  <span class="text-primary">{{ item.reason || '' }}</span>
                </li>
              </ul>
            </div>

            <div v-if="session.plan.value.affectedApps?.length" class="flex flex-col gap-2">
              <div class="font-semibold text-foreground">{{ _t('Affected Apps') }}</div>
              <p class="text-xs text-foreground/80">{{ session.plan.value.affectedApps.join(', ') }}</p>
            </div>

            <div v-if="session.plan.value.risks?.length" class="flex flex-col gap-2">
              <div class="font-semibold text-foreground">{{ _t('Risks') }}</div>
              <ul class="m-0 flex list-none flex-col gap-1.5 p-0 text-xs">
                <li v-for="risk in session.plan.value.risks" :key="risk.code" class="flex flex-wrap items-center gap-2">
                  <span class="rounded bg-amber-500/15 px-1.5 py-0.5 text-[11px] font-medium text-amber-800 dark:text-amber-200">{{
                    risk.code
                  }}</span>
                  <span class="text-foreground/70">{{ risk.message || '' }}</span>
                </li>
              </ul>
            </div>

            <div v-if="session.plan.value.blockers?.length" class="flex flex-col gap-2">
              <div class="font-semibold text-foreground">{{ _t('Blockers') }}</div>
              <ul class="m-0 flex list-none flex-col gap-1.5 p-0 text-xs">
                <li
                  v-for="blocker in session.plan.value.blockers"
                  :key="blocker.code"
                  class="flex flex-wrap items-center gap-2"
                >
                  <span class="rounded bg-destructive/15 px-1.5 py-0.5 text-[11px] font-medium text-destructive">{{
                    blocker.code
                  }}</span>
                  <span class="text-foreground/70">{{ blocker.message || '' }}</span>
                </li>
              </ul>
            </div>

            <button
              v-if="showRegistryOption"
              type="button"
              class="self-start rounded-md border border-border px-2.5 py-1 text-xs hover:bg-muted"
              data-testid="module-op-use-registry"
              :disabled="session.planLoading.value || session.useRegistryOrigin.value"
              @click="onUseRegistry"
            >
              {{
                session.useRegistryOrigin.value
                  ? _t('Using registry version')
                  : _t('Use registry version')
              }}
            </button>
          </template>
          <p v-else class="text-muted-foreground">
            {{ moduleName ? _t('Planning for %s…', moduleName) : _t('Select a module action to plan an operation.') }}
          </p>
        </div>

        <div v-else class="flex flex-col gap-3">
          <div class="rounded-md border px-3 py-2" :class="resultAlertBoxClass">
            {{
              session.step.value === 'progress'
                ? _t('Operation in progress, please wait')
                : session.resultTitle.value
            }}
          </div>
          <div class="flex flex-col gap-2">
            <div class="font-semibold text-foreground">{{ _t('Execution Status') }}</div>
            <div class="module-op-status-row flex flex-wrap items-center gap-2 text-xs text-foreground/80">
              <span class="font-semibold text-foreground">{{ _t('Status:') }}</span>
              <span
                class="rounded px-1.5 py-0.5 text-[11px] font-medium"
                data-testid="module-op-status"
                >{{ session.opStatus.value?.status || '—' }}</span
              >
              <span class="font-semibold text-foreground">{{ _t('Phase:') }}</span>
              <span class="text-foreground/80">{{ phaseLabel }}</span>
            </div>
            <div
              v-if="session.opStatus.value?.resultStatus"
              class="flex flex-wrap gap-2 text-xs"
            >
              <span class="font-semibold text-foreground">{{ _t('Result:') }}</span>
              <span data-testid="module-op-result">{{ session.opStatus.value?.resultStatus }}</span>
            </div>
            <div v-if="session.opStatus.value?.summary" class="flex flex-wrap gap-2 text-xs">
              <span class="font-semibold text-foreground">{{ _t('Summary:') }}</span>
              <span class="text-foreground/80">{{ formatSummary(session.opStatus.value?.summary) }}</span>
            </div>
            <div
              v-if="session.opStatus.value?.errorDomain || session.opStatus.value?.errorCode"
              class="flex flex-wrap gap-2 text-xs"
            >
              <span class="font-semibold text-foreground">{{ _t('Error:') }}</span>
              <span class="text-foreground/80"
                >{{ session.opStatus.value?.errorDomain || '—' }} / {{ session.opStatus.value?.errorCode || '—' }}</span
              >
            </div>
            <div
              v-if="session.opStatus.value?.ReloadTriggered || session.opStatus.value?.ReloadFailed"
              class="flex flex-wrap gap-2 text-xs"
              data-testid="module-op-reload"
            >
              <span class="font-semibold text-foreground">{{ _t('Reload:') }}</span>
              <span class="text-foreground/80">
                {{
                  session.opStatus.value?.ReloadFailed
                    ? _t('Data written; page did not refresh')
                    : _t('Triggered')
                }}
              </span>
            </div>
            <button
              v-if="session.step.value === 'result' && session.opStatus.value?.ReloadFailed"
              type="button"
              class="self-start rounded-md border border-border px-2.5 py-1 text-xs hover:bg-muted"
              data-testid="module-op-reload-retry"
              @click="reloadPage"
            >
              {{ _t('Refresh page') }}
            </button>
          </div>
          <p v-if="session.step.value === 'progress'" class="border-t border-border pt-3 text-xs text-muted-foreground">
            {{ _t('Closing this drawer keeps the operation running; use View Progress on the card.') }}
          </p>
        </div>
      </div>

      <div class="flex justify-end gap-2 border-t border-border px-4 py-3">
        <ChoyButton type="button" variant="outline" :disabled="session.planLoading.value" @click="close">
          {{ session.step.value === 'plan' ? _t('Cancel') : _t('Close') }}
        </ChoyButton>
        <ChoyButton
          v-if="session.step.value === 'plan'"
          type="button"
          data-testid="module-op-confirm"
          :disabled="!session.canConfirm.value"
          @click="onConfirm"
        >
          {{ _t('Confirm') }}
        </ChoyButton>
        <ChoyButton v-else-if="session.step.value === 'result'" type="button" data-testid="module-op-done" @click="close">
          {{ _t('Done') }}
        </ChoyButton>
      </div>
    </ChoySheetContent>
  </ChoySheet>
</template>

<script setup lang="ts">
import { computed, toRef, watch } from 'vue';
import type { WebModelStore } from '@/web/web/stores/modelStore';
import type MetaModuleOperation from '@/meta/service/models/module_operation';
import { createTranslate } from '@/web/web/i18n';
import { useModuleOpDrawerSession } from '../composables/useModuleOpDrawerSession';
import { formatModuleOpSummary } from './module_catalog_chrome';
import {
  ChoyButton,
  ChoySheet,
  ChoySheetContent,
  ChoySheetDescription,
  ChoySheetHeader,
  ChoySheetTitle,
} from '@/web';

defineOptions({ name: 'ModuleOpDrawer' });

const { _t } = createTranslate('meta', { scope: 'web/views/ModuleOpDrawer' });

const open = defineModel<boolean>('open', { default: false });

const props = withDefaults(
  defineProps<{
    operationStore?: WebModelStore<MetaModuleOperation> | null;
    operationId?: string;
    moduleName?: string;
    action?: 'install' | 'upgrade' | 'uninstall' | 'sync' | 'load_demo' | 'progress' | '';
    /** When true, upgrade plan may offer Origin=registry. */
    registryNewerThanLocal?: boolean;
  }>(),
  {
    operationStore: null,
    operationId: '',
    moduleName: '',
    action: '',
    registryNewerThanLocal: false,
  },
);

const emit = defineEmits<{
  settled: [];
}>();

const operationStoreRef = toRef(props, 'operationStore');

const session = useModuleOpDrawerSession({
  operationStore: operationStoreRef as any,
  isOpen: () => open.value,
  msgs: {
    failedToLoadPlan: _t('Failed to load plan'),
    applyFailed: _t('Operation request failed'),
    jobStillRunning: _t('Job is still running in the background; refresh later'),
    serviceRestarting: _t('Service is restarting; status will retry automatically'),
    failedToGetStatus: _t('Failed to get status'),
    completed: _t('Completed'),
    operationFailed: _t('Operation Failed'),
    succeededReloadFailed: _t('Data written; page did not refresh'),
    operationSucceeded: _t('Operation Succeeded'),
  },
  onSettled: () => emit('settled'),
});

const titleText = computed(() => {
  const action = String(props.action || session.activeAction.value || '').trim();
  const name = String(props.moduleName || session.activeModuleName.value || '').trim();
  let label = _t('Module Operation');
  if (action === 'install') label = _t('Install Module');
  else if (action === 'upgrade') label = _t('Upgrade Module');
  else if (action === 'uninstall') label = _t('Uninstall Module');
  else if (action === 'progress') label = _t('Operation Progress');
  else if (action === 'sync') label = _t('Sync Modules');
  else if (action === 'load_demo') label = _t('Load Demo Data');
  return name ? `${label} · ${name}` : label;
});

const sessionOperationId = computed(
  () => String(session.plan.value?.Id || props.operationId || '').trim(),
);

const showRegistryOption = computed(
  () =>
    props.action === 'upgrade' &&
    props.registryNewerThanLocal &&
    session.step.value === 'plan' &&
    !!session.plan.value,
);

const phaseLabel = computed(() => {
  const status = String(session.opStatus.value?.status || '').toLowerCase();
  if (status === 'queued' || status === 'dispatching') return _t('Queued');
  if (status === 'running') return _t('Running');
  if (session.opStatus.value?.ReloadTriggered) return _t('Reload');
  if (status === 'succeeded' || status === 'failed' || status === 'cancelled') return _t('Done');
  return session.opStatus.value?.summary && typeof session.opStatus.value.summary === 'object'
    ? String((session.opStatus.value.summary as any).message || '—')
    : '—';
});

const resultAlertBoxClass = computed(() => {
  if (session.step.value === 'progress') return 'border-border bg-muted/40 text-foreground';
  if (!session.opStatus.value) return 'border-border bg-muted/40 text-foreground';
  if (session.opStatus.value.resultStatus === 'FAILED') {
    return 'border-destructive/40 bg-destructive/10 text-destructive';
  }
  if (session.opStatus.value.resultStatus !== 'SUCCEEDED') {
    return 'border-border bg-muted/40 text-foreground';
  }
  if (session.opStatus.value.ReloadFailed) {
    return 'border-amber-500/40 bg-amber-500/10 text-amber-900 dark:text-amber-100';
  }
  return 'border-emerald-500/40 bg-emerald-500/10 text-emerald-900 dark:text-emerald-100';
});

function formatSummary(summary: unknown) {
  return formatModuleOpSummary(summary);
}

function close() {
  open.value = false;
}

function reloadPage() {
  if (typeof window !== 'undefined') window.location.reload();
}

async function onConfirm() {
  await session.confirmApply();
}

async function onUseRegistry() {
  await session.replanWithRegistry();
}

function isPlanAction(action: string): action is 'install' | 'upgrade' | 'uninstall' {
  return action === 'install' || action === 'upgrade' || action === 'uninstall';
}

watch(
  () =>
    [
      open.value,
      String(props.action || ''),
      String(props.moduleName || ''),
      String(props.operationId || ''),
    ] as const,
  async ([isOpen, action, moduleName, operationId], prev) => {
    if (!isOpen) {
      session.stop();
      return;
    }
    // Ignore operationId-only churn while an in-flight progress/result session is active.
    const sameTarget =
      !!prev &&
      prev[0] === true &&
      prev[1] === action &&
      prev[2] === moduleName &&
      (action === 'progress' ? prev[3] === operationId : true);
    if (sameTarget && session.step.value !== 'plan') return;

    session.reset();
    if (action === 'progress' || (!isPlanAction(action) && operationId)) {
      const id = String(operationId || '').trim();
      if (id) await session.watchProgress(id);
      return;
    }
    if (isPlanAction(action) && moduleName) {
      await session.runPlan({ moduleName, action });
    }
  },
  { immediate: true },
);
</script>
