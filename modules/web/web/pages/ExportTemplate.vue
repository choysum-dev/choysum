<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <ChoyPage v-if="exportTemplateStore" :store="exportTemplateStore">
    <ExportTemplateFormView
      :store="exportTemplateStore"
      :initial-values="initialValues"
      create-action="/web/export-templates/new"
    />
  </ChoyPage>
</template>

<script setup lang="ts">
import ExportTemplateFormView from '../views/ExportTemplateFormView.vue';
import { useScopeManager } from '@/web/web/stores/storeScopeManager';
import { ChoyPage } from '@/web';
import { useExportTemplateDetailPage } from './export_template_detail_page';

defineOptions({ name: 'ExportTemplatePage' });

const { exportTemplateStore } = useExportTemplateDetailPage({
  scopeManager: useScopeManager().menuScopeManager,
});

// Technical admin create defaults; field tree editing stays on ExportPanel.
const initialValues: Record<string, unknown> = {
  Name: null,
  Application: null,
  ModelName: null,
  Fields: [],
  UserId: null,
  ImportCompatible: false,
};
</script>
