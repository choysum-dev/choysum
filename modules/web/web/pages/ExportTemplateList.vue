<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <ChoyPage v-if="exportTemplateStore" :title="pageTitle" :store="exportTemplateStore">
    <ExportTemplateListView :store="exportTemplateStore" />
  </ChoyPage>
</template>

<script setup lang="ts">
import ExportTemplateListView from '../views/ExportTemplateListView.vue';
import { useScopeManager } from '@/web/web/stores/storeScopeManager';
import { ChoyPage } from '@/web';
import { useExportTemplateListPage } from './export_template_list_page';

defineOptions({ name: 'ExportTemplateListPage' });

const { pageTitle, exportTemplateStore } = useExportTemplateListPage({
  scopeManager: useScopeManager().menuScopeManager,
});
</script>
