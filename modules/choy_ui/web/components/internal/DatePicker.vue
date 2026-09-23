<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<script setup lang="ts">
import { computed, ref, watch } from 'vue';
import type { DateValue } from 'reka-ui';
import {
  CalendarCell,
  CalendarCellTrigger,
  CalendarGrid,
  CalendarGridBody,
  CalendarGridHead,
  CalendarGridRow,
  CalendarHeadCell,
  CalendarHeader,
  CalendarHeading,
  CalendarNext,
  CalendarPrev,
  CalendarRoot,
} from 'reka-ui';
import { ChevronLeft, ChevronRight } from 'lucide-vue-next';
import { cn, type ClassValue } from '../../lib/utils';
import Button from '../vendor/ui/button/Button.vue';
import Popover from '../vendor/ui/popover/Popover.vue';
import PopoverContent from '../vendor/ui/popover/PopoverContent.vue';
import PopoverTrigger from '../vendor/ui/popover/PopoverTrigger.vue';
import {
  clearDatePickerValue,
  formatDatePickerValue,
  parseDatePickerValue,
  todayDatePickerValue,
} from './datePickerHelpers';

/**
 * L3 date picker built on @internationalized/date + Reka Calendar.
 * Model is YYYY-MM-DD string or null. Not a public Choy* export.
 */
const props = withDefaults(
  defineProps<{
    class?: ClassValue;
    placeholder?: string;
    disabled?: boolean;
    clearable?: boolean;
  }>(),
  {
    placeholder: 'Pick a date',
    disabled: false,
    clearable: true,
  },
);

const modelValue = defineModel<string | null>({ default: null });
const open = ref(false);

const calendarValue = computed<DateValue | undefined>({
  get() {
    return parseDatePickerValue(modelValue.value) ?? undefined;
  },
  set(next) {
    if (!next) {
      modelValue.value = clearDatePickerValue();
      return;
    }
    // CalendarDate is the DateValue we emit from the Gregorian calendar.
    modelValue.value = formatDatePickerValue(next as Parameters<typeof formatDatePickerValue>[0]);
    open.value = false;
  },
});

const displayText = computed(() => {
  const parsed = parseDatePickerValue(modelValue.value);
  return parsed ? formatDatePickerValue(parsed) : '';
});

function onClear(): void {
  modelValue.value = clearDatePickerValue();
}

function onToday(): void {
  modelValue.value = formatDatePickerValue(todayDatePickerValue());
  open.value = false;
}

watch(
  () => props.disabled,
  (disabled) => {
    if (disabled) {
      open.value = false;
    }
  },
);
</script>

<template>
  <Popover v-model:open="open">
    <div class="flex w-full items-center gap-1">
      <PopoverTrigger as-child>
        <Button
          type="button"
          variant="outline"
          data-anchor="choy.internal.date-picker"
          :disabled="disabled"
          :class="
            cn(
              'choy-date-picker w-full justify-start font-normal',
              !displayText && 'text-foreground/50',
              props.class,
            )
          "
        >
          <span>{{ displayText || placeholder }}</span>
        </Button>
      </PopoverTrigger>
      <button
        v-if="clearable && !!modelValue?.trim() && !disabled"
        type="button"
        class="shrink-0 text-xs text-foreground/50 hover:text-foreground"
        aria-label="Clear date"
        @click="onClear"
      >
        Clear
      </button>
    </div>
    <PopoverContent class="w-auto p-3" align="start">
      <CalendarRoot
        v-slot="{ weekDays, grid }"
        v-model="calendarValue"
        class="choy-date-picker__calendar"
      >
        <CalendarHeader class="flex items-center justify-between gap-2 pb-2">
          <CalendarPrev
            class="inline-flex size-8 items-center justify-center rounded-md border border-border hover:bg-muted"
          >
            <ChevronLeft class="size-4" />
          </CalendarPrev>
          <CalendarHeading class="text-sm font-medium" />
          <CalendarNext
            class="inline-flex size-8 items-center justify-center rounded-md border border-border hover:bg-muted"
          >
            <ChevronRight class="size-4" />
          </CalendarNext>
        </CalendarHeader>
        <div class="flex flex-col gap-4">
          <CalendarGrid
            v-for="month in grid"
            :key="month.value.toString()"
            class="w-full border-collapse space-y-1"
          >
            <CalendarGridHead>
              <CalendarGridRow class="flex">
                <CalendarHeadCell
                  v-for="day in weekDays"
                  :key="day"
                  class="w-8 rounded-md text-[0.7rem] font-normal text-foreground/60"
                >
                  {{ day }}
                </CalendarHeadCell>
              </CalendarGridRow>
            </CalendarGridHead>
            <CalendarGridBody>
              <CalendarGridRow
                v-for="(weekDates, index) in month.rows"
                :key="`week-${index}`"
                class="flex w-full"
              >
                <CalendarCell
                  v-for="weekDate in weekDates"
                  :key="weekDate.toString()"
                  :date="weekDate"
                  class="relative p-0 text-center text-sm"
                >
                  <CalendarCellTrigger
                    :day="weekDate"
                    :month="month.value"
                    class="inline-flex size-8 items-center justify-center rounded-md hover:bg-muted data-[selected]:bg-primary data-[selected]:text-primary-foreground data-[outside-view]:text-foreground/30"
                  />
                </CalendarCell>
              </CalendarGridRow>
            </CalendarGridBody>
          </CalendarGrid>
        </div>
      </CalendarRoot>
      <div class="mt-2 flex justify-end border-t border-border pt-2">
        <Button type="button" variant="ghost" size="sm" @click="onToday">Today</Button>
      </div>
    </PopoverContent>
  </Popover>
</template>
