<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <div class="choy-dogfood-company choy-gallery-token-scope min-h-full bg-background text-foreground">
    <ChoyPage title="Dogfood Company" width="wide">
      <ChoyTabs v-model="activeTab">
        <ChoyTab value="form" label="Form">
          <ChoyCard class="mt-4">
            <GalleryFormShell title="Company" :loading="formLoading">
              <template #breadcrumb>
                <ChoyBreadcrumb
                  :items="[
                    { label: 'Gallery', to: { name: 'ChoyUiGallery' } },
                    { label: 'Dogfood Company' },
                  ]"
                />
              </template>
              <template #statusbar>
                <div
                  class="choy-statusbar-field flex flex-wrap gap-1"
                  role="group"
                  aria-label="State"
                >
                  <ChoyButton
                    v-for="opt in STATE_OPTIONS"
                    :key="opt.value"
                    type="button"
                    size="sm"
                    :variant="state === opt.value ? 'default' : 'outline'"
                    :aria-pressed="state === opt.value"
                    @click="state = opt.value"
                  >
                    {{ opt.label }}
                  </ChoyButton>
                </div>
              </template>
              <template #system-actions>
                <ChoyButton
                  size="sm"
                  type="button"
                  :disabled="formLoading"
                  @click="onSave"
                >
                  Save
                </ChoyButton>
              </template>
              <template #button-box>
                <ChoyButton size="sm" variant="ghost" type="button" @click="activeTab = 'list'">
                  Open list
                </ChoyButton>
              </template>

              <ChoyGrid :cols="12" class="gap-4">
                <ChoyCol :span="6">
                  <ChoyFieldBase
                    label="Name"
                    name="name"
                    required
                    help="Legal or trading name."
                  >
                    <template #default="{ controlId, ariaInvalid, ariaRequired, ariaDescribedby }">
                      <Input
                        :id="controlId"
                        v-model="name"
                        name="name"
                        :aria-invalid="ariaInvalid"
                        :aria-required="ariaRequired"
                        :aria-describedby="ariaDescribedby"
                      />
                    </template>
                  </ChoyFieldBase>
                </ChoyCol>
                <ChoyCol :span="6">
                  <ChoyManyToOneRefField
                    v-model="currencyId"
                    label="Currency"
                    name="currency_id"
                    search-key="dogfood.currency"
                    :search="searchCurrencies"
                    :selected-option="currencyOption"
                    @select="onCurrencySelect"
                  />
                </ChoyCol>
                <ChoyCol :span="4">
                  <ChoyFieldBase label="Founded" name="founded">
                    <template #default="{ controlId, ariaInvalid, ariaRequired, ariaDescribedby }">
                      <DatePicker
                        :id="controlId"
                        v-model="founded"
                        placeholder="Pick a date"
                        clearable
                        :aria-invalid="ariaInvalid"
                        :aria-required="ariaRequired"
                        :aria-describedby="ariaDescribedby"
                      />
                    </template>
                  </ChoyFieldBase>
                </ChoyCol>
                <ChoyCol :span="4">
                  <ChoyFieldBase label="Type" name="partner_type">
                    <template #default="{ controlId, ariaInvalid, ariaRequired, ariaDescribedby }">
                      <Select v-model="partnerType">
                        <SelectTrigger
                          :id="controlId"
                          placeholder="Select type"
                          :aria-invalid="ariaInvalid"
                          :aria-required="ariaRequired"
                          :aria-describedby="ariaDescribedby"
                        />
                        <SelectContent>
                          <SelectItem
                            v-for="opt in TYPE_OPTIONS"
                            :key="opt.value"
                            :value="opt.value"
                          >
                            {{ opt.label }}
                          </SelectItem>
                        </SelectContent>
                      </Select>
                    </template>
                  </ChoyFieldBase>
                </ChoyCol>
                <ChoyCol :span="4">
                  <ChoyFieldBase label="Active" name="active">
                    <template #default="{ controlId, ariaInvalid, ariaRequired, ariaDescribedby }">
                      <Switch
                        :id="controlId"
                        v-model="active"
                        :aria-invalid="ariaInvalid"
                        :aria-required="ariaRequired"
                        :aria-describedby="ariaDescribedby"
                      />
                    </template>
                  </ChoyFieldBase>
                </ChoyCol>
                <ChoyCol :span="6">
                  <ChoyFieldBase
                    label="Share capital"
                    name="capital"
                    :error="capitalInvalidDraft ? 'Invalid amount' : ''"
                  >
                    <template #default="{ controlId, ariaInvalid, ariaRequired, ariaDescribedby }">
                      <Input
                        :id="controlId"
                        :model-value="capitalDisplay"
                        name="capital"
                        inputmode="decimal"
                        :aria-invalid="ariaInvalid || capitalInvalidDraft || undefined"
                        :aria-required="ariaRequired"
                        :aria-describedby="ariaDescribedby"
                        @update:model-value="onCapitalInput"
                        @focus="onCapitalFocus"
                        @blur="onCapitalBlur"
                        @keydown="onCapitalKeydown"
                      />
                    </template>
                  </ChoyFieldBase>
                </ChoyCol>
                <ChoyCol :span="12">
                  <ChoyFieldBase label="Notes" name="notes">
                    <template #default="{ controlId, ariaInvalid, ariaRequired, ariaDescribedby }">
                      <Textarea
                        :id="controlId"
                        v-model="notes"
                        name="notes"
                        :rows="4"
                        :aria-invalid="ariaInvalid"
                        :aria-required="ariaRequired"
                        :aria-describedby="ariaDescribedby"
                      />
                    </template>
                  </ChoyFieldBase>
                </ChoyCol>
                <ChoyCol :span="12">
                  <ChoyHtmlField v-model="htmlNotes" label="HTML notes" name="html_notes" />
                </ChoyCol>
                <ChoyCol :span="6">
                  <ChoyFieldBase label="Meta JSON" name="meta" :error="metaJsonParseError">
                    <template #default="{ controlId, ariaInvalid, ariaRequired, ariaDescribedby }">
                      <Textarea
                        :id="controlId"
                        v-model="metaJsonDraft"
                        name="meta"
                        :rows="8"
                        class="font-mono text-xs"
                        :aria-invalid="ariaInvalid || !!metaJsonParseError || undefined"
                        :aria-required="ariaRequired"
                        :aria-describedby="ariaDescribedby"
                        @blur="commitMetaJsonDraft"
                      />
                    </template>
                  </ChoyFieldBase>
                </ChoyCol>
                <ChoyCol :span="6">
                  <ChoyFieldBase label="Properties" name="properties">
                    <template #default>
                      <div class="flex flex-col gap-3">
                        <div
                          v-for="item in renderablePropItems"
                          :key="item.name"
                          class="flex flex-col gap-1"
                        >
                          <label class="text-sm font-medium text-foreground">
                            {{ item.string || item.name }}
                          </label>
                          <Input
                            v-if="item.type === 'char'"
                            :model-value="propAsString(readPropValue(item.name))"
                            @update:model-value="setPropValue(item.name, $event)"
                          />
                          <Select
                            v-else-if="item.type === 'selection'"
                            :model-value="(readPropValue(item.name) as string | null) ?? null"
                            @update:model-value="setPropValue(item.name, $event)"
                          >
                            <SelectTrigger class="w-full" placeholder="Select…" />
                            <SelectContent>
                              <SelectItem
                                v-for="opt in normalizeSelectionOptions(item.selection)"
                                :key="opt.value"
                                :value="opt.value"
                              >
                                {{ opt.label }}
                              </SelectItem>
                            </SelectContent>
                          </Select>
                        </div>
                      </div>
                    </template>
                  </ChoyFieldBase>
                </ChoyCol>
                <ChoyCol :span="12">
                  <ChoyOneToManyField
                    v-model="contactLines"
                    label="Contacts"
                    :columns="contactColumns"
                    title-field="Title"
                    subtitle-field="Role"
                  />
                </ChoyCol>
                <ChoyCol :span="12">
                  <ChoyManyToManyField
                    v-model="tagIds"
                    label="Currency tags"
                    search-key="dogfood.currency.tags"
                    :search="searchCurrencies"
                    :options="CURRENCIES"
                  />
                </ChoyCol>
              </ChoyGrid>
            </GalleryFormShell>
          </ChoyCard>
        </ChoyTab>

        <ChoyTab value="kanban" label="Kanban">
          <ChoyCard class="mt-4" title="Company pipeline">
            <ChoyKanbanView v-model:lanes="kanbanLanes" />
          </ChoyCard>
        </ChoyTab>

        <ChoyTab value="list" label="List">
          <ChoyCard class="mt-4" title="Companies">
                        <div class="flex flex-col gap-3">
              <div class="flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
                <p class="text-sm text-foreground/70">
                  Selected {{ listSelection.length }} · showing {{ pageRows.length }} of
                  {{ total }}
                </p>
                <ChoySearchView
                  v-model:keyword="searchKeyword"
                  placeholder="Filter by name or country…"
                  @query-update="onSearch"
                />
              </div>
              <DataTable
                v-model:row-selection="listSelection"
                :columns="listColumns"
                :data="pageRows"
                :row-id="companyRowId"
                :height="280"
                @row-click="onRowClick"
              />
            </div>
            <div class="mt-3 flex justify-end">
              <ChoyPagination v-model:page="page" v-model:page-size="pageSize" :total="total" />
            </div>
          </ChoyCard>
        </ChoyTab>
      </ChoyTabs>

      <template #footer>
        <p class="text-sm text-foreground/70">
          Isolation dogfood — Form/List/Search/Kanban + field set (incl. Html/Json/Properties/O2M/M2M)
          on local mocks; no Element Plus / no RPC.
        </p>
      </template>
    </ChoyPage>
    <Toaster />
  </div>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, ref, watch } from 'vue';
import type { ColumnDef } from '@tanstack/vue-table';
import ChoyFieldBase from '../components/field/ChoyFieldBase.vue';
import ChoyHtmlField from '../components/field/ChoyHtmlField.vue';
import ChoyManyToManyField from '../components/field/ChoyManyToManyField.vue';
import ChoyManyToOneField from '../components/field/ChoyManyToOneField.vue';
import ChoyOneToManyField from '../components/field/ChoyOneToManyField.vue';
import Input from '../components/vendor/ui/input/Input.vue';
import Textarea from '../components/vendor/ui/textarea/Textarea.vue';
import Switch from '../components/vendor/ui/switch/Switch.vue';
import Select from '../components/vendor/ui/select/Select.vue';
import SelectContent from '../components/vendor/ui/select/SelectContent.vue';
import SelectItem from '../components/vendor/ui/select/SelectItem.vue';
import SelectTrigger from '../components/vendor/ui/select/SelectTrigger.vue';
import type { RelationOption } from '../components/internal/relationComboboxHelpers';
import {
  normalizeChoyJsonIncoming,
  stringifyChoyJson,
  tryParseChoyJson,
  type ChoyJsonValue
} from '../components/field/jsonFieldHelpers';
import {
  filterRenderablePropertyItems,
  normalizeSelectionOptions,
  writePropertyValue,
  type PropertiesMap
} from '../components/field/propertiesHelpers';
import {
  formatChoyMonetary,
  parseChoyNumber,
  resolveChoyMonetaryPrecision,
  roundChoyDecimal
} from '../components/field/fieldHelpers';
import ChoyButton from '../components/layout/ChoyButton.vue';
import ChoyCard from '../components/layout/ChoyCard.vue';
import ChoyCol from '../components/layout/ChoyCol.vue';
import ChoyGrid from '../components/layout/ChoyGrid.vue';
import ChoyPage from '../components/layout/ChoyPage.vue';
import ChoyTab from '../components/layout/ChoyTab.vue';
import ChoyTabs from '../components/layout/ChoyTabs.vue';
import Toaster from '../components/vendor/ui/toast/Toaster.vue';
import ChoyBreadcrumb from '../components/view/ChoyBreadcrumb.vue';
import GalleryFormShell from './GalleryFormShell.vue';
import ChoyKanbanView from '../components/view/ChoyKanbanView.vue';
import DataTable from '../components/internal/DataTable.vue';
import DatePicker from '../components/internal/DatePicker.vue';
import ChoyPagination from '../components/view/ChoyPagination.vue';
import ChoySearchView from '../components/view/ChoySearchView.vue';
import {
  groupRowsIntoChoyKanbanLanes,
  type ChoyKanbanLane
} from '../components/view/kanbanViewHelpers';
import {
  choyPageOffset,
  choyTotalPages,
  clampChoyPage
} from '../components/view/paginationHelpers';
import {
  filterRowsByKeyword,
  type ChoySearchQuery
} from '../components/view/searchViewHelpers';
import { ChoyMessage } from '../composables/useChoyMessage';

/**
 * Dogfood company form: Form + fields + List/Search/Kanban against local mocks.
 * No RPC — exercises the PR5/PR6 main path in isolation.
 */

type CompanyRow = {
  Id: string;
  name: string;
  country: string;
  active: boolean;
};

const CURRENCIES: RelationOption[] = [
  { id: 'usd', label: 'USD — US Dollar' },
  { id: 'eur', label: 'EUR — Euro' },
  { id: 'cny', label: 'CNY — Chinese Yuan' },
];

const STATE_OPTIONS = [
  { value: 'draft', label: 'Draft' },
  { value: 'confirmed', label: 'Confirmed' },
  { value: 'done', label: 'Done' },
];

const TYPE_OPTIONS = [
  { value: 'company', label: 'Company' },
  { value: 'individual', label: 'Individual' },
];

const allRows = ref<CompanyRow[]>(
  Array.from({ length: 37 }, (_, i) => ({
    Id: `c${i + 1}`,
    name: `Acme ${i + 1}`,
    country: i % 3 === 0 ? 'US' : i % 3 === 1 ? 'DE' : 'CN',
    active: i % 4 !== 0,
  })),
);

const activeTab = ref('form');
const formLoading = ref(false);
const selectedRowId = ref<string | null>(null);
let saveTimer: ReturnType<typeof setTimeout> | null = null;

const name = ref('Acme Holdings');
const notes = ref('Isolation dogfood company record.');
const capital = ref<number | null>(1_250_000.5);
const active = ref(true);
const partnerType = ref<string | null>('company');
const state = ref('draft');
const founded = ref<string | null>('2020-03-15');
const currencyId = ref<string | null>('usd');
const currencyOption = ref<RelationOption | null>(CURRENCIES[0] ?? null);
const htmlNotes = ref<string | null>('<p>Dogfood <em>HTML</em> notes.</p>');
const metaJson = ref<ChoyJsonValue>({ source: 'dogfood', version: 1 });
const metaJsonDraft = ref(stringifyChoyJson(normalizeChoyJsonIncoming(metaJson.value)));
const metaJsonParseError = ref('');
watch(
  () => metaJson.value,
  (next) => {
    const pretty = stringifyChoyJson(normalizeChoyJsonIncoming(next));
    const check = tryParseChoyJson(metaJsonDraft.value, { allowArray: false, nullable: true });
    if (check.ok && stringifyChoyJson(check.value) === pretty) return;
    metaJsonDraft.value = pretty;
    metaJsonParseError.value = '';
  },
);
function commitMetaJsonDraft(): void {
  const result = tryParseChoyJson(metaJsonDraft.value, { allowArray: false, nullable: true });
  if (!result.ok) {
    metaJsonParseError.value = result.error;
    return;
  }
  metaJsonParseError.value = '';
  metaJson.value = result.value;
  metaJsonDraft.value = stringifyChoyJson(result.value);
}

const propItems = [
  { name: 'segment', type: 'char', string: 'Segment', value: 'enterprise' },
  {
    name: 'tier',
    string: 'Tier',
    selection: [['bronze', 'Bronze'], ['silver', 'Silver'], ['gold', 'Gold']],
    value: 'silver',
  },
] as any;
const propsMap = ref<PropertiesMap>(
  Object.assign(Object.create(null), { segment: 'enterprise', tier: 'silver' }),
);
const renderablePropItems = computed(
  () => filterRenderablePropertyItems(propItems ?? []).renderable,
);
function readPropValue(name: string): unknown {
  const map = propsMap.value || Object.create(null);
  if (Object.prototype.hasOwnProperty.call(map, name)) {
    return map[name];
  }
  const item = (propItems ?? []).find((i: { name?: string }) => i?.name === name);
  if (!item) return undefined;
  if (Object.prototype.hasOwnProperty.call(item, 'value')) return item.value;
  if (Object.prototype.hasOwnProperty.call(item, 'default')) return item.default;
  return undefined;
}
function setPropValue(name: string, value: unknown): void {
  propsMap.value = writePropertyValue(propItems ?? [], propsMap.value, name, value);
}
function propAsString(v: unknown): string {
  return v == null ? '' : String(v);
}

const capitalFocused = ref(false);
const capitalEdited = ref(false);
const capitalInvalidDraft = ref(false);
const capitalDraft = ref('');
const monetaryCurrency = computed(() => String(currencyId.value ?? '').trim().toUpperCase());
const capitalDisplay = computed(() => {
  if (capitalFocused.value || capitalEdited.value) {
    return capitalDraft.value;
  }
  return formatChoyMonetary(capital.value, {
    precision: 2,
    currency: monetaryCurrency.value,
  });
});
watch(capital, (next) => {
  const parsedDraft = parseChoyNumber(capitalDraft.value, 'decimal');
  if (
    capitalFocused.value &&
    parsedDraft !== null &&
    next !== null &&
    parsedDraft === next
  ) {
    return;
  }
  capitalEdited.value = false;
  capitalInvalidDraft.value = false;
  capitalDraft.value = next === null || next === undefined ? '' : String(next);
});
function onCapitalFocus(): void {
  capitalFocused.value = true;
  if (!capitalEdited.value) {
    capitalDraft.value =
      capital.value === null || capital.value === undefined ? '' : String(capital.value);
  }
}
function onCapitalBlur(): void {
  capitalFocused.value = false;
  commitCapitalDraft();
}
function commitCapitalDraft(): void {
  if (!capitalEdited.value) {
    capitalDraft.value =
      capital.value === null || capital.value === undefined ? '' : String(capital.value);
    return;
  }
  const text = capitalDraft.value.trim();
  if (!text) {
    capital.value = null;
    capitalDraft.value = '';
    capitalEdited.value = false;
    capitalInvalidDraft.value = false;
    return;
  }
  if (parseChoyNumber(capitalDraft.value, 'decimal') === null) {
    capitalEdited.value = true;
    capitalInvalidDraft.value = true;
    return;
  }
  const rounded = roundChoyDecimal(capitalDraft.value, resolveChoyMonetaryPrecision(2));
  if (!rounded) {
    capitalEdited.value = true;
    capitalInvalidDraft.value = true;
    return;
  }
  capital.value = rounded.value;
  capitalDraft.value = rounded.text;
  capitalEdited.value = false;
  capitalInvalidDraft.value = false;
}
function onCapitalInput(value: string): void {
  capitalEdited.value = true;
  capitalInvalidDraft.value = false;
  capitalDraft.value = value;
}
function onCapitalKeydown(event: KeyboardEvent): void {
  if (event.key !== 'Enter' || event.isComposing || event.keyCode === 229) {
    return;
  }
  event.preventDefault();
  commitCapitalDraft();
}

type ContactLine = { Id: string; Title: string; Role?: string };
const contactLines = ref<ContactLine[]>([
  { Id: 'ct1', Title: 'Ada Lovelace', Role: 'Owner' },
  { Id: 'ct2', Title: 'Grace Hopper', Role: 'Billing' },
]);
const contactColumns: ColumnDef<ContactLine, unknown>[] = [
  { accessorKey: 'Title', header: 'Name', size: 160 },
  { accessorKey: 'Role', header: 'Role', size: 100 },
];
const tagIds = ref<string[]>(['usd']);

const kanbanLanes = ref<ChoyKanbanLane[]>(
  groupRowsIntoChoyKanbanLanes(
    allRows.value.slice(0, 12).map((row, i) => ({
      Id: row.Id,
      Title: row.name,
      State: i % 3 === 0 ? 'draft' : i % 3 === 1 ? 'confirmed' : 'done',
      Note: row.country,
    })),
    {
      laneField: 'State',
      laneDefs: [
        { key: 'draft', label: 'Draft' },
        { key: 'confirmed', label: 'Confirmed' },
        { key: 'done', label: 'Done' },
      ],
      titleField: 'Title',
      subtitleField: 'Note',
    },
  ),
);

const searchKeyword = ref('');
const appliedQuery = ref<ChoySearchQuery>({ keyword: '', filters: [] });
const listSelection = ref<Array<string | number>>([]);
const page = ref(1);
const pageSize = ref(10);

const listColumns: ColumnDef<CompanyRow, unknown>[] = [
  { accessorKey: 'name', header: 'Name', size: 180 },
  { accessorKey: 'country', header: 'Country', size: 100 },
  {
    accessorKey: 'active',
    header: 'Active',
    size: 80,
    cell: ({ getValue }) => (getValue<boolean>() ? 'Yes' : 'No'),
  },
];

function companyRowId(row: CompanyRow): string {
  return row.Id;
}

const filteredRows = computed(() =>
  filterRowsByKeyword(allRows.value, appliedQuery.value.keyword, ['name', 'country']),
);

const total = computed(() => filteredRows.value.length);

const pageRows = computed(() => {
  const size = Math.max(1, Math.floor(pageSize.value) || 1);
  const pages = choyTotalPages(total.value, size);
  const safePage = clampChoyPage(page.value, pages);
  const offset = choyPageOffset(safePage, size);
  return filteredRows.value.slice(offset, offset + size);
});

onBeforeUnmount(() => {
  if (saveTimer !== null) {
    clearTimeout(saveTimer);
    saveTimer = null;
  }
});

async function searchCurrencies(
  query: string,
  opts: { limit: number },
): Promise<RelationOption[]> {
  const q = query.trim().toLowerCase();
  const rows = CURRENCIES.filter(
    (row) => !q || row.label.toLowerCase().includes(q) || row.id.includes(q),
  );
  return rows.slice(0, opts.limit);
}

function onCurrencySelect(option: RelationOption | null): void {
  currencyOption.value = option;
}

function onSearch(query: ChoySearchQuery): void {
  appliedQuery.value = query;
  page.value = 1;
  // The visible row set changes, so ids selected on the previous result set must not linger.
  listSelection.value = [];
  selectedRowId.value = null;
  ChoyMessage.info('Search applied', {
    description: query.keyword ? `Keyword: ${query.keyword}` : 'Cleared keyword filter.',
  });
}

function onSave(): void {
  if (formLoading.value) {
    return;
  }
  const trimmed = name.value.trim();
  if (!trimmed) {
    ChoyMessage.error('Save failed', { description: 'Name is required.' });
    return;
  }
  const selectedId =
    listSelection.value.length === 1 ? String(listSelection.value[0]) : null;
  // The form's loaded row stays the save target across paging; checkbox selection
  // still requires a visible row on the current page.
  const captureRowId =
    (selectedRowId.value !== null &&
    allRows.value.some((row) => row.Id === selectedRowId.value)
      ? selectedRowId.value
      : null) ??
    (selectedId !== null && pageRows.value.some((row) => row.Id === selectedId)
      ? selectedId
      : null);
  const captureActive = active.value;
  const captureCurrencyLabel = currencyOption.value?.label ?? 'no currency';
  if (!captureRowId) {
    ChoyMessage.info('No row selected', {
      description: 'Pick a company from the list before saving.',
    });
    return;
  }
  formLoading.value = true;
  if (saveTimer !== null) {
    clearTimeout(saveTimer);
  }
  saveTimer = setTimeout(() => {
    saveTimer = null;
    formLoading.value = false;
    const row = allRows.value.find((r) => r.Id === captureRowId);
    if (!row) {
      ChoyMessage.info('No row selected', {
        description: 'Pick a company from the list before saving.',
      });
      return;
    }
    row.name = trimmed;
    row.active = captureActive;
    // Keep Kanban cards in sync with the renamed company row.
    for (const lane of kanbanLanes.value) {
      for (const card of lane.cards) {
        if (card.id === captureRowId) {
          card.title = trimmed;
        }
      }
    }
    ChoyMessage.success('Company saved (dogfood)', {
      description: `${trimmed} · ${captureCurrencyLabel}`,
    });
  }, 400);
}

function onRowClick(row: CompanyRow): void {
  selectedRowId.value = row.Id;
  name.value = row.name;
  active.value = row.active;
  activeTab.value = 'form';
  ChoyMessage.info('Loaded from list', { description: row.name });
}
</script>
