<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <FormItem :class="cn('choy-form-item', props.class)">
    <div
      v-if="label || $slots.label || help"
      class="choy-form-item__label inline-flex max-w-full items-center font-normal text-muted-foreground"
    >
      <Label v-if="label || $slots.label" class="min-w-0 text-muted-foreground" :for="controlId || undefined">
        <slot name="label">{{ label }}</slot>
      </Label>
      <button
        v-if="help"
        type="button"
        data-slot="form-help"
        class="m-0 inline-flex cursor-help items-center justify-center border-0 bg-transparent p-0 leading-none text-inherit focus-visible:rounded-sm focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-primary"
        :aria-label="helpLabel || help"
        :title="help"
      >
        <CircleHelp class="size-3.5" />
      </button>
    </div>
    <!-- Do not put controlId on this wrapper — the slotted control owns the id for label association. -->
    <div data-slot="form-control">
      <slot :describedBy="describedBy" :invalid="!!error" />
    </div>
    <p
      v-if="error || reserveError"
      data-slot="form-message"
      :id="messageId || undefined"
      class="text-destructive"
      :class="{ invisible: reserveError && !error }"
      :role="error ? 'alert' : undefined"
      :aria-hidden="reserveError && !error ? true : undefined"
    >
      {{ error || '\u00a0' }}
    </p>
    <p
      v-else-if="description"
      data-slot="form-description"
      :id="descriptionId || undefined"
      class="text-muted-foreground"
    >
      {{ description }}
    </p>
  </FormItem>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import { CircleHelp } from 'lucide-vue-next';
import FormItem from '../vendor/ui/form/FormItem.vue';
import Label from '../vendor/ui/label/Label.vue';
import { cn, type ClassValue } from '../../lib/utils';

defineOptions({ name: 'ChoyFormItem' });

/** Vertical label-above-control; stack gap from --choy-field-stack-gap in index.css. */
const props = defineProps<{
  class?: ClassValue;
  label?: string;
  help?: string;
  helpLabel?: string;
  error?: string;
  /** Keep a reserved error line so sibling fields do not jump when errors appear. */
  reserveError?: boolean;
  description?: string;
  controlId?: string;
}>();

const messageId = computed(() => (props.controlId ? `${props.controlId}-message` : ''));
const descriptionId = computed(() => (props.controlId ? `${props.controlId}-description` : ''));
const describedBy = computed(() => {
  if (props.error && messageId.value) return messageId.value;
  if (!props.error && props.description && descriptionId.value) return descriptionId.value;
  return undefined;
});
</script>
