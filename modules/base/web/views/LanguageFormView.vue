<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <ChoyFormView
    v-bind="{ store, recordId, viewMode, showHeader, createAction }"
    :action-ids="{ create: languageActions.create, edit: languageActions.edit, copy: languageActions.copy, delete: languageActions.delete }"
    :has-action="hasAction"
    v-on="$attrs"
    @update-success="reloadActiveLanguagesAfterLanguageSave"
    @create-success="reloadActiveLanguagesAfterLanguageSave"
  >
    <template #secondary-actions>
      <ChoyButton
        v-if="showActivate"
        type="button"
        size="default"
        data-testid="language-activate"
        :disabled="activeBusy"
        :aria-busy="activeBusy || undefined"
        @click="() => void onActivate()"
      >
        {{ _t('Activate') }}
      </ChoyButton>
      <ChoyButton
        v-else-if="showDeactivate"
        type="button"
        size="default"
        variant="outline"
        data-testid="language-deactivate"
        :disabled="activeBusy"
        :aria-busy="activeBusy || undefined"
        @click="() => void onDeactivate()"
      >
        {{ _t('Deactivate') }}
      </ChoyButton>
    </template>

    <template #default="{ viewMode: mode }">
      <div data-anchor="choy.form.sheet" class="min-h-full space-y-8">
        <section class="space-y-4" data-testid="language-form-identity">
          <h3 class="text-base font-semibold tracking-tight text-foreground">{{ _t('Language Information') }}</h3>
          <ChoyGrid :cols="12">
            <ChoyCol :span="4">
              <ChoyVarcharField :store="store" prop="Name" :rules="requiredRules" />
            </ChoyCol>
            <ChoyCol :span="4">
              <ChoyVarcharField
                v-if="mode === 'create'"
                :store="store"
                prop="Code"
                :rules="requiredRules"
                :placeholder="_t('e.g. en_US')"
                :help="_t('POSIX locale code (e.g. en_US). Catalog languages already exist — open them from the list to activate or edit.')"
                data-testid="language-code-create"
              />
              <ChoyVarcharField
                v-else
                :store="store"
                prop="Code"
                :rules="requiredRules"
                :readonly="isLanguageCodeReadonly(mode)"
                :help="_t('POSIX locale code cannot be changed after create.')"
                data-testid="language-code-readonly"
              />
            </ChoyCol>
            <ChoyCol :span="4">
              <ChoySelectionField
                :store="store"
                prop="Direction"
                :help="_t('Controls text direction (dir) for UI that follows this language.')"
              />
            </ChoyCol>
          </ChoyGrid>
          <ChoyGrid :cols="12">
            <ChoyCol :span="4">
              <ChoyBooleanField
                :store="store"
                prop="IsActive"
                :readonly="mode !== 'create'"
                :help="_t('Enabling a language makes it selectable. It does not install PO / terminology packs. Use Activate / Deactivate in the header after create.')"
              />
            </ChoyCol>
          </ChoyGrid>
        </section>

        <section class="space-y-4" data-testid="language-form-number">
          <h3 class="text-base font-semibold tracking-tight text-foreground">{{ _t('Number Format') }}</h3>
          <ChoyGrid :cols="12">
            <ChoyCol :span="4">
              <ChoyVarcharField :store="store" prop="DecimalSeparator" />
            </ChoyCol>
            <ChoyCol :span="4">
              <ChoyVarcharField :store="store" prop="ThousandSeparator" />
            </ChoyCol>
            <ChoyCol :span="4">
              <ChoySelectionField :store="store" prop="Grouping" />
            </ChoyCol>
          </ChoyGrid>
        </section>

        <section class="space-y-4" data-testid="language-form-datetime">
          <h3 class="text-base font-semibold tracking-tight text-foreground">{{ _t('Date & Time') }}</h3>
          <ChoyGrid :cols="12">
            <ChoyCol :span="4">
              <ChoyVarcharField :store="store" prop="DateFormat" />
            </ChoyCol>
            <ChoyCol :span="4">
              <ChoyVarcharField :store="store" prop="TimeFormat" />
            </ChoyCol>
            <ChoyCol :span="4">
              <ChoySelectionField
                :store="store"
                prop="FirstDayOfWeek"
                data-testid="language-first-day"
              />
            </ChoyCol>
          </ChoyGrid>
        </section>

        <section class="space-y-4" data-testid="language-form-currency-name">
          <h3 class="text-base font-semibold tracking-tight text-foreground">{{ _t('Currency & Person Name') }}</h3>
          <ChoyGrid :cols="12">
            <ChoyCol :span="4">
              <ChoySelectionField :store="store" prop="CurrencySymbolPosition" />
            </ChoyCol>
            <ChoyCol :span="4">
              <ChoyBooleanField :store="store" prop="CurrencySymbolSpacing" />
            </ChoyCol>
            <ChoyCol :span="4">
              <ChoyVarcharField
                :store="store"
                prop="PersonNameFormat"
                :rules="personNameFormatRules"
                placeholder="{FirstName} {LastName}"
              />
            </ChoyCol>
          </ChoyGrid>
        </section>
      </div>
    </template>
  </ChoyFormView>
</template>

<script setup lang="ts">
import { computed, ref, toRef } from 'vue';
import type { RouteLocationRaw } from 'vue-router';
import type { ModelStore } from '@/web/web/stores/modelStore';
import type Language from '@/base/service/models/language';
import { defineModelActions } from '@/core/web/resource';
import { usePermission } from '@/auth/web/composables/usePermission';
import { resolvePageStore } from '@/web/web/composables/usePageContext';
import { createTranslate } from '@/web/web/i18n';
import { ChoyMessage } from '@/web/web/composables/useChoyMessage';
import {
  ChoyBooleanField,
  ChoyButton,
  ChoyCol,
  ChoyFormView,
  ChoyGrid,
  ChoySelectionField,
  ChoyVarcharField,
} from '@/web';
import type { ChoyViewMode as ViewMode } from '@/web';
import {
  activateLanguage,
  createPersonNameFormatRules,
  deactivateLanguage,
  isLanguageCodeReadonly,
  languageRowFields,
  languageRowOverrideFields,
  reloadActiveLanguagesAfterLanguageSave,
  resolveLanguageCurrentRow,
  type LanguageRowOverride,
} from './languageFormHelpers';

defineOptions({ name: 'LanguageFormView', inheritAttrs: true });
const { _t, _lt } = createTranslate('base', { scope: 'web/views/LanguageFormView' });
const requiredRules = computed(() => [{ required: true, message: _t('Required') }]);
const personNameFormatRules = createPersonNameFormatRules(
  _t('Person name format must include {FirstName} and/or {LastName}')
);
const props = withDefaults(
  defineProps<{ store?: ModelStore<Language>; recordId?: string; viewMode?: ViewMode; showHeader?: boolean; createAction?: string | RouteLocationRaw }>(),
  { showHeader: true, createAction: undefined }
);
const languageActions = defineModelActions('base.Language', { entityTitle: _lt('Language') });
const { hasAction } = usePermission();
const store = resolvePageStore(props.store, 'LanguageFormView');
const recordId = toRef(props, 'recordId');
const { viewMode, showHeader, createAction } = props;

const activeBusy = ref(false);
/**
 * Local override after Activate/Deactivate.
 * Scoped via languageRowOverrideFields(id) — stale overrides for other recordIds are ignored.
 */
const rowOverride = ref<LanguageRowOverride | null>(null);

const canWriteLanguage = computed(() => hasAction(languageActions.edit));

const currentRow = computed(() => {
  const overrideFields = languageRowOverrideFields(rowOverride.value, recordId.value);
  if (overrideFields) return overrideFields;
  return resolveLanguageCurrentRow({
    recordId: recordId.value,
    records: (store as any)?.state?.records,
    rows: (store as any)?.state?.result?.rows,
  });
});

const hasMatchedRow = computed(() => {
  const rec = currentRow.value;
  return !!(String(rec.Id || '').trim() || String(rec.Code || '').trim());
});

const showActivate = computed(
  () => canWriteLanguage.value && hasMatchedRow.value && currentRow.value.IsActive !== true
);
const showDeactivate = computed(
  () => canWriteLanguage.value && hasMatchedRow.value && currentRow.value.IsActive === true
);

async function reloadRecord(): Promise<void> {
  const id = String(recordId.value || '').trim();
  if (!id) return;
  if (typeof (store as any)?.Browse !== 'function') return;
  const browsed = await (store as any).Browse(id, ['Id', 'Code', 'IsActive', 'Name']);
  // Drop stale Browse results after the user navigated to another Language.
  if (String(recordId.value || '').trim() !== id) return;
  const fields = languageRowFields(browsed);
  if (fields && (String(fields.Id || '').trim() || String(fields.Code || '').trim())) {
    rowOverride.value = { id, fields: { ...fields } };
  } else {
    rowOverride.value = null;
  }
}

async function onActivate(): Promise<void> {
  if (activeBusy.value) return;
  activeBusy.value = true;
  try {
    await activateLanguage({
      activate: req => (store as any).Activate(req),
      recordId: String(recordId.value || ''),
      reloadRecord,
    });
  } catch (err: any) {
    const message = String(err?.message || err || _t('Activate failed'));
    try {
      ChoyMessage.error(message);
    } catch {
      // Toast host may be absent in FE unit tests.
    }
  } finally {
    activeBusy.value = false;
  }
}

async function onDeactivate(): Promise<void> {
  if (activeBusy.value) return;
  activeBusy.value = true;
  try {
    await deactivateLanguage({
      deactivate: req => (store as any).Deactivate(req),
      recordId: String(recordId.value || ''),
      reloadRecord,
    });
  } catch (err: any) {
    const message = String(err?.message || err || _t('Deactivate failed'));
    try {
      ChoyMessage.error(message);
    } catch {
      // Toast host may be absent in FE unit tests.
    }
  } finally {
    activeBusy.value = false;
  }
}
</script>
