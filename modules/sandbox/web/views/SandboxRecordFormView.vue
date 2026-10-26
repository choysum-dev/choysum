<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <ChoyFormView
    v-bind="{ store, recordId, initialValues, viewMode, showHeader, createAction }"
    :action-ids="{
      create: sandboxRecordActions.create,
      edit: sandboxRecordActions.edit,
      copy: sandboxRecordActions.copy,
      delete: sandboxRecordActions.delete,
    }"
    :has-action="hasAction"
    v-on="$attrs"
  >
    <!--
      Sheet tip chrome + ChoySection `divided` for phase breaks (padless sections,
      not nested cards). Avoid stacking flex gap with divided mt-* (double rhythm).
    -->
    <div data-anchor="choy.form.sheet" class="flex min-h-full flex-col">
      <div class="mb-8 flex flex-col gap-3">
        <ChoyStatusbarField :store="store" prop="Progress" />
        <aside
          class="rounded-md border border-border bg-muted/40 px-3.5 py-2.5 text-xs leading-relaxed text-muted-foreground"
          data-testid="sandbox-onchange-hint"
        >
          {{
            _t(
              'Onchange: Active/State warn → toast; State disables Progress statusbar clicks (options stay visible); Currency/Category rewrite values; negative Int Value → field error; Line Quantity silently clamps to 1…999 (no toast).',
            )
          }}
        </aside>
      </div>

      <ChoySection :title="_t('Identity')">
        <ChoyGrid :cols="12" gap="lg">
          <ChoyCol :span="3">
            <ChoyCharField :store="store" prop="NameChar" :rules="requiredRules" />
          </ChoyCol>
          <ChoyCol :span="3">
            <ChoyVarcharField :store="store" prop="Title" :rules="requiredRules" />
          </ChoyCol>
          <ChoyCol :span="3">
            <ChoyVarcharField :store="store" prop="Code" :rules="requiredRules" :copyable="true" />
          </ChoyCol>
          <ChoyCol :span="3">
            <ChoyBooleanField :store="store" prop="IsActive" widget="switch" />
          </ChoyCol>
          <ChoyCol :span="3">
            <ChoyIntField :store="store" prop="Sequence" />
          </ChoyCol>
          <ChoyCol :span="3">
            <ChoySelectionField :store="store" prop="State" />
          </ChoyCol>
          <ChoyCol :span="12">
            <ChoyTextField :store="store" prop="Notes" orientation="vertical" />
          </ChoyCol>
        </ChoyGrid>
      </ChoySection>

      <ChoySection :title="_t('Numbers')" divided>
        <ChoyGrid :cols="12" gap="lg">
          <ChoyCol :span="3">
            <ChoyIntField :store="store" prop="IntValue" />
          </ChoyCol>
          <ChoyCol :span="3">
            <ChoyBigintField :store="store" prop="BigintValue" />
          </ChoyCol>
          <ChoyCol :span="3">
            <ChoyNumberField :store="store" prop="NumberValue" />
          </ChoyCol>
          <ChoyCol :span="3">
            <ChoyDecimalField :store="store" prop="DecimalValue" />
          </ChoyCol>
          <ChoyCol :span="3">
            <ChoyManyToOneRefField
              :store="store"
              prop="CurrencyId"
              :searchView="CurrencyListView"
              :search-view-title="_t('Select Currency')"
            />
          </ChoyCol>
          <ChoyCol :span="3">
            <ChoyMonetaryField :store="store" prop="Amount" />
          </ChoyCol>
        </ChoyGrid>
      </ChoySection>

      <ChoySection :title="_t('Temporal')" divided>
        <ChoyGrid :cols="12" gap="lg">
          <ChoyCol :span="4">
            <ChoyDateField :store="store" prop="DateValue" />
          </ChoyCol>
          <ChoyCol :span="4">
            <ChoyDatetimeField :store="store" prop="DatetimeValue" mode="datetime" />
          </ChoyCol>
          <ChoyCol :span="4">
            <ChoyTimeField :store="store" prop="TimeValue" />
          </ChoyCol>
        </ChoyGrid>
      </ChoySection>

      <ChoySection :title="_t('Rich Content')" divided>
        <ChoyGrid :cols="12" gap="lg">
          <ChoyCol :span="12">
            <ChoyHtmlField :store="store" prop="BodyHtml" orientation="vertical" />
          </ChoyCol>
          <ChoyCol :span="6">
            <ChoyBinaryField :store="store" prop="Attachment" />
          </ChoyCol>
          <ChoyCol :span="6">
            <ChoyImageField :store="store" prop="Image" />
          </ChoyCol>
          <ChoyCol :span="6">
            <ChoyJsonField :store="store" prop="JsonPayload" />
          </ChoyCol>
          <ChoyCol :span="6">
            <ChoyPropertiesField :store="store" prop="Properties" />
          </ChoyCol>
        </ChoyGrid>
      </ChoySection>

      <ChoySection :title="_t('Depth')" divided>
        <ChoyGrid :cols="12" gap="lg">
          <ChoyCol :span="6">
            <ChoyVarcharField :store="store" prop="CompanyNote" />
          </ChoyCol>
          <ChoyCol :span="6">
            <ChoyManyToOneRefField
              :store="store"
              prop="CompanyId"
              :searchView="CompanyListView"
              :search-view-title="_t('Select Company')"
            />
          </ChoyCol>
          <ChoyCol :span="3">
            <ChoyIntField :store="store" prop="LineCount" :readonly="true" />
          </ChoyCol>
          <ChoyCol :span="3">
            <ChoyVarcharField :store="store" prop="CategoryName" :readonly="true" />
          </ChoyCol>
          <ChoyCol :span="6">
            <ChoyManyToOneField :store="store" prop="PrimaryLineId" :readonly="true">
              <ChoyVarcharField :store="store" prop="PrimaryLineId.Name" />
            </ChoyManyToOneField>
          </ChoyCol>
        </ChoyGrid>
      </ChoySection>

      <ChoySection :title="_t('Relations')" divided>
        <ChoyGrid :cols="12" gap="lg">
          <ChoyCol :span="6">
            <ChoyManyToOneField
              :store="store"
              prop="CategoryId"
              :searchView="SandboxCategoryListView"
              :search-view-title="_t('Select Category')"
            >
              <ChoyVarcharField :store="store" prop="CategoryId.Name" />
            </ChoyManyToOneField>
          </ChoyCol>
          <ChoyCol :span="6">
            <ChoyManyToOneRefField
              :store="store"
              prop="CategoryRefId"
              :searchView="SandboxCategoryListView"
              :search-view-title="_t('Select Category Ref')"
            />
          </ChoyCol>
          <ChoyCol :span="12">
            <ChoyManyToManyField
              :store="store"
              prop="Tags"
              :search-list="SandboxTagListView"
              :search-view-title="_t('Select Tags')"
            >
              <ChoyVarcharField :store="store" prop="Tags.Name" />
              <ChoyVarcharField :store="store" prop="Tags.Code" />
            </ChoyManyToManyField>
          </ChoyCol>
          <ChoyCol :span="12">
            <!--
              ManyToManyRef is string[] on the parent, so TagTableRefIds.Name is not a FieldPath.
              Nested columns bind via field-prefix + row; type the column store as the target model.
            -->
            <ChoyManyToManyRefField
              :store="store"
              prop="TagTableRefIds"
              :search-list="SandboxTagListView"
              :search-view-title="_t('Select Tag Ref Table')"
            >
              <template #default="{ refStore }">
                <ChoyVarcharField :store="(refStore as WebModelStore<SandboxTag> | undefined)" prop="Name" />
                <ChoyVarcharField :store="(refStore as WebModelStore<SandboxTag> | undefined)" prop="Code" />
              </template>
            </ChoyManyToManyRefField>
          </ChoyCol>
          <ChoyCol :span="12">
            <ChoyManyToManyTagsField
              :store="store"
              prop="TagChips"
              :search-list="SandboxTagListView"
              :search-view-title="_t('Select Tag Chips')"
              :tag-label-field="['Name', 'Code', 'Id']"
            />
          </ChoyCol>
          <ChoyCol :span="12">
            <ChoyManyToManyRefTagsField
              :store="store"
              prop="TagRefIds"
              :search-list="SandboxTagListView"
              :search-view-title="_t('Select Tag Refs')"
              :tag-label-field="['Name', 'Code', 'Id']"
            />
          </ChoyCol>
          <ChoyCol :span="12">
            <ChoyManyToManyTreeField
              :store="store"
              prop="Categories"
              :label="_t('Categories')"
              :lazy="false"
              :max-depth="0"
              children-field="Children"
              order-field="Code"
              :root-condition="{
                And: [['ParentId', 'is', null]],
              }"
              :default-expand-all="true"
              :check-strictly="false"
            />
          </ChoyCol>
          <ChoyCol :span="12">
            <ChoyManyToManyRefTreeField
              :store="store"
              prop="CategoryPathIds"
              :label="_t('Category Path')"
              :lazy="false"
              :max-depth="0"
              children-field="Children"
              :root-condition="{
                And: [['ParentId', 'is', null]],
              }"
              :default-expand-all="true"
              :check-strictly="false"
            />
          </ChoyCol>
        </ChoyGrid>
      </ChoySection>

      <ChoySection :title="_t('Lines')" divided>
        <div class="flex flex-col gap-4">
          <SandboxLinesVirtTools :store="store" />
          <ChoyTabs v-model="linesTab">
            <!-- force-mount: both panes register Lines.* so save returnFields stay complete. -->
            <ChoyTab :label="_t('Lines (table)')" value="lines-table" force-mount>
              <ChoyOneToManyField :store="store" prop="Lines" hide-label>
                <ChoyVarcharField :store="store" prop="Lines.Name" />
                <ChoyIntField :store="store" prop="Lines.Quantity" />
                <ChoySelectionField :store="store" prop="Lines.State" />
                <ChoyIntField :store="store" prop="Lines.Sequence" />
              </ChoyOneToManyField>
            </ChoyTab>
            <ChoyTab :label="_t('Lines (kanban)')" value="lines-kanban" force-mount>
              <ChoyOneToManyKanbanField
                :store="store"
                prop="Lines"
                hide-label
                card-title-field="Name"
                card-subtitle-field=""
                :fields="['Quantity', 'State']"
                :default-record="defaultLineRecord"
                :form-view="SandboxLineFormView"
                :create-dialog-title="_t('New Line')"
                :edit-dialog-title="_t('Edit Line')"
                :display-dialog-title="_t('View Line')"
              >
                <template #card="{ item }">
                  <div class="flex flex-col gap-1 text-xs">
                    <div class="truncate text-sm font-medium">{{ item?.Name || _t('Unnamed Line') }}</div>
                    <div class="text-muted-foreground">
                      {{ _t('Qty') }} {{ item?.Quantity ?? 0 }}
                      <span class="text-muted-foreground/60"> · </span>
                      {{ item?.State || '—' }}
                    </div>
                  </div>
                </template>
              </ChoyOneToManyKanbanField>
            </ChoyTab>
          </ChoyTabs>
        </div>
      </ChoySection>
    </div>
  </ChoyFormView>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue';
import type { RouteLocationRaw } from 'vue-router';
import type { WebModelStore } from '@/web/web/stores/modelStore';
import type SandboxRecord from '@/sandbox/service/models/sandbox_record';
import type SandboxLine from '@/sandbox/service/models/sandbox_line';
import type SandboxTag from '@/sandbox/service/models/sandbox_tag';
import CompanyListView from '@/base/web/views/CompanyListView.vue';
import CurrencyListView from '@/base/web/views/CurrencyListView.vue';
import SandboxCategoryListView from './SandboxCategoryListView.vue';
import SandboxTagListView from './SandboxTagListView.vue';
import SandboxLineFormView from './SandboxLineFormView.vue';
import SandboxLinesVirtTools from './SandboxLinesVirtTools.vue';
import { usePermission } from '@/auth/web/composables/usePermission';
import { resolvePageStore } from '@/web/web/composables/usePageContext';
import { createTranslate } from '@/web/web/i18n';
import {
  ChoyBigintField,
  ChoyBinaryField,
  ChoyBooleanField,
  ChoyCharField,
  ChoyCol,
  ChoyDateField,
  ChoyDatetimeField,
  ChoyDecimalField,
  ChoyFormView,
  ChoyGrid,
  ChoyHtmlField,
  ChoyImageField,
  ChoyIntField,
  ChoyJsonField,
  ChoyManyToManyField,
  ChoyManyToManyRefField,
  ChoyManyToManyRefTagsField,
  ChoyManyToManyRefTreeField,
  ChoyManyToManyTagsField,
  ChoyManyToManyTreeField,
  ChoyManyToOneField,
  ChoyManyToOneRefField,
  ChoyMonetaryField,
  ChoyNumberField,
  ChoyOneToManyField,
  ChoyOneToManyKanbanField,
  ChoyPropertiesField,
  ChoySection,
  ChoySelectionField,
  ChoyStatusbarField,
  ChoyTab,
  ChoyTabs,
  ChoyTextField,
  ChoyTimeField,
  ChoyVarcharField,
  type ChoyViewMode as ViewMode,
} from '@/web';
import { sandboxRecordActions } from './sandbox_record_actions';

defineOptions({ name: 'SandboxRecordFormView', inheritAttrs: true });
const { _t } = createTranslate('sandbox', { scope: 'web/views/SandboxRecordFormView' });
const requiredRules = computed(() => [{ required: true, message: _t('Required') }]);
const linesTab = ref('lines-table');

const props = withDefaults(
  defineProps<{
    store?: WebModelStore<SandboxRecord>;
    recordId?: string;
    initialValues?: Partial<SandboxRecord>;
    viewMode?: ViewMode;
    showHeader?: boolean;
    createAction?: string | RouteLocationRaw;
  }>(),
  {
    showHeader: true,
    createAction: undefined,
  },
);

const store = resolvePageStore(props.store, 'SandboxRecordFormView');
const { recordId, initialValues, viewMode, showHeader, createAction } = props;
const { hasAction } = usePermission();

const defaultLineRecord = computed((): Partial<SandboxLine> => ({
  Name: '',
  Quantity: 1,
  State: 'todo',
  Sequence: 10,
}));
</script>
