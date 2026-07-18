<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <OViewContainer :showHeader="showHeader">
    <template #header>
      <div class="o-chart__action-bar">
        <div class="o-chart__actions" v-if="showActions">
          <div class="o-chart__system-actions">
            <slot name="system-actions">
              <el-button v-if="createAction" size="small" plain type="primary" @click="handleCreate">
                <el-icon><Plus /></el-icon>
                {{ _t('New') }}
              </el-button>
              <el-button v-if="refreshAction" size="small" plain @click="handleRefresh">
                <el-icon><Refresh /></el-icon>
                {{ _t('Refresh') }}
              </el-button>
            </slot>
          </div>
          <div class="o-chart__user-actions">
            <slot name="user-actions">
              <slot name="actions" />
            </slot>
          </div>
        </div>
        <div class="o-chart__search" v-if="searchView">
          <component :is="searchView" :store="store" @query-update="onSearch" />
        </div>
        <div class="o-chart__header-right">
          <slot name="header-right" />
        </div>
      </div>
    </template>
    <slot name="fields" />
    <!-- Move controls out of the chart body, using normal document flow to avoid absolute positioning and padding-top compensation -->
    <div class="o-chart__controls" v-if="props.showChartControls && (metricOptions.length || availableChartTypes.length)">
      <label v-if="metricOptions.length" for="o-chart-metric-select" class="o-chart__metric-label">{{ _t('Metric') }}</label>
      <slot name="metric-switcher" :metrics="metricOptions" :current="currentMetricAlias" :change="selectMetric">
        <el-select
          id="o-chart-metric-select"
          v-model="currentMetricAlias"
          size="small"
          class="o-chart__control-select"
          @change="onMetricChange"
          :aria-label="_t('Metric selection')"
        >
          <el-option v-for="m in metricOptions" :key="m.alias" :label="m.label" :value="m.alias" />
        </el-select>
      </slot>
      <slot name="chart-type-switcher" :types="availableChartTypes" :current="currentChartType" :change="selectChartType">
        <el-button-group v-if="availableChartTypes.length" class="o-chart__chart-type-group">
          <el-tooltip v-for="t in availableChartTypes" :key="t" :content="t" placement="bottom">
            <el-button
              size="small"
              :type="currentChartType === t ? 'primary' : 'default'"
              :icon="chartTypeIcons[t] || DefaultChartIcon"
              @click="selectChartType(t)"
              :aria-pressed="currentChartType === t"
              class="o-chart__chart-type-btn"
            />
          </el-tooltip>
        </el-button-group>
      </slot>
      <slot name="stacked-switcher" :stacked="localStacked" :toggle="toggleStacked">
        <el-tooltip :content="_t('Stack')" placement="bottom">
          <el-button
            size="small"
            :type="stackedDisabled ? 'default' : localStacked ? 'primary' : 'default'"
            :icon="LayersOutlined"
            @click="toggleStacked"
            :aria-pressed="localStacked"
            class="o-chart__stacked-btn"
            :disabled="stackedDisabled"
          />
        </el-tooltip>
      </slot>
      <slot name="sort-switcher" :current="currentSort" :change="selectSort">
        <el-button-group v-if="showSortGroup" class="o-chart__sort-group">
          <el-tooltip :content="_t('Original order')" placement="bottom">
            <el-button
              size="small"
              :type="currentSort === 'none' ? 'primary' : 'default'"
              :icon="sortIcons.none"
              @click="selectSort('none')"
              :aria-pressed="currentSort === 'none'"
              class="o-chart__sort-btn"
            />
          </el-tooltip>
          <el-tooltip :content="_t('Ascending')" placement="bottom">
            <el-button
              size="small"
              :type="currentSort === 'asc' ? 'primary' : 'default'"
              :icon="sortIcons.asc"
              @click="selectSort('asc')"
              :aria-pressed="currentSort === 'asc'"
              class="o-chart__sort-btn"
              :disabled="sortDisabled"
            />
          </el-tooltip>
          <el-tooltip :content="_t('Descending')" placement="bottom">
            <el-button
              size="small"
              :type="currentSort === 'desc' ? 'primary' : 'default'"
              :icon="sortIcons.desc"
              @click="selectSort('desc')"
              :aria-pressed="currentSort === 'desc'"
              class="o-chart__sort-btn"
              :disabled="sortDisabled"
            />
          </el-tooltip>
        </el-button-group>
      </slot>
    </div>
    <div ref="chartWrapRef" class="o-chart__body">
      <VChart v-if="option" :option="option" :autoresize="autoResize" :update-options="{ notMerge: true }" class="o-chart__echart" @click="onChartItemClick" />
      <div v-else class="o-chart__empty">{{ _t('No data or grouping not configured') }}</div>
      <div v-if="loading" class="o-chart__overlay">{{ _t('Loading...') }}</div>
      <div v-if="error" class="o-chart__overlay error">{{ _t('Load failed') }}</div>
    </div>
  </OViewContainer>
</template>

<script setup lang="ts" generic="T extends BaseModel">
import { ref, computed, watch, onMounted, nextTick } from 'vue';
import type { WebModelStore } from '@/web/web/stores/modelStore';
import type { BaseModel } from '@/core/rpc';
import type { GroupBySpec, QueryCondition } from '@/core/service/api/query';
import type { QueryUpdatePayload } from '@/web/web/query/types';
import type { OrderByState } from '@/web/web/query/state';
import OViewContainer from '@/web/web/components/view/OViewContainer.vue';
import { createChartController } from '@/web/web/controllers/chartController';
import { awaitFieldSelection } from '@/web/web/query/utils/registry/fieldReady';
import { exportMetrics } from '@/web/web/query/utils/registry/metric';
import { resolveChartAdapter, ensureEChartsRegistered, chartTypeRegistry } from '@/web/web/components/chart/chartTypeAdapter';
import type { IChartTypeAdapter } from '@/web/web/components/chart/chartTypeAdapter';
import type { SearchViewComponent } from '@/web/web/query/types';
import OSearchView from '@/web/web/components/view/OSearchView.vue';
import { Plus, Refresh } from '@element-plus/icons-vue';
import {
  BarChartOutlined,
  ShowChartOutlined,
  PieChartOutlined,
  InsertChartOutlined,
  SortOutlined,
  ArrowUpwardOutlined,
  ArrowDownwardOutlined,
  StackedBarChartOutlined,
  LayersOutlined,
} from '@vicons/material';
import { ElMessage, ElButton, ElTooltip, ElButtonGroup, ElSelect, ElOption, ElIcon } from 'element-plus';
import VChart from 'vue-echarts';
import { createTranslate } from '@/web/web/i18n';

const { _t } = createTranslate('web', { scope: 'web/components/view/OChartView' });
ensureEChartsRegistered();

const props = withDefaults(
  defineProps<{
    store: WebModelStore<T>;
    searchView: SearchViewComponent<T> | typeof OSearchView;
    defaultGroups?: GroupBySpec<T> | GroupBySpec<T>[];
    defaultMetricAlias?: string;
    chartTypes?: string[];
    defaultChartType?: string;
    stacked?: boolean;
    stackMode?: 'absolute' | 'percent';
    palette?: string[];
    autoResize?: boolean;
    showHeader?: boolean;
    showActions?: boolean;
    createAction?: string | any;
    refreshAction?: boolean;
    forcedCondition?: any;
    orderBy?: OrderByState[];
    keywordFields?: string[];
    showChartControls?: boolean; // Controls whether the chart toolbar is shown
  }>(),
  {
    // No default chartTypes; fall back to the registry when unset
    defaultChartType: undefined,
    stacked: true,
    stackMode: 'absolute',
    autoResize: true,
    showHeader: true,
    showActions: true,
    refreshAction: true,
    forcedCondition: undefined,
    orderBy: undefined,
    keywordFields: undefined,
    showChartControls: true,
  }
);

const emit = defineEmits<{
  (e: 'action-error', payload: { action: 'load' | 'refresh' | 'search' | 'sort' | 'metric' | 'chart-type'; error: Error }): void;
  (e: 'metric-change', payload: { alias: string }): void;
  (e: 'chart-type-change', payload: { type: string }): void;
  (e: 'stacked-change', payload: { stacked: boolean }): void;
  (e: 'refresh'): void;
  (e: 'create'): void;
  (e: 'search-change'): void;
  (e: 'chart-item-click', payload: ChartItemClickPayload): void;
}>();

interface ChartItemClickPayload {
  chartType: string;
  categoryIndex?: number;
  categoryLabel?: string;
  seriesName?: string; // Composite path or single-series name
  value: number;
  metricAlias: string;
  metricLabel: string;
  groupDepth: number;
  // Path grouping details in depth order
  path: Array<{
    depth: number; // 0 for the top level, >=1 for deeper levels
    label: string;
    field?: string;
    value: any; // Raw value from labels
  }>;
  // Simplified filter conditions, possibly matching path while preserving field names
  filters: Array<{ field?: string; value: any; label: string; depth: number }>;
  // Original query condition from the aggregate row, normalized to QueryCondition<T>
  condition?: QueryCondition<T>;
}

// Controller
const store = props.store;
const controller = createChartController(store as any);

// Reactive state
const loading = computed(() => controller.vm.loading);
const error = computed(() => controller.vm.error);
const snapshot = computed(() => controller.vm.result);

// Metrics
interface MetricOption {
  alias: string;
  label: string;
  field?: string;
  agg?: string;
}
const metricOptions = ref<MetricOption[]>([]);
const currentMetricAlias = ref<string>('count');

// Chart type handling: fall back to the full registry list when unset or empty
const availableChartTypes = computed(() => {
  const registryTypes = Object.keys(chartTypeRegistry);
  const userTypes = props.chartTypes;
  if (!userTypes || userTypes.length === 0) return registryTypes;
  // Filter invalid types while keeping the user-defined order
  return userTypes.filter(t => registryTypes.includes(t));
});
const currentChartType = ref<string>(
  props.defaultChartType && availableChartTypes.value.includes(props.defaultChartType) ? props.defaultChartType : availableChartTypes.value[0]
);

// Chart type to icon mapping, overrideable via slots
const DefaultChartIcon = InsertChartOutlined;
const chartTypeIcons: Record<string, any> = {
  bar: BarChartOutlined || DefaultChartIcon,
  line: ShowChartOutlined || DefaultChartIcon,
  pie: PieChartOutlined || DefaultChartIcon,
};

// Local stacked state, replacing direct use of props.stacked so users can toggle it interactively
const localStacked = ref<boolean>(!!props.stacked);

// Client-side sort state, meaningful only for bar and line charts based on Y-axis values
type SortState = 'none' | 'asc' | 'desc';
const currentSort = ref<SortState>('none');
const sortIcons: Record<SortState, any> = {
  none: SortOutlined,
  asc: ArrowUpwardOutlined,
  desc: ArrowDownwardOutlined,
};
// Whether to show the sort button group; pie charts can skip sorting because multi-series slices are not sorted
const showSortGroup = computed(() => categories.value.length > 0);
// Disable sorting for pie charts because there is no Y-axis concept
const sortDisabled = computed(() => currentChartType.value === 'pie');
// Disable the stacked toggle unless the chart is bar/line with second-level grouping (multiple series)
const stackedDisabled = computed(() => {
  const typeOk = currentChartType.value === 'bar' || currentChartType.value === 'line';
  const hasSecondLevel = seriesMatrix.value.length > 1; // Multiple series imply second-level grouping
  return !(typeOk && hasSecondLevel);
});

// Data mapping
const categories = ref<string[]>([]);
const seriesMatrix = ref<Array<{ name: string; data: number[] }>>([]);
const groupDepth = ref<number>(1);
const seriesCount = ref<number>(1);
const option = ref<any>(null);
const chartWrapRef = ref<HTMLElement | null>(null);
// Top-level raw rows and composite-path metadata
const topRows = ref<any[]>([]);
// Composite path to metadata, with field and label on each deep node
const comboMeta = ref<Map<string, Array<{ depth: number; label: string; field?: string; value: any }>>>(new Map());
// For each top-level row, map composite paths to conditions; values may be OR-merged and are normalized on click
const perParentComboConditions = ref<Array<Map<string, any>>>([]);
// Debug logging has been removed; add a temporary dlog helper if needed again.

// Save sort baselines so the original order can be restored
const originalCategories = ref<string[]>([]);
const originalSeriesMatrix = ref<Array<{ name: string; data: number[] }>>([]);
const originalTopRows = ref<any[]>([]);
const originalPerParentComboConditions = ref<Array<Map<string, any>>>([]);

// Build metric options once fields registered & after first snapshot
function initMetricOptions() {
  const metas = exportMetrics(store.storeId) || [];
  const arr: MetricOption[] = metas.map(m => ({
    alias: m.alias || `${m.field}_${m.agg}`,
    label: m.alias || `${m.field}:${m.agg}`,
    field: m.field,
    agg: m.agg,
  }));
  // Keep only primary aliases in the dropdown; metricValue still falls back to double-underscore aliases internally
  if (!arr.find(m => m.alias === 'count')) arr.unshift({ alias: 'count', label: 'Count' });
  metricOptions.value = arr;
  if (props.defaultMetricAlias && arr.find(m => m.alias === props.defaultMetricAlias)) {
    currentMetricAlias.value = props.defaultMetricAlias;
  } else if (!arr.find(m => m.alias === currentMetricAlias.value)) {
    currentMetricAlias.value = arr[0]?.alias || 'count';
  }
}

function metricValue(row: any, alias: string): number {
  if (!row) return 0;
  if (alias === 'count') return Number(row.count ?? row.metrics?.__count ?? row.metrics?.count ?? 0);
  // Support alias mapping with single or double underscores
  let v = row.metrics?.[alias];
  if (v == null && alias.includes('_') && !alias.includes('__')) {
    const doubleAlias = alias.replace(/_/g, '__');
    v = row.metrics?.[doubleAlias];
  } else if (v == null && alias.includes('__')) {
    const singleAlias = alias.replace(/__/g, '_');
    v = row.metrics?.[singleAlias];
  }
  return typeof v === 'number' ? v : Number(v ?? 0);
}

function wrapToQueryCondition(raw: any): QueryCondition<T> | undefined {
  if (!raw) return undefined;
  // Single primitive condition: [field, op, value]
  if (Array.isArray(raw) && raw.length === 3 && typeof raw[0] === 'string') {
    const [f, o, v] = raw as [string, any, any];
    return { [f]: [f, o, v] } as any;
  }
  // Recursively normalize Or/And structures
  if (raw.Or && Array.isArray(raw.Or)) {
    const orConds = raw.Or.map((c: any) => wrapToQueryCondition(c)).filter(Boolean) as QueryCondition<T>[];
    return { Or: orConds } as any;
  }
  if (raw.And && Array.isArray(raw.And)) {
    const andConds = raw.And.map((c: any) => wrapToQueryCondition(c)).filter(Boolean) as QueryCondition<T>[];
    return { And: andConds } as any;
  }
  // Object form (field: [..]) is treated as SingleCondition<T>
  if (typeof raw === 'object') return raw as QueryCondition<T>;
  return undefined;
}

function mergeOr(a: QueryCondition<T> | undefined, b: QueryCondition<T> | undefined): QueryCondition<T> | undefined {
  if (!a) return b;
  if (!b) return a;
  // If a or b is an Or node, expand and merge the lists
  const toList = (c: QueryCondition<T>): QueryCondition<T>[] => ((c as any).Or && Array.isArray((c as any).Or) ? (c as any).Or : [c]);
  const merged = [...toList(a), ...toList(b)];
  return { Or: merged } as any;
}

function rebuildData() {
  const snap = snapshot.value;
  if (!snap || snap.kind !== 'group') {
    categories.value = [];
    seriesMatrix.value = [];
    groupDepth.value = 1;
    seriesCount.value = 0;
    option.value = null;
    return;
  }
  const top: any[] = (snap.rows || []).filter((r: any) => Number((r as any).depth) === 0);
  topRows.value = top;
  comboMeta.value = new Map();
  perParentComboConditions.value = [];
  const labelOf = (row: any): string => {
    if (row.labels) {
      const k = Object.keys(row.labels)[0];
      return String(row.labels[k]);
    }
    return String(row.label ?? row.key);
  };
  categories.value = top.map(r => labelOf(r));
  const metricAlias = currentMetricAlias.value;

  // Collect every composite path from depth=1 to leaf nodes, merging the second level and beyond
  const globalComboSet = new Set<string>();
  const perParentValueMaps: Array<Map<string, number>> = []; // Aligned with top row order

  const joiner = ' / ';
  const buildValueMap = (parent: any, parentIndex: number): Map<string, number> => {
    const map = new Map<string, number>();
    const condMap = new Map<string, any>();
    const children = parent.children;
    if (!Array.isArray(children) || !children.length) return map;
    const dfs = (node: any, parts: string[], metaParts: Array<{ depth: number; label: string; field?: string; value: any }>) => {
      const kids = node.children;
      if (Array.isArray(kids) && kids.length) {
        for (const ch of kids) {
          const lblObj = ch.labels || {};
          const field = Object.keys(lblObj)[0];
          dfs(
            ch,
            [...parts, labelOf(ch)],
            [
              ...metaParts,
              {
                depth: Number(ch.depth ?? metaParts.length + 1),
                label: labelOf(ch),
                field,
                value: field ? lblObj[field] : labelOf(ch),
              },
            ]
          );
        }
      } else {
        if (parts.length) {
          const key = parts.join(joiner);
          const prev = map.get(key) || 0;
          map.set(key, prev + metricValue(node, metricAlias));
          if (!comboMeta.value.has(key)) {
            comboMeta.value.set(key, metaParts);
          }
          // Merge conditions with Or when multiple leaves aggregate to the same key
          const leafCondRaw = node.__condition;
          const leafCond = wrapToQueryCondition(leafCondRaw);
          if (leafCond) {
            const prev = condMap.get(key);
            condMap.set(key, mergeOr(prev, leafCond)!);
          }
        }
      }
    };
    for (const ch of children) {
      const lblObj = ch.labels || {};
      const field = Object.keys(lblObj)[0];
      dfs(
        ch,
        [labelOf(ch)],
        [
          {
            depth: Number(ch.depth ?? 1),
            label: labelOf(ch),
            field,
            value: field ? lblObj[field] : labelOf(ch),
          },
        ]
      ); // Start from depth=1
    }
    perParentComboConditions.value[parentIndex] = condMap as any;
    return map;
  };

  for (let i = 0; i < top.length; i++) {
    const p = top[i];
    const vm = buildValueMap(p, i);
    perParentValueMaps.push(vm);
    for (const key of vm.keys()) globalComboSet.add(key);
  }

  const comboList = Array.from(globalComboSet);

  if (!comboList.length) {
    // If there is only one level, or children under depth=1 have no leaves, fall back to a single series
    seriesMatrix.value = [
      {
        name: metricOptions.value.find(m => m.alias === metricAlias)?.label || metricAlias,
        data: top.map(r => metricValue(r, metricAlias)),
      },
    ];
    groupDepth.value = 1;
    seriesCount.value = 1;
    // Save baselines
    originalCategories.value = [...categories.value];
    originalSeriesMatrix.value = seriesMatrix.value.map(s => ({ name: s.name, data: [...s.data] }));
    originalTopRows.value = [...topRows.value];
    originalPerParentComboConditions.value = [...perParentComboConditions.value];
    applySorting();
    buildChartOption();
    return;
  }

  // Composite-path series matrix
  const matrices: Array<{ name: string; data: number[] }> = comboList.map(name => ({ name, data: [] }));
  for (let i = 0; i < top.length; i++) {
    const vm = perParentValueMaps[i];
    for (const series of matrices) {
      series.data.push(vm.get(series.name) ?? 0);
    }
  }
  seriesMatrix.value = matrices;
  seriesCount.value = matrices.length;
  groupDepth.value = 2; // Still visually two-dimensional: X axis plus composite series
  // Save baselines
  originalCategories.value = [...categories.value];
  originalSeriesMatrix.value = seriesMatrix.value.map(s => ({ name: s.name, data: [...s.data] }));
  originalTopRows.value = [...topRows.value];
  originalPerParentComboConditions.value = [...perParentComboConditions.value];
  applySorting();
  buildChartOption();
}

function applySorting() {
  if (!originalCategories.value.length) return;
  if (currentSort.value === 'none' || sortDisabled.value) {
    categories.value = [...originalCategories.value];
    seriesMatrix.value = originalSeriesMatrix.value.map(s => ({ name: s.name, data: [...s.data] }));
    topRows.value = [...originalTopRows.value];
    perParentComboConditions.value = [...originalPerParentComboConditions.value];
    return;
  }
  // Use Y-axis values as the sort base: direct value for one series, sum for multiple series
  const baseSeries = originalSeriesMatrix.value;
  const compositeValues = originalCategories.value.map((_, idx) => {
    if (baseSeries.length === 1) return baseSeries[0].data[idx] ?? 0;
    return baseSeries.reduce((sum, s) => sum + (s.data[idx] ?? 0), 0);
  });
  const indices = originalCategories.value
    .map((_, i) => i)
    .sort((a, b) => (currentSort.value === 'asc' ? compositeValues[a] - compositeValues[b] : compositeValues[b] - compositeValues[a]));
  categories.value = indices.map(i => originalCategories.value[i]);
  topRows.value = indices.map(i => originalTopRows.value[i]);
  perParentComboConditions.value = indices.map(i => originalPerParentComboConditions.value[i]);
  seriesMatrix.value = baseSeries.map(s => ({ name: s.name, data: indices.map(i => s.data[i]) }));
}

function onChartItemClick(params: any) {
  if (!params) return;
  const metricLabel = metricOptions.value.find(m => m.alias === currentMetricAlias.value)?.label || currentMetricAlias.value;
  const metricAlias = currentMetricAlias.value;
  const chartType = currentChartType.value;
  const path: Array<{ depth: number; label: string; field?: string; value: any }> = [];
  const filters: Array<{ field?: string; value: any; label: string; depth: number }> = [];
  let categoryIndex: number | undefined;
  let categoryLabel: string | undefined;
  let seriesName: string | undefined;
  let value = 0;
  let condition: QueryCondition<T> | undefined;

  if (chartType === 'pie') {
    // params.name is the slice name: top-level category for a single series, composite path for multiple series
    seriesName = params.name;
    value = Number(params.value ?? 0);
    if (seriesMatrix.value.length <= 1) {
      // Single series: top-level category
      categoryLabel = seriesName;
      categoryIndex = categories.value.indexOf(categoryLabel || '');
      if (categoryIndex >= 0) {
        const topRow = topRows.value[categoryIndex];
        const lblObj = topRow?.labels || {};
        const field = Object.keys(lblObj)[0];
        const val = field ? lblObj[field] : categoryLabel;
        path.push({ depth: 0, label: categoryLabel || '', field, value: val });
        filters.push({ depth: 0, label: categoryLabel || '', field, value: val });
        condition = wrapToQueryCondition(topRow?.__condition) ?? undefined;
      }
    } else {
      // Multiple series: composite path from the second level onward
      const meta = comboMeta.value.get(seriesName || '') || [];
      for (const m of meta) {
        path.push(m);
        filters.push({ ...m });
      }
      // Aggregate conditions for this composite path across all top-level groups with Or
      const orRaw: any[] = [];
      for (let i = 0; i < perParentComboConditions.value.length; i++) {
        const cond = perParentComboConditions.value[i]?.get(seriesName || '');
        if (cond) orRaw.push(cond);
      }
      const orConds = orRaw.map(c => wrapToQueryCondition(c)).filter(Boolean) as QueryCondition<T>[];
      if (orConds.length === 1) condition = orConds[0];
      else if (orConds.length > 1) condition = { Or: orConds } as any;
    }
  } else {
    // bar / line
    const catIdx = typeof params.dataIndex === 'number' ? params.dataIndex : -1;
    categoryIndex = catIdx >= 0 ? catIdx : undefined;
    categoryLabel = catIdx >= 0 ? categories.value[catIdx] : String(params.name ?? params.axisValue ?? '');
    value = Number(params.value ?? params.data?.value ?? params.data ?? 0);
    const topRow = categoryIndex != null ? topRows.value[categoryIndex] : undefined;
    if (topRow && categoryLabel) {
      const lblObj = topRow.labels || {};
      const field = Object.keys(lblObj)[0];
      const val = field ? lblObj[field] : categoryLabel;
      path.push({ depth: 0, label: categoryLabel || '', field, value: val });
      filters.push({ depth: 0, label: categoryLabel || '', field, value: val });
      condition = wrapToQueryCondition(topRow?.__condition) ?? undefined;
    }
    if (seriesMatrix.value.length > 1 && params.seriesName) {
      seriesName = params.seriesName;
      const meta = comboMeta.value.get(seriesName || '') || [];
      for (const m of meta) {
        path.push(m);
        filters.push({ ...m });
      }
      const raw = categoryIndex != null ? perParentComboConditions.value[categoryIndex]?.get(seriesName || '') : undefined;
      const norm = wrapToQueryCondition(raw);
      if (norm) condition = norm; // Condition for the composite path under a single top-level group
    }
  }

  emit('chart-item-click', {
    chartType,
    categoryIndex,
    categoryLabel,
    seriesName,
    value,
    metricAlias,
    metricLabel,
    groupDepth: groupDepth.value,
    path,
    filters,
    condition,
  });
}

function buildChartOption() {
  if (!categories.value.length || !seriesMatrix.value.length) {
    option.value = null;
    return;
  }
  // Resolve adapter & fallback if unsupported
  let adapter: IChartTypeAdapter | undefined = resolveChartAdapter(currentChartType.value);
  const supportCtx = {
    groupDepth: groupDepth.value,
    stacked: !!localStacked.value,
    seriesCount: seriesCount.value,
    metricAlias: currentMetricAlias.value,
  };
  if (!adapter || !adapter.supports(supportCtx)) {
    // find first supported fallback
    adapter = availableChartTypes.value.map(t => resolveChartAdapter(t)).find(a => a && a.supports(supportCtx));
    if (!adapter) {
      option.value = null;
      return;
    }
    currentChartType.value = adapter.id;
  }
  const metricLabel = metricOptions.value.find(m => m.alias === currentMetricAlias.value)?.label || currentMetricAlias.value;
  let matrix = seriesMatrix.value;
  let percent = false;
  // Do not convert pie charts to stacked percentages; keep raw values for merging
  if (adapter.id !== 'pie' && localStacked.value && props.stackMode === 'percent' && matrix.length > 1) {
    // Percent stacking: sum each category and normalize to percentages in the 0-100 range
    const totals = categories.value.map((_, idx) => matrix.reduce((sum, s) => sum + (s.data[idx] ?? 0), 0));
    matrix = matrix.map(s => ({
      name: s.name,
      data: s.data.map((v, i) => (totals[i] ? +((v / totals[i]) * 100).toFixed(2) : 0)),
    }));
    percent = true;
  }
  option.value = adapter.buildOption({
    categories: categories.value,
    seriesMatrix: matrix,
    metricLabel: percent ? `${metricLabel} (%)` : metricLabel,
    stacked: !!localStacked.value,
    palette: props.palette,
    percent,
  });
}

function onMetricChange() {
  emit('metric-change', { alias: currentMetricAlias.value });
  rebuildData();
  buildChartOption();
}
function selectMetric(a: string) {
  currentMetricAlias.value = a;
  onMetricChange();
}
function onChartTypeChange() {
  emit('chart-type-change', { type: currentChartType.value });
  buildChartOption();
}
function selectChartType(t: string) {
  currentChartType.value = t;
  onChartTypeChange();
}

function toggleStacked() {
  if (stackedDisabled.value) return; // Guard: do not toggle while disabled
  localStacked.value = !localStacked.value;
  emit('stacked-change', { stacked: localStacked.value });
  buildChartOption();
}

function selectSort(s: SortState) {
  currentSort.value = s;
  applySorting();
  buildChartOption();
}

async function handleRefresh() {
  emit('refresh');
  try {
    await controller.apply({
      appliedGroups: (store.state as any)?.queryState?.appliedGroups as any,
      defaultGroups: props.defaultGroups ? (Array.isArray(props.defaultGroups) ? props.defaultGroups : [props.defaultGroups]) : undefined,
      keyword: (store.state as any)?.queryState?.keyword,
      keywordFields: props.keywordFields,
      forcedCondition: props.forcedCondition,
      orderBy: props.orderBy,
    });
    rebuildData();
  } catch (e) {
    const err = e instanceof Error ? e : new Error(String(e));
    emit('action-error', { action: 'refresh', error: err });
    ElMessage.error(_t('Chart refresh failed'));
  }
}

async function handleCreate() {
  if (!props.createAction) return;
  try {
    // External route navigation follows the other views; this branch only does a simple push
    // Because router generic typing is not imported here, fall back to any
    const r: any = await import('vue-router').then(m => m.useRouter()).catch(() => null);
    if (r) await r.push(props.createAction as any);
    emit('create');
  } catch (e) {
    const err = e instanceof Error ? e : new Error(String(e));
    emit('action-error', { action: 'load', error: err });
  }
}

function onSearch(payload: QueryUpdatePayload<T>) {
  emit('search-change');
  controller
    .apply({
      appliedGroups: payload.appliedGroups as any,
      defaultGroups: props.defaultGroups ? (Array.isArray(props.defaultGroups) ? props.defaultGroups : [props.defaultGroups]) : undefined,
      keyword: payload.keyword,
      keywordFields: props.keywordFields,
      forcedCondition: props.forcedCondition,
      orderBy: props.orderBy,
    })
    .then(() => {
      rebuildData();
    })
    .catch(e => {
      const err = e instanceof Error ? e : new Error(String(e));
      emit('action-error', { action: 'search', error: err });
    });
}

onMounted(async () => {
  const qs: any = (store.state as any)?.queryState || {};
  if (props.orderBy) qs.orderBy = props.orderBy;
  // Wait for field registration before extracting metrics
  await awaitFieldSelection(store, { requireNonEmpty: false });
  initMetricOptions();
  // Remove the explicit apply call and rely entirely on OSearchView query-update for first-frame loading
  // await controller.apply({ ... });
  rebuildData();
  buildChartOption();
  await nextTick();
});

watch(
  () => snapshot.value?.planHash,
  () => {
    rebuildData();
  }
);
watch(currentMetricAlias, () => buildChartOption());
watch(currentChartType, () => buildChartOption());
watch(currentSort, () => {
  applySorting();
  buildChartOption();
});
watch(localStacked, () => buildChartOption());
</script>

<style scoped lang="scss">
.o-chart__action-bar {
  display: grid;
  grid-template-columns: auto 1fr auto;
  align-items: center;
  gap: 12px;
  padding-bottom: 4px;
  border-bottom: 1px solid var(--el-border-color-light);
  min-height: 40px;
}
.o-chart__actions {
  display: flex;
  align-items: center;
  gap: 16px;
}
.o-chart__system-actions {
  display: flex;
  align-items: center;
  gap: 8px;
}
.o-chart__user-actions {
  display: flex;
  align-items: center;
  gap: 8px;
}
.o-chart__search {
  display: flex;
  justify-content: center;
  align-items: center;
  min-width: 240px;
}
.o-chart__header-right {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: 8px;
}
.o-chart__body {
  position: relative;
  width: 100%;
  height: 100%;
  min-height: 260px;
}
.o-chart__echart {
  width: 100%;
  height: 100%;
  min-height: 260px;
}
.o-chart__controls {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 4px 6px 6px 6px;
  flex-wrap: nowrap;
  background: var(--el-color-white);
  border-bottom: 1px dashed var(--el-border-color-light);
  margin-bottom: 4px;
}
.o-chart__control-select {
  min-width: 140px;
  width: 160px;
  max-width: 220px;
  flex: 0 0 160px; /* Avoid stretching to fill the flex row */
}
.o-chart__metric-label {
  font-size: 12px;
  color: var(--el-text-color-secondary);
  user-select: none;
  padding-left: 2px;
  padding-right: 4px;
  line-height: 1;
  white-space: nowrap;
  word-break: keep-all;
  display: inline-flex;
  align-items: center;
}
.o-chart__chart-type-group {
  display: inline-flex;
  flex-wrap: nowrap;
  white-space: nowrap;
}
.o-chart__chart-type-btn {
  min-width: 40px; // Widen the button to give the icon more visual weight
  padding: 6px; // Increase padding
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 18px; // vicons use 1em sizing, so increasing font-size enlarges the icon
}
.o-chart__stacked-btn {
  min-width: 40px;
  padding: 6px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  margin-left: 4px;
  font-size: 18px;
}
.o-chart__sort-group {
  display: inline-flex;
  flex-wrap: nowrap;
  white-space: nowrap;
  margin-left: 4px;
}
.o-chart__sort-btn {
  min-width: 40px;
  padding: 6px;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 18px;
}
/* Ensure SVG icons scale with font size as well; scoped styles require :deep */
.o-chart__chart-type-btn :deep(svg),
.o-chart__stacked-btn :deep(svg),
.o-chart__sort-btn :deep(svg) {
  width: 1.15em;
  height: 1.15em;
}
.o-chart__empty {
  width: 100%;
  height: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--el-text-color-secondary);
  font-size: 13px;
}
.o-chart__overlay {
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 13px;
  background: rgba(255, 255, 255, 0.6);
  backdrop-filter: blur(2px);
}
.o-chart__overlay.error {
  color: #d03050;
}
@media (max-width: 768px) {
  .o-chart__action-bar {
    grid-template-columns: 1fr;
    grid-auto-rows: auto;
  }
  .o-chart__search {
    order: 2;
  }
}
</style>
