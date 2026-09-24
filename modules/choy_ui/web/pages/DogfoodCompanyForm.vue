<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<script setup lang="ts">
import { computed, onBeforeUnmount, ref } from 'vue';
import type { ColumnDef } from '@tanstack/vue-table';
import '../styles/tokens.css';
import '../styles/preflight-policy.css';
// Produced by web build (EnsureChoyTailwindCSS); not committed.
import '../styles/choy-tailwind.generated.css';
import ChoyBooleanField from '../components/field/ChoyBooleanField.vue';
import ChoyDateField from '../components/field/ChoyDateField.vue';
import ChoyManyToOneField from '../components/field/ChoyManyToOneField.vue';
import ChoyMonetaryField from '../components/field/ChoyMonetaryField.vue';
import ChoySelectionField from '../components/field/ChoySelectionField.vue';
import ChoyStatusbarField from '../components/field/ChoyStatusbarField.vue';
import ChoyTextField from '../components/field/ChoyTextField.vue';
import ChoyVarcharField from '../components/field/ChoyVarcharField.vue';
import type { RelationOption } from '../components/internal/relationComboboxHelpers';
import ChoyButton from '../components/layout/ChoyButton.vue';
import ChoyCard from '../components/layout/ChoyCard.vue';
import ChoyCol from '../components/layout/ChoyCol.vue';
import ChoyGrid from '../components/layout/ChoyGrid.vue';
import ChoyPage from '../components/layout/ChoyPage.vue';
import ChoyTab from '../components/layout/ChoyTab.vue';
import ChoyTabs from '../components/layout/ChoyTabs.vue';
import Toaster from '../components/vendor/ui/toast/Toaster.vue';
import ChoyBreadcrumb from '../components/view/ChoyBreadcrumb.vue';
import ChoyFormView from '../components/view/ChoyFormView.vue';
import ChoyListView from '../components/view/ChoyListView.vue';
import ChoyPagination from '../components/view/ChoyPagination.vue';
import ChoySearchView from '../components/view/ChoySearchView.vue';
import {
  choyPageOffset,
  choyTotalPages,
  clampChoyPage,
} from '../components/view/paginationHelpers';
import {
  filterRowsByKeyword,
  type ChoySearchQuery,
} from '../components/view/searchViewHelpers';
import { ChoyMessage } from '../composables/useChoyMessage';

/**
 * Dogfood company form: Form + fields + List/Search/Pagination against local mocks.
 * No RPC — exercises the PR5 main path in isolation.
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

const searchKeyword = ref('');
const appliedQuery = ref<ChoySearchQuery>({ keyword: '', filters: [] });
const listSelection = ref<Array<string | number>>([]);
const page = ref(1);
const pageSize = ref(10);

const monetaryCurrency = computed(() => {
  // No currency selected → no suffix; don't imply USD on the amount.
  return String(currencyId.value ?? '').trim().toUpperCase();
});

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

<template>
  <div class="choy-dogfood-company choy-gallery-token-scope min-h-full bg-background text-foreground">
    <ChoyPage title="Dogfood Company" width="wide">
      <ChoyTabs v-model="activeTab">
        <ChoyTab value="form" label="Form">
          <ChoyCard class="mt-4">
            <ChoyFormView title="Company" :loading="formLoading">
              <template #breadcrumb>
                <ChoyBreadcrumb
                  :items="[
                    { label: 'Gallery', to: { name: 'ChoyUiGallery' } },
                    { label: 'Dogfood Company' },
                  ]"
                />
              </template>
              <template #statusbar>
                <ChoyStatusbarField v-model="state" :options="STATE_OPTIONS" label="" />
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
                  <ChoyVarcharField
                    v-model="name"
                    label="Name"
                    name="name"
                    required
                    help="Legal or trading name."
                  />
                </ChoyCol>
                <ChoyCol :span="6">
                  <ChoyManyToOneField
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
                  <ChoyDateField v-model="founded" label="Founded" name="founded" />
                </ChoyCol>
                <ChoyCol :span="4">
                  <ChoySelectionField
                    v-model="partnerType"
                    label="Type"
                    name="partner_type"
                    :options="TYPE_OPTIONS"
                  />
                </ChoyCol>
                <ChoyCol :span="4">
                  <ChoyBooleanField
                    v-model="active"
                    label="Active"
                    name="active"
                    widget="switch"
                  />
                </ChoyCol>
                <ChoyCol :span="6">
                  <ChoyMonetaryField
                    v-model="capital"
                    label="Share capital"
                    name="capital"
                    :currency="monetaryCurrency"
                    :precision="2"
                  />
                </ChoyCol>
                <ChoyCol :span="12">
                  <ChoyTextField v-model="notes" label="Notes" name="notes" />
                </ChoyCol>
              </ChoyGrid>
            </ChoyFormView>
          </ChoyCard>
        </ChoyTab>

        <ChoyTab value="list" label="List">
          <ChoyCard class="mt-4" title="Companies">
            <ChoyListView
              v-model:row-selection="listSelection"
              :columns="listColumns"
              :data="pageRows"
              :row-id="(row) => row.Id"
              :height="280"
              @row-click="onRowClick"
            >
              <template #search>
                <ChoySearchView
                  v-model:keyword="searchKeyword"
                  placeholder="Filter by name or country…"
                  @query-update="onSearch"
                />
              </template>
              <template #header>
                <p class="text-sm text-foreground/70">
                  Selected {{ listSelection.length }} · showing {{ pageRows.length }} of
                  {{ total }}
                </p>
              </template>
            </ChoyListView>
            <div class="mt-3 flex justify-end">
              <ChoyPagination v-model:page="page" v-model:page-size="pageSize" :total="total" />
            </div>
          </ChoyCard>
        </ChoyTab>
      </ChoyTabs>

      <template #footer>
        <p class="text-sm text-foreground/70">
          Isolation dogfood — Form/List/Search + field main set on local mocks; no Element Plus /
          no RPC.
        </p>
      </template>
    </ChoyPage>
    <Toaster />
  </div>
</template>
