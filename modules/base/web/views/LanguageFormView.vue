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
                :help="_t('Enabling a language makes it selectable. It does not install PO / terminology packs.')"
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
import { isValidPersonNameFormat } from '@/core/service/utils/person_name';
import { isLanguageCodeReadonly } from './languageFormHelpers';

defineOptions({ name: 'LanguageFormView', inheritAttrs: true });
const { _t, _lt } = createTranslate('base', { scope: 'web/views/LanguageFormView' });
const requiredRules = computed(() => [{ required: true, message: _t('Required') }]);
const personNameFormatRules = computed(() => [
  {
    validator: (_rule: unknown, value: unknown, cb: (error?: Error) => void) => {
      if (!isValidPersonNameFormat(value)) {
        return cb(
          new Error(_t('Person name format must include {FirstName} and/or {LastName}'))
        );
      }
      cb();
    },
    trigger: ['change', 'blur'],
  },
]);
const props = withDefaults(
  defineProps<{ store?: ModelStore<Language>; recordId?: string; viewMode?: ViewMode; showHeader?: boolean; createAction?: string | RouteLocationRaw }>(),
  { showHeader: true, createAction: undefined }
);
const languageActions = defineModelActions('base.Language', { entityTitle: _lt('Language') });
const { hasAction } = usePermission();
const store = resolvePageStore(props.store, 'LanguageFormView');
const { recordId, viewMode, showHeader, createAction } = props;

/** Refresh FE format overlays so list/number displays pick up Language field changes. */
async function onLanguageSaved() {
  try {
    await useI18nStore().loadActiveLanguages();
  } catch {
    // Best-effort; next full page init will reload formats.
  }
}
</script>
