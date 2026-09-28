<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <ODateField v-if="storeMode" v-bind="(storeBind as any)" value-format="YYYY-MM-DD" />
  <ChoyFieldBase
    v-else
    v-bind="($attrs as any)"
    data-anchor="choy.date-field"
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
      <DatePicker
        v-model="model"
        :id="controlId"
        :placeholder="placeholder"
        :disabled="disabled || readonly"
        :clearable="clearable && !readonly"
        :aria-invalid="ariaInvalid"
        :aria-required="ariaRequired"
        :aria-describedby="ariaDescribedby"
      />
    </template>
  </ChoyFieldBase>
</template>

<script setup lang="ts">
import { useAttrs } from 'vue';
import DatePicker from '../internal/DatePicker.vue';
import type { ClassValue } from '../../lib/utils';
import type { WebModelStore } from '@/web/web/stores/modelStore';
import { useChoyStoreFieldBinding } from '@/web/web/composables/choyStoreMode';
import ChoyFieldBase from './ChoyFieldBase.vue';
import ODateField from './ODateField.vue';
import {
  choyFieldChromeDefaults,
  type ChoyFieldChromeProps,
} from './fieldHelpers';

defineOptions({ name: 'ChoyDateField', inheritAttrs: false });

/**
 * Date field (YYYY-MM-DD). Store+prop hosts ODateField; otherwise L3 DatePicker chrome.
 */
const props = withDefaults(
  defineProps<
    ChoyFieldChromeProps & {
      class?: ClassValue;
      placeholder?: string;
      clearable?: boolean;
      store?: WebModelStore<any>;
      prop?: string;
      binding?: unknown;
      rules?: unknown[];
      vColumnProps?: Record<string, unknown>;
    }
  >(),
  {
    ...choyFieldChromeDefaults,
    placeholder: 'Pick a date',
    clearable: true,
  },
);

const attrs = useAttrs();
const { storeMode, storeBind } = useChoyStoreFieldBinding(props as any, attrs as Record<string, unknown>);

const model = defineModel<string | null>({ default: null });
</script>
