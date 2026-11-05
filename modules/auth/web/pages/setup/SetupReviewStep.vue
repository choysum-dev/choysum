<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <div class="setup-review-step flex flex-col gap-4 text-sm">
    <section class="setup-review-section" data-section="admin">
      <h4 class="setup-review-title">{{ _t('Administrator') }}</h4>
      <dl class="setup-review-list">
        <div v-for="row in adminRows" :key="row.key" class="setup-review-row" :data-field="row.key">
          <dt>{{ row.label }}</dt>
          <dd :class="{ 'text-foreground/50': !row.value }">{{ row.value || _t('Not provided') }}</dd>
        </div>
      </dl>
    </section>
    <section class="setup-review-section" data-section="organization">
      <h4 class="setup-review-title">{{ _t('Organization') }}</h4>
      <dl class="setup-review-list">
        <div v-for="row in orgRows" :key="row.key" class="setup-review-row" :data-field="row.key">
          <dt>{{ row.label }}</dt>
          <dd :class="{ 'text-foreground/50': !row.value }">{{ row.value || _t('Not provided') }}</dd>
        </div>
      </dl>
    </section>
    <p class="setup-review-currency-note text-foreground/70" role="note">
      {{ _t('The base currency is used for accounting and cannot be changed once transactions exist.') }}
    </p>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import { createTranslate } from '@/web/web/i18n';
import type { SetupSelectionItem } from '../setup_options';

const { _t } = createTranslate('auth', { scope: 'web/pages/setup/SetupReviewStep' });

/** Read-only summary of the wizard values before submit. */
const props = defineProps<{
  formData: Record<string, unknown>;
  selections: Record<'CountryCode' | 'CurrencyCode' | 'Timezone' | 'OrgLanguageCode', SetupSelectionItem[]>;
}>();

function text(key: string): string {
  return String(props.formData[key] ?? '').trim();
}

function selectionLabel(key: keyof typeof props.selections): string {
  const value = text(key);
  if (!value) return '';
  return props.selections[key].find(item => item.value === value)?.label ?? value;
}

const adminRows = computed(() => [
  { key: 'Username', label: _t('Username'), value: text('Username') },
  { key: 'Email', label: _t('Email'), value: text('Email') },
]);

const orgRows = computed(() => [
  { key: 'OrgName', label: _t('Organization Name'), value: text('OrgName') },
  { key: 'CountryCode', label: _t('Country / Region'), value: selectionLabel('CountryCode') },
  { key: 'CurrencyCode', label: _t('Base Currency'), value: selectionLabel('CurrencyCode') },
  { key: 'Timezone', label: _t('Business Timezone'), value: selectionLabel('Timezone') },
  { key: 'OrgLanguageCode', label: _t('Organization Language'), value: selectionLabel('OrgLanguageCode') },
]);
</script>

<style scoped>
.setup-review-title {
  margin-bottom: 0.5rem;
  font-weight: 600;
}

.setup-review-list {
  display: flex;
  flex-direction: column;
  gap: 0.375rem;
}

.setup-review-row {
  display: grid;
  grid-template-columns: 8rem minmax(0, 1fr);
  gap: 0.75rem;
}

.setup-review-row dt {
  color: var(--color-muted-foreground);
}

.setup-review-row dd {
  overflow-wrap: anywhere;
}
</style>
