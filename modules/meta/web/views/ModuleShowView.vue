<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <!-- Module show is read-only: omit mutating action-ids so Edit/Copy/Delete stay hidden. -->
  <ChoyFormView
    v-bind="{ store, recordId, viewMode: 'display' as const, showHeader }"
  >
    <ChoyCard :title="_t('Overview')" class="mb-3.5">
      <ChoyGrid :cols="12">
        <ChoyCol :span="4">
          <ChoyVarcharField :store="store" prop="ModuleName" />
        </ChoyCol>
        <ChoyCol :span="4">
          <ChoyVarcharField :store="store" prop="Application" />
        </ChoyCol>
        <ChoyCol :span="4">
          <ChoyVarcharField :store="store" prop="Category" />
        </ChoyCol>
      </ChoyGrid>
      <ChoyGrid :cols="12">
        <ChoyCol :span="12">
          <ChoyTextField :store="store" prop="Summary" />
        </ChoyCol>
      </ChoyGrid>
      <ChoyGrid :cols="12">
        <ChoyCol :span="6">
          <ChoyVarcharField :store="store" prop="Author" />
        </ChoyCol>
        <ChoyCol :span="6">
          <ChoyTextField :store="store" prop="License" />
        </ChoyCol>
      </ChoyGrid>
    </ChoyCard>

    <ChoyCard :title="_t('Dependencies')" class="mb-3.5">
      <ChoyGrid :cols="12">
        <ChoyCol :span="6">
          <ChoyJsonField :store="store" prop="DependsJson" />
        </ChoyCol>
        <ChoyCol :span="6">
          <ChoyJsonField :store="store" prop="DependentsJson" />
        </ChoyCol>
      </ChoyGrid>
    </ChoyCard>

    <ChoyCard :title="_t('Versions & Source')" class="mb-3.5">
      <ChoyGrid :cols="12">
        <ChoyCol :span="3">
          <ChoyVarcharField :store="store" prop="InstalledVersion" />
        </ChoyCol>
        <ChoyCol :span="3">
          <ChoyVarcharField :store="store" prop="LocalVersion" />
        </ChoyCol>
        <ChoyCol :span="3">
          <ChoyVarcharField :store="store" prop="RegistryVersion" />
        </ChoyCol>
        <ChoyCol :span="3">
          <ChoyVarcharField :store="store" prop="LocalPath" />
        </ChoyCol>
      </ChoyGrid>
      <ChoyGrid :cols="12">
        <ChoyCol :span="3">
          <ChoyBooleanField :store="store" prop="Installed" />
        </ChoyCol>
        <ChoyCol :span="3">
          <ChoyBooleanField :store="store" prop="UpdateAvailable" />
        </ChoyCol>
        <ChoyCol :span="3">
          <ChoyBooleanField :store="store" prop="Available" />
        </ChoyCol>
        <ChoyCol :span="3">
          <ChoyTextField :store="store" prop="UnavailableReason" />
        </ChoyCol>
      </ChoyGrid>
    </ChoyCard>

    <ChoyCard :title="_t('Sync')" class="mb-3.5">
      <ChoyGrid :cols="12">
        <ChoyCol :span="4">
          <ChoyDatetimeField :store="store" prop="LastSyncAt" />
        </ChoyCol>
        <ChoyCol :span="4">
          <ChoyVarcharField :store="store" prop="ActiveOperationId" />
        </ChoyCol>
        <ChoyCol :span="2">
          <ChoyBooleanField :store="store" prop="HasDemo" />
        </ChoyCol>
        <ChoyCol :span="2">
          <ChoyBooleanField :store="store" prop="DemoLoaded" />
        </ChoyCol>
      </ChoyGrid>
    </ChoyCard>
  </ChoyFormView>
</template>

<script setup lang="ts">
import type { WebModelStore } from '@/web/web/stores/modelStore';
import type MetaModuleCatalog from '@/meta/service/models/module_catalog';
import { resolvePageStore } from '@/web/web/composables/usePageContext';
import { createTranslate } from '@/web/web/i18n';
import {
  ChoyBooleanField,
  ChoyCard,
  ChoyCol,
  ChoyDatetimeField,
  ChoyFormView,
  ChoyGrid,
  ChoyJsonField,
  ChoyTextField,
  ChoyVarcharField,
} from '@/web';

defineOptions({ name: 'ModuleShowView', inheritAttrs: true });

const { _t } = createTranslate('meta', { scope: 'web/views/ModuleShowView' });

const props = withDefaults(
  defineProps<{
    store?: WebModelStore<MetaModuleCatalog>;
    recordId?: string;
    showHeader?: boolean;
  }>(),
  { showHeader: true },
);

const store = resolvePageStore(props.store, 'ModuleShowView');
const { recordId, showHeader } = props;
</script>
