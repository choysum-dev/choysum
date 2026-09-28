<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <div class="module-kanban-view">
    <div class="sr-only" aria-hidden="true">
      <ChoyVirtualField :store="store" prop="ModuleName" />
      <ChoyVirtualField :store="store" prop="Version" />
      <ChoyVirtualField :store="store" prop="LocalVersion" />
      <ChoyVirtualField :store="store" prop="RegistryVersion" />
      <ChoyVirtualField :store="store" prop="InstalledStatus" />
      <ChoyVirtualField :store="store" prop="InstalledVersion" />
      <ChoyVirtualField :store="store" prop="Available" />
      <ChoyVirtualField :store="store" prop="OriginTypes" />
      <ChoyVirtualField :store="store" prop="LastSyncAt" />
      <ChoyVirtualField :store="store" prop="ManifestJson" />
    </div>

    <ChoyKanbanView
      v-model:lanes="choyLanes"
      :show-header="showHeader"
      :show-actions="true"
      :readonly="true"
      @card-click="onCardClick"
    >
      <template #system-actions />

      <template #user-actions>
        <div class="flex items-center gap-1">
          <ChoyButton size="sm" :title="_t('Board View')" @click="toKanban">
            <LayoutGrid class="size-4" aria-hidden="true" />
          </ChoyButton>
          <ChoyButton
            v-if="canRoute('meta.route.module_list')"
            variant="outline"
            size="sm"
            :title="_t('List View')"
            @click="toList"
          >
            <List class="size-4" aria-hidden="true" />
          </ChoyButton>
          <ChoyButton
            v-if="canRoute('meta.route.module_history')"
            variant="outline"
            size="sm"
            :title="_t('Operation History')"
            @click="toHistory"
          >
            <History class="size-4" aria-hidden="true" />
          </ChoyButton>
          <ChoyButton
            v-if="hasAction(moduleSyncIndexAction)"
            variant="outline"
            size="sm"
            :title="_t('Sync Index')"
            :disabled="syncLoading"
            @click="onSyncIndex"
          >
            <RefreshCw class="size-4" :class="{ 'animate-spin': syncLoading }" aria-hidden="true" />
          </ChoyButton>
        </div>
      </template>

      <template #search>
        <ChoySearchView :store="store" @query-update="onSearch" />
      </template>

      <template #card="{ card }">
        <div class="module-card flex flex-col gap-1.5 text-xs">
          <div class="module-card__title flex items-center justify-between gap-2">
            <span class="name text-sm font-semibold text-foreground">{{ recordField(card, 'ModuleName') }}</span>
            <span
              class="module-card__status shrink-0 rounded px-1.5 py-0.5 text-[11px] font-medium"
              :class="statusBadgeClass(String(payloadRecord(card).InstalledStatus ?? ''), payloadRecord(card).Available)"
            >
              {{ statusLabel(String(payloadRecord(card).InstalledStatus ?? ''), payloadRecord(card).Available) }}
            </span>
          </div>
          <div class="flex justify-between gap-3 text-foreground/70">
            <span>{{ _t('Local:') }} {{ recordField(card, 'LocalVersion') || '—' }}</span>
            <span>{{ _t('Registry:') }} {{ recordField(card, 'RegistryVersion') || '—' }}</span>
          </div>
          <div class="flex justify-between gap-3 text-foreground/70">
            <span>{{ _t('Display:') }} {{ recordField(card, 'Version') || '—' }}</span>
            <span>{{ _t('Installed:') }} {{ recordField(card, 'InstalledVersion') || '—' }}</span>
          </div>
          <div class="flex justify-between gap-3 text-foreground/70">
            <span>{{ _t('Origin:') }} {{ recordField(card, 'OriginTypes') || recordField(card, 'OriginType') || 'local' }}</span>
            <span>{{ _t('Synced:') }} {{ formatDate(recordField(card, 'LastSyncAt')) || '—' }}</span>
          </div>
          <div class="min-h-[2rem] text-foreground/80">
            {{ manifestSummary(recordField(card, 'ManifestJson')) || _t('No description') }}
          </div>
          <div class="flex flex-wrap gap-2 pt-1">
            <ChoyButton
              v-if="!isInstalled(String(recordField(card, 'InstalledStatus') || '')) && hasAction(moduleInstallAction)"
              size="sm"
              variant="outline"
              :disabled="recordField(card, 'Available') === false"
              @click.stop="onActionClick('install', payloadRecord(card))"
            >
              {{ _t('Install') }}
            </ChoyButton>
            <ChoyButton
              v-if="isInstalled(String(recordField(card, 'InstalledStatus') || '')) && hasAction(moduleUpgradeAction)"
              size="sm"
              variant="outline"
              @click.stop="onActionClick('upgrade', payloadRecord(card))"
            >
              {{ _t('Upgrade') }}
            </ChoyButton>
            <ChoyButton
              v-if="isInstalled(String(recordField(card, 'InstalledStatus') || '')) && hasAction(moduleUninstallAction)"
              size="sm"
              variant="outline"
              class="text-destructive hover:text-destructive"
              @click.stop="onActionClick('uninstall', payloadRecord(card))"
            >
              {{ _t('Uninstall') }}
            </ChoyButton>
          </div>
        </div>
      </template>

      <template #card-empty>
        <div class="text-xs opacity-60">{{ _t('No modules') }}</div>
      </template>
    </ChoyKanbanView>

    <Teleport to="body">
      <div
        v-if="dialogVisible"
        class="module-op-dialog-overlay fixed inset-0 z-[100] flex items-center justify-center bg-black/50 p-4"
        role="presentation"
      >
        <div
          ref="dialogRef"
          tabindex="-1"
          class="module-op-dialog flex max-h-[90vh] w-full max-w-[680px] flex-col rounded-lg border border-border bg-background shadow-lg outline-none"
          role="dialog"
          aria-modal="true"
          :aria-labelledby="dialogTitleId"
          @click.stop
          @keydown="onDialogKeydown"
        >
          <div class="border-b border-border px-6 py-4">
            <h2 :id="dialogTitleId" class="text-lg font-semibold text-foreground">{{ dialogTitle }}</h2>
          </div>

          <div class="flex-1 overflow-y-auto px-6 py-4">
            <div v-if="dialogStep === 'plan'" class="flex flex-col gap-3">
              <div v-if="planLoading" class="flex flex-col gap-2">
                <div v-for="n in 6" :key="n" class="h-4 animate-pulse rounded bg-muted" />
              </div>
              <template v-else>
                <div
                  v-if="plan?.blockers?.length"
                  class="rounded-md border border-destructive/40 bg-destructive/10 px-3 py-2 text-sm text-destructive"
                >
                  {{ _t('Resolve blockers before continuing') }}
                </div>
                <div
                  v-else
                  class="rounded-md border border-border bg-muted/40 px-3 py-2 text-sm text-foreground"
                >
                  {{ _t('Confirm the impact of this operation') }}
                </div>

                <div class="flex flex-col gap-2">
                  <div class="text-sm font-semibold text-foreground">{{ _t('Affected Modules') }}</div>
                  <ul class="m-0 flex list-none flex-col gap-1.5 p-0 text-xs text-foreground/80">
                    <li v-for="item in plan?.affectedModules || []" :key="item.moduleName" class="flex flex-wrap items-center gap-2">
                      <span class="font-semibold">{{ item.moduleName }}</span>
                      <span class="text-foreground/60">{{ item.currentVersion || '—' }}</span>
                      <span class="text-foreground/60">{{ item.targetVersion || '' }}</span>
                      <span class="text-primary">{{ item.reason || '' }}</span>
                    </li>
                  </ul>
                </div>

                <div v-if="plan?.risks?.length" class="flex flex-col gap-2">
                  <div class="text-sm font-semibold text-foreground">{{ _t('Risks') }}</div>
                  <ul class="m-0 flex list-none flex-col gap-1.5 p-0 text-xs">
                    <li v-for="risk in plan?.risks" :key="risk.code" class="flex flex-wrap items-center gap-2">
                      <span class="rounded bg-amber-500/15 px-1.5 py-0.5 text-[11px] font-medium text-amber-800 dark:text-amber-200">{{
                        risk.code
                      }}</span>
                      <span class="text-foreground/70">{{ risk.message || '' }}</span>
                    </li>
                  </ul>
                </div>

                <div v-if="plan?.blockers?.length" class="flex flex-col gap-2">
                  <div class="text-sm font-semibold text-foreground">{{ _t('Blockers') }}</div>
                  <ul class="m-0 flex list-none flex-col gap-1.5 p-0 text-xs">
                    <li v-for="blocker in plan?.blockers" :key="blocker.code" class="flex flex-wrap items-center gap-2">
                      <span class="rounded bg-destructive/15 px-1.5 py-0.5 text-[11px] font-medium text-destructive">{{
                        blocker.code
                      }}</span>
                      <span class="text-foreground/70">{{ blocker.message || '' }}</span>
                    </li>
                  </ul>
                </div>

                <label v-if="action === 'install'" class="flex cursor-pointer items-center gap-2 text-sm">
                  <input v-model="withDemo" type="checkbox" class="size-4 rounded border-border" />
                  {{ _t('Include demo data') }}
                </label>
              </template>
            </div>

            <div v-else class="flex flex-col gap-3">
              <div
                class="rounded-md border px-3 py-2 text-sm"
                :class="resultAlertBoxClass"
              >
                {{
                  dialogStep === 'progress'
                    ? _t('Operation in progress, please wait')
                    : resultTitle
                }}
              </div>
              <div class="flex flex-col gap-2">
                <div class="text-sm font-semibold text-foreground">{{ _t('Execution Status') }}</div>
                <div class="status-row flex flex-wrap items-center gap-2 text-xs text-foreground/80">
                  <span class="font-semibold text-foreground">{{ _t('Status:') }}</span>
                  <span
                    class="module-op-status-badge rounded px-1.5 py-0.5 text-[11px] font-medium"
                    :class="statusBadgeClass(opStatus?.status)"
                    >{{ opStatus?.status || '—' }}</span
                  >
                  <span class="font-semibold text-foreground">{{ _t('Result:') }}</span>
                  <span
                    class="module-op-status-badge rounded px-1.5 py-0.5 text-[11px] font-medium"
                    :class="statusBadgeClass(opStatus?.resultStatus)"
                    >{{ opStatus?.resultStatus || '—' }}</span
                  >
                </div>
                <div v-if="opStatus?.summary" class="status-row flex flex-wrap gap-2 text-xs">
                  <span class="font-semibold text-foreground">{{ _t('Summary:') }}</span>
                  <span class="value text-foreground/80">{{ formatSummary(opStatus?.summary) }}</span>
                </div>
                <div
                  v-if="opStatus?.failureKind && opStatus?.failureKind !== 'NONE'"
                  class="status-row flex flex-wrap gap-2 text-xs"
                >
                  <span class="font-semibold text-foreground">{{ _t('Failure Kind:') }}</span>
                  <span class="value text-foreground/80">{{ opStatus?.failureKind }}</span>
                </div>
                <div v-if="opStatus?.errorDomain || opStatus?.errorCode" class="status-row flex flex-wrap gap-2 text-xs">
                  <span class="font-semibold text-foreground">{{ _t('Error:') }}</span>
                  <span class="value text-foreground/80"
                    >{{ opStatus?.errorDomain || '—' }} / {{ opStatus?.errorCode || '—' }}</span
                  >
                </div>
                <div v-if="opStatus?.ReloadTriggered" class="status-row flex flex-wrap gap-2 text-xs">
                  <span class="font-semibold text-foreground">{{ _t('Reload:') }}</span>
                  <span class="value text-foreground/80">{{
                    opStatus?.ReloadFailed ? _t('Trigger Failed') : _t('Triggered')
                  }}</span>
                </div>
              </div>
              <div v-if="dialogStep === 'progress'" class="border-t border-border pt-3 text-xs text-foreground/60">
                {{ _t('Do not refresh; status updates automatically.') }}
              </div>
            </div>
          </div>

          <div class="flex justify-end gap-2 border-t border-border px-6 py-4">
            <ChoyButton type="button" variant="outline" :disabled="planLoading" @click="closeDialog">{{ _t('Cancel') }}</ChoyButton>
            <ChoyButton
              v-if="dialogStep === 'plan'"
              type="button"
              :disabled="!!plan?.blockers?.length || planLoading || executeLoading"
              @click="submitOperation"
            >
              {{ _t('Confirm') }}
            </ChoyButton>
            <ChoyButton v-else type="button" @click="closeDialog">{{ _t('Done') }}</ChoyButton>
          </div>
        </div>
      </div>
    </Teleport>
  </div>
</template>

<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref, useId, watch } from 'vue';
import { useRouter } from 'vue-router';
import { History, LayoutGrid, List, RefreshCw } from 'lucide-vue-next';
import type { WebModelStore } from '@/web/web/stores/modelStore';
import type MetaModule from '@/meta/service/models/module';
import type MetaModuleIndex from '@/meta/service/models/module_index';
import type { ClientModelProps } from '@/core/rpc/types';
import { defineAction } from '@/core/web/resource';
import { usePermission } from '@/auth/web/composables/usePermission';
import { restoreDialogFocus } from '@/auth/web/components/preferences/dialog_focus_restore';
import { trapDialogTabKey } from '@/auth/web/components/preferences/dialog_focus_trap';
import { resolvePageStore } from '@/web/web/composables/usePageContext';
import { createTranslate } from '@/web/web/i18n';
import {
  ChoyButton,
  ChoyKanbanView,
  ChoyMessage,
  ChoySearchView,
  ChoyVirtualField,
  type ChoyKanbanCard,
  type ChoyKanbanLane,
} from '@/web';
import { createKanbanController } from '@/web/web/controllers/kanbanController';
import { awaitFieldSelection } from '@/web/web/query/utils/registry/fieldReady';
import type { ChoySearchQuery } from '@/web/web/components/view/searchViewHelpers';
import type { Lane } from '@/web/web/query/types';
import {
  createModuleOpProgressSession,
  type ModuleOpStatusSnapshot,
} from '../composables/useModuleOpProgress';
import { createModuleKanbanOpProgressHooks } from '../composables/moduleKanbanOpProgress';
import {
  captureDialogFocusTarget,
  createLaneSyncGate,
  createPlanDialogSessionGate,
  formatModuleKanbanDate,
  formatModuleOpSummary,
  isModuleInstalled,
  manifestSummaryText,
  moduleStatusBadgeClass,
  resolveModuleKanbanCardId,
  resolveModuleKanbanCardKey,
  shouldRecoverStaleKanbanSearch,
} from './module_kanban_chrome';

defineOptions({ name: 'ModuleKanbanView' });

const { _t, _lt } = createTranslate('meta', { scope: 'web/views/ModuleKanbanView' });

const props = withDefaults(
  defineProps<{ store?: WebModelStore<MetaModuleIndex>; moduleStore: WebModelStore<MetaModule>; showHeader?: boolean }>(),
  { showHeader: true },
);
const store = resolvePageStore(props.store, 'ModuleKanbanView');
const { showHeader } = props;
const moduleStore = props.moduleStore;

const keywordFields = ['ModuleName', 'Version', 'OriginType', 'OriginRef'];

const router = useRouter();
const controller = createKanbanController(store as any);
const choyLanes = ref<ChoyKanbanLane[]>([]);
const laneSyncGate = createLaneSyncGate();
let searchSeq = 0;
let searchInFlight = 0;
let lastSearchQuery: ChoySearchQuery | null = null;
const searchPending = ref(false);

const dialogTitleId = useId();
const dialogRef = ref<HTMLElement | null>(null);
let dialogFocusRestore: HTMLElement | null = null;

const moduleInstallAction = defineAction('meta.action.module_install', {
  title: _lt('Install Module'),
  requires: [{ model: 'meta.MetaModule', method: 'RequestInstall' }],
});
const moduleUpgradeAction = defineAction('meta.action.module_upgrade', {
  title: _lt('Upgrade Module'),
  requires: [{ model: 'meta.MetaModule', method: 'RequestUpgrade' }],
});
const moduleUninstallAction = defineAction('meta.action.module_uninstall', {
  title: _lt('Uninstall Module'),
  requires: [{ model: 'meta.MetaModule', method: 'RequestUninstall' }],
});
const moduleSyncIndexAction = defineAction('meta.action.module_sync_index', {
  title: _lt('Sync Module Index'),
  requires: [{ model: 'meta.MetaModuleIndex', method: 'RequestSync' }],
});
const { canRoute, hasAction } = usePermission();

type ModuleAction = 'install' | 'uninstall' | 'upgrade';

type PlanOperationResp = {
  baseRevision: string;
  affectedModules: Array<{ moduleName: string; reason?: string; currentVersion?: string; targetVersion?: string }>;
  risks: Array<{ code: string; level: string; message?: string; params?: Record<string, any> }>;
  blockers: Array<{ code: string; level: string; message?: string; params?: Record<string, any> }>;
};

type OpStatusResp = ModuleOpStatusSnapshot;

const dialogVisible = ref(false);
const dialogStep = ref<'plan' | 'progress' | 'result'>('plan');
const planLoading = ref(false);
const executeLoading = ref(false);
const planDialogSession = createPlanDialogSessionGate();
const plan = ref<PlanOperationResp | null>(null);
const opStatus = ref<OpStatusResp | null>(null);
const action = ref<ModuleAction>('install');
const targetModule = ref<ClientModelProps<MetaModuleIndex> | null>(null);
const withDemo = ref(false);

const opProgress = createModuleOpProgressSession(
  createModuleKanbanOpProgressHooks({
    fetchStatus: async (jobId) => (await (moduleStore as any).GetOpStatus(jobId)) as OpStatusResp,
    isDialogOpen: () => dialogVisible.value,
    setOpStatus: (status) => {
      opStatus.value = status;
    },
    setDialogStep: (step) => {
      dialogStep.value = step;
    },
    warn: (message) => {
      ChoyMessage.warning(message);
    },
    error: (message) => {
      ChoyMessage.error(message);
    },
    messages: {
      jobStillRunning: () => _t('Job is still running in the background; refresh later'),
      serviceRestarting: () => _t('Service is restarting; status will retry automatically'),
      failedToGetStatus: () => _t('Failed to get status'),
    },
  }),
);

const dialogTitle = computed(() => {
  const actionLabel =
    action.value === 'install' ? _t('Install Module') : action.value === 'uninstall' ? _t('Uninstall Module') : _t('Upgrade Module');
  return `${actionLabel} · ${targetModule.value?.ModuleName || ''}`.trim();
});

const resultTitle = computed(() => {
  if (!opStatus.value) return _t('Completed');
  if (opStatus.value.resultStatus === 'FAILED') return _t('Operation Failed');
  return opStatus.value.ReloadFailed ? _t('Succeeded but reload failed') : _t('Operation Succeeded');
});

const resultAlertBoxClass = computed(() => {
  if (dialogStep.value === 'progress') {
    return 'border-border bg-muted/40 text-foreground';
  }
  if (!opStatus.value) return 'border-border bg-muted/40 text-foreground';
  if (opStatus.value.resultStatus === 'FAILED') {
    return 'border-destructive/40 bg-destructive/10 text-destructive';
  }
  if (opStatus.value.ReloadFailed) {
    return 'border-amber-500/40 bg-amber-500/10 text-amber-900 dark:text-amber-100';
  }
  return 'border-emerald-500/40 bg-emerald-500/10 text-emerald-900 dark:text-emerald-100';
});

function rowToCard(row: unknown, index: number, laneKey: string): ChoyKanbanCard {
  const payload = resolveRowPayload(row);
  const id = resolveModuleKanbanCardKey(payload, laneKey, index);
  return {
    id,
    title: String(payload.ModuleName ?? id),
    laneKey,
    payload,
  };
}

function resolveRowPayload(row: unknown): Record<string, unknown> {
  if (!row || typeof row !== 'object') return {};
  const bag = row as Record<string, unknown>;
  const payload = bag.payload;
  if (payload && typeof payload === 'object') return payload as Record<string, unknown>;
  return bag;
}

async function syncLanesFromController(): Promise<void> {
  if ((await laneSyncGate.enter()) === 'waited') return;
  try {
    do {
      laneSyncGate.beginPass();
      const laneList = controller.lanes.value;
      if (!laneList.length) {
        const rows =
          controller.vm.result?.kind === 'search' ? ((controller.vm.result.rows as any[]) || []) : [];
        choyLanes.value = [
          {
            key: 'all',
            label: _t('All'),
            cards: rows.map((row, index) => rowToCard(row, index, 'all')),
          },
        ];
        continue;
      }
      await Promise.all(
        laneList.map(l =>
          controller.preloadLane(l.key).catch(error => {
            // Keep the board usable, but leave a log trail for partial lane failures.
            console.error(`Module kanban lane preload failed: ${l.key}`, error);
            return undefined;
          }),
        ),
      );
      choyLanes.value = laneList.map(lane => ({
        key: lane.key,
        label: laneLabel(lane),
        remain: controller.getLaneRemain(lane),
        cards: (controller.laneRecords.value[lane.key] || []).map((row, index) => rowToCard(row, index, lane.key)),
      }));
    } while (laneSyncGate.shouldResync());
  } finally {
    laneSyncGate.leave();
  }
}

watch(
  () => controller.lanes.value,
  () => {
    void syncLanesFromController();
  },
  { deep: true },
);

onMounted(async () => {
  try {
    await (store as any).RequestSync({ IfStale: true });
  } catch {
    // sync unavailable — silently skip, page remains usable
  }
  try {
    await awaitFieldSelection(store, { requireNonEmpty: true });
    await controller.setKeywordFields(keywordFields);
    await controller.apply({ keywordFields });
    await syncLanesFromController();
  } catch (e) {
    ChoyMessage.error(_t('Failed to load kanban'));
    console.error('Module kanban load failed:', e);
  }
});

async function onSearch(query: ChoySearchQuery) {
  lastSearchQuery = query;
  const seq = ++searchSeq;
  searchInFlight++;
  searchPending.value = true;
  try {
    await controller.apply({
      keyword: query.keyword,
      appliedFilters: (query.appliedFilters || []) as any,
      appliedGroups: query.appliedGroups as any,
      keywordFields,
    });
  } catch (e) {
    if (seq === searchSeq) {
      ChoyMessage.error(_t('Failed to load kanban'));
      console.error('Module kanban search failed:', e);
    }
  } finally {
    searchInFlight--;
  }

  if (
    shouldRecoverStaleKanbanSearch({
      completedSeq: seq,
      latestSeq: searchSeq,
      inFlight: searchInFlight,
    }) &&
    lastSearchQuery
  ) {
    const recoverSeq = searchSeq;
    searchInFlight++;
    try {
      await controller.apply({
        keyword: lastSearchQuery.keyword,
        appliedFilters: (lastSearchQuery.appliedFilters || []) as any,
        appliedGroups: lastSearchQuery.appliedGroups as any,
        keywordFields,
      });
      if (recoverSeq === searchSeq) await syncLanesFromController();
    } catch (e) {
      if (recoverSeq === searchSeq) {
        ChoyMessage.error(_t('Failed to load kanban'));
        console.error('Module kanban search recover failed:', e);
      }
    } finally {
      searchInFlight--;
      if (searchInFlight === 0) searchPending.value = false;
    }
    return;
  }

  if (seq !== searchSeq) {
    if (searchInFlight === 0) searchPending.value = false;
    return;
  }
  try {
    await syncLanesFromController();
  } catch (e) {
    if (seq === searchSeq) {
      ChoyMessage.error(_t('Failed to load kanban'));
      console.error('Module kanban lane sync failed:', e);
    }
  } finally {
    if (seq === searchSeq) searchPending.value = false;
  }
}

function toKanban() {
  router.push('/meta/modules');
}

function toList() {
  router.push('/meta/modules/list');
}

function toHistory() {
  router.push('/meta/modules/history');
}

function payloadOf(card: ChoyKanbanCard): Record<string, unknown> {
  return (card.payload ?? {}) as Record<string, unknown>;
}

function payloadRecord(card: ChoyKanbanCard): ClientModelProps<MetaModuleIndex> {
  return payloadOf(card) as ClientModelProps<MetaModuleIndex>;
}

function recordField(card: ChoyKanbanCard, field: string): unknown {
  return payloadOf(card)[field];
}

function onCardClick(card: ChoyKanbanCard) {
  const id = resolveModuleKanbanCardId(payloadOf(card));
  if (id) router.push(`/meta/modules/${id}`);
}

function statusBadgeClass(status?: string, available?: boolean): string {
  return moduleStatusBadgeClass(status, available);
}

function statusLabel(status?: string, available?: boolean) {
  if (available === false) return _t('Unavailable');
  const val = String(status || '').toLowerCase();
  if (val === 'installed') return _t('Installed');
  if (val === 'uninstalled') return _t('Not Installed');
  if (val === 'disabled') return _t('Disabled');
  if (val === 'broken') return _t('Broken');
  if (val === 'succeeded') return _t('Succeeded');
  if (val === 'failed') return _t('Failed');
  if (val === 'dispatching') return _t('Dispatching');
  if (val === 'queued') return _t('Queued');
  return status || _t('Unknown');
}

function isInstalled(status?: string) {
  return isModuleInstalled(status);
}

function formatDate(dt?: unknown) {
  return formatModuleKanbanDate(dt);
}

function formatSummary(summary: unknown) {
  return formatModuleOpSummary(summary);
}

async function onActionClick(nextAction: ModuleAction, record: ClientModelProps<MetaModuleIndex>) {
  resetDialog();
  action.value = nextAction;
  targetModule.value = record;
  withDemo.value = false;
  dialogFocusRestore = captureDialogFocusTarget();
  dialogVisible.value = true;
  dialogStep.value = 'plan';
  const requestSeq = planDialogSession.begin();
  planLoading.value = true;
  await nextTick();
  dialogRef.value?.focus();
  try {
    const nextPlan = (await (moduleStore as any).PlanOperation({
      action: nextAction,
      moduleName: record.ModuleName,
      withDemo: nextAction === 'install' ? withDemo.value : false,
    })) as PlanOperationResp;
    if (!planDialogSession.isCurrent(requestSeq)) return;
    plan.value = nextPlan;
  } catch (error: any) {
    if (!planDialogSession.isCurrent(requestSeq)) return;
    ChoyMessage.error(error?.message || _t('Failed to load plan'));
    closeDialog();
  } finally {
    if (planDialogSession.isCurrent(requestSeq)) planLoading.value = false;
  }
}

async function submitOperation() {
  if (!targetModule.value) return;
  executeLoading.value = true;
  dialogStep.value = 'progress';
  try {
    let jobId = '';
    const moduleName = targetModule.value.ModuleName;
    if (action.value === 'install') jobId = await (moduleStore as any).RequestInstall({ ModuleName: moduleName, WithDemo: withDemo.value });
    else if (action.value === 'uninstall') jobId = await (moduleStore as any).RequestUninstall({ ModuleName: moduleName });
    else jobId = await (moduleStore as any).RequestUpgrade({ ModuleName: moduleName });
    executeLoading.value = false;
    await opProgress.watch(jobId);
  } catch (error: any) {
    executeLoading.value = false;
    dialogStep.value = 'result';
    opStatus.value = {
      status: 'failed',
      resultStatus: 'FAILED',
      errorDomain: 'CLIENT',
      errorCode: 'REQUEST_FAILED',
    } as OpStatusResp;
    ChoyMessage.error(error?.message || _t('Operation request failed'));
  }
}

function resetDialog() {
  plan.value = null;
  opStatus.value = null;
  dialogStep.value = 'plan';
  planLoading.value = false;
  executeLoading.value = false;
  opProgress.stop();
}

function onDialogClose() {
  opProgress.stop();
}

function closeDialog() {
  planDialogSession.invalidate();
  dialogVisible.value = false;
  planLoading.value = false;
  onDialogClose();
  const restore = dialogFocusRestore;
  dialogFocusRestore = null;
  restoreDialogFocus(restore);
}

function onDialogKeydown(event: KeyboardEvent) {
  if (event.key === 'Escape') {
    closeDialog();
    return;
  }
  const root = dialogRef.value;
  if (root) trapDialogTabKey(event, root as HTMLElement);
}

function manifestSummary(raw: unknown) {
  return manifestSummaryText(raw);
}

watch(dialogVisible, async (open) => {
  if (!open) return;
  await nextTick();
  dialogRef.value?.focus();
});

const syncLoading = ref(false);

async function onSyncIndex() {
  if (syncLoading.value) return;
  syncLoading.value = true;
  try {
    const jobId = await (store as any).RequestSync({ Force: true, IfStale: false });
    ChoyMessage.success(jobId ? _t('Sync job triggered: all:%s', String(jobId)) : _t('Sync job triggered'));
  } catch (error: any) {
    ChoyMessage.warning(_t('Sync failed: %s', String(error?.message || 'request failed')));
  } finally {
    syncLoading.value = false;
  }
}

function laneLabel(lane: Lane | ChoyKanbanLane): string {
  return String((lane as Lane).label ?? (lane as ChoyKanbanLane).label ?? (lane as Lane).key ?? '');
}

onBeforeUnmount(() => {
  planDialogSession.invalidate();
  opProgress.stop();
});

defineExpose({
  onActionClick,
  submitOperation,
  resetDialog,
  onDialogClose,
  dialogVisible,
  dialogStep,
  opStatus,
  action,
  targetModule,
});
</script>
