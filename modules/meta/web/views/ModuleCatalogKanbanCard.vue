<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <div
    class="module-catalog-card flex flex-col gap-1.5 text-xs"
    data-testid="module-catalog-card"
  >
    <div class="flex flex-col gap-1.5">
      <div class="flex items-center justify-between gap-2">
        <span
          class="truncate text-sm font-medium text-foreground"
          data-testid="module-catalog-card-name"
        >{{ moduleCatalogKanbanCardModuleName(card) }}</span>
        <ChoyBadge
          :variant="statusVariant"
          class="shrink-0 px-1.5 py-0 text-[10px] font-normal"
          data-testid="module-catalog-card-status"
        >
          {{ statusLabel }}
        </ChoyBadge>
      </div>
      <div
        class="truncate text-muted-foreground"
        :title="moduleCatalogKanbanCardVersionLine(card)"
      >{{ moduleCatalogKanbanCardVersionLine(card) }}</div>
      <div
        v-if="moduleCatalogKanbanCardSummary(card)"
        class="line-clamp-2 text-muted-foreground"
      >{{ moduleCatalogKanbanCardSummary(card) }}</div>
      <div
        v-if="moduleCatalogKanbanCardApplication(card)"
        class="truncate text-muted-foreground/80"
        data-testid="module-catalog-card-application"
      >{{ moduleCatalogKanbanCardApplication(card) }}</div>
      <div
        v-if="moduleCatalogKanbanCardAuthor(card) || moduleCatalogKanbanCardLicense(card)"
        class="flex items-center justify-between gap-2 text-muted-foreground/80"
        data-testid="module-catalog-card-attribution"
      >
        <span
          v-if="moduleCatalogKanbanCardAuthor(card)"
          class="min-w-0 truncate"
          :title="moduleCatalogKanbanCardAuthor(card)"
          data-testid="module-catalog-card-author"
        >{{ moduleCatalogKanbanCardAuthor(card) }}</span>
        <span
          v-if="moduleCatalogKanbanCardLicense(card)"
          class="min-w-0 shrink-0 truncate text-right"
          :class="moduleCatalogKanbanCardAuthor(card) ? 'max-w-[45%]' : 'ml-auto'"
          :title="moduleCatalogKanbanCardLicense(card)"
          data-testid="module-catalog-card-license"
        >{{ moduleCatalogKanbanCardLicense(card) }}</span>
      </div>
    </div>
    <div class="flex flex-wrap items-center justify-start gap-1 pt-2">
      <ChoyButton
        v-if="primaryAction"
        type="button"
        size="xs"
        :variant="primaryAction === 'progress' ? 'outline' : 'default'"
        :data-testid="`module-catalog-card-primary-${primaryAction}`"
        @click.stop="emit('primary')"
      >
        {{ primaryLabel }}
      </ChoyButton>
      <ChoyButton
        v-if="showUninstall"
        type="button"
        size="xs"
        variant="outline-destructive"
        data-testid="module-catalog-card-uninstall"
        @click.stop="emit('uninstall')"
      >
        {{ _t('Uninstall') }}
      </ChoyButton>
    </div>
  </div>
</template>

<script setup lang="ts">
import { createTranslate } from '@/web/web/i18n';
import type { ChoyKanbanCard } from '@/web';
import { ChoyBadge, ChoyButton } from '@/web';
import {
  moduleCatalogKanbanCardApplication,
  moduleCatalogKanbanCardAuthor,
  moduleCatalogKanbanCardLicense,
  moduleCatalogKanbanCardModuleName,
  moduleCatalogKanbanCardSummary,
  moduleCatalogKanbanCardVersionLine,
} from './module_catalog_kanban_view';

defineOptions({ name: 'ModuleCatalogKanbanCard', inheritAttrs: false });

const { _t } = createTranslate('meta', { scope: 'web/views/ModuleCatalogKanbanView' });

defineProps<{
  card: ChoyKanbanCard;
  statusLabel: string;
  statusVariant: 'default' | 'secondary' | 'destructive' | 'outline';
  primaryAction: 'install' | 'upgrade' | 'progress' | null;
  primaryLabel: string;
  showUninstall: boolean;
}>();

const emit = defineEmits<{
  primary: [];
  uninstall: [];
}>();
</script>
