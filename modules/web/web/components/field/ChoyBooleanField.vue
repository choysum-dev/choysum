<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <BooleanField v-if="storeMode" v-bind="(storeBind as any)" />
  <ChoyFieldBase
    v-else
    v-bind="($attrs as any)"
    data-anchor="choy.boolean-field"
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
      <div class="flex items-center gap-2">
        <Checkbox
          v-if="widget === 'checkbox'"
          :id="controlId"
          :model-value="model"
          :disabled="disabled || readonly"
          :aria-invalid="ariaInvalid"
          :aria-required="ariaRequired"
          :aria-describedby="ariaDescribedby"
          @update:model-value="onCheckboxUpdate"
        />
        <Switch
          v-else
          :id="controlId"
          v-model="model"
          :disabled="disabled || readonly"
          :aria-invalid="ariaInvalid"
          :aria-required="ariaRequired"
          :aria-describedby="ariaDescribedby"
        />
      </div>
    </template>
  </ChoyFieldBase>
</template>

<script setup lang="ts">
import { useAttrs } from 'vue';
import Checkbox from '../vendor/ui/checkbox/Checkbox.vue';
import Switch from '../vendor/ui/switch/Switch.vue';
import type { ClassValue } from '../../lib/utils';
import type { WebModelStore } from '@/web/web/stores/modelStore';
import { useChoyStoreFieldBinding } from '@/web/web/composables/choyStoreMode';
import ChoyFieldBase from './ChoyFieldBase.vue';
import BooleanField from './BooleanField.vue';
import {
  choyFieldChromeDefaults,
  type ChoyFieldChromeProps,
} from './fieldHelpers';

defineOptions({ name: 'ChoyBooleanField', inheritAttrs: false });

/**
 * Boolean field. Store+prop hosts OBooleanField; otherwise checkbox/switch chrome.
 */
const props = withDefaults(
  defineProps<
    ChoyFieldChromeProps & {
      class?: ClassValue;
      widget?: 'checkbox' | 'switch';
      store?: WebModelStore<any>;
      prop?: string;
      binding?: unknown;
    }
  >(),
  {
    ...choyFieldChromeDefaults,
    widget: 'checkbox',
  },
);

const attrs = useAttrs();
const { storeMode, storeBind } = useChoyStoreFieldBinding(props as any, attrs as Record<string, unknown>);

const model = defineModel<boolean>({ default: false });

function onCheckboxUpdate(value: boolean | 'indeterminate'): void {
  model.value = value === true;
}
</script>
