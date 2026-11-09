<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <ChoyDialog :open="open" @update:open="onOpenChange">
    <ChoyDialogContent
      class="module-op-dialog flex max-h-[min(85vh,40rem)] w-[min(92vw,32rem)] max-w-[min(92vw,32rem)] flex-col gap-0 overflow-hidden p-0"
      data-testid="module-op-dialog"
      :show-close-button="true"
    >
      <ChoyDialogHeader class="shrink-0 space-y-1 border-b border-border px-6 py-4 text-start">
        <ChoyDialogTitle>{{ titleText }}</ChoyDialogTitle>
      </ChoyDialogHeader>

      <div class="flex min-h-0 flex-1 flex-col gap-3 overflow-y-auto px-6 py-4 text-sm">
        <div v-if="step === 'plan'" class="flex flex-col gap-3">
          <div v-if="planLoading" class="flex flex-col gap-2" data-testid="module-op-plan-loading">
            <div v-for="n in 5" :key="n" class="h-4 animate-pulse rounded bg-muted" />
          </div>
          <template v-else-if="plan">
            <div
              v-if="plan.blockers?.length"
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
                  v-for="item in plan.affectedModules"
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

            <div v-if="plan.affectedApps?.length" class="flex flex-col gap-2">
              <div class="font-semibold text-foreground">{{ _t('Affected Apps') }}</div>
              <p class="text-xs text-foreground/80">{{ plan.affectedApps.join(', ') }}</p>
            </div>

            <div v-if="plan.risks?.length" class="flex flex-col gap-2">
              <div class="font-semibold text-foreground">{{ _t('Risks') }}</div>
              <ul class="m-0 flex list-none flex-col gap-1.5 p-0 text-xs">
                <li v-for="risk in plan.risks" :key="risk.code" class="flex flex-wrap items-center gap-2">
                  <span
                    class="rounded bg-amber-500/15 px-1.5 py-0.5 text-[11px] font-medium text-amber-800 dark:text-amber-200"
                    >{{ risk.code }}</span
                  >
                  <span class="text-foreground/70">{{ risk.message || '' }}</span>
                </li>
              </ul>
            </div>

            <div v-if="plan.blockers?.length" class="flex flex-col gap-2">
              <div class="font-semibold text-foreground">{{ _t('Blockers') }}</div>
              <ul class="m-0 flex list-none flex-col gap-1.5 p-0 text-xs">
                <li
                  v-for="blocker in plan.blockers"
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
              :disabled="planLoading || useRegistryOrigin"
              @click="onUseRegistry"
            >
              {{
                useRegistryOrigin
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
          <div
            class="rounded-md border px-3 py-2"
            data-testid="module-op-alert"
            :class="resultAlertBoxClass"
          >
            {{ resultTitle }}
          </div>
          <div class="flex flex-col gap-2">
            <div class="font-semibold text-foreground">{{ _t('Execution Status') }}</div>
            <span class="sr-only" data-testid="module-op-status">{{ opStatus?.status || '' }}</span>
            <div
              v-if="opStatus?.resultStatus"
              class="flex flex-wrap gap-2 text-xs"
            >
              <span class="font-semibold text-foreground">{{ _t('Result:') }}</span>
              <span data-testid="module-op-result">{{ opStatus?.resultStatus }}</span>
            </div>
            <div v-if="opStatus?.summary" class="flex flex-wrap gap-2 text-xs">
              <span class="font-semibold text-foreground">{{ _t('Summary:') }}</span>
              <span class="text-foreground/80">{{ formatSummary(opStatus?.summary) }}</span>
            </div>
            <div
              v-if="opStatus?.errorDomain || opStatus?.errorCode"
              class="flex flex-wrap gap-2 text-xs"
            >
              <span class="font-semibold text-foreground">{{ _t('Error:') }}</span>
              <span class="text-foreground/80"
                >{{ opStatus?.errorDomain || '—' }} / {{ opStatus?.errorCode || '—' }}</span
              >
            </div>
          </div>
        </div>
      </div>

      <div class="flex shrink-0 justify-end gap-2 border-t border-border px-6 py-3">
        <ChoyButton
          type="button"
          variant="outline"
          :disabled="planLoading || executeLoading"
          data-testid="module-op-cancel"
          @click="close"
        >
          {{ step === 'plan' ? _t('Cancel') : _t('Close') }}
        </ChoyButton>
        <ChoyButton
          v-if="step === 'plan'"
          type="button"
          data-testid="module-op-confirm"
          :variant="action === 'uninstall' ? 'destructive' : 'default'"
          :disabled="!canConfirm || executeLoading"
          @click="onConfirm"
        >
          {{ _t('Confirm') }}
        </ChoyButton>
        <ChoyButton
          v-else-if="step === 'result'"
          type="button"
          data-testid="module-op-done"
          @click="close"
        >
          {{ _t('Done') }}
        </ChoyButton>
      </div>
    </ChoyDialogContent>
  </ChoyDialog>
</template>

<script setup lang="ts">
import { computed, toRef, watch } from 'vue';
import type { WebModelStore } from '@/web/web/stores/modelStore';
import type MetaModuleOperation from '@/meta/service/models/module_operation';
import { createTranslate } from '@/web/web/i18n';
import { useModuleCatalogOpDrawerSession } from '../composables/useModuleCatalogOpDrawerSession';
import { formatModuleOpSummary } from './module_catalog_chrome';
import {
  ChoyButton,
  ChoyDialog,
  ChoyDialogContent,
  ChoyDialogHeader,
  ChoyDialogTitle,
} from '@/web';

defineOptions({ name: 'ModuleCatalogOpDialog' });

// Keep OpDrawer i18n scope so existing pot/zh_CN entries continue to resolve.
const { _t } = createTranslate('meta', { scope: 'web/views/ModuleCatalogOpDrawer' });

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
  settled: [status?: { resultStatus?: string; partialFailure?: boolean } | null];
  /** Fired after Apply succeeds so hosts can refresh ActiveOperationId while progress is locked. */
  applied: [operationId: string];
}>();

const operationStoreRef = toRef(props, 'operationStore');

const session = useModuleCatalogOpDrawerSession({
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
    operationSucceededPartial: _t('Module list updated with warnings'),
  },
  onSettled: status => emit('settled', status),
  onApplied: operationId => emit('applied', operationId),
});

const {
  step,
  planLoading,
  executeLoading,
  plan,
  opStatus,
  activeModuleName,
  activeAction,
  useRegistryOrigin,
  canConfirm,
  resultTitle,
} = session;

const titleText = computed(() => {
  const action = String(props.action || activeAction.value || '').trim();
  const name = String(props.moduleName || activeModuleName.value || '').trim();
  let label = _t('Module Operation');
  if (action === 'install') label = _t('Install Module');
  else if (action === 'upgrade') label = _t('Upgrade Module');
  else if (action === 'uninstall') label = _t('Uninstall Module');
  else if (action === 'load_demo') label = _t('Load Demo Data');
  return name ? `${label} · ${name}` : label;
});

const showRegistryOption = computed(
  () =>
    props.action === 'upgrade' &&
    props.registryNewerThanLocal &&
    step.value === 'plan' &&
    !!plan.value,
);

const resultAlertBoxClass = computed(() => {
  if (!opStatus.value) return 'border-border bg-muted/40 text-foreground';
  if (opStatus.value.resultStatus === 'FAILED') {
    return 'border-destructive/40 bg-destructive/10 text-destructive';
  }
  return 'border-border bg-muted/40 text-foreground';
});

function formatSummary(summary: unknown) {
  if (summary && typeof summary === 'object' && (summary as any).syncResult) {
    const sync = (summary as any).syncResult as {
      total: number;
      success: number;
      failed: number;
      catalogRows: number;
    };
    if (sync.failed > 0) {
      return _t(
        'Synced %s/%s sources (%s failed); catalog has %s modules',
        String(sync.success),
        String(sync.total),
        String(sync.failed),
        String(sync.catalogRows),
      );
    }
    return _t('Synced %s sources; catalog has %s modules', String(sync.total), String(sync.catalogRows));
  }
  return formatModuleOpSummary(summary);
}

function onOpenChange(next: boolean) {
  open.value = next;
}

function close() {
  open.value = false;
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
      props.operationStore,
    ] as const,
  async ([isOpen, action, moduleName, operationId], prev) => {
    if (!isOpen) {
      session.stop();
      return;
    }
    const sameTarget =
      !!prev &&
      prev[0] === true &&
      prev[1] === action &&
      prev[2] === moduleName &&
      prev[4] === props.operationStore;
    if (sameTarget && step.value !== 'plan') return;

    session.reset();
    // Progress / sync monitors run on the page strip; confirm dialog only plans.
    if (action === 'progress' || action === 'sync') {
      const id = String(operationId || '').trim();
      open.value = false;
      if (id) emit('applied', id);
      return;
    }
    if (action === 'load_demo') {
      await session.runPlanDemo();
      return;
    }
    if (isPlanAction(action) && moduleName) {
      await session.runPlan({ moduleName, action });
    }
  },
  { immediate: true },
);
</script>
