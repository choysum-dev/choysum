<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <ChoyFormView
    v-bind="{ store, recordId, initialValues, viewMode, showHeader, createAction }"
    :action-ids="{
      create: tagActions.create,
      edit: tagActions.edit,
      copy: tagActions.copy,
      delete: tagActions.delete,
    }"
    :has-action="hasAction"
    v-on="$attrs"
  >
    <ChoySection :title="_t('Tag')">
      <ChoyGrid :cols="12">
        <ChoyCol :span="6">
          <ChoyVarcharField :store="store" prop="Name" :rules="requiredRules" />
        </ChoyCol>
        <ChoyCol :span="6">
          <ChoyVarcharField :store="store" prop="Code" :rules="requiredRules" />
        </ChoyCol>
        <ChoyCol :span="6">
          <ChoyBooleanField :store="store" prop="IsActive" />
        </ChoyCol>
      </ChoyGrid>
    </ChoySection>
  </ChoyFormView>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import type { RouteLocationRaw } from 'vue-router';
import type { WebModelStore } from '@/web/web/stores/modelStore';
import type SandboxTag from '@/sandbox/service/models/sandbox_tag';
import { usePermission } from '@/auth/web/composables/usePermission';
import { resolvePageStore } from '@/web/web/composables/usePageContext';
import { createTranslate } from '@/web/web/i18n';
import {
  ChoyBooleanField,
  ChoyCol,
  ChoyFormView,
  ChoyGrid,
  ChoySection,
  ChoyVarcharField,
  type ChoyViewMode as ViewMode,
} from '@/web';
import { sandboxTagActions } from './sandbox_tag_actions';

defineOptions({ name: 'SandboxTagFormView', inheritAttrs: true });
const { _t } = createTranslate('sandbox', { scope: 'web/views/SandboxTagFormView' });
const requiredRules = computed(() => [{ required: true, message: _t('Required') }]);

const props = withDefaults(
  defineProps<{
    store?: WebModelStore<SandboxTag>;
    recordId?: string;
    initialValues?: Partial<SandboxTag>;
    viewMode?: ViewMode;
    showHeader?: boolean;
    createAction?: string | RouteLocationRaw;
  }>(),
  {
    showHeader: true,
    createAction: undefined,
  },
);

const tagActions = sandboxTagActions;
const { hasAction } = usePermission();
const store = resolvePageStore(props.store, 'SandboxTagFormView');
const { recordId, initialValues, viewMode, showHeader, createAction } = props;
</script>
