<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <ChoyViewContainer
    data-anchor="choy.chart-view"
    data-region="chart-view"
    :class="cn('choy-chart-view', 'gap-5', props.class)"
  >
    <template v-if="showHeader" #header>
      <div data-anchor="choy.chart.view-chrome">
        <ChoyViewActionBar
          layout="thirds"
          class="gap-2 border-b-0 [border-bottom-width:0px]"
          :system-actions-label="_t('System actions')"
          :user-actions-label="_t('User actions')"
        >
          <template v-if="showActions || pageTitle || showTitleOverflow" #system-actions>
            <slot v-if="showActions" name="system-actions">
              <!-- Keep New mounted during busy so the action bar does not flash. -->
              <ChoyButton
                v-if="showCreate"
                type="button"
                size="default"
                variant="default"
                :disabled="boardBusy"
                @click="emit('create')"
              >
                <Plus class="size-4" aria-hidden="true" />
                {{ resolvedCreateLabel }}
              </ChoyButton>
            </slot>

            <h1
              v-if="pageTitle"
              :id="pageTitleId || undefined"
              data-slot="page-title"
              class="choy-chart-view__page-title min-w-0 max-w-[10rem] truncate text-base font-semibold text-foreground sm:max-w-[14rem]"
            >
              {{ pageTitle }}
            </h1>
            <ChoyPageTitleActions
              v-if="showTitleOverflow"
              :has-io-menu="!!pageTitleActionsIo"
              :action-import="pageTitleActionsIo?.actionImport ?? false"
              :action-export="pageTitleActionsIo?.actionExport ?? false"
              :action-import-upload-hint="pageTitleActionsIo?.actionImportUploadHint"
              :action-import-column-mapping="pageTitleActionsIo?.actionImportColumnMapping"
              :action-list-ref="(pageTitleActionsIo?.actionListRef as any)"
              :action-company-id="pageTitleActionsIo?.actionCompanyId"
              :store="store"
            >
              <template v-if="showRefreshInOverflow || $slots['overflow-actions']" #overflow>
                <DropdownMenuItem
                  v-if="showRefreshInOverflow"
                  data-testid="chart-overflow-refresh"
                  :disabled="boardBusy"
                  @select="onRefresh"
                >
                  <RefreshCw class="size-4" aria-hidden="true" />
                  {{ resolvedRefreshLabel }}
                </DropdownMenuItem>
                <slot name="overflow-actions" />
              </template>
            </ChoyPageTitleActions>
          </template>

          <template v-if="showActions && !!$slots['user-actions']" #user-actions>
            <slot name="user-actions" />
          </template>

          <!-- Keep #center registered so trailing controls stay in the last column. -->
          <template #center>
            <ChoySearchFit v-if="$slots.search">
              <slot name="search" :on-query-update="onSearch" />
            </ChoySearchFit>
            <div v-else class="min-w-0" />
          </template>

          <template v-if="$slots['header-right']" #header-right>
            <slot name="header-right" />
          </template>
        </ChoyViewActionBar>
      </div>
    </template>

    <div
      v-if="showChartControls"
      class="flex min-h-control flex-nowrap items-center justify-between gap-3 pb-3"
      data-region="chart-controls"
    >
      <!-- Odoo-like: measure left (fixed width); stack / sort / type right. -->
      <slot
        name="metric-switcher"
        :metrics="resolvedMetrics"
        :current="localMetric"
        :change="selectMetric"
      >
        <div v-if="resolvedMetrics.length" class="w-[9.5rem] shrink-0">
          <Select
            class="block w-full"
            :model-value="localMetric"
            @update:model-value="onMetricSelect"
          >
            <SelectTrigger
              class="box-border h-control min-h-control w-full py-0 leading-none"
              data-testid="choy-chart-metric"
              :aria-label="_t('Metric selection')"
              :placeholder="_t('Metric')"
            >
              {{ selectedMetricLabel }}
            </SelectTrigger>
            <SelectContent>
              <SelectItem
                v-for="m in resolvedMetrics"
                :key="m.alias"
                :value="m.alias"
              >
                {{ m.label }}
              </SelectItem>
            </SelectContent>
          </Select>
        </div>
      </slot>

      <TooltipProvider :delay-duration="200">
        <div class="flex h-control shrink-0 items-center gap-3">
          <!-- Odoo: Cumulative only on line charts; any category axis. -->
          <slot
            v-if="cumulativeAvailable"
            name="cumulative-switcher"
            :cumulative="localCumulative"
            :toggle="toggleCumulative"
          >
            <div
              class="flex h-control items-center gap-2 text-sm leading-none text-foreground"
              data-testid="choy-chart-cumulative"
            >
              <span>{{ _t('Cumulative') }}</span>
              <Switch
                v-model="localCumulative"
                :aria-label="_t('Cumulative')"
              />
            </div>
          </slot>

          <!-- Pie: hide stack. Bar needs 2+ series; line keeps stack for area fill (Odoo). -->
          <slot
            v-if="!stackedDisabled"
            name="stacked-switcher"
            :stacked="localStacked"
            :disabled="stackedDisabled"
            :toggle="toggleStacked"
          >
            <div
              class="flex h-control items-center gap-2 text-sm leading-none text-foreground"
              data-testid="choy-chart-stack"
            >
              <span>{{ _t('Stack') }}</span>
              <Switch
                v-model="localStacked"
                :aria-label="_t('Stack')"
              />
            </div>
          </slot>

          <!-- Pie: sort does not apply; hide instead of disabled chrome. -->
          <slot
            v-if="!sortDisabled"
            name="sort-switcher"
            :current="localSort"
            :disabled="sortDisabled"
            :change="selectSort"
          >
            <ChoyButtonGroup :aria-label="_t('Sort')" data-testid="choy-chart-sort-group">
              <Tooltip v-for="opt in sortOptions" :key="opt.value">
                <TooltipTrigger as-child>
                  <ChoyButton
                    type="button"
                    size="icon"
                    variant="outline"
                    :class="localSort === opt.value ? chartControlActive : chartControlIdle"
                    :aria-pressed="localSort === opt.value"
                    :aria-label="opt.label"
                    @click="selectSort(opt.value)"
                  >
                    <component :is="opt.icon" class="size-4" aria-hidden="true" />
                  </ChoyButton>
                </TooltipTrigger>
                <TooltipContent side="bottom" :side-offset="6">
                  {{ opt.label }}
                </TooltipContent>
              </Tooltip>
            </ChoyButtonGroup>
          </slot>

          <slot
            name="chart-type-switcher"
            :types="availableTypes"
            :current="localChartType"
            :change="selectChartType"
          >
            <ChoyButtonGroup
              v-if="availableTypes.length"
              :aria-label="_t('Chart type')"
              data-testid="choy-chart-type-group"
            >
              <Tooltip v-for="t in availableTypes" :key="t">
                <TooltipTrigger as-child>
                  <ChoyButton
                    type="button"
                    size="icon"
                    variant="outline"
                    :class="localChartType === t ? chartControlActive : chartControlIdle"
                    :aria-pressed="localChartType === t"
                    :aria-label="chartTypeLabels[t]"
                    :data-testid="`choy-chart-type-${t}`"
                    @click="selectChartType(t)"
                  >
                    <component :is="chartTypeIcons[t]" class="size-4" aria-hidden="true" />
                  </ChoyButton>
                </TooltipTrigger>
                <TooltipContent side="bottom" :side-offset="6">
                  {{ chartTypeLabels[t] }}
                </TooltipContent>
              </Tooltip>
            </ChoyButtonGroup>
          </slot>
        </div>
      </TooltipProvider>
    </div>

    <ChoyViewBusyHost
      ref="chartBodyRef"
      :busy="boardBusy && !errorText"
      class="relative flex min-h-0 min-w-0 w-full flex-1 flex-col"
      data-region="chart-body"
    >
      <ChartContainer
        v-if="spec"
        :config="spec.config"
        class="relative min-h-0 w-full flex-1"
      >
          <div
          v-if="spec.kind === 'bar' || spec.kind === 'line'"
          class="flex min-h-0 w-full min-w-0 flex-1"
          data-region="chart-xy-frame"
          @mousemove="onXyPointerMove"
          @mouseleave="onXyPointerLeave"
        >
          <!-- Crisp HTML vertical metric label; SVG rotate(-90) blurs CJK. -->
          <div
            v-if="spec.metricLabel"
            class="flex w-6 shrink-0 items-center justify-center self-stretch"
            data-region="chart-y-label"
            aria-hidden="true"
          >
            <span
              class="select-none text-xs leading-none text-muted-foreground [writing-mode:vertical-rl]"
            >
              {{ spec.metricLabel }}
            </span>
          </div>
          <div
            ref="xyPlotHostRef"
            class="relative h-full min-h-[280px] min-w-0 w-full flex-1"
            data-region="chart-xy-plot"
          >
            <VisXYContainer
              :data="xyRows"
              :height="plotHeight"
              :x-domain="xyXDomain"
              :y-domain="xyYDomain"
              :clip-path-extend="xyClipPathExtend"
              class="h-full min-h-[280px] min-w-0 w-full"
            >
              <VisGroupedBar
                v-if="spec.kind === 'bar' && !spec.stacked"
                :x="xAccessor"
                :y="xyYAccessors"
                :color="xyColors"
                :group-max-width="barLayout.barMaxWidth"
                :group-padding="barLayout.groupPadding"
                :bar-padding="barLayout.barPadding"
                :rounded-corners="4"
                :events="groupedBarEvents"
              />
              <VisStackedBar
                v-else-if="spec.kind === 'bar' && spec.stacked"
                :x="xAccessor"
                :y="xyYAccessors"
                :color="xyColors"
                :bar-max-width="barLayout.barMaxWidth"
                :bar-padding="barLayout.barPadding"
                :rounded-corners="4"
                :events="stackedBarEvents"
              />
              <template v-else-if="spec.kind === 'line'">
                <!--
                  Odoo: stacked line → area fill (+ Unovis stacks when `y` is an array);
                  unstacked → stroke only. Line/Scatter use stacked tops so strokes
                  ride the fill ridges.
                -->
                <VisArea
                  v-if="spec.stacked"
                  :x="xAccessor"
                  :y="xyYAccessors"
                  :color="xyColors"
                  :curve-type="CurveType.MonotoneX"
                  :opacity="CHOY_CHART_LINE_AREA_OPACITY"
                />
                <VisLine
                  :x="xAccessor"
                  :y="linePlotYAccessors"
                  :color="xyColors"
                  :curve-type="CurveType.MonotoneX"
                  :line-width="CHOY_CHART_LINE_WIDTH"
                  :events="lineEvents"
                />
                <VisScatter
                  :x="xAccessor"
                  :y="linePlotYAccessors"
                  :color="xyColors"
                  :size="CHOY_CHART_LINE_POINT_SIZE"
                  stroke-color="var(--choy-color-background)"
                  :stroke-width="1.5"
                />
              </template>
              <VisAxis
                type="x"
                :x="xAccessor"
                :tick-format="xTickFormat"
                :tick-line="false"
                :domain-line="false"
                :grid-line="false"
                tick-text-color="var(--choy-color-foreground)"
                tick-text-font-size="12px"
                :tick-text-hide-overlapping="true"
              />
              <VisAxis
                type="y"
                :tick-line="false"
                :domain-line="false"
                :grid-line="true"
                :num-ticks="5"
                tick-text-color="var(--choy-color-muted-foreground)"
                tick-text-font-size="12px"
                :tick-format="yTickFormat"
              />
              <!-- Line hover uses a teleported overlay tip; VisTooltip stays for bars. -->
              <VisTooltip
                v-if="spec.kind !== 'line'"
                :triggers="xyTooltipTriggers"
                class-name="choy-chart-tooltip"
                :container="visTooltipContainer"
                horizontal-placement="auto"
                vertical-placement="top"
                :follow-cursor="true"
              />
            </VisXYContainer>
            <!-- Odoo-like category column cue while line hover is active. -->
            <div
              v-if="spec.kind === 'line' && lineHoverBand"
              data-testid="choy-chart-line-band"
              class="pointer-events-none absolute z-[1] bg-foreground/[0.04]"
              :style="{
                left: `${lineHoverBand.left}px`,
                top: `${lineHoverBand.top}px`,
                width: `${lineHoverBand.width}px`,
                height: `${lineHoverBand.height}px`,
              }"
            />
          </div>
        </div>

        <VisSingleContainer
          v-else-if="spec.kind === 'pie'"
          :data="pieRows"
          :height="plotHeight"
          class="h-full min-h-[280px] w-full"
        >
          <!-- arcWidth 0 → solid pie (Odoo graph); non-zero would be a donut ring. -->
          <!-- Solid pie: hide Unovis donut track (showBackground) so no gray outer ring. -->
          <VisDonut
            :value="(d: { value: number }) => d.value"
            :color="(d: { color: string }) => d.color"
            :arc-width="0"
            :show-background="false"
            :pad-angle="0"
            :corner-radius="0"
            :events="donutEvents"
          />
          <VisTooltip
            :triggers="pieTooltipTriggers"
            class-name="choy-chart-tooltip"
            :container="visTooltipContainer"
            horizontal-placement="auto"
            vertical-placement="top"
            :follow-cursor="true"
          />
        </VisSingleContainer>

        <ChartLegendContent
          :class="
            spec.kind === 'pie'
              ? 'absolute end-3 top-3 z-[1] flex-col flex-nowrap items-end justify-end gap-1.5 pt-0 text-xs'
              : undefined
          "
        />

        <!-- Fixed tip on body so overflow:hidden ancestors cannot clip the last category. -->
        <Teleport to="body">
          <div
            v-if="spec.kind === 'line' && lineHoverTip"
            ref="lineHoverTipEl"
            class="choy-chart-tooltip choy-chart-tooltip--floating pointer-events-none"
            data-testid="choy-chart-line-tooltip"
            :style="{
              left: `${lineHoverTip.x}px`,
              top: `${lineHoverTip.y}px`,
            }"
            v-html="lineHoverTip.html"
          />
        </Teleport>
      </ChartContainer>

      <ChoyEmpty
        v-else-if="!boardBusy && !errorText"
        class="min-h-[280px] w-full flex-1 border-none"
        :description="resolvedEmptyLabel"
      />

      <div
        v-if="errorText"
        class="absolute inset-0 flex flex-col items-center justify-center gap-2 bg-background/70 text-sm text-destructive"
        role="alert"
      >
        <span>{{ errorText }}</span>
        <ChoyButton size="sm" variant="outline" @click="onRefresh">
          {{ _t('Retry') }}
        </ChoyButton>
      </div>
    </ChoyViewBusyHost>
  </ChoyViewContainer>
</template>

<script setup lang="ts">
import {
  computed,
  nextTick,
  onBeforeUnmount,
  onMounted,
  ref,
  useSlots,
  watch,
  type Component,
} from 'vue';
import {
  ArrowDown,
  ArrowUp,
  BarChart3,
  Equal,
  LineChart,
  PieChart,
  Plus,
  RefreshCw,
} from 'lucide-vue-next';
import {
  VisArea,
  VisAxis,
  VisDonut,
  VisGroupedBar,
  VisLine,
  VisScatter,
  VisSingleContainer,
  VisStackedBar,
  VisTooltip,
  VisXYContainer
} from '@unovis/vue';
import { CurveType, Donut, GroupedBar, Line, Scatter, StackedBar } from '@unovis/ts';
import type { ClassValue } from '../../lib/utils';
import { cn } from '../../lib/utils';
import type { BaseModel } from '@/core/rpc';
import type { GroupBySpec, QueryCondition } from '@/core/service/api/query';
import type { WebModelStore } from '@/web/web/stores/modelStore';
import type { OrderByState } from '@/web/web/query/state';
import type { ChoySearchQuery } from './searchViewHelpers';
import ChoyButton from '../layout/ChoyButton.vue';
import ChoyButtonGroup from '../layout/ChoyButtonGroup.vue';
import ChoyEmpty from '../layout/ChoyEmpty.vue';
import ChoyPageTitleActions from '../layout/ChoyPageTitleActions.vue';
import ChoyViewContainer from '@/web/web/components/view/ChoyViewContainer.vue';
import ChoyViewActionBar from '@/web/web/components/view/ChoyViewActionBar.vue';
import ChoySearchFit from '@/web/web/components/view/ChoySearchFit.vue';
import ChoyViewBusyHost from '@/web/web/components/view/ChoyViewBusyHost.vue';
import { useProvideViewScope } from '@/web/web/components/view/viewScopeKeys';
import { createTranslate } from '@/web/web/i18n';
import {
  ChartContainer,
  ChartLegendContent
} from '../vendor/ui/chart';
import Select from '../vendor/ui/select/Select.vue';
import SelectContent from '../vendor/ui/select/SelectContent.vue';
import SelectItem from '../vendor/ui/select/SelectItem.vue';
import SelectTrigger from '../vendor/ui/select/SelectTrigger.vue';
import Switch from '../vendor/ui/switch/Switch.vue';
import DropdownMenuItem from '../vendor/ui/dropdown-menu/DropdownMenuItem.vue';
import Tooltip from '../vendor/ui/tooltip/Tooltip.vue';
import TooltipContent from '../vendor/ui/tooltip/TooltipContent.vue';
import TooltipProvider from '../vendor/ui/tooltip/TooltipProvider.vue';
import TooltipTrigger from '../vendor/ui/tooltip/TooltipTrigger.vue';
import { createChartController } from '@/web/web/controllers/chartController';
import { resolvePageStore, useHostPageTitleInView } from '@/web/web/composables/usePageContext';
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
  accumulateChartSeries,
  CHOY_CHART_LINE_AREA_OPACITY,
  CHOY_CHART_LINE_CLIP_EXTEND,
  CHOY_CHART_LINE_POINT_SIZE,
  CHOY_CHART_LINE_WIDTH,
  CHOY_CHART_MIN_PLOT_HEIGHT,
  chartCategoryAxisIsTemporal,
  chartSpecToPieRows,
  chartSpecToXyRows,
  buildChartXyTooltipHtml,
  formatChartCompactNumber,
  formatChartTooltipHtml,
  formatChartTooltipPiePercent,
  formatChartTooltipValue,
  normalizeSeriesToPercent,
  resolveChartBarLayout,
  resolveChartFloatingTooltipPosition,
  resolveChartPlotHeight,
  resolveChartTooltipViewportSize,
  resolveChartXDomain,
  resolveStackedYDomain,
  resolveGroupedXyClickTarget,
  resolveClickClientX,
  resolveLineClickCategory,
  resolveLineFlushHoverBand,
  resolveLinePlotHoverTarget,
  resolveXyPlotClientRect,
  resolveStackedXyClickTarget,
  rowCategoryIndex,
  sortChartCategories,
  sortChartCategoriesByTime,
  stackSeriesTopsAtCategory,
  type ChoyChartItemClickPayload,
  type ChoyChartMetricOption
} from './chartViewHelpers';
import { groupRowsToChartSeries } from './chartStoreHelpers';

/**
 * Store-bound chart view. Requires :store or a page-provided store.
 * Owns createChartController and maps grouped snapshots into Unovis series.
 */
const { _t } = createTranslate('web', { scope: 'web/components/view/ChartView' });

useProvideViewScope({ kind: 'Chart', mode: 'display' });

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
    /** Running total along the category axis (line charts only, like Odoo). */
    cumulative?: boolean;
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
    /** Optional remap for category / series labels after group snapshot mapping. */
    formatLabel?: (label: string) => string;
    onLoadError?: (error: unknown) => void;
    onSearchError?: (error: unknown) => void;
  }>(),
  {
    metrics: () => [],
    metricAlias: '',
    chartTypes: () => ['bar', 'line', 'pie'],
    chartType: 'bar',
    stacked: true,
    cumulative: false,
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

const slots = useSlots();
const { pageTitle, pageTitleId, pageTitleActionsIo } = useHostPageTitleInView(
  () => props.showHeader,
);
const showRefreshInOverflow = computed(
  () => props.showActions && props.showHeader && props.showRefresh,
);
const showTitleOverflow = computed(
  () =>
    !!pageTitleActionsIo.value ||
    showRefreshInOverflow.value ||
    !!slots['overflow-actions'],
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
  'cumulative-change': [cumulative: boolean];
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

const chartBodyRef = ref<{ $el?: HTMLElement } | HTMLElement | null>(null);
const plotHeight = ref(CHOY_CHART_MIN_PLOT_HEIGHT);
let chartBodyObserver: ResizeObserver | null = null;

function chartBodyEl(): HTMLElement | null {
  const r = chartBodyRef.value;
  if (!r) return null;
  if (r instanceof HTMLElement) return r;
  const el = (r as { $el?: unknown }).$el;
  return el instanceof HTMLElement ? el : null;
}

function syncPlotHeight(): void {
  const el = chartBodyEl();
  const h = el?.clientHeight ?? 0;
  plotHeight.value = resolveChartPlotHeight(h);
}

const localStacked = ref(props.stacked);
const localCumulative = ref(props.cumulative);
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

/** Explicit trigger text so SelectValue does not flash the raw alias before items mount. */
const selectedMetricLabel = computed(() => {
  const hit = resolvedMetrics.value.find((m) => m.alias === localMetric.value);
  return hit?.label || localMetric.value || _t('Metric');
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
    arr.unshift({ alias: 'count', label: _t('Count') });
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
  clearLinePlotHover();
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
  const fmt = props.formatLabel;
  if (fmt) {
    categories.value = mapped.categories.map((c) => fmt(c));
    seriesMatrix.value = mapped.seriesMatrix.map((s) => ({
      name: fmt(s.name),
      data: s.data,
    }));
  } else {
    categories.value = mapped.categories;
    seriesMatrix.value = mapped.seriesMatrix;
  }
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
watch(localStacked, (v) => {
  if (v !== props.stacked) emit('stacked-change', v);
});
watch(
  () => props.cumulative,
  (v) => {
    localCumulative.value = v;
  },
);
watch(localCumulative, (v) => {
  if (v !== props.cumulative) emit('cumulative-change', v);
});
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
      emit('chart-type-change', next);
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

/** Pie: no stack. Bar: need 2+ series. Line: always (area fill / Y stack, Odoo). */
const stackedDisabled = computed(
  () =>
    localChartType.value === 'pie' ||
    (localChartType.value === 'bar' && seriesMatrix.value.length <= 1),
);

const sortDisabled = computed(() => localChartType.value === 'pie');

/** First applied/default groupby is temporal → keep timeline order when sort is none. */
const activeChartGroups = computed(() => {
  const fromStore = (store.state as { queryState?: { appliedGroups?: unknown } } | undefined)
    ?.queryState?.appliedGroups;
  if (Array.isArray(fromStore) && fromStore.length) return fromStore;
  const defaults = props.defaultGroups;
  if (Array.isArray(defaults)) return defaults;
  return defaults != null ? [defaults] : [];
});

const categoryAxisIsTemporal = computed(() => chartCategoryAxisIsTemporal(activeChartGroups.value));

/** Odoo: Cumulative toggle only for line charts. */
const cumulativeAvailable = computed(() => localChartType.value === 'line');

watch(cumulativeAvailable, (ok) => {
  if (!ok && localCumulative.value) localCumulative.value = false;
});

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
    localChartType.value === 'bar' &&
    matrix.length > 1;
  if (!sortDisabled.value && localSort.value !== 'none') {
    const sorted = sortChartCategories(cats, matrix, localSort.value);
    cats = sorted.categories;
    matrix = sorted.seriesMatrix;
    categoryOrder = sorted.order;
  } else if (!sortDisabled.value && categoryAxisIsTemporal.value) {
    // Default timeline order when the user has not picked value sort.
    const sorted = sortChartCategoriesByTime(cats, matrix);
    cats = sorted.categories;
    matrix = sorted.seriesMatrix;
    categoryOrder = sorted.order;
  }
  if (localCumulative.value && cumulativeAvailable.value) {
    matrix = accumulateChartSeries(cats, matrix);
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

/** Few categories → wider bars / less padding (closer to Odoo graph density). */
const barLayout = computed(() =>
  resolveChartBarLayout(spec.value?.kind === 'bar' ? spec.value.categories.length : 0),
);

/** Bar: band centers. Line: first category at x=0 (flush to the Y-axis, like Odoo). */
const xyXDomain = computed<[number, number] | undefined>(() => {
  const s = spec.value;
  if (!s || s.kind === 'pie') return undefined;
  return resolveChartXDomain(s.categories.length, s.kind === 'line' ? 'line' : 'bar');
});

/** Widen Unovis clip slightly for line markers at the flush domain edges. */
const xyClipPathExtend = computed(() =>
  spec.value?.kind === 'line' ? CHOY_CHART_LINE_CLIP_EXTEND : 2,
);

/** Stacked bar/line plots need a domain from the column sum; otherwise Unovis auto-scale. */
const xyYDomain = computed<[number, number] | undefined>(() => {
  const s = spec.value;
  if (!s || s.kind === 'pie' || !s.stacked) return undefined;
  if (s.kind !== 'bar' && s.kind !== 'line') return undefined;
  return resolveStackedYDomain(
    xyRows.value,
    s.series.map((series) => series.key),
    { percent: s.percent },
  );
});

const xyYAccessors = computed(() => {
  if (!spec.value) return [];
  return spec.value.series.map((s) => (d: Record<string, string | number>) => Number(d[s.key]) || 0);
});

/** Line stroke/points: stacked tops when stacked; raw values otherwise. */
const linePlotYAccessors = computed(() => {
  const s = spec.value;
  if (!s || s.kind !== 'line') return [];
  const keys = s.series.map((series) => series.key);
  if (!s.stacked) {
    return keys.map((key) => (d: Record<string, string | number>) => Number(d[key]) || 0);
  }
  return keys.map((_, seriesIdx) => (d: Record<string, string | number>) => {
    let sum = 0;
    for (let i = 0; i <= seriesIdx; i++) {
      sum += Number(d[keys[i]!]) || 0;
    }
    return sum;
  });
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

function yTickFormat(v: number): string {
  if (spec.value?.percent) return `${v}%`;
  return formatChartCompactNumber(v);
}

function xTickFormat(v: number): string {
  const cats = spec.value?.categories ?? [];
  const raw = Number(v);
  if (!cats.length || !Number.isFinite(raw)) return '';
  const idx = Math.round(raw);
  if (Math.abs(raw - idx) > 1e-6 || idx < 0 || idx >= cats.length) return '';
  return cats[idx]!;
}

// Match ChoyViewSwitcher: muted fill for pressed toggles (not primary green).
const chartControlActive = 'z-[1] bg-muted text-foreground hover:bg-muted';
const chartControlIdle = 'text-muted-foreground';

const chartTypeIcons: Record<ChoyChartKind, Component> = {
  bar: BarChart3,
  line: LineChart,
  pie: PieChart,
};

const chartTypeLabels = computed(() => ({
  bar: _t('Bar'),
  line: _t('Line'),
  pie: _t('Pie'),
}));

const sortOptions = computed(() => [
  { value: 'none' as const, label: _t('No sort'), icon: Equal },
  { value: 'asc' as const, label: _t('Sort ascending'), icon: ArrowUp },
  { value: 'desc' as const, label: _t('Sort descending'), icon: ArrowDown },
]);

function selectMetric(alias: string): void {
  if (alias === localMetric.value) return;
  if (resolvedMetrics.value.length > 0 && !resolvedMetrics.value.some((m) => m.alias === alias)) return;
  localMetric.value = alias;
  emit('metric-change', alias);
}

function onMetricSelect(value: unknown): void {
  if (value == null || value === '') return;
  selectMetric(String(value));
}

function selectChartType(type: ChoyChartKind): void {
  if (!availableTypes.value.includes(type) || type === localChartType.value) return;
  localChartType.value = type;
  emit('chart-type-change', type);
}

function toggleStacked(): void {
  if (stackedDisabled.value) return;
  localStacked.value = !localStacked.value;
}

function toggleCumulative(): void {
  if (!cumulativeAvailable.value) return;
  localCumulative.value = !localCumulative.value;
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
  // Minimal FE DOM may lack SVGSVGElement; treat tagName=svg as the plot root.
  const svg =
    target instanceof Element
      ? target.tagName?.toLowerCase() === 'svg'
        ? target
        : target.closest('svg')
      : null;
  const clientX = resolveClickClientX(ev);
  if (!svg || clientX == null) return;
  const rect =
    target instanceof Element
      ? target.getBoundingClientRect()
      : (svg as Element).getBoundingClientRect();
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
  if (idx < 0 && clickedName != null) {
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

/**
 * Hover highlight — VisTooltip / line-plot overlay keep mark + panel in sync.
 * Bars/pie: brightness. Scatter: halo ring only (no SVG scale — that fought
 * Unovis `translate(...)` and left end markers looking inconsistently sized).
 */
const CHOY_CHART_HOVER_MARK_CLASS = 'choy-chart-mark--hover';

let hoverMarkEl: Element | null = null;

function isStyledDomEl(el: Element): el is Element & { style: CSSStyleDeclaration } {
  return typeof (el as { style?: unknown }).style === 'object' && (el as { style?: unknown }).style != null;
}

function clearHoverMarkStyle(el: Element): void {
  el.classList.remove(CHOY_CHART_HOVER_MARK_CLASS);
  if (isStyledDomEl(el)) {
    el.style.removeProperty('filter');
  }
  const paints = [el, ...Array.from(el.querySelectorAll('path, circle, rect'))];
  for (const paint of paints) {
    if (!isStyledDomEl(paint)) continue;
    paint.style.removeProperty('stroke');
    paint.style.removeProperty('stroke-width');
    paint.style.removeProperty('paint-order');
    paint.style.removeProperty('filter');
  }
}

function resolveScatterGroup(el: Element): Element {
  if (el.tagName.toLowerCase() === 'g' && el.hasAttribute('transform')) return el;
  // Minimal FE DOM may not support attribute selectors on closest().
  try {
    return el.closest('g[transform]') ?? el;
  } catch {
    return el;
  }
}

function applyScatterHoverMark(el: Element): void {
  const g = resolveScatterGroup(el);
  g.classList.add(CHOY_CHART_HOVER_MARK_CLASS);
  // Emphasize the glyph inside the translated group — keep Unovis transform intact.
  const paint = g.querySelector('path, circle, rect') ?? g;
  if (isStyledDomEl(paint)) {
    paint.style.setProperty('stroke', 'var(--choy-color-background)', 'important');
    paint.style.setProperty('stroke-width', '4px', 'important');
    paint.style.setProperty('paint-order', 'stroke fill', 'important');
    paint.style.setProperty('filter', 'brightness(1.15)', 'important');
  }
  hoverMarkEl = g;
}

/**
 * @param kind `scatter` uses a halo ring; `fill` only brightens (bars/pie).
 */
function applyHoverMarkEl(
  el: Element | null | undefined,
  kind: 'fill' | 'scatter' = 'fill',
): void {
  const target = el && kind === 'scatter' ? resolveScatterGroup(el) : el;
  if (hoverMarkEl && hoverMarkEl !== target) {
    clearHoverMarkStyle(hoverMarkEl);
    hoverMarkEl = null;
  }
  if (!target) return;
  if (kind === 'scatter') {
    applyScatterHoverMark(target);
    return;
  }
  target.classList.add(CHOY_CHART_HOVER_MARK_CLASS);
  if (isStyledDomEl(target)) {
    target.style.setProperty('filter', 'brightness(1.22) saturate(1.12)', 'important');
  }
  hoverMarkEl = target;
}

function clearChartHover(): void {
  if (hoverMarkEl) {
    clearHoverMarkStyle(hoverMarkEl);
    hoverMarkEl = null;
  }
  // Sweep any leftover hover marks if Unovis recreated nodes mid-hover.
  const host = xyPlotHostRef.value;
  if (!host) return;
  for (const el of host.querySelectorAll(`.${CHOY_CHART_HOVER_MARK_CLASS}`)) {
    clearHoverMarkStyle(el);
  }
}

/** VisTooltip calls triggers as `(datum, index, nodes)`. */
function highlightTooltipMark(
  index: number | undefined,
  nodes: ArrayLike<Element> | undefined,
  kind: 'fill' | 'scatter' = 'fill',
): void {
  if (typeof index !== 'number' || !Number.isFinite(index) || !nodes) return;
  const el = nodes[Math.round(index)];
  applyHoverMarkEl(el instanceof Element ? el : null, kind);
}

function readDomDatum(el: Element): unknown {
  return (el as Element & { __data__?: unknown }).__data__;
}

function scatterPointMatches(
  d: unknown,
  categoryIdx: number,
  seriesIdx: number,
): boolean {
  if (!d || typeof d !== 'object') return false;
  const row = d as XyRow & {
    _point?: { pointIndex?: number; groupIndex?: number };
  };
  const point = row._point;
  const pi =
    typeof point?.pointIndex === 'number' && Number.isFinite(point.pointIndex)
      ? Math.round(point.pointIndex)
      : rowCategoryIndex(row);
  const gi =
    typeof point?.groupIndex === 'number' && Number.isFinite(point.groupIndex)
      ? Math.round(point.groupIndex)
      : 0;
  return pi === categoryIdx && gi === seriesIdx;
}

/**
 * Locate the Scatter vertex for a category/series so line-path tooltips can
 * highlight the matching point without requiring a direct point hit.
 */
function findScatterPointEl(
  root: Element | null | undefined,
  categoryIdx: number,
  seriesIdx: number,
): Element | null {
  if (!root) return null;
  const raw = String(Scatter.selectors.point || '');
  const className = raw.startsWith('.') ? raw.slice(1) : raw;
  const escape =
    typeof CSS !== 'undefined' && typeof CSS.escape === 'function'
      ? CSS.escape
      : (s: string) => s.replace(/[^a-zA-Z0-9_-]/g, '\\$&');
  let nodes: Element[] = [];
  if (className) {
    try {
      nodes = Array.from(root.querySelectorAll(`.${escape(className)}`));
    } catch {
      nodes = Array.from(root.querySelectorAll(`.${className}`));
    }
  }
  // Also scan translated groups — selector class may be hashed / nested.
  // Minimal FE DOM may not support attribute selectors; ignore and fall through.
  if (!nodes.length) {
    try {
      nodes = Array.from(root.querySelectorAll('g[transform]'));
    } catch {
      nodes = [];
    }
  }
  for (const node of nodes) {
    if (scatterPointMatches(readDomDatum(node), categoryIdx, seriesIdx)) {
      return node;
    }
  }
  // Fallback when __data__ is unavailable: Unovis lays out series-major points.
  const nCat = spec.value?.categories.length ?? 0;
  const nSer = spec.value?.series.length ?? 0;
  if (nCat > 0 && nSer > 0 && nodes.length >= nCat * nSer) {
    const seriesMajor = seriesIdx * nCat + categoryIdx;
    const categoryMajor = categoryIdx * nSer + seriesIdx;
    return nodes[seriesMajor] ?? nodes[categoryMajor] ?? null;
  }
  return null;
}

function highlightScatterPointForTooltip(
  categoryIdx: number | null,
  seriesIdx: number | undefined,
  root: Element | null | undefined,
): void {
  if (categoryIdx == null || seriesIdx == null) {
    clearChartHover();
    return;
  }
  applyHoverMarkEl(findScatterPointEl(root, categoryIdx, seriesIdx), 'scatter');
}

const xyPlotHostRef = ref<HTMLElement | null>(null);
const lineHoverTipEl = ref<HTMLElement | null>(null);
/** Portal target so Unovis VisTooltip (bar/pie) is not clipped by plot overflow. */
const chartTooltipPortal = ref<HTMLElement | undefined>(undefined);
/**
 * VisTooltip `container` typing can disagree with the app DOM lib's HTMLElement
 * (duplicate lib.dom merges). Bridge with an unknown cast at the prop boundary.
 */
const visTooltipContainer = computed(
  (): HTMLElement | undefined =>
    chartTooltipPortal.value as unknown as HTMLElement | undefined,
);

type LineHoverTip = { html: string; x: number; y: number };
type LineHoverBand = { left: number; top: number; width: number; height: number };

const lineHoverTip = ref<LineHoverTip | null>(null);
const lineHoverBand = ref<LineHoverBand | null>(null);

/** Fallback tip size before the floating node is measured. */
const LINE_HOVER_TIP_FALLBACK = { width: 180, height: 72 };

function clearLinePlotHover(): void {
  clearChartHover();
  lineHoverTip.value = null;
  lineHoverBand.value = null;
}

/**
 * Fixed viewport placement for the teleported line tip (client coordinates).
 */
function placeLineHoverTip(html: string, clientX: number, clientY: number): void {
  const measured = lineHoverTipEl.value?.getBoundingClientRect();
  const hasMeasured = !!(measured && measured.width > 0 && measured.height > 0);
  const tipSize = hasMeasured
    ? { width: measured!.width, height: measured!.height }
    : LINE_HOVER_TIP_FALLBACK;
  const viewport = resolveChartTooltipViewportSize();
  const placed = resolveChartFloatingTooltipPosition({
    cursorX: clientX,
    cursorY: clientY,
    hostWidth: viewport.width,
    hostHeight: viewport.height,
    tipWidth: tipSize.width,
    tipHeight: tipSize.height,
  });
  const prevHtml = lineHoverTip.value?.html;
  lineHoverTip.value = { html, x: placed.x, y: placed.y };
  // Remeasure only when the tip is new/unmeasured or the HTML size may have changed.
  if (hasMeasured && prevHtml === html) return;
  void nextTick(() => {
    if (!lineHoverTip.value || lineHoverTip.value.html !== html) return;
    const tipNow = lineHoverTipEl.value;
    if (!tipNow) return;
    const tipNowRect = tipNow.getBoundingClientRect();
    const again = resolveChartFloatingTooltipPosition({
      cursorX: clientX,
      cursorY: clientY,
      hostWidth: viewport.width,
      hostHeight: viewport.height,
      tipWidth: tipNowRect.width,
      tipHeight: tipNowRect.height,
    });
    lineHoverTip.value = { html, x: again.x, y: again.y };
  });
}

function linePlotYDomain(): [number, number] {
  const domain = xyYDomain.value;
  if (domain) return domain;
  const s = spec.value;
  if (!s || s.kind === 'pie') return [0, 1];
  let max = 0;
  for (const series of s.series) {
    for (const raw of series.values) {
      const v = Number(raw);
      if (Number.isFinite(v) && v > max) max = v;
    }
  }
  return [0, max > 0 ? max : 1];
}

/**
 * Full-plot line hover (Odoo-like category columns): tip + vertex highlight
 * track the pointer anywhere over the SVG, not only on the thin stroke.
 */
function updateLinePlotHover(event: MouseEvent): void {
  const host = xyPlotHostRef.value;
  const current = spec.value;
  if (!host || !current || current.kind !== 'line') {
    clearLinePlotHover();
    return;
  }
  const svg = host.querySelector('svg');
  if (!svg) {
    clearLinePlotHover();
    return;
  }
  const plotRect = resolveXyPlotClientRect(svg);
  const hostRect = host.getBoundingClientRect();
  if (plotRect.width <= 0 || plotRect.height <= 0) {
    clearLinePlotHover();
    return;
  }
  if (
    event.clientX < plotRect.left ||
    event.clientX > plotRect.right ||
    event.clientY < plotRect.top ||
    event.clientY > plotRect.bottom
  ) {
    clearLinePlotHover();
    return;
  }
  const nCat = current.categories.length;
  const relX = (event.clientX - plotRect.left) / plotRect.width;
  const relY = (event.clientY - plotRect.top) / plotRect.height;
  const { categoryIdx, seriesIdx } = resolveLinePlotHoverTarget({
    relX,
    relY,
    categoryCount: nCat,
    seriesValuesAtCategory: (idx) => {
      const raw = current.series.map((series) => Number(series.values[idx]) || 0);
      return current.stacked ? stackSeriesTopsAtCategory(raw) : raw;
    },
    yDomain: linePlotYDomain(),
  });
  const html = buildXyTooltipHtml(categoryIdx, seriesIdx);
  highlightScatterPointForTooltip(categoryIdx, seriesIdx, svg);
  lineHoverBand.value = resolveLineFlushHoverBand({
    categoryIdx,
    categoryCount: nCat,
    plotLeft: plotRect.left,
    plotTop: plotRect.top,
    plotWidth: plotRect.width,
    plotHeight: plotRect.height,
    hostLeft: hostRect.left,
    hostTop: hostRect.top,
  });
  if (html) placeLineHoverTip(html, event.clientX, event.clientY);
}

function onXyPointerMove(event: MouseEvent): void {
  if (spec.value?.kind === 'line') {
    updateLinePlotHover(event);
  }
}

function onXyPointerLeave(): void {
  clearLinePlotHover();
}

watch(
  () => spec.value?.kind,
  () => {
    clearLinePlotHover();
  },
);

const groupedBarEvents = computed(() => ({
  [GroupedBar.selectors.bar]: {
    click: onGroupedBarClick,
    mouseout: clearChartHover,
  },
}));
const stackedBarEvents = computed(() => ({
  [StackedBar.selectors.bar]: {
    click: onStackedBarClick,
    mouseout: clearChartHover,
  },
}));
const lineEvents = computed(() => ({
  [Line.selectors.line]: { click: onLineClick },
}));
const donutEvents = computed(() => ({
  [Donut.selectors.segment]: {
    click: onPieSegmentClick,
    mouseout: clearChartHover,
  },
}));

/**
 * Odoo-like hover panel: category title; multi-series shows every series at that category.
 * `seriesIdx` is kept for call-site highlight sync; tip rows come from {@link buildChartXyTooltipHtml}.
 */
function buildXyTooltipHtml(
  categoryIdx: number | null,
  seriesIdx: number | undefined,
): string | null {
  void seriesIdx;
  const current = spec.value;
  if (!current || current.kind === 'pie') return null;
  return buildChartXyTooltipHtml(
    {
      categories: current.categories,
      series: current.series,
      config: current.config,
      metricLabel: current.metricLabel,
      percent: current.percent,
      palette: CHOY_CHART_DEFAULT_PALETTE,
    },
    categoryIdx,
  );
}

function onStackedBarTooltip(
  d:
    | (XyRow & { stackIndex?: number; datum?: XyRow; index?: number })
    | undefined,
  index?: number,
  nodes?: ArrayLike<Element>,
): string | null {
  if (!spec.value) {
    clearChartHover();
    return null;
  }
  // Tooltip reads the raw D3 datum (`{ datum, index, stackIndex }`); clicks get a mapped flatten.
  const row = (d?.datum ?? d) as XyRow | undefined;
  const stackIndex =
    typeof d?.stackIndex === 'number'
      ? d.stackIndex
      : typeof (row as { stackIndex?: number } | undefined)?.stackIndex === 'number'
        ? (row as { stackIndex?: number }).stackIndex
        : undefined;
  const categoryFromEvent =
    typeof d?.index === 'number' && Number.isFinite(d.index) ? Math.round(d.index) : undefined;
  const { categoryIdx, seriesIdx } = resolveStackedXyClickTarget(
    row,
    categoryFromEvent,
    stackIndex,
  );
  const html = buildXyTooltipHtml(categoryIdx, seriesIdx);
  if (html) highlightTooltipMark(index, nodes);
  else clearChartHover();
  return html;
}

function onGroupedBarTooltip(
  d: XyRow | undefined,
  flatIndex?: number,
  nodes?: ArrayLike<Element>,
): string | null {
  if (!spec.value) {
    clearChartHover();
    return null;
  }
  const { categoryIdx, seriesIdx } = resolveGroupedXyClickTarget(
    d,
    flatIndex,
    spec.value.series.length,
  );
  const html = buildXyTooltipHtml(categoryIdx, seriesIdx);
  if (html) highlightTooltipMark(flatIndex, nodes);
  else clearChartHover();
  return html;
}

function onPieSegmentTooltip(
  d: {
    data?: { key?: string; name?: string; value?: number; color?: string };
    index?: number;
    key?: string;
    name?: string;
    value?: number;
    color?: string;
  },
  index?: number,
  nodes?: ArrayLike<Element>,
): string | null {
  const current = spec.value;
  const slices = current?.slices;
  if (!current || current.kind !== 'pie' || !slices?.length) {
    clearChartHover();
    return null;
  }
  const clickedKey = d?.data?.key ?? d?.key;
  const clickedName = d?.data?.name ?? d?.name;
  let idx =
    clickedKey != null ? slices.findIndex((sl) => sl.key === clickedKey) : -1;
  if (idx < 0 && clickedName != null) {
    idx = slices.findIndex((sl) => sl.name === clickedName);
  }
  if (idx < 0) {
    idx =
      typeof d?.index === 'number' && Number.isFinite(d.index)
        ? Math.round(d.index)
        : -1;
  }
  if (idx < 0) {
    clearChartHover();
    return null;
  }
  const slice = slices[idx];
  if (!slice) {
    clearChartHover();
    return null;
  }
  const color = current.config[slice.key]?.color || CHOY_CHART_DEFAULT_PALETTE[0] || '#888';
  const total = slices.reduce((sum, sl) => sum + Math.max(0, Number(sl.value) || 0), 0);
  const value = Number(slice.value) || 0;
  const valueText = `${formatChartTooltipValue(value)} (${formatChartTooltipPiePercent(value, total)})`;
  const html = formatChartTooltipHtml({
    title: slice.name,
    rows: [
      {
        color,
        label: current.metricLabel || slice.name,
        valueText,
      },
    ],
  });
  highlightTooltipMark(index, nodes);
  return html;
}

const xyTooltipTriggers = computed(() => {
  const current = spec.value;
  if (!current || current.kind === 'pie' || current.kind === 'line') return {};
  if (current.kind === 'bar' && current.stacked) {
    return { [StackedBar.selectors.bar]: onStackedBarTooltip };
  }
  return { [GroupedBar.selectors.bar]: onGroupedBarTooltip };
});

const pieTooltipTriggers = computed(() => ({
  [Donut.selectors.segment]: onPieSegmentTooltip,
}));

function onViewportDismissLineHover(): void {
  clearLinePlotHover();
}

onMounted(() => {
  if (typeof document !== 'undefined') {
    chartTooltipPortal.value = document.body;
  }
  if (typeof window !== 'undefined') {
    // Fixed tip on body can linger across scroll/route without mouseleave.
    window.addEventListener('scroll', onViewportDismissLineHover, true);
    window.addEventListener('resize', onViewportDismissLineHover);
  }
  const Observer = (globalThis as { ResizeObserver?: typeof ResizeObserver }).ResizeObserver;
  const el = chartBodyEl();
  if (Observer && el) {
    chartBodyObserver = new Observer(() => {
      syncPlotHeight();
    });
    chartBodyObserver.observe(el);
    syncPlotHeight();
  }
  if (!props.autoBootstrap) {
    initMetricOptions();
    return;
  }
  void bootstrap().catch((error) => props.onLoadError?.(error));
});

onBeforeUnmount(() => {
  clearLinePlotHover();
  if (typeof window !== 'undefined') {
    window.removeEventListener('scroll', onViewportDismissLineHover, true);
    window.removeEventListener('resize', onViewportDismissLineHover);
  }
  chartBodyObserver?.disconnect();
  chartBodyObserver = null;
});

defineExpose({
  bootstrap,
  applyQuery,
  rebuildFromSnapshot,
  boardBusy,
  controller,
  plotHeight,
  /** Test / debug hook: run the line-plot hover path with a synthetic pointer. */
  onXyPointerMove,
  clearLinePlotHover,
  // FE unit coverage hooks (QuickJS host cannot drive Unovis pointer events).
  selectChartType,
  selectSort,
  selectMetric,
  toggleStacked,
  toggleCumulative,
  onMetricSelect,
  onSearch,
  onRefresh,
  initMetricOptions,
  syncPlotHeight,
  xAccessor,
  xTickFormat,
  yTickFormat,
  onLineClick,
  onStackedBarClick,
  onGroupedBarClick,
  onPieSegmentClick,
  onStackedBarTooltip,
  onGroupedBarTooltip,
  onPieSegmentTooltip,
  applyHoverMarkEl,
  clearChartHover,
  highlightTooltipMark,
  findScatterPointEl,
  highlightScatterPointForTooltip,
  placeLineHoverTip,
  updateLinePlotHover,
  onXyPointerLeave,
  onViewportDismissLineHover,
  linePlotYDomain,
  normalizeDefaultGroups,
});
</script>

<!-- Unovis Tooltip mounts beside the plot; unscoped so className matches. -->
<style>
/* Bar/pie hover mark (scatter uses a painted halo; no transform scale). */
.choy-chart-view .choy-chart-mark--hover {
  filter: brightness(1.22) saturate(1.12);
}

/* Solid pie: no donut track / segment hairline (Unovis defaults). */
.choy-chart-view {
  --vis-donut-segment-stroke-width: 0px;
  --vis-donut-segment-stroke-color: transparent;
  --vis-donut-background-color: transparent;
}

/*
 * Shared tooltip chrome for VisTooltip (bar/pie) and the line floating tip.
 * Unovis reads the --vis-tooltip-* vars; floating tip needs the same paint
 * applied directly (it is not hosted by Unovis).
 */
.choy-chart-tooltip {
  --vis-tooltip-background-color: var(--choy-color-background);
  --vis-tooltip-border-color: var(--choy-color-border);
  --vis-tooltip-text-color: var(--choy-color-foreground);
  --vis-tooltip-padding: 0;
  --vis-tooltip-border-radius: calc(var(--radius) - 2px);
  --vis-tooltip-box-shadow: 0 4px 12px rgba(15, 23, 42, 0.12);
  --vis-tooltip-backdrop-filter: none;

  box-sizing: border-box;
  min-width: max-content;
  padding: 0;
  color: var(--choy-color-foreground);
  background-color: var(--choy-color-background);
  border: 1px solid var(--choy-color-border);
  border-radius: calc(var(--radius) - 2px);
  box-shadow: 0 4px 12px rgba(15, 23, 42, 0.12);
}

/* Teleported line tip: fixed to the viewport so shell overflow cannot clip it. */
.choy-chart-tooltip--floating {
  position: fixed;
  z-index: 50;
}

.choy-chart-tooltip__panel {
  display: flex;
  flex-direction: column;
  gap: 0.35rem;
  min-width: 8rem;
  padding: 0.5rem 0.65rem;
  font-size: 0.75rem;
  line-height: 1.25;
}

.choy-chart-tooltip__title {
  font-weight: 500;
  color: var(--choy-color-muted-foreground);
}

.choy-chart-tooltip__row {
  display: flex;
  align-items: center;
  gap: 0.4rem;
  color: var(--choy-color-foreground);
}

.choy-chart-tooltip__swatch {
  display: inline-block;
  width: 0.55rem;
  height: 0.55rem;
  flex-shrink: 0;
  border-radius: 2px;
}

.choy-chart-tooltip__label {
  min-width: 0;
  flex: 1 1 auto;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-weight: 600;
}

.choy-chart-tooltip__value {
  flex: 0 0 auto;
  margin-inline-start: 0.5rem;
  font-weight: 600;
  font-variant-numeric: tabular-nums;
  color: var(--choy-color-foreground);
}
</style>
