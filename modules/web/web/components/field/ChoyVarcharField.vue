<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <OVarCharField v-if="storeMode" v-bind="(storeBind as any)" />
  <ChoyFieldBase
    v-else
    v-bind="($attrs as any)"
    data-anchor="choy.varchar-field"
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
      <Input
        :id="controlId"
        v-model="model"
        :name="name || undefined"
        :placeholder="placeholder"
        :disabled="disabled"
        :readonly="!!readonly"
        :aria-invalid="ariaInvalid"
        :aria-required="ariaRequired"
        :aria-describedby="ariaDescribedby"
      />
    </template>
  </ChoyFieldBase>
</template>

<script setup lang="ts">
import { computed, useAttrs } from 'vue';
import Input from '../vendor/ui/input/Input.vue';
import type { ClassValue } from '../../lib/utils';
import type { WebModelStore } from '@/web/web/stores/modelStore';
import { isChoyStoreFieldBinding } from '@/web/web/composables/choyStoreMode';
import ChoyFieldBase from './ChoyFieldBase.vue';
import OVarCharField from './OVarCharField.vue';
import {
  choyFieldChromeDefaults,
  type ChoyFieldChromeProps,
} from './fieldHelpers';

defineOptions({ name: 'ChoyVarcharField', inheritAttrs: false });

const props = withDefaults(
  defineProps<
    ChoyFieldChromeProps & {
      class?: ClassValue;
      placeholder?: string;
      store?: WebModelStore<any>;
      prop?: string;
      binding?: unknown;
      rules?: unknown[];
      vColumnProps?: Record<string, unknown>;
      maxLength?: number;
      nullable?: boolean;
      renderMode?: string;
    }
  >(),
  {
    ...choyFieldChromeDefaults,
    placeholder: '',
  },
);

const attrs = useAttrs();
const storeMode = computed(() => isChoyStoreFieldBinding(props));
const storeBind = computed(() => ({ ...attrs, ...props }) as any);

const model = defineModel<string>({ default: '' });
</script>
