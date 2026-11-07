<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <ChoySheet v-model:open="open">
    <ChoySheetContent
      side="right"
      class="module-op-drawer w-full sm:max-w-lg"
      data-testid="module-op-drawer"
    >
      <ChoySheetHeader>
        <ChoySheetTitle>{{ titleText }}</ChoySheetTitle>
        <ChoySheetDescription v-if="operationId">
          {{ _t('Operation %s', operationId) }}
        </ChoySheetDescription>
      </ChoySheetHeader>
      <div class="flex min-h-0 flex-1 flex-col gap-3 px-4 pb-4 text-sm text-muted-foreground">
        <p v-if="!operationId && !moduleName">
          {{ _t('Select a module action to plan an operation.') }}
        </p>
        <p v-else-if="moduleName && !operationId">
          {{ _t('Planning for %s…', moduleName) }}
        </p>
        <p v-else>
          {{ _t('Operation progress and results appear here.') }}
        </p>
      </div>
    </ChoySheetContent>
  </ChoySheet>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import { createTranslate } from '@/web/web/i18n';
import {
  ChoySheet,
  ChoySheetContent,
  ChoySheetDescription,
  ChoySheetHeader,
  ChoySheetTitle,
} from '@/web';

defineOptions({ name: 'ModuleOpDrawer' });

const { _t } = createTranslate('meta', { scope: 'web/views/ModuleOpDrawer' });

const open = defineModel<boolean>('open', { default: false });

const props = defineProps<{
  operationId?: string;
  moduleName?: string;
  action?: 'install' | 'upgrade' | 'uninstall' | 'sync' | 'load_demo' | 'progress' | '';
}>();

const titleText = computed(() => {
  const action = String(props.action || '').trim();
  if (action === 'install') return _t('Install Module');
  if (action === 'upgrade') return _t('Upgrade Module');
  if (action === 'uninstall') return _t('Uninstall Module');
  if (action === 'progress') return _t('Operation Progress');
  if (action === 'sync') return _t('Sync Modules');
  if (action === 'load_demo') return _t('Load Demo Data');
  return _t('Module Operation');
});
</script>
