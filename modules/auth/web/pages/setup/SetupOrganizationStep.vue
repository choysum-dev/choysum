<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <ChoyFieldGroup class="setup-fields setup-organization-step" :style="{ gap: '0' }">
    <div class="setup-org-name">
      <ChoyVarcharField
        :store="store"
        prop="OrgName"
        :label="_t('Organization Name')"
        orientation="vertical"
        :placeholder="_t('Enter organization name')"
        autocomplete="organization"
        name="organization"
        id="setup-org-name"
        buffer-strategy="live"
        :nullable="false"
        :show-word-limit="false"
        :required="true"
        show-inline-error
        :rules="orgNameRules"
      />
    </div>
    <div class="setup-country">
      <ChoySelectionField
        :store="store"
        prop="CountryCode"
        :label="_t('Country / Region')"
        orientation="vertical"
        :placeholder="_t('Optional')"
        clearable
        show-inline-error
        :rules="countryRules"
      />
    </div>
    <div class="setup-currency">
      <ChoySelectionField
        :store="store"
        prop="CurrencyCode"
        :label="_t('Base Currency')"
        orientation="vertical"
        :placeholder="_t('Select a base currency')"
        :help="_t('Company accounting currency. It cannot be changed once transactions exist.')"
        :required="true"
        show-inline-error
        :rules="currencyRules"
      />
    </div>
    <div class="setup-timezone">
      <ChoySelectionField
        :store="store"
        prop="Timezone"
        :label="_t('Business Timezone')"
        orientation="vertical"
        :placeholder="_t('Select a business timezone')"
        :help="_t('Used for business dates and scheduled jobs; independent of personal display timezones.')"
        :required="true"
        show-inline-error
        :rules="timezoneRules"
      />
    </div>
    <div class="setup-org-language">
      <ChoySelectionField
        :store="store"
        prop="OrgLanguageCode"
        :label="_t('Organization Language')"
        orientation="vertical"
        :placeholder="_t('Select an organization language')"
        :help="_t('Language used for the organization name and shared business documents.')"
        :required="true"
        show-inline-error
        :rules="orgLanguageRules"
      />
    </div>
  </ChoyFieldGroup>
</template>

<script setup lang="ts">
import { computed, watch } from 'vue';
import { ChoyFieldGroup, ChoySelectionField, ChoyVarcharField, type LocalFormStore } from '@/web';
import { createTranslate } from '@/web/web/i18n';
import {
  setupCurrencyRules,
  setupOrgLanguageRules,
  setupOrgNameRules,
  setupTimezoneRules,
  withSetupServerError,
} from '../setup_form';

const { _t } = createTranslate('auth', { scope: 'web/pages/setup/SetupOrganizationStep' });

/**
 * Organization fields of the setup wizard. Picking a country fills its default
 * currency until the user chooses a currency explicitly (`currencyTouched`).
 */
const props = defineProps<{
  store: LocalFormStore;
  formData: Record<string, unknown>;
  /** Server field errors keyed by local field name. */
  serverErrors: Record<string, string>;
  /** Country code → default currency code. */
  countryCurrencies: Record<string, string>;
  currencyTouched: boolean;
}>();

const emit = defineEmits<{
  /** The user picked a currency by hand; stop following the country default. */
  (e: 'currency-touched'): void;
}>();

const orgNameRules = computed(() => withSetupServerError(setupOrgNameRules(_t), props.serverErrors.OrgName));
const countryRules = computed(() => withSetupServerError([], props.serverErrors.CountryCode));
const currencyRules = computed(() => withSetupServerError(setupCurrencyRules(_t), props.serverErrors.CurrencyCode));
const timezoneRules = computed(() => withSetupServerError(setupTimezoneRules(_t), props.serverErrors.Timezone));
const orgLanguageRules = computed(() =>
  withSetupServerError(setupOrgLanguageRules(_t), props.serverErrors.OrgLanguageCode),
);

let autoCurrency: string | null = null;

function applyCurrency(code: string) {
  autoCurrency = code;
  props.formData.CurrencyCode = code;
  props.store.setField('CurrencyCode', code);
}

watch(
  () => props.formData.CountryCode,
  (country, prev) => {
    if (country === prev || props.currencyTouched) return;
    const code = props.countryCurrencies[String(country ?? '')];
    if (code && code !== props.formData.CurrencyCode) applyCurrency(code);
  },
);

watch(
  () => props.formData.CurrencyCode,
  (currency, prev) => {
    if (currency === prev) return;
    if (autoCurrency !== null && currency === autoCurrency) {
      autoCurrency = null;
      return;
    }
    autoCurrency = null;
    if (!props.currencyTouched) emit('currency-touched');
  },
);
</script>
