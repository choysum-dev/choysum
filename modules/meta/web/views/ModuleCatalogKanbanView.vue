<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <div
    class="module-catalog-kanban-view flex h-full min-h-0 min-w-0 flex-1 flex-col"
    data-testid="module-catalog-view"
  >
    <div class="sr-only" aria-hidden="true">
      <ChoyVirtualField :store="store" prop="ModuleName" />
      <ChoyVirtualField :store="store" prop="Installed" />
      <ChoyVirtualField :store="store" prop="InstalledVersion" />
      <ChoyVirtualField :store="store" prop="LocalVersion" />
      <ChoyVirtualField :store="store" prop="RegistryVersion" />
      <ChoyVirtualField :store="store" prop="UpdateAvailable" />
      <ChoyVirtualField :store="store" prop="Application" />
      <ChoyVirtualField :store="store" prop="Author" />
      <ChoyVirtualField :store="store" prop="License" />
      <ChoyVirtualField :store="store" prop="Summary" />
    </div>

    <ChoyKanbanView
      :store="store"
      :show-header="showHeader"
      :show-actions="true"
      readonly
      :flat-lane-label="_t('All')"
      :map-row-to-card="rowToCard"
      :resolve-move-record-id="resolveMoveRecordId"
      :on-load-error="onLoadError"
      :on-search-error="onSearchError"
      :on-load-more-error="onLoadMoreError"
      @card-click="onCardClick"
    >
      <template #search="{ onQueryUpdate }">
        <ChoySearchView
          class="min-w-0 w-full"
          :store="store"
          :default-groups="defaultGroups"
          @query-update="onQueryUpdate"
        />
      </template>

      <!-- After title + ⋮ in the system tray (no ActionBar separator). -->
      <template v-if="showHeader" #secondary-actions>
        <ChoyButton
          v-if="hasAction(moduleSyncAction)"
          type="button"
          size="default"
          variant="outline"
          data-testid="module-catalog-sync"
          :disabled="syncBusy"
          :aria-busy="syncBusy || undefined"
          @click="onSync"
        >
          <RefreshCw
            class="size-4"
            :class="{ 'animate-spin': syncBusy }"
            aria-hidden="true"
          />
          {{ _t('Update Module List') }}
        </ChoyButton>
        <ChoyButton
          v-if="hasAction(moduleLoadDemoAction) && hasPendingDemo"
          type="button"
          size="default"
          variant="outline"
          data-testid="module-catalog-load-demo"
          @click="onLoadDemo"
        >
          {{ _t('Load Demo Data') }}
        </ChoyButton>
      </template>

      <template v-if="showHeader" #header-right>
        <span
          v-if="lastSyncAtLabel"
          class="text-[11px] text-muted-foreground"
          data-testid="module-catalog-last-sync"
        >
          {{ _t('Last sync %s', lastSyncAtLabel) }}
        </span>
        <ChoyViewSwitcher
          :aria-label="_t('View switcher')"
          :active="activeView"
          :options="viewSwitcherOptions"
          @select="onViewSelect"
        />
      </template>

      <template #card="{ card }">
        <!--
          Intrinsic height (no h-full / min-h-0): flat board scrolls the grid;
          card body must not shrink-clip meta rows when the viewport is short.
        -->
        <div
          class="module-catalog-card flex flex-col gap-1.5 text-xs"
          data-testid="module-catalog-card"
        >
          <div class="flex flex-col gap-1.5">
            <div class="flex items-center justify-between gap-2">
              <span
                class="truncate text-sm font-medium text-foreground"
                data-testid="module-catalog-card-name"
              >{{ cardModuleName(card) }}</span>
              <ChoyBadge
                :variant="statusBadgeVariant(cardPayload(card))"
                class="shrink-0 px-1.5 py-0 text-[10px] font-normal"
                data-testid="module-catalog-card-status"
              >
                {{ cardStatusLabel(card) }}
              </ChoyBadge>
            </div>
            <div
              class="truncate text-muted-foreground"
              :title="cardVersionLine(card)"
            >{{ cardVersionLine(card) }}</div>
            <div
              v-if="cardSummary(card)"
              class="line-clamp-2 text-muted-foreground"
            >{{ cardSummary(card) }}</div>
            <div
              v-if="cardApplication(card)"
              class="truncate text-muted-foreground/80"
              data-testid="module-catalog-card-application"
            >{{ cardApplication(card) }}</div>
            <div
              v-if="cardAuthor(card) || cardLicense(card)"
              class="flex items-center justify-between gap-2 text-muted-foreground/80"
              data-testid="module-catalog-card-attribution"
            >
              <span
                v-if="cardAuthor(card)"
                class="min-w-0 truncate"
                :title="cardAuthor(card)"
                data-testid="module-catalog-card-author"
              >{{ cardAuthor(card) }}</span>
              <span
                v-if="cardLicense(card)"
                class="min-w-0 shrink-0 truncate text-right"
                :class="cardAuthor(card) ? 'max-w-[45%]' : 'ml-auto'"
                :title="cardLicense(card)"
                data-testid="module-catalog-card-license"
              >{{ cardLicense(card) }}</span>
            </div>
          </div>
          <div class="flex flex-wrap items-center justify-start gap-1 pt-2">
            <ChoyButton
              v-if="primaryActionFor(cardPayload(card))"
              type="button"
              size="xs"
              :variant="primaryActionFor(cardPayload(card)) === 'progress' ? 'outline' : 'default'"
              :data-testid="`module-catalog-card-primary-${primaryActionFor(cardPayload(card))}`"
              @click.stop="onPrimaryAction(cardPayload(card))"
            >
              {{ primaryActionLabel(primaryActionFor(cardPayload(card))) }}
            </ChoyButton>
            <ChoyButton
              v-if="canUninstall(cardPayload(card)) && hasAction(moduleUninstallAction)"
              type="button"
              size="xs"
              variant="outline-destructive"
              data-testid="module-catalog-card-uninstall"
              @click.stop="onUninstall(cardPayload(card))"
            >
              {{ _t('Uninstall') }}
            </ChoyButton>
          </div>
        </div>
      </template>

      <template #card-empty>
        <div class="text-xs opacity-60">{{ _t('No modules match this filter.') }}</div>
      </template>
    </ChoyKanbanView>

    <ModuleCatalogOpDialog
      v-model:open="drawerOpen"
      :operation-store="operationStore"
      :operation-id="drawerOperationId"
      :module-name="drawerModuleName"
      :action="drawerAction"
      :registry-newer-than-local="drawerRegistryNewerThanLocal"
      @applied="onOpApplied"
      @settled="onDrawerSettled"
    />
    <ModuleCatalogOpProgressStrip
      :busy="progressBusy"
      :title="progressTitle"
      :headline="progressHeadline"
      :value="progressValue"
    />
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, toRef } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import { RefreshCw } from 'lucide-vue-next';
import type { WebModelStore } from '@/web/web/stores/modelStore';
import type MetaModuleCatalog from '@/meta/service/models/module_catalog';
import type MetaModuleOperation from '@/meta/service/models/module_operation';
import { resolvePageStore } from '@/web/web/composables/usePageContext';
import { usePermission } from '@/auth/web/composables/usePermission';
import { createTranslate } from '@/web/web/i18n';
import { useModuleCatalogViewModel } from '../composables/useModuleCatalogViewModel';
import { moduleLoadDemoAction, moduleSyncAction, moduleUninstallAction } from './module_catalog_actions';
import ModuleCatalogOpDialog from './ModuleCatalogOpDialog.vue';
import ModuleCatalogOpProgressStrip from './ModuleCatalogOpProgressStrip.vue';
import {
  formatModuleCatalogVersionLine,
  moduleCatalogCardApplication,
  moduleCatalogCardAuthor,
  moduleCatalogCardLicense,
  moduleCatalogCardSummary,
} from './module_catalog_chrome';
import {
  ChoyBadge,
  ChoyButton,
  ChoyKanbanView,
  ChoyMessage,
  ChoySearchView,
  ChoyViewSwitcher,
  ChoyVirtualField,
  useChoyViewRouteSwitch,
  type ChoyKanbanCard,
  type ChoyViewSwitcherOption,
} from '@/web';
import type { GroupBySpec } from '@/web/web/query/types';

defineOptions({ name: 'ModuleCatalogKanbanView', inheritAttrs: false });

const { _t } = createTranslate('meta', { scope: 'web/views/ModuleCatalogKanbanView' });
const router = useRouter();
const route = useRoute();
const { hasAction } = usePermission();

const props = withDefaults(
  defineProps<{
    store?: WebModelStore<MetaModuleCatalog>;
    operationStore?: WebModelStore<MetaModuleOperation> | null;
    showHeader?: boolean;
    initialOperationId?: string;
  }>(),
  { showHeader: true, operationStore: null },
);

const store = resolvePageStore(props.store, 'ModuleCatalogKanbanView');
const showHeader = toRef(props, 'showHeader');
// Prefer the prop binding in the template (`operationStore`) so we pass the
// concrete Pinia store — wrapping it in computed() can leave a ComputedRef
// unwrapped and break session.store.Plan().

const { active: activeView, defaultGroups: routeGroups, selectView: onViewSelect } =
  useChoyViewRouteSwitch({ store });
const defaultGroups = routeGroups.value as GroupBySpec<MetaModuleCatalog>[];

const viewSwitcherOptions = computed<ChoyViewSwitcherOption[]>(() => [
  { value: 'kanban', label: _t('Kanban View') },
  { value: 'list', label: _t('List View') },
]);

const {
  drawerOpen,
  drawerOperationId,
  drawerModuleName,
  drawerAction,
  drawerRegistryNewerThanLocal,
  syncBusy,
  lastSyncAtLabel,
  onSync,
  onPrimaryAction,
  canUninstall,
  onUninstall,
  hasPendingDemo,
  onLoadDemo,
  onOpApplied,
  onDrawerSettled,
  openDrawer,
  closeDrawer,
  progressBusy,
  progressTitle,
  progressHeadline,
  progressValue,
  toShow,
  rowRecord,
  statusBadgeVariant,
  statusLabel,
  primaryActionFor,
  primaryActionLabel,
  bootFromRoute,
} = useModuleCatalogViewModel({
  store,
  operationStore: toRef(props, 'operationStore'),
  route,
  router,
  initialOperationId: () => String(props.initialOperationId || ''),
  msgs: {
    unknown: _t('Unknown'),
    install: _t('Install'),
    upgrade: _t('Upgrade'),
    progress: _t('View Progress'),
    installed: _t('Installed'),
    updateAvailable: _t('Update available'),
    uninstalled: _t('Not Installed'),
    unavailable: _t('Unavailable'),
    syncFailed: msg => _t('Sync failed: %s', msg),
    // Prefer the sync summary as the toast body; fall back when summary is empty.
    syncSettledOk: summary => (summary ? summary : _t('Module list updated')),
    syncSettledPartial: summary =>
      summary ? summary : _t('Module list updated with warnings'),
    syncSettledFailed: msg => _t('Sync failed: %s', msg),
    syncJobStillRunning: _t('Job is still running in the background; refresh later'),
    syncServiceRestarting: _t('Service is restarting; status will retry automatically'),
    syncFailedToGetStatus: _t('Failed to get status'),
    formatSyncSummaryOk: sync =>
      _t('Module catalog updated, %s modules', String(sync.catalogRows)),
    formatSyncSummaryPartial: sync =>
      _t(
        'Module catalog updated, %s modules (%s failed)',
        String(sync.catalogRows),
        String(sync.failed),
      ),
    opSettledOk: _t('Operation Succeeded'),
    opSettledFailed: msg => _t('Operation Failed: %s', msg),
    opJobStillRunning: _t('Job is still running in the background; refresh later'),
    opServiceRestarting: _t('Service is restarting; status will retry automatically'),
    opFailedToGetStatus: _t('Failed to get status'),
    opReloadingService: () => _t('Almost done, refreshing…'),
    opExecuting: () => _t('Working on it, please wait…'),
    opQueued: () => _t('Getting ready…'),
    opFinishing: () => _t('Wrapping up…'),
    opInProgress: () => _t('Please wait…'),
    opProgressTitle: (action, moduleName) => {
      const name = String(moduleName || '').trim();
      if (action === 'install') return name ? _t('Installing %s', name) : _t('Installing…');
      if (action === 'upgrade') return name ? _t('Upgrading %s', name) : _t('Upgrading…');
      if (action === 'uninstall') return name ? _t('Uninstalling %s', name) : _t('Uninstalling…');
      if (action === 'load_demo') return _t('Loading demo data…');
      if (action === 'progress') return name ? _t('Working on %s', name) : _t('Working…');
      return name ? _t('Working on %s', name) : _t('Working…');
    },
  },
});

function rowToCard(row: unknown, index: number, laneKey: string): ChoyKanbanCard {
  const payload = rowRecord(row);
  const id = String(payload.Id ?? payload.ModuleName ?? `${laneKey}-${index}`);
  return {
    id,
    title: String(payload.ModuleName || id),
    laneKey,
    payload,
  };
}

function resolveMoveRecordId(cards: ReadonlyArray<ChoyKanbanCard>, moveCardId: string): string {
  const hit = cards.find(c => String(c.id) === String(moveCardId));
  const payload = (hit?.payload || {}) as Record<string, unknown>;
  return String(payload.Id ?? moveCardId ?? '').trim();
}

function cardPayload(card: ChoyKanbanCard): Record<string, unknown> {
  const p = card.payload;
  return p && typeof p === 'object' ? (p as Record<string, unknown>) : {};
}

function cardModuleName(card: ChoyKanbanCard): string {
  return String(cardPayload(card).ModuleName || card.title || '').trim();
}

function cardSummary(card: ChoyKanbanCard): string {
  return moduleCatalogCardSummary(cardPayload(card));
}

function cardApplication(card: ChoyKanbanCard): string {
  return moduleCatalogCardApplication(cardPayload(card));
}

function cardAuthor(card: ChoyKanbanCard): string {
  return moduleCatalogCardAuthor(cardPayload(card));
}

function cardLicense(card: ChoyKanbanCard): string {
  return moduleCatalogCardLicense(cardPayload(card));
}

function cardVersionLine(card: ChoyKanbanCard): string {
  return formatModuleCatalogVersionLine(cardPayload(card));
}

function cardStatusLabel(card: ChoyKanbanCard): string {
  return statusLabel(cardPayload(card));
}

function onCardClick(card: ChoyKanbanCard) {
  toShow(cardPayload(card));
}

function onLoadError(e: unknown) {
  ChoyMessage.error(_t('Failed to load modules'));
  console.error('Module catalog kanban load failed:', e);
}

function onSearchError(e: unknown) {
  ChoyMessage.error(_t('Failed to load modules'));
  console.error('Module catalog kanban search failed:', e);
}

function onLoadMoreError(e: unknown) {
  ChoyMessage.error(_t('Failed to load more'));
  console.error('Module catalog kanban load-more failed:', e);
}

onMounted(() => {
  bootFromRoute();
});

defineExpose({
  openDrawer,
  closeDrawer,
});
</script>
