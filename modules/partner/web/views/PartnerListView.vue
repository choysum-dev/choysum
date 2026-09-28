<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <ChoyListView
    ref="listRef"
    v-bind="$attrs"
    :store="store"
    :searchView="ChoySearchView"
    :action-ids="{ create: partnerActions.create, delete: partnerActions.delete }"
    :has-action="hasAction"
    @row-click="onRowClick"
  >
    <ChoyVColumn type="selection" :vColumnProps="{ align: 'center' }" />
    <ChoyVColumn type="index" :vColumnProps="{ align: 'right' }" />

    <ChoyVarcharField :store="store" prop="Name" :vColumnProps="{ minWidth: 180 }" />
    <ChoyVarcharField :store="store" prop="Code" :vColumnProps="{ minWidth: 120 }" />
    <ChoyManyToOneField :store="store" prop="CompanyId" :vColumnProps="{ minWidth: 180 }" />
    <ChoyNumberField :store="store" prop="CustomerRank" mode="integer" />
    <ChoyNumberField :store="store" prop="SupplierRank" mode="integer" />
    <ChoyBooleanField :store="store" prop="IsActive" />
    <ChoyDatetimeField :store="store" prop="UpdatedAt" mode="datetime" :vColumnProps="{ minWidth: 160 }" />
  </ChoyListView>
</template>

<script setup lang="ts">
import { useRouter } from 'vue-router';
import type { WebModelStore } from '@/web/web/stores/modelStore';
import type Partner from '@/partner/service/models/partner';
import { useListViewExpose } from '@/web/web/composables/useListView';
import { resolvePageStore } from '@/web/web/composables/usePageContext';
import { usePermission } from '@/auth/web/composables/usePermission';
import { createTranslate } from '@/web/web/i18n';
import {
  ChoyBooleanField,
  ChoyDatetimeField,
  ChoyListView,
  ChoyManyToOneField,
  ChoyNumberField,
  ChoySearchView,
  ChoyVColumn,
  ChoyVarcharField,
} from '@/web';
import { partnerActions, partnerOpenDetailAction } from './partner_actions';

defineOptions({ name: 'PartnerListView', inheritAttrs: true });
const { _t } = createTranslate('partner', { scope: 'web/views/PartnerListView' });

/**
 * Props consumed by the partner list view.
 * `store` falls back to ChoyPage provided store when omitted.
 */
const props = defineProps<{
  store?: WebModelStore<Partner>;
}>();

const store = resolvePageStore(props.store, 'PartnerListView');

const router = useRouter();

/**
 * Action descriptor used to open the partner detail page from the list.
 */
const { hasAction } = usePermission();

/**
 * Opens the clicked partner row when the actor has detail access.
 */
function onRowClick(row: Partner) {
  if (!hasAction(partnerOpenDetailAction)) {
    return;
  }
  const id = String((row as { Id?: string })?.Id || '').trim();
  if (!id) return;
  router.push(`/partner/partners/${id}`);
}

const { listRef, expose } = useListViewExpose<Partner>();
defineExpose(expose);
</script>
