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
    <!--
      Idle mirrors ChoyPagination (one clickable value / total cluster).
      Edit only swaps the current digit into an input; / total stays visible.
    -->
    <div
      class="choy-form-pager__position inline-flex items-center gap-1 whitespace-nowrap text-sm tabular-nums text-muted-foreground"
      aria-live="polite"
    >
      <Input
        v-if="editing"
        ref="currentInputRef"
        type="text"
        inputmode="numeric"
        class="choy-form-pager__current-input box-border h-control w-auto shrink-0 bg-background px-1.5 text-center text-sm tabular-nums shadow-none focus-visible:border-ring focus-visible:ring-0 focus-visible:ring-offset-0"
        :style="{ width: currentInputWidth }"
        :disabled="busy"
        :aria-label="currentEditAriaLabel"
        data-testid="choy-form-pager-current-input"
        v-model="editDraft"
        @keydown="onEditKeydown"
        @blur="commitEdit"
      />
      <button
        v-else
        ref="currentIdleRef"
        type="button"
        class="choy-form-pager__current-idle inline-flex h-control w-auto shrink-0 items-center gap-1 rounded-md border border-transparent bg-transparent px-1 text-sm tabular-nums focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring disabled:pointer-events-none disabled:opacity-50"
        :disabled="busy || normalizedTotal <= 0"
        :aria-label="currentEditAriaLabel"
        data-testid="choy-form-pager-current"
        @click="startEdit"
      >
        <span
          class="choy-form-pager__current-value whitespace-nowrap text-foreground underline decoration-muted-foreground/60 underline-offset-4 hover:decoration-foreground"
        >
          {{ currentLabel }}
        </span>
        <span aria-hidden="true">/</span>
        <span class="choy-form-pager__total tabular-nums">{{ totalLabel }}</span>
      </button>
      <template v-if="editing">
        <span aria-hidden="true">/</span>
        <span class="choy-form-pager__total tabular-nums">{{ totalLabel }}</span>
      </template>
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
import { computed, nextTick, ref } from 'vue';
import { ChevronLeft, ChevronRight } from 'lucide-vue-next';
import ChoyButton from '../layout/ChoyButton.vue';
import ChoyButtonGroup from '../layout/ChoyButtonGroup.vue';
import Input from '../vendor/ui/input/Input.vue';
import { createTranslate } from '../../i18n';
import {
  formPagerCurrentInputCh,
  parseFormPagerCurrentDraft,
  resolvePagerInputEl,
} from './formPagerHelpers';

const { _t } = createTranslate('web', { scope: 'web/components/view/ChoyFormPager' });

/**
 * Form collection pager: editable 1-based current index + total + prev/next.
 * Visual chrome mirrors ChoyPagination so List↔Form action-bar right edge stays calm.
 */
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
  /** 1-based position within the collection. */
  goto: [position: number];
}>();

const editing = ref(false);
const editDraft = ref('');
const currentInputRef = ref<{ $el?: HTMLInputElement } | HTMLInputElement | null>(null);
const currentIdleRef = ref<HTMLButtonElement | null>(null);
const suppressBlurCommit = ref(false);

function currentInputEl(): HTMLInputElement | null {
  return resolvePagerInputEl(currentInputRef.value);
}

const normalizedCurrent = computed(() => {
  const n = Math.trunc(Number(props.current) || 0);
  return Math.max(0, n);
});
const normalizedTotal = computed(() => {
  const n = Math.trunc(Number(props.total) || 0);
  return Math.max(0, n);
});
const currentLabel = computed(() => String(normalizedCurrent.value));
const totalLabel = computed(() => String(normalizedTotal.value));
const idleClusterLabel = computed(() => `${currentLabel.value} / ${totalLabel.value}`);
const currentInputWidth = computed(() => {
  const ch = formPagerCurrentInputCh({
    draft: editDraft.value,
    currentLabel: currentLabel.value,
    totalLabel: totalLabel.value,
  });
  return `${ch}ch`;
});
const currentEditLabel = computed(() => _t('Go to record'));
const currentEditAriaLabel = computed(
  () => `${currentEditLabel.value}: ${idleClusterLabel.value}`,
);
const prevLabel = computed(() => _t('Previous record'));
const nextLabel = computed(() => _t('Next record'));

async function startEdit(): Promise<void> {
  if (props.busy || normalizedTotal.value <= 0) return;
  editing.value = true;
  editDraft.value = currentLabel.value;
  await nextTick();
  const el = currentInputEl();
  if (el && typeof el.focus === 'function') el.focus();
  if (el && typeof el.select === 'function') el.select();
}

async function startEditWith(draft: string): Promise<void> {
  await startEdit();
  editDraft.value = draft;
}

async function focusCurrentIdle(): Promise<void> {
  await nextTick();
  const btn = currentIdleRef.value;
  if (btn && typeof btn.focus === 'function') btn.focus();
}

function cancelEdit(): void {
  suppressBlurCommit.value = true;
  editing.value = false;
  editDraft.value = '';
  queueMicrotask(() => {
    suppressBlurCommit.value = false;
  });
  // Escape closes the input; restore focus to the idle control.
  void focusCurrentIdle();
}

function commitEdit(opts?: { restoreFocus?: boolean }): void {
  if (suppressBlurCommit.value || !editing.value) return;
  // Enter unmounts the input and can fire blur; ignore that follow-up commit.
  suppressBlurCommit.value = true;
  const draft = editDraft.value;
  editing.value = false;
  editDraft.value = '';
  queueMicrotask(() => {
    suppressBlurCommit.value = false;
  });
  // Blur (Tab / click away) must not steal focus from the user's next target.
  if (opts?.restoreFocus) void focusCurrentIdle();
  if (draft.trim() === currentLabel.value) return;
  const next = parseFormPagerCurrentDraft(draft, normalizedTotal.value);
  if (next == null || next === normalizedCurrent.value) return;
  emit('goto', next);
}

function onEditKeydown(event: KeyboardEvent): void {
  if (event.key === 'Enter') {
    event.preventDefault();
    commitEdit({ restoreFocus: true });
  } else if (event.key === 'Escape') {
    event.preventDefault();
    cancelEdit();
  }
}

defineExpose({
  startEdit,
  startEditWith,
  commitEdit,
  cancelEdit,
  onEditKeydown,
});
</script>
