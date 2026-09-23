<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<script setup lang="ts">
import { onUnmounted } from 'vue';
import { ToastClose, ToastDescription, ToastProvider, ToastRoot, ToastTitle, ToastViewport } from 'reka-ui';
import { clearToasts, dismiss, useToastStore } from './useToast';

const toasts = useToastStore();

onUnmounted(() => {
  clearToasts();
});
</script>

<template>
  <ToastProvider>
    <ToastRoot
      v-for="item in toasts"
      :key="item.id"
      v-model:open="item.open"
      :duration="item.duration"
      data-slot="toast"
      class="group pointer-events-auto relative flex w-full items-center justify-between gap-2 overflow-hidden rounded-md border border-border bg-background p-4 text-foreground shadow-lg"
      @update:open="(value: boolean) => !value && dismiss(item.id)"
    >
      <div class="grid gap-1">
        <ToastTitle class="text-sm font-semibold">{{ item.title }}</ToastTitle>
        <ToastDescription v-if="item.description" class="text-sm text-foreground/70">
          {{ item.description }}
        </ToastDescription>
      </div>
      <ToastClose
        class="rounded-md p-1 opacity-70 transition-opacity hover:opacity-100 focus:outline-none focus:ring-2 focus:ring-ring"
        aria-label="Close"
      />
    </ToastRoot>
    <ToastViewport
      data-slot="toast-viewport"
      class="fixed bottom-0 right-0 z-[100] flex max-h-screen w-full flex-col-reverse gap-2 p-4 sm:max-w-[420px]"
    />
  </ToastProvider>
</template>
