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
              data-testid="module-op-plan-blocker-banner"
            >
              {{ _t('Resolve blockers before continuing') }}
            </div>

            <div class="flex flex-col gap-2" data-testid="module-op-changing-modules">
              <div class="text-sm font-semibold text-foreground">{{ _t('Modules that will change') }}</div>
              <ul
                v-if="plan.changingModules?.length"
                class="m-0 flex list-none flex-col gap-2 p-0 text-sm"
              >
                <li
                  v-for="item in plan.changingModules"
                  :key="item.moduleName"
                  class="flex flex-col gap-0.5"
                  :class="item.role === 'target' ? '' : 'ms-3 border-l border-border/80 ps-2.5'"
                  data-testid="module-op-changing-row"
                  :data-role="item.role || ''"
                >
                  <div class="flex flex-wrap items-center gap-x-2 gap-y-0.5">
                    <span
                      class="text-foreground"
                      :class="item.role === 'target' ? 'font-semibold' : 'font-medium text-foreground/90'"
                      >{{ item.moduleName }}</span
                    >
                    <span
                      v-if="formatVersionLine(item)"
                      class="font-mono text-xs text-muted-foreground"
                      >{{ formatVersionLine(item) }}</span
                    >
                    <span
                      v-if="item.origin === 'registry'"
                      class="text-xs text-muted-foreground"
                      >({{ formatOrigin(item.origin) }})</span
                    >
                    <ChoyBadge
                      v-if="changingEffectBadge(item)"
                      variant="outline"
                      class="h-5 px-1.5 text-[11px] font-normal"
                      :class="changingEffectBadgeClass(item)"
                      data-testid="module-op-effect-badge"
                    >
                      {{ changingEffectBadge(item) }}
                    </ChoyBadge>
                  </div>
                </li>
              </ul>
              <p v-else class="m-0 text-sm text-muted-foreground">{{ _t('No module changes required') }}</p>
            </div>

            <div
              v-if="plan.satisfiedModules?.length"
              class="flex flex-col gap-1.5"
              data-testid="module-op-satisfied-modules"
            >
              <button
                type="button"
                class="flex items-center gap-1 self-start text-sm text-muted-foreground hover:text-foreground"
                data-testid="module-op-satisfied-toggle"
                :aria-expanded="satisfiedOpen"
                @click="satisfiedOpen = !satisfiedOpen"
              >
                <ChevronRight
                  class="size-3.5 shrink-0 transition-transform"
                  :class="{ 'rotate-90': satisfiedOpen }"
                  aria-hidden="true"
                />
                <span>
                  {{
                    _t(
                      'Also %s dependencies with no changes',
                      String(plan.satisfiedModules.length),
                    )
                  }}
                </span>
              </button>
              <ul
                v-if="satisfiedOpen"
                class="m-0 ms-4 flex list-none flex-col gap-0.5 border-l border-border ps-3 p-0 text-sm text-muted-foreground"
                data-testid="module-op-satisfied-list"
              >
                <li
                  v-for="item in plan.satisfiedModules"
                  :key="item.moduleName"
                  class="flex flex-wrap items-center gap-x-2"
                >
                  <span>{{ item.moduleName }}</span>
                  <span
                    v-if="satisfiedVersionLine(item)"
                    class="font-mono text-xs text-muted-foreground"
                    >{{ satisfiedVersionLine(item) }}</span
                  >
                  <ChoyBadge
                    variant="outline"
                    class="h-5 px-1.5 text-[11px] font-normal"
                    data-testid="module-op-satisfied-badge"
                  >
                    {{ _t('Installed') }}
                  </ChoyBadge>
                </li>
              </ul>
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
            <div
              v-if="opStatus?.ReloadTriggered"
              class="flex flex-wrap gap-2 text-xs"
              data-testid="module-op-reload"
            >
              <span class="font-semibold text-foreground">{{ _t('Reload:') }}</span>
              <span class="text-foreground/80">{{
                opStatus?.ReloadFailed ? _t('Trigger Failed') : _t('Triggered')
              }}</span>
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
        <template v-else-if="step === 'result'">
          <ChoyButton
            v-if="opStatus?.ReloadFailed"
            type="button"
            variant="outline"
            data-testid="module-op-reload-retry"
            @click="onReloadRetry"
          >
            {{ _t('Retry page refresh') }}
          </ChoyButton>
          <ChoyButton type="button" data-testid="module-op-done" @click="close">
            {{ _t('Done') }}
          </ChoyButton>
        </template>
      </div>
    </ChoyDialogContent>
  </ChoyDialog>
</template>

<script setup lang="ts">
import { computed, ref, toRef, watch } from 'vue';
import { ChevronRight } from 'lucide-vue-next';
import type { WebModelStore } from '@/web/web/stores/modelStore';
import type MetaModuleOperation from '@/meta/service/models/module_operation';
import { createTranslate } from '@/web/web/i18n';
import { useModuleCatalogOpDrawerSession } from '../composables/useModuleCatalogOpDrawerSession';
import { formatModuleOpSummary } from './module_catalog_chrome';
import { formatPlanVersionDisplay, shouldShowPlanVersionArrow } from './module_op_plan_impact';
import type { ModuleOpPlanModuleView } from './module_op_plan_impact';
import {
  ChoyBadge,
  ChoyButton,
  ChoyDialog,
  ChoyDialogContent,
  ChoyDialogHeader,
  ChoyDialogTitle,
  ChoyMessage,
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
const satisfiedOpen = ref(false);

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

function formatOrigin(origin?: string): string {
  const o = String(origin || '').trim();
  if (o === 'registry') return _t('registry');
  if (o === 'local') return _t('local');
  return o;
}

function changingEffectBadge(item: ModuleOpPlanModuleView): string {
  switch (String(item.effect || '').trim()) {
    case 'install':
      return _t('Will install');
    case 'upgrade':
      return _t('Will upgrade');
    case 'uninstall':
      return _t('Will uninstall');
    default:
      return '';
  }
}

/** Outline base + light semantic tint; solid color stays on the confirm CTA. */
function changingEffectBadgeClass(item: ModuleOpPlanModuleView): string {
  switch (String(item.effect || '').trim()) {
    case 'uninstall':
      return 'border-destructive/45 text-destructive';
    case 'upgrade':
      return 'border-border text-muted-foreground';
    case 'install':
      return 'border-primary/45 text-primary';
    default:
      return '';
  }
}

function formatVersionLine(item: ModuleOpPlanModuleView): string {
  const effect = String(item.effect || '').trim();
  if (effect === 'uninstall') return '';
  const current = formatPlanVersionDisplay(item.currentVersion);
  const target = formatPlanVersionDisplay(item.targetVersion);
  if (!target) return '';
  if (shouldShowPlanVersionArrow(item)) {
    return `${current || '—'} → ${target}`;
  }
  return target;
}

/** Already-satisfied deps: prefer installed (current) version, else target. */
function satisfiedVersionLine(item: ModuleOpPlanModuleView): string {
  return formatPlanVersionDisplay(item.currentVersion) || formatPlanVersionDisplay(item.targetVersion);
}

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

const showRegistryOption = computed(() => {
  const action = String(props.action || activeAction.value || '').trim();
  return (
    action === 'upgrade' &&
    props.registryNewerThanLocal &&
    step.value === 'plan' &&
    !!plan.value
  );
});

const resultAlertBoxClass = computed(() => {
  if (!opStatus.value) return 'border-border bg-muted/40 text-foreground';
  if (opStatus.value.resultStatus === 'FAILED') {
    return 'border-destructive/40 bg-destructive/10 text-destructive';
  }
  return 'border-border bg-muted/40 text-foreground';
});

function formatSummary(summary: unknown) {
  if (summary && typeof summary === 'object' && (summary as any).syncResult) {
    const raw = (summary as any).syncResult as {
      total?: number;
      success?: number;
      failed?: number;
      catalogRows?: number;
    };
    const failed = Number(raw.failed) || 0;
    const catalogRows = Number(raw.catalogRows) || 0;
    if (failed > 0) {
      return _t(
        'Module catalog updated, %s modules (%s failed)',
        String(catalogRows),
        String(failed),
      );
    }
    return _t('Module catalog updated, %s modules', String(catalogRows));
  }
  return formatModuleOpSummary(summary);
}

function onOpenChange(next: boolean) {
  open.value = next;
}

function close() {
  open.value = false;
}

function onReloadRetry() {
  if (typeof window !== 'undefined' && typeof window.location?.reload === 'function') {
    window.location.reload();
  }
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
  () => plan.value?.Id,
  () => {
    satisfiedOpen.value = false;
  },
);

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
      satisfiedOpen.value = false;
      return;
    }
    const sameTarget =
      !!prev &&
      prev[0] === true &&
      prev[1] === action &&
      prev[2] === moduleName &&
      prev[3] === operationId &&
      prev[4] === props.operationStore;
    if (sameTarget && step.value !== 'plan') return;

    session.reset();
    // Progress / sync monitors run on the page strip; confirm dialog only plans.
    if (action === 'progress' || action === 'sync') {
      const id = String(operationId || '').trim();
      // Emit before close so the host can start progress before this watcher re-enters on close.
      if (id) {
        emit('applied', id);
      } else {
        ChoyMessage.warning(_t('Failed to get status'));
      }
      open.value = false;
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

defineExpose({
  formatOrigin,
  changingEffectBadge,
  changingEffectBadgeClass,
  formatVersionLine,
  formatSummary,
  resultAlertBoxClass,
  onOpenChange,
  opStatus,
});
</script>
