<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <OTextField v-if="storeMode" v-bind="(storeBind as any)" />
  <ChoyFieldBase
    v-else
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
import { computed, useAttrs } from 'vue';
import Textarea from '../vendor/ui/textarea/Textarea.vue';
import type { ClassValue } from '../../lib/utils';
import type { WebModelStore } from '@/web/web/stores/modelStore';
import { isChoyStoreFieldBinding } from '@/web/web/composables/choyStoreMode';
import ChoyFieldBase from './ChoyFieldBase.vue';
import OTextField from './OTextField.vue';
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
const storeMode = computed(() => isChoyStoreFieldBinding(props));
const storeBind = computed(() => ({ ...attrs, ...props }) as any);

const model = defineModel<string>({ default: '' });
</script>
