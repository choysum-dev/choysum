<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <div
    data-anchor="choy.form-pager"
    data-testid="choy-form-pager"
    class="choy-form-pager flex h-control shrink-0 flex-nowrap items-center gap-2 text-sm text-foreground"
  >
    <div
      class="inline-flex items-center gap-1 whitespace-nowrap text-sm tabular-nums text-muted-foreground"
      aria-live="polite"
    >
      <span class="text-foreground">{{ currentLabel }}</span>
      <span aria-hidden="true">/</span>
      <span>{{ totalLabel }}</span>
    </div>
    <ChoyButtonGroup>
      <ChoyButton
        type="button"
        variant="outline"
        size="icon"
        data-testid="choy-form-pager-prev"
        :disabled="!canPrev || busy"
        :aria-label="prevLabel"
        @click="emit('prev')"
      >
        <ChevronLeft class="size-4 rtl:rotate-180" aria-hidden="true" />
      </ChoyButton>
      <ChoyButton
        type="button"
        variant="outline"
        size="icon"
        data-testid="choy-form-pager-next"
        :disabled="!canNext || busy"
        :aria-label="nextLabel"
        @click="emit('next')"
      >
        <ChevronRight class="size-4 rtl:rotate-180" aria-hidden="true" />
      </ChoyButton>
    </ChoyButtonGroup>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import { ChevronLeft, ChevronRight } from 'lucide-vue-next';
import ChoyButton from '../layout/ChoyButton.vue';
import ChoyButtonGroup from '../layout/ChoyButtonGroup.vue';
import { createTranslate } from '../../i18n';

const { _t } = createTranslate('web', { scope: 'web/components/view/ChoyFormPager' });

const props = withDefaults(
  defineProps<{
    current: number;
    total: number;
    canPrev?: boolean;
    canNext?: boolean;
    busy?: boolean;
  }>(),
  {
    canPrev: false,
    canNext: false,
    busy: false,
  }
);

const emit = defineEmits<{
  prev: [];
  next: [];
}>();

const currentLabel = computed(() => String(Math.max(0, Math.trunc(Number(props.current) || 0))));
const totalLabel = computed(() => String(Math.max(0, Math.trunc(Number(props.total) || 0))));
const prevLabel = computed(() => _t('Previous record'));
const nextLabel = computed(() => _t('Next record'));
</script>
