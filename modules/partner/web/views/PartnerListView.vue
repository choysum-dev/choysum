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
    detail-path="/partner/partners/:id"
    :can-open-detail="canOpenPartnerDetail"
  >
    <ChoyTableColumn type="selection" :vColumnProps="{ align: 'center' }" />
    <ChoyTableColumn type="index" :vColumnProps="{ align: 'right' }" />

    <ChoyVarcharField :store="store" prop="Name" />
    <ChoyVarcharField :store="store" prop="Code" />
    <ChoyManyToOneRefField :store="store" prop="CompanyId" />
    <ChoyIntField :store="store" prop="CustomerRank" />
    <ChoyIntField :store="store" prop="SupplierRank" />
    <ChoyBooleanField :store="store" prop="IsActive" />
    <ChoyDatetimeField :store="store" prop="UpdatedAt" mode="datetime" />
  </ChoyListView>
</template>

<script setup lang="ts">
import type { ModelStore } from '@/web/web/stores/modelStore';
import type Partner from '@/partner/service/models/partner';
import { useListViewExpose } from '@/web/web/composables/useListView';
import { resolvePageStore } from '@/web/web/composables/usePageContext';
import { usePermission } from '@/auth/web/composables/usePermission';
import { createTranslate } from '@/web/web/i18n';
import {
  ChoyBooleanField,
  ChoyDatetimeField,
  ChoyListView,
  ChoySearchView,
  ChoyTableColumn,
  ChoyVarcharField,
  ChoyIntField,
  ChoyManyToOneRefField,
} from '@/web';
import { partnerActions, partnerOpenDetailAction } from './partner_actions';

defineOptions({ name: 'PartnerListView', inheritAttrs: true });
const { _t } = createTranslate('partner', { scope: 'web/views/PartnerListView' });

const props = defineProps<{
  store?: ModelStore<Partner>;
}>();

const store = resolvePageStore(props.store, 'PartnerListView');
const { hasAction } = usePermission();

function canOpenPartnerDetail() {
  return hasAction(partnerOpenDetailAction);
}

const { listRef, expose } = useListViewExpose<Partner>();
defineExpose(expose);
</script>
