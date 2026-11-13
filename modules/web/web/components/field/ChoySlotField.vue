<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <ChoyFieldBase
    class="choy-slot-field"
    :binding="binding"
    :label="label"
    :help="help"
    :orientation="orientation"
    :hide-label="hideLabel"
    :hide-help="hideHelp"
    :id="id"
    :label-for-control="false"
    render-mode="form"
    :show-inline-error="false"
  >
    <template #edit>
      <div class="choy-slot-field__control min-w-0 w-full">
        <slot />
      </div>
    </template>
  </ChoyFieldBase>
</template>

<script setup lang="ts">
import ChoyFieldBase, { type FieldOrientation } from './ChoyFieldBase.vue';
import { useDetachedField } from '@/web/web/composables/useDetachedField';

defineOptions({ name: 'ChoySlotField' });

const props = withDefaults(
  defineProps<{
    label?: string;
    help?: string;
    orientation?: FieldOrientation;
    hideLabel?: boolean;
    hideHelp?: boolean;
    id?: string;
    /** Synthetic prop for detached binding ids (default `_slot`). */
    prop?: string;
  }>(),
  {
    orientation: 'horizontal',
    hideLabel: false,
    hideHelp: false,
  },
);

// Detached chrome binding; FieldBase generics expect BaseModel — slot fields are untyped.
const binding = useDetachedField({ prop: props.prop ?? '_slot' }) as any;
</script>
