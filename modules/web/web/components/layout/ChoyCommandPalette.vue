<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<script setup lang="ts">
import { onMounted, onUnmounted, ref } from 'vue'
import { Search } from 'lucide-vue-next'
import { createTranslate } from '../../i18n'
import ChoyButton from './ChoyButton.vue'
import { ChoyDialog as Dialog, ChoyDialogContent as DialogContent, ChoyDialogDescription as DialogDescription, ChoyDialogHeader as DialogHeader, ChoyDialogTitle as DialogTitle } from './choyDialog'
import { shouldToggleCommandPalette } from './choyCommandPaletteHotkey'

/**
 * Thin Command Palette shell: Ctrl/Cmd+K + header trigger.
 * Uses Dialog chrome for now; swap to Command* when FE SFC props are local.
 */
const { _t } = createTranslate('web', { scope: 'web/components/layout/ChoyCommandPalette' })

const open = ref(false)

function onKeydown(event: KeyboardEvent) {
  if (!shouldToggleCommandPalette(event)) return
  event.preventDefault()
  open.value = !open.value
}

onMounted(() => {
  document.addEventListener('keydown', onKeydown)
})
onUnmounted(() => {
  document.removeEventListener('keydown', onKeydown)
})

defineExpose({ onKeydown })
</script>

<template>
  <ChoyButton
    variant="ghost"
    size="icon"
    type="button"
    :aria-label="_t('Open command palette')"
    data-testid="choy-shell-command-trigger"
    @click="open = true"
  >
    <Search class="size-4" aria-hidden="true" />
  </ChoyButton>
  <Dialog v-model:open="open">
    <DialogContent class="overflow-hidden p-0 sm:max-w-lg" data-testid="choy-shell-command-dialog">
      <DialogHeader class="sr-only">
        <DialogTitle>{{ _t('Command palette') }}</DialogTitle>
        <DialogDescription>{{ _t('Search…') }}</DialogDescription>
      </DialogHeader>
      <div class="flex flex-col">
        <input
          type="search"
          class="border-b border-border bg-transparent px-3 py-3 text-sm outline-none"
          :placeholder="_t('Type a command or search…')"
          :aria-label="_t('Command palette')"
          data-testid="choy-shell-command-input"
        >
        <p class="text-muted-foreground px-3 py-6 text-center text-sm">
          {{ _t('No results') }}
        </p>
      </div>
    </DialogContent>
  </Dialog>
</template>
