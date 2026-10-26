<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <ChoyFormView
    v-bind="{ store, recordId, initialValues, viewMode, showHeader }"
    v-on="$attrs"
  >
    <ChoySection :title="_t('Line')">
      <ChoyGrid :cols="12">
        <ChoyCol :span="6">
          <ChoyVarcharField :store="store" prop="Name" :rules="requiredRules" />
        </ChoyCol>
        <ChoyCol :span="3">
          <ChoyIntField :store="store" prop="Quantity" />
        </ChoyCol>
        <ChoyCol :span="3">
          <ChoyIntField :store="store" prop="Sequence" />
        </ChoyCol>
        <ChoyCol :span="6">
          <ChoySelectionField :store="store" prop="State" />
        </ChoyCol>
      </ChoyGrid>
    </ChoySection>
  </ChoyFormView>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import type { WebModelStore } from '@/web/web/stores/modelStore';
import type SandboxLine from '@/sandbox/service/models/sandbox_line';
import { resolvePageStore } from '@/web/web/composables/usePageContext';
import { createTranslate } from '@/web/web/i18n';
import {
  ChoyCol,
  ChoyFormView,
  ChoyGrid,
  ChoyIntField,
  ChoySection,
  ChoySelectionField,
  ChoyVarcharField,
  type ChoyViewMode as ViewMode,
} from '@/web';

defineOptions({ name: 'SandboxLineFormView', inheritAttrs: true });
const { _t } = createTranslate('sandbox', { scope: 'web/views/SandboxLineFormView' });
const requiredRules = computed(() => [{ required: true, message: _t('Required') }]);

const props = withDefaults(
  defineProps<{
    store?: WebModelStore<SandboxLine>;
    recordId?: string;
    initialValues?: Partial<SandboxLine>;
    viewMode?: ViewMode;
    showHeader?: boolean;
  }>(),
  {
    showHeader: false,
  },
);

const store = resolvePageStore(props.store, 'SandboxLineFormView');
const { recordId, initialValues, viewMode, showHeader } = props;
</script>
