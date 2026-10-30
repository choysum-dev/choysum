<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <ChoyFormView
    v-bind="{ store, recordId, viewMode, showHeader }"
    :action-ids="{ edit: editAction }"
    :has-action="hasAction"
    v-on="$attrs"
    @update-success="onSaved"
  >
    <div data-anchor="choy.form.sheet" class="min-h-full">
      <ChoySection :title="_t('Term')">
        <ChoyGrid :cols="12">
          <ChoyCol :span="6">
            <ChoyVarcharField :store="store" prop="Module" :readonly="true" />
          </ChoyCol>
          <ChoyCol :span="6">
            <ChoyVarcharField :store="store" prop="Lang" :readonly="true" />
          </ChoyCol>
          <ChoyCol :span="12">
            <ChoyVarcharField :store="store" prop="Scope" :readonly="true" />
          </ChoyCol>
          <ChoyCol :span="12">
            <ChoyTextField :store="store" prop="Src" :readonly="true" />
          </ChoyCol>
          <ChoyCol :span="12">
            <ChoyTextField :store="store" prop="Value" />
          </ChoyCol>
          <ChoyCol :span="6">
            <ChoyVarcharField :store="store" prop="Kind" :readonly="true" />
          </ChoyCol>
          <ChoyCol :span="6">
            <ChoyVarcharField :store="store" prop="Source" :readonly="true" />
          </ChoyCol>
          <ChoyCol :span="12">
            <ChoyTextField :store="store" prop="Comments" :readonly="true" />
          </ChoyCol>
        </ChoyGrid>
      </ChoySection>
    </div>
  </ChoyFormView>
</template>

<script setup lang="ts">
import type { WebModelStore } from '@/web/web/stores/modelStore';
import { defineAction } from '@/core/web/resource';
import { usePermission } from '@/auth/web/composables/usePermission';
import { resolvePageStore } from '@/web/web/composables/usePageContext';
import { createTranslate } from '@/web/web/i18n';
import { useI18nStore } from '@/web/web/stores/i18nStore';
import {
  ChoyCol,
  ChoyFormView,
  ChoyGrid,
  ChoySection,
  ChoyTextField,
  ChoyVarcharField,
} from '@/web';
import type { ChoyViewMode as ViewMode } from '@/web';

defineOptions({ name: 'TranslationTermFormView', inheritAttrs: true });

const { _t, _lt } = createTranslate('web', { scope: 'web/views/TranslationTermFormView' });

const props = withDefaults(
  defineProps<{
    store?: WebModelStore<any>;
    application: string;
    recordId?: string;
    viewMode?: ViewMode;
    showHeader?: boolean;
  }>(),
  { showHeader: true }
);

// Shared across apps: extract requires a static resourceId (UI_DECL_005).
const editAction = defineAction('web.action.translation_term_edit', {
  title: _lt('Edit Translation Term'),
  defaultRoles: ['sys.admin'],
});

const store = resolvePageStore(props.store, 'TranslationTermFormView');
const { hasAction } = usePermission();
const { recordId, viewMode, showHeader } = props;

async function onSaved() {
  try {
    await useI18nStore().reloadTerminology();
  } catch {
    /* best-effort SPA catalog refresh */
  }
}
</script>
