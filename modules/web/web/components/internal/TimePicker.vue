<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <Popover v-model:open="open">
    <div class="relative w-full min-w-0">
      <PopoverTrigger as-child>
        <Button
          type="button"
          variant="outline"
          data-anchor="choy.internal.time-picker"
          :id="id || undefined"
          :disabled="disabled"
          :aria-invalid="props['aria-invalid']"
          :aria-required="props['aria-required']"
          :aria-describedby="props['aria-describedby']"
          :class="
            cn(
              'choy-time-picker w-full min-w-0 justify-between font-normal',
              !displayText && 'text-muted-foreground',
              props.class,
            )
          "
        >
          <span class="min-w-0 flex-1 truncate text-start tabular-nums">{{
            displayText || effectivePlaceholder
          }}</span>
          <span class="inline-flex shrink-0 items-center gap-1">
            <span v-if="showClear" class="size-3.5" aria-hidden="true" />
            <Clock class="size-4 opacity-50" aria-hidden="true" />
          </span>
        </Button>
      </PopoverTrigger>
      <button
        v-if="showClear"
        type="button"
        class="choy-time-picker__clear absolute end-9 top-1/2 inline-flex -translate-y-1/2 text-foreground/50 hover:text-foreground"
        :aria-label="_t('Clear')"
        data-testid="choy-time-picker-clear"
        @click.stop.prevent="onClear"
      >
        <X class="size-3.5" aria-hidden="true" />
      </button>
    </div>
    <PopoverContent class="w-auto p-0" align="start">
      <div class="flex gap-1 px-2 py-3" data-slot="time-picker-columns">
        <div
          ref="hourListEl"
          class="choy-time-picker__hours flex max-h-[16.5rem] flex-col gap-0.5 overflow-y-auto pe-1"
          role="listbox"
          :aria-label="_t('Hour')"
        >
          <button
            v-for="h in hours"
            :key="`h-${h}`"
            type="button"
            role="option"
            :aria-selected="h === selectedHour"
            :data-selected="h === selectedHour ? 'true' : undefined"
            :class="timeItemClass(h === selectedHour)"
            @click="onSelectHour(h)"
          >
            {{ String(h).padStart(2, '0') }}
          </button>
        </div>
        <div
          ref="minuteListEl"
          class="choy-time-picker__minutes flex max-h-[16.5rem] flex-col gap-0.5 overflow-y-auto border-s border-border px-1"
          role="listbox"
          :aria-label="_t('Minute')"
        >
          <button
            v-for="m in minutes"
            :key="`m-${m}`"
            type="button"
            role="option"
            :aria-selected="m === selectedMinute"
            :data-selected="m === selectedMinute ? 'true' : undefined"
            :class="timeItemClass(m === selectedMinute)"
            @click="onSelectMinute(m)"
          >
            {{ String(m).padStart(2, '0') }}
          </button>
        </div>
        <div
          ref="secondListEl"
          class="choy-time-picker__seconds flex max-h-[16.5rem] flex-col gap-0.5 overflow-y-auto border-s border-border ps-1"
          role="listbox"
          :aria-label="_t('Second')"
        >
          <button
            v-for="s in seconds"
            :key="`s-${s}`"
            type="button"
            role="option"
            :aria-selected="s === selectedSecond"
            :data-selected="s === selectedSecond ? 'true' : undefined"
            :class="timeItemClass(s === selectedSecond)"
            @click="onSelectSecond(s)"
          >
            {{ String(s).padStart(2, '0') }}
          </button>
        </div>
      </div>
      <div class="flex items-center justify-between border-t border-border px-3 py-2">
        <Button type="button" variant="ghost" size="sm" @click="onClear">
          {{ _t('Clear') }}
        </Button>
        <Button type="button" variant="ghost" size="sm" @click="onNow">
          {{ _t('Now') }}
        </Button>
      </div>
    </PopoverContent>
  </Popover>
</template>

<script setup lang="ts">
import { computed, nextTick, ref, watch } from 'vue';
import { Clock, X } from 'lucide-vue-next';
import { cn, type ClassValue } from '../../lib/utils';
import { createTranslate } from '@/web/web/i18n';
import Button from '../vendor/ui/button/Button.vue';
import Popover from '../vendor/ui/popover/Popover.vue';
import PopoverContent from '../vendor/ui/popover/PopoverContent.vue';
import PopoverTrigger from '../vendor/ui/popover/PopoverTrigger.vue';
import {
  clearTimePickerValue,
  formatTimePickerDisplay,
  nowTimePickerValue,
  parseTimePickerValue,
  withTimePickerParts,
} from './timePickerHelpers';

const { _t } = createTranslate('web', { scope: 'web/components/internal/TimePicker' });

const hours = Array.from({ length: 24 }, (_, i) => i);
const minutes = Array.from({ length: 60 }, (_, i) => i);
const seconds = Array.from({ length: 60 }, (_, i) => i);

/**
 * L3 time picker: Popover + hour/minute/second columns.
 * Model is HH:mm:ss string or null (local wall clock).
 */
const props = withDefaults(
  defineProps<{
    class?: ClassValue;
    placeholder?: string;
    disabled?: boolean;
    clearable?: boolean;
    id?: string;
    'aria-invalid'?: boolean | 'true' | 'false';
    'aria-required'?: boolean | 'true' | 'false';
    'aria-describedby'?: string;
  }>(),
  {
    placeholder: undefined,
    disabled: false,
    clearable: true,
  },
);

const modelValue = defineModel<string | null>({ default: null });
const open = ref(false);
const hourListEl = ref<HTMLElement | null>(null);
const minuteListEl = ref<HTMLElement | null>(null);
const secondListEl = ref<HTMLElement | null>(null);

const effectivePlaceholder = computed(() => props.placeholder || _t('Select time'));
const showClear = computed(
  () => props.clearable && !!String(modelValue.value ?? '').trim() && !props.disabled,
);
const parsed = computed(() => parseTimePickerValue(modelValue.value));
const selectedHour = computed(() => parsed.value?.hour ?? 0);
const selectedMinute = computed(() => parsed.value?.minute ?? 0);
const selectedSecond = computed(() => parsed.value?.second ?? 0);

const displayText = computed(() => {
  const formatted = formatTimePickerDisplay(modelValue.value);
  if (formatted) return formatted;
  return String(modelValue.value ?? '').trim();
});

function timeItemClass(selected: boolean): string {
  return cn(
    'inline-flex h-8 w-10 shrink-0 items-center justify-center rounded-md text-sm tabular-nums',
    selected ? 'bg-primary text-primary-foreground' : 'hover:bg-muted',
  );
}

function scrollSelectedIntoView(): void {
  for (const root of [hourListEl.value, minuteListEl.value, secondListEl.value]) {
    const btn = root?.querySelector('[data-selected="true"]') as HTMLElement | null;
    btn?.scrollIntoView({ block: 'center' });
  }
}

function onSelectHour(hour: number): void {
  const next = withTimePickerParts(modelValue.value, { hour });
  if (next) modelValue.value = next;
}

function onSelectMinute(minute: number): void {
  const next = withTimePickerParts(modelValue.value, { minute });
  if (next) modelValue.value = next;
}

function onSelectSecond(second: number): void {
  const next = withTimePickerParts(modelValue.value, { second });
  if (next) modelValue.value = next;
}

function onClear(): void {
  modelValue.value = clearTimePickerValue();
  open.value = false;
}

function onNow(): void {
  modelValue.value = nowTimePickerValue();
  open.value = false;
}

watch(open, async (isOpen) => {
  if (!isOpen) return;
  await nextTick();
  scrollSelectedIntoView();
});

watch(
  () => props.disabled,
  (disabled) => {
    if (disabled) open.value = false;
  },
);
</script>
