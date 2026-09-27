<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <!-- Store id mode (default): OManyToOneRefField. Record mode: OManyToOneField. -->
  <OManyToOneRefField v-if="storeMode && valueMode !== 'record'" v-bind="(storeBind as any)" />
  <OManyToOneField v-else-if="storeMode" v-bind="(storeBind as any)" />
  <ChoyFieldBase
    v-else
    data-anchor="choy.many-to-one-field"
    :class="props.class"
    :label="label"
    :help="help"
    :required="!!required"
    :readonly="!!readonly"
    :disabled="disabled"
    :error="error"
    :name="name"
    :visible="visible"
  >
    <template #default="{ controlId, ariaInvalid, ariaRequired, ariaDescribedby }">
      <RelationCombobox
        v-model="model"
        :id="controlId"
        :search="search!"
        :search-key="searchKey"
        :selected-option="selectedOption"
        :page-size="pageSize"
        :search-more="searchMore"
        :placeholder="placeholder"
        :disabled="disabled || !!readonly"
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
import { computed, useAttrs } from 'vue';
import RelationCombobox from '../internal/RelationCombobox.vue';
import type {
  RelationNameSearchFn,
  RelationOption,
} from '../internal/relationComboboxHelpers';
import type { ClassValue } from '../../lib/utils';
import type { WebModelStore } from '@/web/web/stores/modelStore';
import { isChoyStoreFieldBinding } from '@/web/web/composables/choyStoreMode';
import ChoyFieldBase from './ChoyFieldBase.vue';
import OManyToOneField from './OManyToOneField.vue';
import OManyToOneRefField from './OManyToOneRefField.vue';
import {
  choyFieldChromeDefaults,
  type ChoyFieldChromeProps,
} from './fieldHelpers';

defineOptions({ name: 'ChoyManyToOneField', inheritAttrs: false });

/**
 * Many-to-one field. Store+prop hosts Ref (id) or record O* engines via valueMode.
 * Chrome mode uses RelationCombobox + v-model id.
 */
const props = withDefaults(
  defineProps<
    ChoyFieldChromeProps & {
      class?: ClassValue;
      placeholder?: string;
      search?: RelationNameSearchFn;
      searchKey?: string;
      selectedOption?: RelationOption | null;
      pageSize?: number;
      searchMore?: boolean;
      clearable?: boolean;
      store?: WebModelStore<any>;
      prop?: string;
      binding?: unknown;
      /** `id` → OManyToOneRefField; `record` → OManyToOneField. Default `id`. */
      valueMode?: 'id' | 'record';
    }
  >(),
  {
    ...choyFieldChromeDefaults,
    placeholder: 'Search…',
    selectedOption: null,
    pageSize: 20,
    searchMore: true,
    clearable: true,
    valueMode: 'id',
  },
);

const attrs = useAttrs();
const storeMode = computed(() => isChoyStoreFieldBinding(props));
const storeBind = computed(() => ({ ...attrs, ...props }) as any);

const model = defineModel<string | null>({ default: null });

const emit = defineEmits<{
  'search-more': [query: string];
  'search-error': [message: string | null];
  select: [option: RelationOption | null];
}>();
</script>
