<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<script setup lang="ts">
import Checkbox from '../vendor/ui/checkbox/Checkbox.vue';
import Switch from '../vendor/ui/switch/Switch.vue';
import type { ClassValue } from '../../lib/utils';
import ChoyFieldBase from './ChoyFieldBase.vue';
import {
  choyFieldChromeDefaults,
  type ChoyFieldChromeProps,
} from './fieldHelpers';

/**
 * Boolean field rendered as checkbox (default) or switch.
 */
const props = withDefaults(
  defineProps<
    ChoyFieldChromeProps & {
      class?: ClassValue;
      widget?: 'checkbox' | 'switch';
    }
  >(),
  {
    ...choyFieldChromeDefaults,
    widget: 'checkbox',
  },
);

const model = defineModel<boolean>({ default: false });
</script>

<template>
  <ChoyFieldBase
    data-anchor="choy.boolean-field"
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
    <template #default="{ controlId, ariaInvalid, ariaDescribedby }">
      <div class="flex items-center gap-2">
        <Checkbox
          v-if="widget === 'checkbox'"
          :id="controlId"
          v-model="model"
          :disabled="disabled || readonly"
          :aria-invalid="ariaInvalid"
          :aria-describedby="ariaDescribedby"
        />
        <Switch
          v-else
          :id="controlId"
          v-model="model"
          :disabled="disabled || readonly"
          :aria-invalid="ariaInvalid"
          :aria-describedby="ariaDescribedby"
        />
      </div>
    </template>
  </ChoyFieldBase>
</template>
