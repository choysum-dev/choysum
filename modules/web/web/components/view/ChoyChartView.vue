<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <div
    data-anchor="choy.chart-view"
    data-region="chart-view"
    :class="cn('choy-chart-view flex flex-col gap-3', props.class)"
  >
    <div
      v-if="showHeader"
      class="flex flex-wrap items-center justify-between gap-2 min-h-[var(--choy-control-height)]"
    >
      <div v-if="showActions" class="flex flex-wrap items-center gap-2">
        <ChoyActionTray
          class="choy-chart__system-actions"
          :aria-label="_t('System actions')"
        >
          <slot name="system-actions">
            <ChoyButton
              v-if="showCreate"
              size="sm"
              variant="outline"
              @click="emit('create')"
            >
              {{ resolvedCreateLabel }}
            </ChoyButton>
            <ChoyButton
              v-if="showRefresh"
              size="sm"
              variant="outline"
              :disabled="boardBusy"
              @click="onRefresh"
            >
              {{ resolvedRefreshLabel }}
            </ChoyButton>
          </slot>
        </ChoyActionTray>
        <ChoyActionTray
          class="choy-chart__user-actions"
          :aria-label="_t('User actions')"
        >
          <slot name="user-actions" />
        </ChoyActionTray>
      </div>
      <div class="min-w-0 flex-1">
        <slot name="search" :on-query-update="onSearch" />
      </div>
      <slot name="header-right" />
    </div>

    <div
      v-if="showChartControls"
      class="flex flex-wrap items-center gap-2 min-h-[var(--choy-control-height)]"
      data-region="chart-controls"
    >
      <slot
        name="metric-switcher"
        :metrics="resolvedMetrics"
        :current="localMetric"
        :change="selectMetric"
      >
        <label
          v-if="resolvedMetrics.length"
          class="flex items-center gap-2 text-xs text-muted-foreground"
        >
          {{ _t('Metric') }}
          <select
            class="choy-input h-control"
            :value="localMetric"
            :aria-label="_t('Metric selection')"
            @change="selectMetric(($event.target as HTMLSelectElement).value)"
          >
            <option
              v-for="m in resolvedMetrics"
              :key="m.alias"
              :value="m.alias"
            >
              {{ m.label }}
            </option>
          </select>
        </label>
      </slot>

      <slot
        name="chart-type-switcher"
        :types="availableTypes"
        :current="localChartType"
        :change="selectChartType"
      >
        <div
          v-if="availableTypes.length"
          class="inline-flex overflow-hidden rounded-md border border-border"
          role="group"
          :aria-label="_t('Chart type')"
        >
          <ChoyButton
            v-for="t in availableTypes"
            :key="t"
            size="sm"
            :variant="localChartType === t ? 'default' : 'ghost'"
            class="rounded-none"
            :aria-pressed="localChartType === t"
            @click="selectChartType(t)"
          >
            {{ t }}
          </ChoyButton>
        </div>
      </slot>

      <slot
        name="stacked-switcher"
        :stacked="localStacked"
        :disabled="stackedDisabled"
        :toggle="toggleStacked"
      >
        <ChoyButton
          size="sm"
          :variant="localStacked && !stackedDisabled ? 'default' : 'outline'"
          :disabled="stackedDisabled"
          :aria-pressed="localStacked"
          @click="toggleStacked"
        >
          {{ _t('Stack') }}
        </ChoyButton>
      </slot>

      <slot
        name="sort-switcher"
        :current="localSort"
        :disabled="sortDisabled"
        :change="selectSort"
      >
        <div
          class="inline-flex overflow-hidden rounded-md border border-border"
          role="group"
          :aria-label="_t('Sort')"
        >
          <ChoyButton
            size="sm"
            class="rounded-none"
            :variant="localSort === 'none' ? 'default' : 'ghost'"
            :disabled="sortDisabled"
            :aria-pressed="localSort === 'none'"
            :aria-label="_t('No sort')"
            :title="_t('No sort')"
            @click="selectSort('none')"
          >
            ∅
          </ChoyButton>
          <ChoyButton
            size="sm"
            class="rounded-none"
            :variant="localSort === 'asc' ? 'default' : 'ghost'"
            :disabled="sortDisabled"
            :aria-pressed="localSort === 'asc'"
            :aria-label="_t('Sort ascending')"
            :title="_t('Sort ascending')"
            @click="selectSort('asc')"
          >
            ↑
          </ChoyButton>
          <ChoyButton
            size="sm"
            class="rounded-none"
            :variant="localSort === 'desc' ? 'default' : 'ghost'"
            :disabled="sortDisabled"
            :aria-pressed="localSort === 'desc'"
            :aria-label="_t('Sort descending')"
            :title="_t('Sort descending')"
            @click="selectSort('desc')"
          >
            ↓
          </ChoyButton>
        </div>
      </slot>
    </div>

    <div class="relative min-h-[280px] w-full" data-region="chart-body">
      <ChartContainer
        v-if="spec"
        :config="spec.config"
        class="min-h-[280px] w-full"
      >
        <VisXYContainer
          v-if="spec.kind === 'bar' || spec.kind === 'line'"
          :data="xyRows"
          :height="280"
          class="h-[280px] w-full"
        >
          <VisGroupedBar
            v-if="spec.kind === 'bar' && !spec.stacked"
            :x="xAccessor"
            :y="xyYAccessors"
            :color="xyColors"
            :rounded-corners="2"
            :events="groupedBarEvents"
          />
          <VisStackedBar
            v-else-if="spec.kind === 'bar' && spec.stacked"
            :x="xAccessor"
            :y="xyYAccessors"
            :color="xyColors"
            :rounded-corners="2"
            :events="stackedBarEvents"
          />
          <template v-else-if="spec.kind === 'line'">
            <VisArea
              v-if="spec.stacked"
              :x="xAccessor"
              :y="xyYAccessors"
              :color="xyColors"
              :opacity="0.25"
            />
            <VisLine
              :x="xAccessor"
              :y="xyYAccessors"
              :color="xyColors"
              :events="lineEvents"
            />
          </template>
          <VisAxis
            type="x"
            :x="xAccessor"
            :tick-format="xTickFormat"
            :tick-line="false"
            :domain-line="false"
            :grid-line="false"
          />
          <VisAxis
            type="y"
            :label="spec.metricLabel"
            :tick-line="false"
            :domain-line="false"
            :grid-line="true"
            :tick-format="spec.percent ? ((v: number) => `${v}%`) : undefined"
          />
        </VisXYContainer>

        <VisSingleContainer
          v-else-if="spec.kind === 'pie'"
          :data="pieRows"
          :height="280"
          class="h-[280px] w-full"
        >
          <VisDonut
            :value="(d: { value: number }) => d.value"
            :color="(d: { color: string }) => d.color"
            :arc-width="40"
            :events="donutEvents"
          />
        </VisSingleContainer>

        <ChartLegendContent />
      </ChartContainer>

      <ChoyEmpty
        v-else
        class="min-h-[280px] w-full border-none"
        :description="resolvedEmptyLabel"
      />

      <div
        v-if="boardBusy && !errorText"
        class="absolute inset-0 flex flex-col items-center justify-center gap-2 bg-background/60"
        role="status"
        :aria-label="_t('Loading...')"
      >
        <ChoySpinner :label="_t('Loading...')" />
      </div>
      <div
        v-else-if="errorText"
        class="absolute inset-0 flex flex-col items-center justify-center gap-2 bg-background/70 text-sm text-destructive"
        role="alert"
      >
        <span>{{ errorText }}</span>
        <ChoyButton size="sm" variant="outline" @click="onRefresh">
          {{ _t('Retry') }}
        </ChoyButton>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue';
import {
  VisArea,
  VisAxis,
  VisDonut,
  VisGroupedBar,
  VisLine,
  VisSingleContainer,
  VisStackedBar,
  VisXYContainer
} from '@unovis/vue';
import { Donut, GroupedBar, Line, StackedBar } from '@unovis/ts';
import type { ClassValue } from '../../lib/utils';
import { cn } from '../../lib/utils';
import type { BaseModel } from '@/core/rpc';
import type { GroupBySpec, QueryCondition } from '@/core/service/api/query';
import type { WebModelStore } from '@/web/web/stores/modelStore';
import type { OrderByState } from '@/web/web/query/state';
import type { ChoySearchQuery } from './searchViewHelpers';
import ChoyButton from '../layout/ChoyButton.vue';
import ChoyActionTray from '@/web/web/components/layout/ChoyActionTray.vue';
import ChoyEmpty from '../layout/ChoyEmpty.vue';
import ChoySpinner from '../layout/ChoySpinner.vue';
import { createTranslate } from '@/web/web/i18n';
import {
  ChartContainer,
  ChartLegendContent
} from '../vendor/ui/chart';
import { createChartController } from '@/web/web/controllers/chartController';
import { resolvePageStore } from '@/web/web/composables/usePageContext';
import { awaitFieldSelection } from '@/web/web/query/utils/registry/fieldReady';
import { exportMetrics } from '@/web/web/query/utils/registry/metric';
import {
  availableChartTypes,
  CHOY_CHART_DEFAULT_PALETTE,
  resolveChartAdapter,
  type ChoyChartKind,
  type ChoyChartSeries,
  type ChoyChartSort,
  type ChoyChartSpec
} from './chart/chartTypeAdapter';
import {
  chartSpecToPieRows,
  chartSpecToXyRows,
  normalizeSeriesToPercent,
  resolveGroupedXyClickTarget,
  resolveClickClientX,
  resolveLineClickCategory,
  resolveStackedXyClickTarget,
  sortChartCategories,
  type ChoyChartItemClickPayload,
  type ChoyChartMetricOption
} from './chartViewHelpers';
import { groupRowsToChartSeries } from './chartStoreHelpers';

/**
 * Store-bound chart view. Requires :store or a page-provided store.
 * Owns createChartController and maps grouped snapshots into Unovis series.
 */
const { _t } = createTranslate('web', { scope: 'web/components/view/ChartView' });

const props = withDefaults(
  defineProps<{
    class?: ClassValue;
    store?: WebModelStore<any>;
    defaultGroups?: GroupBySpec<any> | GroupBySpec<any>[];
    keywordFields?: string[];
    forcedCondition?: QueryCondition<any> | QueryCondition<any>[];
    orderBy?: OrderByState[];
    metrics?: ChoyChartMetricOption[];
    metricAlias?: string;
    chartTypes?: ChoyChartKind[];
    chartType?: ChoyChartKind;
    stacked?: boolean;
    stackMode?: 'absolute' | 'percent';
    sort?: ChoyChartSort;
    groupDepth?: number;
    palette?: string[];
    emptyLabel?: string;
    showHeader?: boolean;
    showActions?: boolean;
    showChartControls?: boolean;
    createLabel?: string;
    refreshLabel?: string;
    showCreate?: boolean;
    showRefresh?: boolean;
    autoBootstrap?: boolean;
    beforeBootstrap?: () => Promise<void>;
    onLoadError?: (error: unknown) => void;
    onSearchError?: (error: unknown) => void;
  }>(),
  {
    metrics: () => [],
    metricAlias: '',
    chartTypes: () => ['bar', 'line', 'pie'],
    chartType: 'bar',
    stacked: true,
    stackMode: 'absolute',
    sort: 'none',
    groupDepth: 0,
    showHeader: true,
    showActions: true,
    showChartControls: true,
    showCreate: false,
    showRefresh: true,
    autoBootstrap: true,
  },
);

const resolvedCreateLabel = computed(() => props.createLabel || _t('New'));
const resolvedRefreshLabel = computed(() => props.refreshLabel || _t('Refresh'));
const resolvedEmptyLabel = computed(
  () => props.emptyLabel || _t('No data or grouping not configured'),
);

const emit = defineEmits<{
  'metric-change': [alias: string];
  'chart-type-change': [type: ChoyChartKind];
  'stacked-change': [stacked: boolean];
  'sort-change': [sort: ChoyChartSort];
  'chart-item-click': [payload: ChoyChartItemClickPayload];
  refresh: [];
  create: [];
  'query-update': [query: ChoySearchQuery];
}>();

const store = resolvePageStore(props.store, 'ChoyChartView');
const controller = createChartController(store as WebModelStore<BaseModel>);

const categories = ref<string[]>([]);
const seriesMatrix = ref<ChoyChartSeries[]>([]);
const registryMetrics = ref<ChoyChartMetricOption[]>([]);

const localStacked = ref(props.stacked);
const localSort = ref<ChoyChartSort>(props.sort);
const localChartType = ref<ChoyChartKind>(props.chartType);
const localMetric = ref(props.metricAlias || 'count');

const boardBusy = computed(() => !!controller.vm.loading);
const errorText = computed(() => {
  const err = controller.vm.error;
  if (err == null) return null;
  if (err instanceof Error) return err.message || String(err);
  return String(err);
});

const resolvedMetrics = computed(() => {
  if (props.metrics.length) return props.metrics;
  return registryMetrics.value;
});

function normalizeDefaultGroups(): Array<GroupBySpec<any>> | undefined {
  const g = props.defaultGroups;
  if (!g) return undefined;
  return Array.isArray(g) ? g : [g];
}

function initMetricOptions(): void {
  const metas = exportMetrics(store.storeId) || [];
  const arr: ChoyChartMetricOption[] = metas.map((m) => ({
    alias: m.alias || `${m.field}_${m.agg}`,
    label: m.alias || `${m.field}:${m.agg}`,
  }));
  if (!arr.find((m) => m.alias === 'count')) {
    arr.unshift({ alias: 'count', label: 'Count' });
  }
  registryMetrics.value = arr;
  const preferred = props.metricAlias;
  if (preferred && arr.some((m) => m.alias === preferred)) {
    localMetric.value = preferred;
  } else if (!arr.some((m) => m.alias === localMetric.value)) {
    localMetric.value = arr[0]?.alias || 'count';
  }
}

function rebuildFromSnapshot(): void {
  const snap = controller.vm.result;
  const rows = snap?.kind === 'group' ? ((snap.rows as any[]) || []) : [];
  const metricLabel =
    resolvedMetrics.value.find((m) => m.alias === localMetric.value)?.label ||
    localMetric.value ||
    'Value';
  const mapped = groupRowsToChartSeries(rows, {
    metricAlias: localMetric.value,
    metricLabel,
    groupDepth: props.groupDepth,
  });
  categories.value = mapped.categories;
  seriesMatrix.value = mapped.seriesMatrix;
}

async function applyQuery(overrides?: {
  keyword?: string;
  appliedFilters?: any[];
  appliedGroups?: Array<GroupBySpec<any>>;
}): Promise<void> {
  await controller.apply({
    appliedGroups: overrides?.appliedGroups,
    appliedFilters: overrides?.appliedFilters,
    defaultGroups: normalizeDefaultGroups(),
    keyword: overrides?.keyword ?? (store.state as any)?.queryState?.keyword,
    keywordFields: props.keywordFields,
    forcedCondition: props.forcedCondition as any,
    orderBy: props.orderBy,
  });
  rebuildFromSnapshot();
}

async function bootstrap(): Promise<void> {
  if (props.beforeBootstrap) await props.beforeBootstrap();
  await awaitFieldSelection(store, { requireNonEmpty: false });
  initMetricOptions();
  try {
    await applyQuery();
  } catch (error) {
    props.onLoadError?.(error);
  }
}

async function onRefresh(): Promise<void> {
  emit('refresh');
  try {
    await applyQuery({
      appliedGroups: (store.state as any)?.queryState?.appliedGroups as any,
      keyword: (store.state as any)?.queryState?.keyword,
    });
  } catch (error) {
    props.onLoadError?.(error);
  }
}

function onSearch(query: ChoySearchQuery): void {
  emit('query-update', query);
  void applyQuery({
    keyword: query.keyword,
    appliedFilters: query.appliedFilters as any,
    appliedGroups: query.appliedGroups as any,
  }).catch((error) => {
    props.onSearchError?.(error);
  });
}

watch(
  () => props.stacked,
  (v) => {
    localStacked.value = v;
  },
);
watch(
  () => props.sort,
  (v) => {
    localSort.value = v;
  },
);
watch(
  () => props.chartType,
  (v) => {
    const next = availableTypes.value.includes(v) ? v : availableTypes.value[0];
    if (!next) {
      localChartType.value = v;
      return;
    }
    if (next !== localChartType.value) {
      localChartType.value = next;
      if (next !== v) {
        emit('chart-type-change', next);
      }
    }
  },
);
watch(
  () => props.metricAlias,
  (v) => {
    if (!v) return;
    const known = resolvedMetrics.value.some((m) => m.alias === v);
    const next = known ? v : resolvedMetrics.value[0]?.alias ?? v;
    localMetric.value = next;
    if (next !== v) {
      emit('metric-change', next);
    }
  },
);
watch(
  resolvedMetrics,
  (metrics) => {
    const known = metrics.some((m) => m.alias === localMetric.value);
    if (known) return;
    const next = metrics[0]?.alias;
    if (!next || next === localMetric.value) return;
    localMetric.value = next;
    emit('metric-change', next);
  },
  { deep: true },
);
watch(localMetric, () => {
  rebuildFromSnapshot();
});
watch(
  () => controller.vm.result,
  () => {
    rebuildFromSnapshot();
  },
);

const supportCtx = computed(() => ({
  groupDepth: Math.max(1, props.groupDepth + 1),
  stacked: localStacked.value,
  seriesCount: seriesMatrix.value.length,
  metricAlias: localMetric.value || resolvedMetrics.value[0]?.alias || 'count',
}));

const availableTypes = computed(() => {
  const supported = new Set(availableChartTypes(supportCtx.value));
  return [...new Set(props.chartTypes)].filter((t) => supported.has(t));
});

watch(
  availableTypes,
  (types) => {
    if (!types.includes(localChartType.value) && types.length) {
      localChartType.value = types[0]!;
      emit('chart-type-change', localChartType.value);
    }
  },
  { immediate: true },
);

const metricLabel = computed(() => {
  const hit = resolvedMetrics.value.find((m) => m.alias === localMetric.value);
  return hit?.label || localMetric.value || 'Value';
});

const stackedDisabled = computed(
  () => localChartType.value === 'pie' || seriesMatrix.value.length <= 1,
);

const sortDisabled = computed(() => localChartType.value === 'pie');

const prepared = computed(() => {
  let cats = categories.value.slice();
  let matrix = seriesMatrix.value.map((s) => ({
    name: s.name,
    data: (s.data || []).slice(),
  }));
  let categoryOrder = cats.map((_, idx) => idx);
  const percent =
    localStacked.value &&
    props.stackMode === 'percent' &&
    localChartType.value !== 'pie' &&
    matrix.length > 1;
  if (!sortDisabled.value && localSort.value !== 'none') {
    const sorted = sortChartCategories(cats, matrix, localSort.value);
    cats = sorted.categories;
    matrix = sorted.seriesMatrix;
    categoryOrder = sorted.order;
  }
  if (percent) {
    matrix = normalizeSeriesToPercent(cats, matrix);
  }
  return { categories: cats, seriesMatrix: matrix, percent, categoryOrder };
});

const spec = computed<ChoyChartSpec | null>(() => {
  if (!availableTypes.value.includes(localChartType.value)) return null;
  const adapter = resolveChartAdapter(localChartType.value);
  if (!adapter) return null;
  if (!adapter.supports(supportCtx.value)) return null;
  const { categories: cats, seriesMatrix: matrix, percent } = prepared.value;
  if (!cats.length || !matrix.length) return null;
  const built = adapter.build({
    categories: cats,
    seriesMatrix: matrix,
    metricLabel: metricLabel.value,
    stacked: localStacked.value,
    palette: props.palette,
    percent,
  });
  if (built.kind === 'pie' && !built.slices?.length) return null;
  return built;
});

const xyRows = computed(() => (spec.value && spec.value.kind !== 'pie' ? chartSpecToXyRows(spec.value) : []));
const pieRows = computed(() => (spec.value?.kind === 'pie' ? chartSpecToPieRows(spec.value) : []));

const xyYAccessors = computed(() => {
  if (!spec.value) return [];
  return spec.value.series.map((s) => (d: Record<string, string | number>) => Number(d[s.key]) || 0);
});

const xyColors = computed(() => {
  if (!spec.value) return [];
  return spec.value.series.map(
    (s) => spec.value!.config[s.key]?.color || CHOY_CHART_DEFAULT_PALETTE[0],
  );
});

type XyRow = Record<string, string | number>;

function xAccessor(d: XyRow): number {
  return Number(d.index) || 0;
}

function xTickFormat(v: number): string {
  const cats = spec.value?.categories ?? [];
  const raw = Number(v);
  if (!cats.length || !Number.isFinite(raw)) return '';
  const idx = Math.round(raw);
  if (Math.abs(raw - idx) > 1e-6 || idx < 0 || idx >= cats.length) return '';
  return cats[idx]!;
}

function selectMetric(alias: string): void {
  if (alias === localMetric.value) return;
  if (resolvedMetrics.value.length > 0 && !resolvedMetrics.value.some((m) => m.alias === alias)) return;
  localMetric.value = alias;
  emit('metric-change', alias);
}

function selectChartType(type: ChoyChartKind): void {
  if (!availableTypes.value.includes(type) || type === localChartType.value) return;
  localChartType.value = type;
  emit('chart-type-change', type);
}

function toggleStacked(): void {
  if (stackedDisabled.value) return;
  localStacked.value = !localStacked.value;
  emit('stacked-change', localStacked.value);
}

function selectSort(next: ChoyChartSort): void {
  if (sortDisabled.value || next === localSort.value) return;
  localSort.value = next;
  emit('sort-change', next);
}

type UnovisClickEvent = MouseEvent | PointerEvent | TouchEvent | WheelEvent;

function emitXyClick(categoryIdx: number, seriesIdx: number | undefined): void {
  if (!spec.value) return;
  const idx = Number.isFinite(categoryIdx) ? Math.round(categoryIdx) : -1;
  const category = spec.value.categories[idx];
  if (idx < 0 || category == null) return;
  const resolvedSeriesIdx =
    seriesIdx != null &&
    Number.isFinite(seriesIdx) &&
    seriesIdx >= 0 &&
    seriesIdx < spec.value.series.length
      ? Math.round(seriesIdx)
      : undefined;
  const series =
    resolvedSeriesIdx == null ? undefined : spec.value.series[resolvedSeriesIdx];
  const originIdx = prepared.value.categoryOrder[idx] ?? idx;
  emit('chart-item-click', {
    chartType: spec.value.kind,
    category,
    seriesName: series?.name,
    value: series?.values[idx],
    categoryIndex: originIdx,
    seriesIndex: resolvedSeriesIdx,
    path: [category],
  });
}

function onStackedBarClick(
  d: (XyRow & { stackIndex?: number }) | undefined,
  _event: UnovisClickEvent,
  categoryIdx?: number,
): void {
  if (!spec.value) return;
  const { categoryIdx: idx, seriesIdx } = resolveStackedXyClickTarget(
    d,
    categoryIdx,
    d?.stackIndex,
  );
  if (idx == null) return;
  emitXyClick(idx, seriesIdx);
}

function onGroupedBarClick(
  d: XyRow,
  _event: UnovisClickEvent,
  flatIndex?: number,
): void {
  if (!spec.value) return;
  const { categoryIdx, seriesIdx } = resolveGroupedXyClickTarget(
    d,
    flatIndex,
    spec.value.series.length,
  );
  if (categoryIdx == null) return;
  emitXyClick(categoryIdx, seriesIdx);
}

function onLineClick(
  _data: unknown,
  event: UnovisClickEvent,
  seriesIdx?: number,
): void {
  if (!spec.value) return;
  const rows = xyRows.value;
  if (!rows.length) return;
  const ev = event as MouseEvent & TouchEvent;
  const target = (ev.currentTarget ?? ev.target) as Element | null;
  const svg =
    target instanceof SVGSVGElement
      ? target
      : target instanceof Element
        ? target.closest('svg')
        : null;
  const clientX = resolveClickClientX(ev);
  if (!svg || clientX == null) return;
  const rect = (target instanceof Element ? target : svg).getBoundingClientRect();
  if (rect.width <= 0) return;
  const rel = (clientX - rect.left) / rect.width;
  const categoryIdx = resolveLineClickCategory(rel, rows.length);
  const nSeries = spec.value.series.length;
  let si =
    typeof seriesIdx === 'number' && Number.isFinite(seriesIdx)
      ? Math.round(seriesIdx)
      : undefined;
  if (nSeries === 1) si = 0;
  if (si != null && (si < 0 || si >= nSeries)) si = undefined;
  emitXyClick(categoryIdx, si);
}

function onPieSegmentClick(d: {
  data?: { key?: string; name?: string };
  index?: number;
  key?: string;
  name?: string;
}): void {
  const current = spec.value;
  const slices = current?.slices;
  if (!current || !slices) return;
  const clickedKey = d?.data?.key ?? d?.key;
  const clickedName = d?.data?.name ?? d?.name;
  let idx =
    clickedKey != null ? slices.findIndex((sl) => sl.key === clickedKey) : -1;
  if (idx < 0 && clickedKey == null && clickedName != null) {
    idx = slices.findIndex((sl) => sl.name === clickedName);
  }
  if (idx < 0) {
    idx =
      typeof d?.index === 'number' && Number.isFinite(d.index)
        ? Math.round(d.index)
        : -1;
  }
  if (idx < 0) return;
  const slice = slices[idx];
  if (!slice) return;
  const originIdx =
    typeof slice.categoryIndex === 'number' && Number.isFinite(slice.categoryIndex)
      ? Math.round(slice.categoryIndex)
      : idx;
  const multiSeries = current.series.length > 1;
  emit('chart-item-click', {
    chartType: 'pie',
    category: multiSeries ? undefined : slice.name,
    seriesName: multiSeries ? slice.name : current.series[0]?.name,
    value: slice.value,
    categoryIndex: multiSeries ? undefined : originIdx,
    seriesIndex: multiSeries ? originIdx : 0,
    path: [slice.name],
  });
}

const groupedBarEvents = computed(() => ({
  [GroupedBar.selectors.bar]: { click: onGroupedBarClick },
}));
const stackedBarEvents = computed(() => ({
  [StackedBar.selectors.bar]: { click: onStackedBarClick },
}));
const lineEvents = computed(() => ({
  [Line.selectors.line]: { click: onLineClick },
}));
const donutEvents = computed(() => ({
  [Donut.selectors.segment]: { click: onPieSegmentClick },
}));

onMounted(() => {
  if (!props.autoBootstrap) {
    initMetricOptions();
    return;
  }
  void bootstrap().catch((error) => props.onLoadError?.(error));
});

defineExpose({
  bootstrap,
  applyQuery,
  rebuildFromSnapshot,
  boardBusy,
  controller,
});
</script>
