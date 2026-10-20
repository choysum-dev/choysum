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
          data-anchor="choy.internal.datetime-picker"
          :id="id || undefined"
          :disabled="disabled"
          :aria-invalid="props['aria-invalid']"
          :aria-required="props['aria-required']"
          :aria-describedby="props['aria-describedby']"
          :class="
            cn(
              'choy-datetime-picker w-full min-w-0 justify-between font-normal',
              !displayText && 'text-muted-foreground',
              props.class,
            )
          "
        >
          <span class="min-w-0 flex-1 truncate text-start">{{ displayText || effectivePlaceholder }}</span>
          <span class="inline-flex shrink-0 items-center gap-1">
            <span v-if="showClear" class="size-3.5" aria-hidden="true" />
            <CalendarClock class="size-4 opacity-50" aria-hidden="true" />
          </span>
        </Button>
      </PopoverTrigger>
      <button
        v-if="showClear"
        type="button"
        class="choy-datetime-picker__clear absolute end-9 top-1/2 inline-flex -translate-y-1/2 text-foreground/50 hover:text-foreground"
        :aria-label="_t('Clear')"
        data-testid="choy-datetime-picker-clear"
        @click.stop.prevent="onClear"
      >
        <X class="size-3.5" aria-hidden="true" />
      </button>
    </div>
    <PopoverContent class="w-auto p-0" align="start">
      <div class="flex">
        <div class="p-3">
          <CalendarRoot
            v-slot="{ weekDays, grid }"
            v-model="calendarValue"
            :locale="calendarLocale"
            :min-value="datePickerMinValue"
            :max-value="datePickerMaxValue"
            class="choy-datetime-picker__calendar"
          >
            <CalendarHeader class="flex items-center justify-between gap-2 pb-2">
              <CalendarPrev
                :aria-label="_t('Previous month')"
                class="inline-flex size-8 items-center justify-center rounded-md border border-border hover:bg-muted"
              >
                <ChevronLeft class="size-4" />
              </CalendarPrev>
              <CalendarHeading class="text-sm font-medium" />
              <CalendarNext
                :aria-label="_t('Next month')"
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
                      class="w-8 rounded-md text-[0.7rem] font-normal text-muted-foreground"
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
                        class="inline-flex size-8 items-center justify-center rounded-md hover:bg-muted data-[selected]:bg-primary data-[selected]:text-primary-foreground data-[outside-view]:text-muted-foreground/50"
                      />
                    </CalendarCell>
                  </CalendarGridRow>
                </CalendarGridBody>
              </CalendarGrid>
            </div>
          </CalendarRoot>
        </div>

        <div
          class="flex border-s border-border py-3 pe-2 ps-1"
          data-slot="datetime-picker-time"
        >
          <div
            ref="hourListEl"
            class="choy-datetime-picker__hours flex max-h-[16.5rem] flex-col gap-0.5 overflow-y-auto pe-1"
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
              :data-testid="h === selectedHour ? 'choy-datetime-hour-selected' : undefined"
              :class="
                cn(
                  'inline-flex h-8 w-10 shrink-0 items-center justify-center rounded-md text-sm tabular-nums',
                  h === selectedHour
                    ? 'bg-primary text-primary-foreground'
                    : 'hover:bg-muted',
                )
              "
              @click="onSelectHour(h)"
            >
              {{ String(h).padStart(2, '0') }}
            </button>
          </div>
          <div
            ref="minuteListEl"
            class="choy-datetime-picker__minutes flex max-h-[16.5rem] flex-col gap-0.5 overflow-y-auto ps-1"
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
              :data-testid="m === selectedMinute ? 'choy-datetime-minute-selected' : undefined"
              :class="
                cn(
                  'inline-flex h-8 w-10 shrink-0 items-center justify-center rounded-md text-sm tabular-nums',
                  m === selectedMinute
                    ? 'bg-primary text-primary-foreground'
                    : 'hover:bg-muted',
                )
              "
              @click="onSelectMinute(m)"
            >
              {{ String(m).padStart(2, '0') }}
            </button>
          </div>
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
import { CalendarClock, ChevronLeft, ChevronRight, X } from 'lucide-vue-next';
import { CalendarDate } from '@internationalized/date';
import { cn, type ClassValue } from '../../lib/utils';
import { createTranslate } from '@/web/web/i18n';
import { useI18nStore } from '@/web/web/stores/i18nStore';
import Button from '../vendor/ui/button/Button.vue';
import Popover from '../vendor/ui/popover/Popover.vue';
import PopoverContent from '../vendor/ui/popover/PopoverContent.vue';
import PopoverTrigger from '../vendor/ui/popover/PopoverTrigger.vue';
import {
  clearDateTimePickerValue,
  formatDateTimePickerDisplay,
  nowDateTimePickerValue,
  parseDateTimePickerValue,
  withDateTimePickerDate,
  withDateTimePickerTime,
} from './dateTimePickerHelpers';

const { _t } = createTranslate('web', { scope: 'web/components/internal/DateTimePicker' });

/** Reka Calendar / Intl locale; falls back when Pinia is unavailable (FE unit mounts). */
function resolveCalendarLocale(): string {
  try {
    return String(useI18nStore().localeCode || '').trim() || 'en';
  } catch {
    return 'en';
  }
}

const calendarLocale = computed(() => resolveCalendarLocale());

const datePickerMinValue = new CalendarDate(1, 1, 1);
const datePickerMaxValue = new CalendarDate(9999, 12, 31);

const hours = Array.from({ length: 24 }, (_, i) => i);
const minutes = Array.from({ length: 60 }, (_, i) => i);

/**
 * L3 datetime picker: Popover + Calendar + hour/minute columns.
 * Model is YYYY-MM-DDTHH:mm string or null (local wall clock).
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

const effectivePlaceholder = computed(() => props.placeholder || _t('Select date and time'));
const showClear = computed(
  () => props.clearable && !!String(modelValue.value ?? '').trim() && !props.disabled,
);
const parsed = computed(() => parseDateTimePickerValue(modelValue.value));

const selectedHour = computed(() => parsed.value?.hour ?? 0);
const selectedMinute = computed(() => parsed.value?.minute ?? 0);

const calendarValue = computed<DateValue | undefined>({
  get() {
    const p = parsed.value;
    if (!p) return undefined;
    return new CalendarDate(p.year, p.month, p.day);
  },
  set(next) {
    if (!next) {
      modelValue.value = clearDateTimePickerValue();
      return;
    }
    const nextValue = withDateTimePickerDate(modelValue.value, {
      year: next.year,
      month: next.month,
      day: next.day,
    });
    if (!nextValue) return;
    modelValue.value = nextValue;
    // Keep the panel open so the user can adjust time after picking a day.
  },
});

const displayText = computed(() => {
  const formatted = formatDateTimePickerDisplay(modelValue.value);
  if (formatted) return formatted;
  return String(modelValue.value ?? '').trim();
});

function scrollSelectedIntoView(): void {
  const hourBtn = hourListEl.value?.querySelector('[data-selected="true"]') as HTMLElement | null;
  const minuteBtn = minuteListEl.value?.querySelector('[data-selected="true"]') as HTMLElement | null;
  hourBtn?.scrollIntoView({ block: 'center' });
  minuteBtn?.scrollIntoView({ block: 'center' });
}

function onSelectHour(hour: number): void {
  const next = withDateTimePickerTime(modelValue.value, { hour });
  if (next) modelValue.value = next;
}

function onSelectMinute(minute: number): void {
  const next = withDateTimePickerTime(modelValue.value, { minute });
  if (next) modelValue.value = next;
}

function onClear(): void {
  modelValue.value = clearDateTimePickerValue();
  open.value = false;
}

function onNow(): void {
  modelValue.value = nowDateTimePickerValue();
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
