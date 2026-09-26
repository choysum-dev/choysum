<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <ChoyFieldBase
    data-anchor="choy.many-to-one-field"
    :class="props.class"
    :label="label"
    :help="help"
    :required="required"
    :readonly="readonly"
    :disabled="disabled"
    :error="error"
    :name="name"
    :visible="visible"
  >
    <template #default="{ controlId, ariaInvalid, ariaRequired, ariaDescribedby }">
      <RelationCombobox
        v-model="model"
        :id="controlId"
        :search="search"
        :search-key="searchKey"
        :selected-option="selectedOption"
        :page-size="pageSize"
        :search-more="searchMore"
        :placeholder="placeholder"
        :disabled="disabled || readonly"
        :clearable="clearable && !readonly"
        :aria-invalid="ariaInvalid"
        :aria-required="ariaRequired"
        :aria-describedby="ariaDescribedby"
        @search-more="emit('search-more', $event)"
        @search-error="emit('search-error', $event)"
        @select="emit('select', $event)"
      />
    </template>
  </ChoyFieldBase>
</template>

<script setup lang="ts">
import RelationCombobox from '../internal/RelationCombobox.vue';
import type {
  RelationNameSearchFn,
  RelationOption,
} from '../internal/relationComboboxHelpers';
import type { ClassValue } from '../../lib/utils';
import ChoyFieldBase from './ChoyFieldBase.vue';
import {
  choyFieldChromeDefaults,
  type ChoyFieldChromeProps,
} from './fieldHelpers';

/**
 * Many-to-one relation field wrapping L3 RelationCombobox.
 */
const props = withDefaults(
  defineProps<
    ChoyFieldChromeProps & {
      class?: ClassValue;
      placeholder?: string;
      search: RelationNameSearchFn;
      searchKey?: string;
      selectedOption?: RelationOption | null;
      pageSize?: number;
      searchMore?: boolean;
      clearable?: boolean;
    }
  >(),
  {
    ...choyFieldChromeDefaults,
    placeholder: 'Search…',
    selectedOption: null,
    pageSize: 20,
    searchMore: true,
    clearable: true,
  },
);

const model = defineModel<string | null>({ default: null });

const emit = defineEmits<{
  'search-more': [query: string];
  'search-error': [message: string | null];
  select: [option: RelationOption | null];
}>();
</script>
