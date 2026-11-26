<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <div
    v-if="hasIoMenu || $slots.overflow || $slots.default || menuTitle"
    class="choy-page__title-actions flex min-w-0 shrink-0 items-center gap-2"
  >
    <ChoyPageIoMenu
      v-if="hasIoMenu || $slots.overflow || menuTitle"
      :action-import="hasIoMenu ? actionImport : false"
      :action-export="hasIoMenu ? actionExport : false"
      :action-import-upload-hint="hasIoMenu ? actionImportUploadHint : undefined"
      :action-import-column-mapping="hasIoMenu ? actionImportColumnMapping : undefined"
      :action-list-ref="hasIoMenu ? actionListRef : undefined"
      :action-company-id="hasIoMenu ? actionCompanyId : undefined"
      :store="store"
    >
      <template v-if="menuTitle" #trigger="{ open, ariaLabel }">
        <button
          type="button"
          data-testid="page-io-menu-trigger"
          data-slot="view-title-menu-trigger"
          class="group inline-flex min-h-control min-w-0 max-w-full items-center gap-1.5 rounded-md px-2.5 text-start outline-none transition-colors hover:bg-accent focus-visible:ring-2 focus-visible:ring-ring/50"
          :class="open ? 'bg-accent' : undefined"
          :title="menuTitle || undefined"
          :aria-label="ariaLabel"
          :aria-expanded="open"
          :aria-haspopup="true"
        >
          <span
            :id="menuTitleId || undefined"
            data-slot="page-title"
            :class="menuTitleClass"
          >
            {{ menuTitle }}
          </span>
          <ChevronDown
            class="size-3.5 shrink-0 text-muted-foreground transition-[opacity,transform] duration-150 group-hover:opacity-100"
            :class="open ? 'opacity-100 rotate-180' : 'opacity-70'"
            aria-hidden="true"
          />
        </button>
      </template>
      <slot name="overflow" />
    </ChoyPageIoMenu>
    <slot />
  </div>
</template>

<script setup lang="ts">
import { ChevronDown } from 'lucide-vue-next';
import ChoyPageIoMenu from './ChoyPageIoMenu.vue';
import type { PageIoMenuListRef } from './ChoyPageIoMenu.vue';
import type { ClassValue } from '../../lib/utils';

/**
 * Shared title-actions row for ChoyPage and view chrome.
 * `#overflow` items land inside the menu; default slot stays beside the trigger.
 * When `menuTitle` is set (View Action Bar), the title itself is the menu trigger.
 */
defineProps<{
  hasIoMenu: boolean;
  actionImport: boolean;
  actionExport: boolean;
  actionImportUploadHint?: string;
  actionImportColumnMapping?: Record<string, string>;
  actionListRef?: PageIoMenuListRef | null;
  actionCompanyId?: string;
  store?: { storeId?: string; fullModelName?: string; state?: { result?: { total?: number } } };
  /** When set, use the page title as the overflow/IO menu trigger (View chrome). */
  menuTitle?: string | null;
  menuTitleId?: string | null;
  menuTitleClass?: ClassValue;
}>();
</script>
