<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <Stepper
    :model-value="activeIndex + 1"
    :linear="false"
    data-anchor="choy.stepper"
    :data-readonly="props.readonly ? 'true' : undefined"
    :class="cn('choy-stepper w-full items-start', props.class)"
    @update:model-value="onStepperUpdate"
  >
    <StepperItem
      v-for="(step, index) in props.steps"
      :key="step.value"
      :step="index + 1"
      :disabled="!isSelectable(index)"
      :completed="index < activeIndex"
      :data-step="step.value"
      class="choy-stepper__item relative flex min-w-0 flex-1 flex-col items-center"
    >
      <StepperTrigger class="choy-stepper__trigger">
        <StepperIndicator class="choy-stepper__indicator">
          <Check v-if="index < activeIndex" class="size-4" aria-hidden="true" />
          <span v-else>{{ index + 1 }}</span>
        </StepperIndicator>
        <StepperTitle class="choy-stepper__title">{{ step.label }}</StepperTitle>
        <StepperDescription v-if="step.description" class="choy-stepper__description">
          {{ step.description }}
        </StepperDescription>
      </StepperTrigger>
      <StepperSeparator
        v-if="index < props.steps.length - 1"
        class="choy-stepper__separator absolute h-px"
      />
    </StepperItem>
  </Stepper>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import { Check } from 'lucide-vue-next';
import {
  Stepper,
  StepperDescription,
  StepperIndicator,
  StepperItem,
  StepperSeparator,
  StepperTitle,
  StepperTrigger,
} from '../vendor/ui/stepper';
import { cn, type ClassValue } from '../../lib/utils';

export type ChoyStepperStep = {
  value: string;
  label: string;
  description?: string;
};

/**
 * Horizontal step indicator bound to a step `value` via v-model.
 * With `linear` (default) only earlier steps are clickable, so moving forward
 * stays with the host (validate, then set v-model). `readonly` disables every step.
 */
const props = withDefaults(
  defineProps<{
    steps: readonly ChoyStepperStep[];
    linear?: boolean;
    readonly?: boolean;
    class?: ClassValue;
  }>(),
  {
    linear: true,
    readonly: false,
  },
);

const modelValue = defineModel<string>();

const activeIndex = computed(() => {
  const idx = props.steps.findIndex(step => step.value === modelValue.value);
  return idx >= 0 ? idx : 0;
});

function isSelectable(index: number): boolean {
  if (props.readonly) return false;
  if (!props.linear) return true;
  return index <= activeIndex.value;
}

function onStepperUpdate(next: number | undefined): void {
  if (typeof next !== 'number') return;
  const index = next - 1;
  const step = props.steps[index];
  if (!step || !isSelectable(index)) return;
  modelValue.value = step.value;
}
</script>

<style scoped>
.choy-stepper {
  container-type: inline-size;
}

.choy-stepper :deep(.choy-stepper__trigger) {
  position: relative;
  z-index: 1;
  max-width: 100%;
}

.choy-stepper :deep(.choy-stepper__title) {
  font-size: 0.875rem;
  font-weight: 500;
  color: var(--color-muted-foreground);
  text-align: center;
  overflow-wrap: anywhere;
}

/* Narrow hosts: keep every indicator, but only the active step shows its title and description. */
@container (max-width: 18rem) {
  .choy-stepper :deep([data-state='active'] .choy-stepper__trigger) {
    max-width: none;
  }

  .choy-stepper :deep([data-state='active'] .choy-stepper__title) {
    white-space: nowrap;
  }

  .choy-stepper :deep([data-state='inactive'] .choy-stepper__title),
  .choy-stepper :deep([data-state='completed'] .choy-stepper__title),
  .choy-stepper :deep([data-state='inactive'] .choy-stepper__description),
  .choy-stepper :deep([data-state='completed'] .choy-stepper__description) {
    position: absolute;
    width: 1px;
    height: 1px;
    padding: 0;
    margin: -1px;
    overflow: hidden;
    clip: rect(0, 0, 0, 0);
    white-space: nowrap;
    border: 0;
  }
}

.choy-stepper :deep([data-state='active'] .choy-stepper__title),
.choy-stepper :deep([data-state='completed'] .choy-stepper__title) {
  color: var(--color-foreground);
}

.choy-stepper :deep(.choy-stepper__indicator) {
  width: 2rem;
  height: 2rem;
  border: 1px solid var(--color-border);
  background-color: var(--color-background);
  color: var(--color-muted-foreground);
  font-size: 0.875rem;
  font-weight: 600;
}

.choy-stepper :deep([data-state='active'] .choy-stepper__indicator) {
  border-color: var(--color-primary);
  background-color: var(--color-primary);
  color: var(--color-primary-foreground);
}

.choy-stepper :deep([data-state='completed'] .choy-stepper__indicator) {
  border-color: var(--color-primary);
  background-color: var(--color-background);
  color: var(--color-primary);
}

.choy-stepper :deep(.choy-stepper__separator) {
  top: 1.25rem;
  left: calc(50% + 1.25rem);
  right: calc(-50% + 1.25rem);
  background-color: var(--color-border);
}

.choy-stepper :deep([data-state='completed'] .choy-stepper__separator) {
  background-color: var(--color-primary);
}

.choy-stepper[data-readonly='true'] :deep(.choy-stepper__indicator),
.choy-stepper[data-readonly='true'] :deep(.choy-stepper__separator) {
  opacity: 1;
}
</style>
