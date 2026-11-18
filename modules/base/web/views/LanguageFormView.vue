<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <ChoyFormView
    ref="formViewRef"
    v-bind="{ store, recordId, viewMode, showHeader, createAction }"
    :action-ids="{ create: languageActions.create, edit: languageActions.edit, copy: languageActions.copy, delete: languageActions.delete }"
    :has-action="hasAction"
    v-on="$attrs"
    @update-success="reloadActiveLanguagesAfterLanguageSave"
    @create-success="reloadActiveLanguagesAfterLanguageSave"
  >
    <template #secondary-actions="{ formData, recordId: slotRecordId, loading }">
      <ChoyButton
        v-if="!loading && showActivateFor(formData, slotRecordId)"
        type="button"
        size="default"
        data-testid="language-activate"
        :disabled="activeBusy || reloadBusy"
        :aria-busy="activeBusy || undefined"
        @click="() => void onActivate(slotRecordId)"
      >
        <CircleCheck class="size-4" aria-hidden="true" />
        {{ _t('Activate') }}
      </ChoyButton>
      <ChoyButton
        v-else-if="!loading && showDeactivateFor(formData, slotRecordId)"
        type="button"
        size="default"
        variant="outline"
        data-testid="language-deactivate"
        :disabled="activeBusy || reloadBusy"
        :aria-busy="activeBusy || undefined"
        @click="() => void onDeactivate(slotRecordId)"
      >
        <CircleOff class="size-4" aria-hidden="true" />
        {{ _t('Deactivate') }}
      </ChoyButton>
      <ChoyButton
        v-if="!loading && showReloadTerminologyFor(formData, slotRecordId)"
        type="button"
        size="default"
        variant="outline"
        data-testid="language-reload-terminology"
        :disabled="activeBusy || reloadBusy"
        :aria-busy="reloadBusy || undefined"
        @click="() => void onReloadTerminology(slotRecordId, formData)"
      >
        <Languages class="size-4" aria-hidden="true" />
        {{ _t('Update translations') }}
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
import { computed, ref } from 'vue';
import type { RouteLocationRaw } from 'vue-router';
import { CircleCheck, CircleOff, Languages } from 'lucide-vue-next';
import type { ModelStore } from '@/web/web/stores/modelStore';
import type Language from '@/base/service/models/language';
import { defineModelActions } from '@/core/web/resource';
import { usePermission } from '@/auth/web/composables/usePermission';
import { resolvePageStore } from '@/web/web/composables/usePageContext';
import { createTranslate } from '@/web/web/i18n';
import { ChoyMessage } from '@/web/web/composables/useChoyMessage';
import { useI18nStore } from '@/web/web/stores/i18nStore';
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
  mergeLanguageFormFields,
  buildLanguageReloadTerminologyToast,
  reloadActiveLanguagesAfterLanguageSave,
  reloadLanguageTerminology,
  resolveLanguageCurrentRow,
  shouldPreferBrowseReloadAfterLanguageAction,
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
const { recordId, viewMode, showHeader, createAction } = props;

type FormViewApi = {
  reloadDisplay?: () => Promise<void>;
  refresh?: () => Promise<void>;
  getViewMode?: () => string | null | undefined;
};
const formViewRef = ref<FormViewApi | null>(null);

const activeBusy = ref(false);
const reloadBusy = ref(false);
/**
 * Local override after Activate/Deactivate when form reload is unavailable.
 * Scoped via languageRowOverrideFields(id) — stale overrides for other recordIds are ignored.
 */
const rowOverride = ref<LanguageRowOverride | null>(null);

const canWriteLanguage = computed(() => hasAction(languageActions.edit));

function resolveRowFields(
  formData: Record<string, unknown> | undefined,
  slotRecordId: string | undefined
): { id: string; fields: Record<string, unknown> } {
  const id = String(slotRecordId || '').trim();
  const overrideFields = languageRowOverrideFields(rowOverride.value, id);
  if (overrideFields) return { id, fields: overrideFields };
  const fromStore = resolveLanguageCurrentRow({
    recordId: id,
    records: (store as any)?.state?.records,
    rows: (store as any)?.state?.result?.rows,
  });
  return {
    id,
    fields: mergeLanguageFormFields({ recordId: id, fromStore, formData }),
  };
}

function rowMatched(fields: Record<string, unknown>): boolean {
  return !!(String(fields.Id || '').trim() || String(fields.Code || '').trim());
}

function showActivateFor(formData: Record<string, unknown> | undefined, slotRecordId: string | undefined): boolean {
  if (!canWriteLanguage.value) return false;
  const { id, fields } = resolveRowFields(formData, slotRecordId);
  return !!id && rowMatched(fields) && fields.IsActive !== true;
}

function showDeactivateFor(formData: Record<string, unknown> | undefined, slotRecordId: string | undefined): boolean {
  if (!canWriteLanguage.value) return false;
  const { id, fields } = resolveRowFields(formData, slotRecordId);
  return !!id && rowMatched(fields) && fields.IsActive === true;
}

/** Update translations is available for any matched row (active or not). */
function showReloadTerminologyFor(
  formData: Record<string, unknown> | undefined,
  slotRecordId: string | undefined
): boolean {
  if (!canWriteLanguage.value) return false;
  const { id, fields } = resolveRowFields(formData, slotRecordId);
  return !!id && rowMatched(fields);
}

async function reloadRecordViaBrowse(id: string): Promise<void> {
  const rid = String(id || '').trim();
  if (!rid) return;
  if (typeof (store as any)?.Browse !== 'function') return;
  const browsed = await (store as any).Browse(rid, ['Id', 'Code', 'IsActive', 'Name']);
  const fields = languageRowFields(browsed);
  if (fields && (String(fields.Id || '').trim() || String(fields.Code || '').trim())) {
    rowOverride.value = { id: rid, fields: { ...fields } };
  } else if (!rowOverride.value || rowOverride.value.id === rid) {
    // Do not clear an override that belongs to a different record after navigation.
    rowOverride.value = null;
  }
}

/** Prefer silent form reload so the sheet IsActive matches the header CTA. */
async function reloadRecord(id: string): Promise<void> {
  const form = formViewRef.value;
  // Keep unsaved edit/create drafts; Browse override still flips the header CTA.
  if (shouldPreferBrowseReloadAfterLanguageAction(form?.getViewMode?.())) {
    await reloadRecordViaBrowse(id);
    return;
  }
  try {
    if (form?.reloadDisplay) {
      await form.reloadDisplay();
      rowOverride.value = null;
      return;
    }
    if (form?.refresh) {
      await form.refresh();
      rowOverride.value = null;
      return;
    }
  } catch {
    // Fall through to Browse override so the header CTA can still flip.
  }
  await reloadRecordViaBrowse(id);
}

async function onActivate(slotRecordId: string | undefined): Promise<void> {
  if (activeBusy.value || reloadBusy.value) return;
  const id = String(slotRecordId || '').trim();
  if (!id) return;
  const activate = (store as any)?.Activate;
  if (typeof activate !== 'function') {
    try {
      ChoyMessage.error(_t('Activate failed'));
    } catch {
      // Toast host may be absent in FE unit tests.
    }
    return;
  }
  activeBusy.value = true;
  try {
    await activateLanguage({
      activate: req => activate.call(store, req),
      recordId: id,
      reloadRecord: () => reloadRecord(id),
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

async function onDeactivate(slotRecordId: string | undefined): Promise<void> {
  if (activeBusy.value || reloadBusy.value) return;
  const id = String(slotRecordId || '').trim();
  if (!id) return;
  const deactivate = (store as any)?.Deactivate;
  if (typeof deactivate !== 'function') {
    try {
      ChoyMessage.error(_t('Deactivate failed'));
    } catch {
      // Toast host may be absent in FE unit tests.
    }
    return;
  }
  activeBusy.value = true;
  try {
    await deactivateLanguage({
      deactivate: req => deactivate.call(store, req),
      recordId: id,
      reloadRecord: () => reloadRecord(id),
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

async function onReloadTerminology(
  slotRecordId: string | undefined,
  formData: Record<string, unknown> | undefined
): Promise<void> {
  if (reloadBusy.value || activeBusy.value) return;
  const { id, fields } = resolveRowFields(formData, slotRecordId);
  if (!id) return;
  const reload = (store as any)?.ReloadTerminology;
  if (typeof reload !== 'function') {
    try {
      ChoyMessage.error(_t('Update translations failed'));
    } catch {
      // Toast host may be absent in FE unit tests.
    }
    return;
  }
  reloadBusy.value = true;
  try {
    const i18n = useI18nStore();
    // Prefer LanguageId so an unsaved Code draft cannot retarget the reload.
    await reloadLanguageTerminology({
      reload: params => reload.call(store, params),
      languageId: id,
      currentLang: () => i18n.lang,
      revalidateIfCurrent: () => i18n.revalidateTerminologyIfStale(),
      onDone: stats => {
        const toast = buildLanguageReloadTerminologyToast({
          code: String(stats?.Code || fields.Code || '').trim() || id,
          modulesScanned: Number(stats?.ModulesScanned || 0),
          modulesWithPo: Number(stats?.ModulesWithPo || 0),
          upserted: Number(stats?.Upserted || 0),
          isActive: fields.IsActive === true ? true : fields.IsActive === false ? false : null,
        });
        try {
          const opts = toast.description ? { description: toast.description } : undefined;
          if (toast.level === 'success') ChoyMessage.success(toast.title, opts);
          else ChoyMessage.warning(toast.title, opts);
        } catch {
          // Toast host may be absent in FE unit tests.
        }
      },
    });
  } catch (err: any) {
    const message = String(err?.message || err || _t('Update translations failed'));
    try {
      ChoyMessage.error(message);
    } catch {
      // Toast host may be absent in FE unit tests.
    }
  } finally {
    reloadBusy.value = false;
  }
}
</script>
