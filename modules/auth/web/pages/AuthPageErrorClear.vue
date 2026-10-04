<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<script setup lang="ts">
/**
 * Clears a page-level auth error when FormView draft credentials change.
 * Must bind values from the FormView slot `formData` (draft), not createLocalFormStore:
 * field edits write through form-root into the draft, not the store.
 */
import { watchClearPageErrorOnCredentialChange } from './login_form';

const props = defineProps<{
  username?: unknown;
  password?: unknown;
  email?: unknown;
  confirmPassword?: unknown;
  hasError: boolean;
}>();

const emit = defineEmits<{ clear: [] }>();

watchClearPageErrorOnCredentialChange(
  () => [props.username, props.password, props.email, props.confirmPassword] as const,
  {
    getError: () => (props.hasError ? '1' : ''),
    setError: () => emit('clear'),
  },
);
</script>

<template></template>
