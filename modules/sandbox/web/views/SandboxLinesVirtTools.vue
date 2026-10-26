<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <div
    class="mb-3 flex w-full min-w-0 flex-col gap-2 sm:flex-row sm:items-center sm:justify-between"
    data-testid="sandbox-lines-virt-tools"
  >
    <p class="text-xs text-muted-foreground sm:max-w-[40rem]">
      {{ _t('Short Lines stay content-sized. Open VIRT-LINES or fill rows to exercise capped table / kanban virtualize (not saved until you Save).') }}
    </p>
    <div v-if="isEdit" class="flex shrink-0 flex-wrap items-center gap-2">
      <ChoyButton
        type="button"
        size="sm"
        variant="outline"
        data-testid="sandbox-lines-fill"
        @click="fillVirtLines"
      >{{ _t('Fill 20 lines') }}</ChoyButton>
      <ChoyButton
        type="button"
        size="sm"
        variant="ghost"
        data-testid="sandbox-lines-clear-filled"
        @click="clearFilledLines"
      >{{ _t('Clear filled lines') }}</ChoyButton>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import type { WebModelStore } from '@/web/web/stores/modelStore';
import type SandboxRecord from '@/sandbox/service/models/sandbox_record';
import ChoyButton from '@/web/web/components/layout/ChoyButton.vue';
import { useField } from '@/web/web/composables/useField';
import { createTranslate } from '@/web/web/i18n';

/** Session fill marker; seed rows use `Virt seed …` and are left alone. */
const VIRT_FILL_PREFIX = 'Virt fill ';

defineOptions({ name: 'SandboxLinesVirtTools' });
const { _t } = createTranslate('sandbox', { scope: 'web/views/SandboxLinesVirtTools' });

const props = defineProps<{
  store: WebModelStore<SandboxRecord>;
}>();

const binding = useField<SandboxRecord, 'Lines'>({ store: props.store, prop: 'Lines' });
binding.registerFields(['Lines.Name', 'Lines.Quantity', 'Lines.State', 'Lines.Sequence']);

const { getItems } = binding.asMutableArray<Record<string, unknown>>();
const isEdit = computed(() => !!binding.env?.isEditMode);

function nextSequenceBase(items: Record<string, unknown>[]): number {
  let max = 0;
  for (const row of items) {
    const n = Number(row?.Sequence);
    if (Number.isFinite(n) && n > max) max = n;
  }
  return max;
}

function fillVirtLines() {
  const items = [...(getItems() || [])];
  let seq = nextSequenceBase(items);
  const start = items.filter((r) => String(r?.Name || '').startsWith(VIRT_FILL_PREFIX)).length;
  for (let i = 0; i < 20; i += 1) {
    seq += 10;
    items.push({
      Name: `${VIRT_FILL_PREFIX}${start + i + 1}`,
      Quantity: 1,
      State: 'todo',
      Sequence: seq,
    });
  }
  (binding.fieldRef() as { value: unknown }).value = items;
}

function clearFilledLines() {
  const kept = (getItems() || []).filter(
    (row) => !String(row?.Name || '').startsWith(VIRT_FILL_PREFIX),
  );
  (binding.fieldRef() as { value: unknown }).value = kept;
}
</script>
