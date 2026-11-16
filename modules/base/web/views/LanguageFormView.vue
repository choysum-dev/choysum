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
    @update-success="onLanguageSaved"
    @create-success="onLanguageSaved"
  >
    <template #default="{ formData, viewMode: mode }">
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
                :help="_t('Enabling a language makes it selectable. It does not install PO / terminology packs.')"
              />
            </ChoyCol>
          </ChoyGrid>
        </section>

        <section class="space-y-4" data-testid="language-form-number">
          <h3 class="text-base font-semibold tracking-tight text-foreground">{{ _t('Number Format') }}</h3>
          <ChoyGrid :cols="12">
            <ChoyCol :span="4">
              <ChoyVarcharField :store="store" prop="DecimalSeparator" :placeholder="'.'" />
              <div class="mt-1.5 flex flex-wrap gap-1.5">
                <button
                  v-for="preset in decimalSeparatorPresets"
                  :key="`dec-${preset}`"
                  type="button"
                  class="rounded-md border border-border bg-background px-2 py-0.5 text-xs text-muted-foreground hover:text-foreground"
                  data-testid="language-decimal-preset"
                  @click="applyPreset(formData, 'DecimalSeparator', preset)"
                >
                  {{ preset }}
                </button>
              </div>
            </ChoyCol>
            <ChoyCol :span="4">
              <ChoyVarcharField :store="store" prop="ThousandSeparator" :placeholder="','" />
              <div class="mt-1.5 flex flex-wrap gap-1.5">
                <button
                  v-for="preset in thousandSeparatorPresets"
                  :key="`thou-${preset.label}`"
                  type="button"
                  class="rounded-md border border-border bg-background px-2 py-0.5 text-xs text-muted-foreground hover:text-foreground"
                  data-testid="language-thousand-preset"
                  @click="applyPreset(formData, 'ThousandSeparator', preset.value)"
                >
                  {{ preset.label }}
                </button>
              </div>
            </ChoyCol>
            <ChoyCol :span="4">
              <ChoySelectionField :store="store" prop="Grouping" />
            </ChoyCol>
          </ChoyGrid>
          <p class="text-sm text-muted-foreground" data-testid="language-preview-number">
            {{ _t('Preview') }}: {{ numberPreview(formData) }}
          </p>
        </section>

        <section class="space-y-4" data-testid="language-form-datetime">
          <h3 class="text-base font-semibold tracking-tight text-foreground">{{ _t('Date & Time') }}</h3>
          <ChoyGrid :cols="12">
            <ChoyCol :span="4">
              <ChoyVarcharField :store="store" prop="DateFormat" :placeholder="'YYYY-MM-DD'" />
              <div class="mt-1.5 flex flex-wrap gap-1.5">
                <button
                  v-for="preset in dateFormatPresets"
                  :key="preset"
                  type="button"
                  class="rounded-md border border-border bg-background px-2 py-0.5 text-xs text-muted-foreground hover:text-foreground"
                  data-testid="language-date-preset"
                  @click="applyPreset(formData, 'DateFormat', preset)"
                >
                  {{ preset }}
                </button>
              </div>
            </ChoyCol>
            <ChoyCol :span="4">
              <ChoyVarcharField :store="store" prop="TimeFormat" :placeholder="'HH:mm:ss'" />
              <div class="mt-1.5 flex flex-wrap gap-1.5">
                <button
                  v-for="preset in timeFormatPresets"
                  :key="preset"
                  type="button"
                  class="rounded-md border border-border bg-background px-2 py-0.5 text-xs text-muted-foreground hover:text-foreground"
                  data-testid="language-time-preset"
                  @click="applyPreset(formData, 'TimeFormat', preset)"
                >
                  {{ preset }}
                </button>
              </div>
            </ChoyCol>
            <ChoyCol :span="4">
              <ChoySelectionField
                :store="store"
                prop="FirstDayOfWeek"
                data-testid="language-first-day"
              />
            </ChoyCol>
          </ChoyGrid>
          <p class="text-sm text-muted-foreground" data-testid="language-preview-datetime">
            {{ _t('Preview') }}: {{ datePreview(formData) }} · {{ timePreview(formData) }}
          </p>
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
              <ChoySelectionField
                :store="store"
                prop="PersonNameFormat"
                :selection="personNameSelection"
              />
            </ChoyCol>
          </ChoyGrid>
          <p class="text-sm text-muted-foreground" data-testid="language-preview-currency">
            {{ _t('Currency preview') }}: {{ currencyPreview(formData) }}
          </p>
          <p class="text-sm text-muted-foreground" data-testid="language-preview-person-name">
            {{ _t('Person name preview') }}: {{ personNamePreview(formData) }}
          </p>
        </section>
      </div>
    </template>
  </ChoyFormView>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import type { RouteLocationRaw } from 'vue-router';
import type { ModelStore } from '@/web/web/stores/modelStore';
import type Language from '@/base/service/models/language';
import { defineModelActions } from '@/core/web/resource';
import { usePermission } from '@/auth/web/composables/usePermission';
import { resolvePageStore } from '@/web/web/composables/usePageContext';
import { createTranslate } from '@/web/web/i18n';
import { useI18nStore } from '@/web/web/stores/i18nStore';
import {
  ChoyBooleanField,
  ChoyCol,
  ChoyFormView,
  ChoyGrid,
  ChoySelectionField,
  ChoyVarcharField,
} from '@/web';
import type { ChoyViewMode as ViewMode } from '@/web';
import {
  DATE_FORMAT_PRESETS,
  DECIMAL_SEPARATOR_PRESETS,
  isLanguageCodeReadonly,
  previewCurrency,
  previewDate,
  previewNumber,
  previewPersonName,
  previewTime,
  PERSON_NAME_SELECTION,
  TIME_FORMAT_PRESETS,
  type LanguageFormPreviewSource,
} from './languageFormHelpers';

defineOptions({ name: 'LanguageFormView', inheritAttrs: true });
const { _t, _lt } = createTranslate('base', { scope: 'web/views/LanguageFormView' });
const requiredRules = computed(() => [{ required: true, message: _t('Required') }]);
const props = withDefaults(
  defineProps<{ store?: ModelStore<Language>; recordId?: string; viewMode?: ViewMode; showHeader?: boolean; createAction?: string | RouteLocationRaw }>(),
  { showHeader: true, createAction: undefined }
);
const languageActions = defineModelActions('base.Language', { entityTitle: _lt('Language') });
const { hasAction } = usePermission();
const store = resolvePageStore(props.store, 'LanguageFormView');
const { recordId, viewMode, showHeader, createAction } = props;

const dateFormatPresets = DATE_FORMAT_PRESETS;
const timeFormatPresets = TIME_FORMAT_PRESETS;
const decimalSeparatorPresets = DECIMAL_SEPARATOR_PRESETS;
const thousandSeparatorPresets = [
  { value: ',', label: ',' },
  { value: '.', label: '.' },
  { value: ' ', label: '␣' },
  { value: "'", label: "'" },
  { value: '', label: '∅' },
] as const;
const personNameSelection = [...PERSON_NAME_SELECTION];

function applyPreset(formData: Record<string, unknown>, path: string, value: string) {
  formData[path] = value;
}

function asPreviewSource(formData: Record<string, unknown> | null | undefined): LanguageFormPreviewSource {
  return (formData || {}) as LanguageFormPreviewSource;
}

function numberPreview(formData: Record<string, unknown>) {
  return previewNumber(asPreviewSource(formData));
}
function currencyPreview(formData: Record<string, unknown>) {
  return previewCurrency(asPreviewSource(formData));
}
function datePreview(formData: Record<string, unknown>) {
  return previewDate(asPreviewSource(formData));
}
function timePreview(formData: Record<string, unknown>) {
  return previewTime(asPreviewSource(formData));
}
function personNamePreview(formData: Record<string, unknown>) {
  return previewPersonName(asPreviewSource(formData));
}

/** Refresh FE format overlays so list/number displays pick up Language field changes. */
async function onLanguageSaved() {
  try {
    await useI18nStore().loadActiveLanguages();
  } catch {
    // Best-effort; next full page init will reload formats.
  }
}
</script>
