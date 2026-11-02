<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <!--
    Display-only Form for meta.MetaModelData. Full external id key is Module.Name.
    Omitting mutating action-ids hides Edit/Create/Copy/Delete (opt-in slots).
    Host loaders own the write path.
  -->
  <ChoyFormView
    v-bind="{ store, recordId, viewMode, showHeader }"
    :show-actions="true"
    v-on="$attrs"
  >
    <div data-anchor="choy.form.sheet" class="flex min-h-full flex-col gap-8">
      <ChoySection :title="_t('External Id')">
        <ChoyGrid :cols="12">
          <ChoyCol :span="6">
            <ChoyVarcharField
              :store="store"
              prop="Module"
              :label="_t('Module')"
              :help="_t('Full key is Module.Name (first dot splits module vs name).')"
            />
          </ChoyCol>
          <ChoyCol :span="6">
            <ChoyVarcharField :store="store" prop="Name" :label="_t('Name')" />
          </ChoyCol>
        </ChoyGrid>
      </ChoySection>

      <ChoySection :title="_t('Target')">
        <ChoyGrid :cols="12">
          <ChoyCol :span="6">
            <ChoyVarcharField :store="store" prop="Application" :label="_t('Application')" />
          </ChoyCol>
          <ChoyCol :span="6">
            <ChoyVarcharField :store="store" prop="ModelName" :label="_t('Model Name')" />
          </ChoyCol>
          <ChoyCol :span="6">
            <ChoyManyToOneField :store="store" prop="ModelId" :label="_t('Model')" />
          </ChoyCol>
          <ChoyCol :span="6">
            <ChoyVarcharField :store="store" prop="ResId" :label="_t('Resource Id')" />
          </ChoyCol>
          <ChoyCol :span="6">
            <ChoyBooleanField
              :store="store"
              prop="NoUpdate"
              :label="_t('No Update')"
              widget="switch"
            />
          </ChoyCol>
        </ChoyGrid>
      </ChoySection>
    </div>
  </ChoyFormView>
</template>

<script setup lang="ts">
import type { WebModelStore } from '@/web/web/stores/modelStore';
import { resolvePageStore } from '@/web/web/composables/usePageContext';
import { createTranslate } from '@/web/web/i18n';
import {
  ChoyBooleanField,
  ChoyCol,
  ChoyFormView,
  ChoyGrid,
  ChoyManyToOneField,
  ChoySection,
  ChoyVarcharField,
  type ChoyViewMode as ViewMode,
} from '@/web';

defineOptions({ name: 'MetaModelDataFormView', inheritAttrs: true });
const { _t } = createTranslate('meta', { scope: 'web/views/MetaModelDataFormView' });

const props = withDefaults(
  defineProps<{
    store?: WebModelStore<any>;
    recordId?: string;
    viewMode?: ViewMode;
    showHeader?: boolean;
  }>(),
  {
    showHeader: true,
    viewMode: 'display',
  }
);

const store = resolvePageStore(props.store, 'MetaModelDataFormView');
const { recordId, viewMode, showHeader } = props;
</script>
