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
  </ChoyFieldGroup>
</template>

<script setup lang="ts">
import { computed, watch } from 'vue';
import { ChoyFieldGroup, ChoySelectionField, ChoyVarcharField, type LocalFormStore } from '@/web';
import { createTranslate } from '@/web/web/i18n';
import {
  setupCurrencyRules,
  setupOrgNameRules,
  setupTimezoneRules,
  setupRuleTranslate,
  withSetupServerError,
} from '../setup_form';
import { createCurrencyFollower } from '../setup_options';

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

const orgNameRules = computed(() => withSetupServerError(setupOrgNameRules(setupRuleTranslate), props.serverErrors.OrgName));
const countryRules = computed(() => withSetupServerError([], props.serverErrors.CountryCode));
const currencyRules = computed(() => withSetupServerError(setupCurrencyRules(setupRuleTranslate), props.serverErrors.CurrencyCode));
const timezoneRules = computed(() => withSetupServerError(setupTimezoneRules(setupRuleTranslate), props.serverErrors.Timezone));

const currencyFollower = createCurrencyFollower({
  formData: () => props.formData,
  setField: (prop, value) => props.store.setField(prop, value),
  countryCurrencies: () => props.countryCurrencies,
  currencyTouched: () => props.currencyTouched,
  onTouched: () => emit('currency-touched'),
});

watch(() => props.formData.CountryCode, currencyFollower.onCountryChange);
watch(() => props.formData.CurrencyCode, currencyFollower.onCurrencyChange);
</script>
