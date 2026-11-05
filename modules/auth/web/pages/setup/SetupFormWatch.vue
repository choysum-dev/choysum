<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<script setup lang="ts">
/**
 * Observes the FormView draft (slot `formData`) for the setup wizard.
 * Emits `change` with the fields whose values changed, and `ready` once with the
 * draft object so the page can read values outside the slot.
 */
import { onMounted, watch } from 'vue';
import { SETUP_STEP_FIELDS } from '../setup_form';

const WATCHED_FIELDS = [...SETUP_STEP_FIELDS.admin, ...SETUP_STEP_FIELDS.organization];

const props = defineProps<{
  formData: Record<string, unknown>;
}>();

const emit = defineEmits<{
  change: [fields: string[]];
  ready: [formData: Record<string, unknown>];
}>();

onMounted(() => emit('ready', props.formData));
watch(
  () => props.formData,
  formData => emit('ready', formData),
);

watch(
  () => WATCHED_FIELDS.map(name => props.formData[name]),
  (next, prev) => {
    const changed = WATCHED_FIELDS.filter((_, i) => next[i] !== prev?.[i]);
    if (changed.length) emit('change', changed);
  },
);
</script>

<template></template>
