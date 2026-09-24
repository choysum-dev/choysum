<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<script setup lang="ts">
import { ref, watch } from 'vue';
import Dialog from '../vendor/ui/dialog/Dialog.vue';
import DialogContent from '../vendor/ui/dialog/DialogContent.vue';
import DialogDescription from '../vendor/ui/dialog/DialogDescription.vue';
import DialogTitle from '../vendor/ui/dialog/DialogTitle.vue';
import Input from '../vendor/ui/input/Input.vue';
import ChoyButton from '../layout/ChoyButton.vue';

export type ChoyCompanyValueRow = {
  company: string;
  value: string;
};

/**
 * Dialog for editing per-company field values.
 */
const props = withDefaults(
  defineProps<{
    fieldLabel?: string;
    rows?: ChoyCompanyValueRow[];
  }>(),
  {
    fieldLabel: '',
    rows: () => [],
  },
);

const open = defineModel<boolean>('open', { default: false });

const emit = defineEmits<{
  save: [rows: ChoyCompanyValueRow[]];
}>();

const draft = ref<ChoyCompanyValueRow[]>([]);
/** True until the user edits a draft value while the dialog is open. */
const draftPristine = ref(true);

function seedDraft(): void {
  draft.value = (props.rows ?? []).map((r) => ({
    company: String(r.company ?? ''),
    value: String(r.value ?? ''),
  }));
  draftPristine.value = true;
}

watch(
  open,
  (isOpen) => {
    if (isOpen) {
      seedDraft();
    }
  },
  { immediate: true },
);

watch(
  () => props.rows,
  () => {
    if (open.value && draftPristine.value) {
      seedDraft();
    }
  },
);

function onDraftInput(): void {
  draftPristine.value = false;
}

function onSave(): void {
  emit(
    'save',
    draft.value.map((r) => ({ company: r.company, value: r.value })),
  );
  open.value = false;
}

function onCancel(): void {
  open.value = false;
}
</script>

<template>
  <Dialog v-model:open="open">
    <DialogContent data-anchor="choy.field-company-values-dialog" class="max-w-md">
      <DialogTitle>
        Company values{{ fieldLabel ? `: ${fieldLabel}` : '' }}
      </DialogTitle>
      <DialogDescription>
        Edit company-specific values for this field.
      </DialogDescription>
      <div class="flex max-h-72 flex-col gap-3 overflow-auto py-2">
        <div
          v-for="(row, index) in draft"
          :key="`${row.company}-${index}`"
          class="grid grid-cols-[8rem_1fr] items-center gap-2"
        >
          <span class="text-sm font-medium text-foreground/70">{{ row.company || '—' }}</span>
          <Input
            v-model="row.value"
            :aria-label="`Value for ${row.company}`"
            @update:model-value="onDraftInput"
          />
        </div>
        <p v-if="!draft.length" class="text-sm text-foreground/60">No company values.</p>
      </div>
      <div class="flex justify-end gap-2">
        <ChoyButton type="button" variant="outline" @click="onCancel">Cancel</ChoyButton>
        <ChoyButton type="button" @click="onSave">Save</ChoyButton>
      </div>
    </DialogContent>
  </Dialog>
</template>
