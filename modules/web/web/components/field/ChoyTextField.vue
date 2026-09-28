<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <TextField v-if="storeMode" v-bind="(storeBind as any)" />
  <ChoyFieldBase
    v-else
    v-bind="($attrs as any)"
    data-anchor="choy.text-field"
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
      <Textarea
        :id="controlId"
        v-model="model"
        :name="name || undefined"
        :placeholder="placeholder"
        :rows="rows"
        :disabled="disabled"
        :readonly="readonly"
        :aria-invalid="ariaInvalid"
        :aria-required="ariaRequired"
        :aria-describedby="ariaDescribedby"
      />
    </template>
  </ChoyFieldBase>
</template>

<script setup lang="ts">
import { useAttrs } from 'vue';
import Textarea from '../vendor/ui/textarea/Textarea.vue';
import type { ClassValue } from '../../lib/utils';
import type { WebModelStore } from '@/web/web/stores/modelStore';
import { useChoyStoreFieldBinding } from '@/web/web/composables/choyStoreMode';
import ChoyFieldBase from './ChoyFieldBase.vue';
import TextField from './TextField.vue';
import {
  choyFieldChromeDefaults,
  type ChoyFieldChromeProps,
} from './fieldHelpers';

defineOptions({ name: 'ChoyTextField', inheritAttrs: false });

/**
 * Multi-line string field. Store+prop hosts OTextField; otherwise chrome v-model.
 */
const props = withDefaults(
  defineProps<
    ChoyFieldChromeProps & {
      class?: ClassValue;
      placeholder?: string;
      rows?: number;
      store?: WebModelStore<any>;
      prop?: string;
      binding?: unknown;
    }
  >(),
  {
    ...choyFieldChromeDefaults,
    placeholder: '',
    rows: 4,
  },
);

const attrs = useAttrs();
const { storeMode, storeBind } = useChoyStoreFieldBinding(props as any, attrs as Record<string, unknown>);

const model = defineModel<string>({ default: '' });
</script>
