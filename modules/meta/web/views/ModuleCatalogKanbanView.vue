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
      <ChoyVirtualField :store="store" prop="Application" />
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
          :disabled="syncLoading"
          :aria-busy="syncLoading || undefined"
          @click="onSync"
        >
          <RefreshCw
            class="size-4"
            :class="{ 'animate-spin': syncLoading }"
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
        <div
          class="module-catalog-card flex h-full min-h-0 flex-1 flex-col gap-1.5 text-xs"
          data-testid="module-catalog-card"
        >
          <div class="flex min-h-0 flex-1 flex-col gap-1.5">
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
            >{{ cardApplication(card) }}</div>
          </div>
          <!-- Pin actions to the card footer so missing Application/Summary does not shift them. -->
          <div class="mt-auto flex flex-wrap items-center justify-start gap-1 pt-2">
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

    <ModuleCatalogOpDrawer
      v-model:open="drawerOpen"
      :operation-store="operationStore"
      :operation-id="drawerOperationId"
      :module-name="drawerModuleName"
      :action="drawerAction"
      :registry-newer-than-local="drawerRegistryNewerThanLocal"
      @settled="onDrawerSettled"
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
import ModuleCatalogOpDrawer from './ModuleCatalogOpDrawer.vue';
import {
  formatModuleCatalogVersionLine,
  moduleCatalogCardApplication,
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
  syncLoading,
  lastSyncAtLabel,
  onSync,
  onPrimaryAction,
  canUninstall,
  onUninstall,
  hasPendingDemo,
  onLoadDemo,
  refreshCatalog,
  openDrawer,
  closeDrawer,
  toShow,
  rowRecord,
  statusBadgeVariant,
  statusLabel,
  primaryActionFor,
  primaryActionLabel,
  bootFromRoute,
} = useModuleCatalogViewModel({
  store,
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
    syncTriggered: id => _t('Sync job triggered: %s', id),
    syncFailed: msg => _t('Sync failed: %s', msg),
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

function cardVersionLine(card: ChoyKanbanCard): string {
  return formatModuleCatalogVersionLine(cardPayload(card));
}

function cardStatusLabel(card: ChoyKanbanCard): string {
  return statusLabel(cardPayload(card));
}

function onCardClick(card: ChoyKanbanCard) {
  toShow(cardPayload(card));
}

function onDrawerSettled() {
  void refreshCatalog();
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
