<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<script setup lang="ts">
import { computed, ref, watch } from 'vue';
import {
  VisArea,
  VisAxis,
  VisDonut,
  VisGroupedBar,
  VisLine,
  VisSingleContainer,
  VisStackedBar,
  VisXYContainer,
} from '@unovis/vue';
import { Donut, GroupedBar, Line, StackedBar } from '@unovis/ts';
import type { ClassValue } from '../../lib/utils';
import { cn } from '../../lib/utils';
import ChoyButton from '../layout/ChoyButton.vue';
import {
  ChartContainer,
  ChartLegendContent,
} from '../vendor/ui/chart';
import {
  availableChartTypes,
  CHOY_CHART_DEFAULT_PALETTE,
  resolveChartAdapter,
  type ChoyChartKind,
  type ChoyChartSeries,
  type ChoyChartSort,
  type ChoyChartSpec,
} from './chart/chartTypeAdapter';
import {
  chartSpecToPieRows,
  chartSpecToXyRows,
  normalizeSeriesToPercent,
  resolveGroupedXyClickTarget,
  resolveLineClickCategory,
  resolveStackedXyClickTarget,
  sortChartCategories,
  type ChoyChartItemClickPayload,
  type ChoyChartMetricOption,
} from './chartViewHelpers';

/**
 * Isolation ChartView: host-owned categories/series; Unovis render (no ECharts).
 */
const props = withDefaults(
  defineProps<{
    class?: ClassValue;
    categories?: string[];
    seriesMatrix?: ChoyChartSeries[];
    metrics?: ChoyChartMetricOption[];
    metricAlias?: string;
    chartTypes?: ChoyChartKind[];
    chartType?: ChoyChartKind;
    stacked?: boolean;
    stackMode?: 'absolute' | 'percent';
    sort?: ChoyChartSort;
    groupDepth?: number;
    palette?: string[];
    loading?: boolean;
    error?: string | null;
    emptyLabel?: string;
    showHeader?: boolean;
    showActions?: boolean;
    showChartControls?: boolean;
    createLabel?: string;
    refreshLabel?: string;
    showCreate?: boolean;
    showRefresh?: boolean;
  }>(),
  {
    categories: () => [],
    seriesMatrix: () => [],
    metrics: () => [],
    metricAlias: '',
    chartTypes: () => ['bar', 'line', 'pie'],
    chartType: 'bar',
    stacked: true,
    stackMode: 'absolute',
    sort: 'none',
    groupDepth: 1,
    loading: false,
    error: null,
    emptyLabel: 'No data or grouping not configured',
    showHeader: true,
    showActions: true,
    showChartControls: true,
    createLabel: 'New',
    refreshLabel: 'Refresh',
    showCreate: false,
    showRefresh: true,
  },
);

const emit = defineEmits<{
  'metric-change': [alias: string];
  'chart-type-change': [type: ChoyChartKind];
  'stacked-change': [stacked: boolean];
  'sort-change': [sort: ChoyChartSort];
  'chart-item-click': [payload: ChoyChartItemClickPayload];
  refresh: [];
  create: [];
}>();

const localStacked = ref(props.stacked);
const localSort = ref<ChoyChartSort>(props.sort);
const localChartType = ref<ChoyChartKind>(props.chartType);
const localMetric = ref(props.metricAlias);

watch(
  () => props.stacked,
  v => {
    localStacked.value = v;
  },
);
watch(
  () => props.sort,
  v => {
    localSort.value = v;
  },
);
watch(
  () => props.chartType,
  v => {
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
  v => {
    const known = props.metrics.some(m => m.alias === v);
    const next = known ? v : props.metrics[0]?.alias ?? v;
    localMetric.value = next;
    if (next !== v) {
      emit('metric-change', next);
    }
  },
  { immediate: true },
);
watch(
  () => props.metrics,
  metrics => {
    const known = metrics.some(m => m.alias === localMetric.value);
    if (known) return;
    const next = metrics[0]?.alias;
    if (!next || next === localMetric.value) return;
    localMetric.value = next;
    emit('metric-change', next);
  },
  { deep: true },
);

const supportCtx = computed(() => ({
  groupDepth: props.groupDepth,
  stacked: localStacked.value,
  seriesCount: props.seriesMatrix.length,
  metricAlias: localMetric.value || props.metrics[0]?.alias || 'count',
}));

const availableTypes = computed(() => {
  const supported = new Set(availableChartTypes(supportCtx.value));
  // Preserve host-declared order; dedupe and keep allowlist semantics.
  return [...new Set(props.chartTypes)].filter(t => supported.has(t));
});

watch(
  availableTypes,
  types => {
    if (!types.includes(localChartType.value) && types.length) {
      localChartType.value = types[0]!;
      emit('chart-type-change', localChartType.value);
    }
  },
  { immediate: true },
);

const metricLabel = computed(() => {
  const hit = props.metrics.find(m => m.alias === localMetric.value);
  return hit?.label || localMetric.value || 'Value';
});

const stackedDisabled = computed(
  () => localChartType.value === 'pie' || props.seriesMatrix.length <= 1,
);

const sortDisabled = computed(() => localChartType.value === 'pie');

const prepared = computed(() => {
  let categories = props.categories.slice();
  let seriesMatrix = props.seriesMatrix.map(s => ({
    name: s.name,
    data: (s.data || []).slice(),
  }));
  const percent =
    localStacked.value &&
    props.stackMode === 'percent' &&
    localChartType.value !== 'pie' &&
    seriesMatrix.length > 1;
  // Sort on raw totals first; percent columns all sum to 100 and would no-op sort.
  if (!sortDisabled.value && localSort.value !== 'none') {
    const sorted = sortChartCategories(categories, seriesMatrix, localSort.value);
    categories = sorted.categories;
    seriesMatrix = sorted.seriesMatrix;
  }
  if (percent) {
    seriesMatrix = normalizeSeriesToPercent(categories, seriesMatrix);
  }
  return { categories, seriesMatrix, percent };
});

const spec = computed<ChoyChartSpec | null>(() => {
  // Never render a kind that the data context or the host allowlist rejects.
  if (!availableTypes.value.includes(localChartType.value)) return null;
  const adapter = resolveChartAdapter(localChartType.value);
  if (!adapter) return null;
  if (!adapter.supports(supportCtx.value)) return null;
  const { categories, seriesMatrix, percent } = prepared.value;
  if (!categories.length || !seriesMatrix.length) return null;
  const built = adapter.build({
    categories,
    seriesMatrix,
    metricLabel: metricLabel.value,
    stacked: localStacked.value,
    palette: props.palette,
    percent,
  });
  // Pie drops non-positive slices; an all-zero dataset must show empty state.
  if (built.kind === 'pie' && !built.slices?.length) return null;
  return built;
});

const xyRows = computed(() => (spec.value && spec.value.kind !== 'pie' ? chartSpecToXyRows(spec.value) : []));
const pieRows = computed(() => (spec.value?.kind === 'pie' ? chartSpecToPieRows(spec.value) : []));

const xyYAccessors = computed(() => {
  if (!spec.value) return [];
  return spec.value.series.map(s => (d: Record<string, string | number>) => Number(d[s.key]) || 0);
});

const xyColors = computed(() => {
  if (!spec.value) return [];
  return spec.value.series.map(
    s => spec.value!.config[s.key]?.color || CHOY_CHART_DEFAULT_PALETTE[0],
  );
});

type XyRow = Record<string, string | number>;

function xAccessor(d: XyRow): number {
  return Number(d.index) || 0;
}

function xTickFormat(v: number): string {
  const idx = Math.round(Number(v));
  return spec.value?.categories[idx] ?? String(v);
}

function selectMetric(alias: string): void {
  localMetric.value = alias;
  emit('metric-change', alias);
}

function selectChartType(type: ChoyChartKind): void {
  if (!availableTypes.value.includes(type)) return;
  localChartType.value = type;
  emit('chart-type-change', type);
}

function toggleStacked(): void {
  if (stackedDisabled.value) return;
  localStacked.value = !localStacked.value;
  emit('stacked-change', localStacked.value);
}

function selectSort(next: ChoyChartSort): void {
  if (sortDisabled.value) return;
  localSort.value = next;
  emit('sort-change', next);
}

/** Unovis VisEventCallback: (datum, event, index, elements). */
type UnovisClickEvent = MouseEvent | PointerEvent | TouchEvent | WheelEvent;

function emitXyClick(categoryIdx: number, seriesIdx: number | undefined): void {
  if (!spec.value) return;
  const idx = Number.isFinite(categoryIdx) ? Math.round(categoryIdx) : -1;
  const category = spec.value.categories[idx];
  if (idx < 0 || category == null) return;
  // Stacked/grouped event indexes can be stale or out of range: never emit a
  // series index that has no matching series.
  const resolvedSeriesIdx =
    seriesIdx != null &&
    Number.isFinite(seriesIdx) &&
    seriesIdx >= 0 &&
    seriesIdx < spec.value.series.length
      ? Math.round(seriesIdx)
      : undefined;
  const series =
    resolvedSeriesIdx == null ? undefined : spec.value.series[resolvedSeriesIdx];
  emit('chart-item-click', {
    chartType: spec.value.kind,
    category,
    seriesName: series?.name,
    value: series?.values[idx],
    categoryIndex: idx,
    seriesIndex: resolvedSeriesIdx,
    path: [category],
  });
}

/**
 * StackedBar click: datum is mapped with stackIndex + original row; index is category.
 */
function onStackedBarClick(
  d: XyRow & { stackIndex?: number },
  _event: UnovisClickEvent,
  categoryIdx?: number,
): void {
  if (!spec.value) return;
  const { categoryIdx: idx, seriesIdx } = resolveStackedXyClickTarget(
    d,
    categoryIdx,
    d.stackIndex,
  );
  if (idx == null) return;
  emitXyClick(idx, seriesIdx);
}

/**
 * GroupedBar click: datum is the XY row; index is the flattened bar element index
 * (category * seriesCount + series). Prefer row.index for category.
 */
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

/**
 * Line path click: datum is the transformed series polyline (not a category row).
 * Map pointer X within the SVG to the nearest category; third arg is series index.
 */
function onLineClick(
  _data: unknown,
  event: UnovisClickEvent,
  seriesIdx?: number,
): void {
  if (!spec.value) return;
  const rows = xyRows.value;
  if (!rows.length) return;
  const mouse = event as MouseEvent;
  const target = (mouse.currentTarget ?? mouse.target) as Element | null;
  const svg =
    target instanceof SVGSVGElement
      ? target
      : target instanceof Element
        ? target.closest('svg')
        : null;
  if (!svg || typeof mouse.clientX !== 'number') return;
  // The clicked path spans the data x-domain more tightly than the full SVG
  // box (which also covers axis gutters).
  const rect = (target instanceof Element ? target : svg).getBoundingClientRect();
  // Zero/negative width (hidden tab, pre-layout) makes clientX meaningless.
  if (rect.width <= 0) return;
  const rel = (mouse.clientX - rect.left) / rect.width;
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

/**
 * Donut segment click: datum is DonutArcDatum; original row is d.data, slice index is d.index.
 * Second callback arg is the DOM event (not an index).
 */
function onPieSegmentClick(d: {
  data?: { key?: string; name?: string };
  index?: number;
  key?: string;
  name?: string;
}): void {
  if (!spec.value?.slices) return;
  let idx =
    typeof d?.index === 'number' && Number.isFinite(d.index)
      ? Math.round(d.index)
      : spec.value.slices.findIndex(
          sl =>
            sl.key === d?.data?.key ||
            sl.name === d?.data?.name ||
            sl.key === d?.key ||
            sl.name === d?.name,
        );
  if (idx < 0) return;
  const slice = spec.value.slices[idx];
  if (!slice) return;
  const originIdx =
    typeof slice.categoryIndex === 'number' && Number.isFinite(slice.categoryIndex)
      ? Math.round(slice.categoryIndex)
      : idx;
  emit('chart-item-click', {
    chartType: 'pie',
    category: slice.name,
    seriesName: slice.name,
    value: slice.value,
    categoryIndex: originIdx,
    seriesIndex: originIdx,
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
</script>

<template>
  <div
    data-anchor="choy.chart-view"
    data-region="chart-view"
    :class="cn('choy-chart-view flex flex-col gap-3', props.class)"
  >
    <div
      v-if="showHeader"
      class="flex flex-wrap items-center justify-between gap-2"
    >
      <div v-if="showActions" class="flex flex-wrap items-center gap-2">
        <slot name="system-actions">
          <ChoyButton
            v-if="showCreate"
            size="sm"
            variant="outline"
            @click="emit('create')"
          >
            {{ createLabel }}
          </ChoyButton>
          <ChoyButton
            v-if="showRefresh"
            size="sm"
            variant="outline"
            @click="emit('refresh')"
          >
            {{ refreshLabel }}
          </ChoyButton>
        </slot>
        <slot name="user-actions" />
      </div>
      <div class="min-w-0 flex-1">
        <slot name="search" />
      </div>
      <slot name="header-right" />
    </div>

    <div
      v-if="showChartControls"
      class="flex flex-wrap items-center gap-2"
      data-region="chart-controls"
    >
      <slot
        name="metric-switcher"
        :metrics="metrics"
        :current="localMetric"
        :change="selectMetric"
      >
        <label
          v-if="metrics.length"
          class="flex items-center gap-2 text-xs text-muted-foreground"
        >
          Metric
          <select
            class="h-8 rounded-md border border-input bg-background px-2 text-sm text-foreground"
            :value="localMetric"
            aria-label="Metric selection"
            @change="selectMetric(($event.target as HTMLSelectElement).value)"
          >
            <option
              v-for="m in metrics"
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
          aria-label="Chart type"
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
        :toggle="toggleStacked"
      >
        <ChoyButton
          size="sm"
          :variant="localStacked && !stackedDisabled ? 'default' : 'outline'"
          :disabled="stackedDisabled"
          :aria-pressed="localStacked"
          @click="toggleStacked"
        >
          Stack
        </ChoyButton>
      </slot>

      <slot
        name="sort-switcher"
        :current="localSort"
        :change="selectSort"
      >
        <div
          class="inline-flex overflow-hidden rounded-md border border-border"
          role="group"
          aria-label="Sort"
        >
          <ChoyButton
            size="sm"
            class="rounded-none"
            :variant="localSort === 'none' ? 'default' : 'ghost'"
            :disabled="sortDisabled"
            :aria-pressed="localSort === 'none'"
            aria-label="No sort"
            title="No sort"
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
            aria-label="Sort ascending"
            title="Sort ascending"
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
            aria-label="Sort descending"
            title="Sort descending"
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

      <div
        v-else
        class="flex min-h-[280px] items-center justify-center text-sm text-muted-foreground"
      >
        {{ emptyLabel }}
      </div>

      <div
        v-if="loading && !error"
        class="absolute inset-0 flex items-center justify-center bg-background/60 text-sm text-muted-foreground"
      >
        Loading…
      </div>
      <div
        v-else-if="error"
        class="absolute inset-0 flex items-center justify-center bg-background/70 text-sm text-destructive"
        role="alert"
      >
        {{ error }}
      </div>
    </div>
  </div>
</template>
