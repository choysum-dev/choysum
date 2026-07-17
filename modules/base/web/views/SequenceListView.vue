<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <OListView
    ref="listRef"
    v-bind="$attrs"
    :store="store"
    :searchView="OSearchView"
    :action-ids="{ create: sequenceActions.create, delete: sequenceActions.delete }"
    :has-action="hasAction"
    @row-click="onRowClick"
  >
    <OVColumn type="selection" :vColumnProps="{ align: 'center' }" />
    <OVColumn type="index" :vColumnProps="{ align: 'right' }" />
    <OVarCharField :store="store" prop="Name" :label="_t('Name')" />
    <OVarCharField :store="store" prop="Code" :label="_t('Code')" />
    <OManyToOneField :store="store" prop="CompanyId" :label="_t('Company')"><OVarCharField :store="store" prop="CompanyId.Name" :label="_t('Company')" /></OManyToOneField>
    <OVarCharField :store="store" prop="Prefix" :label="_t('Prefix')" />
    <OVarCharField :store="store" prop="Suffix" :label="_t('Suffix')" />
    <OIntField :store="store" prop="Padding" :label="_t('Padding Length')" />
    <OBigintField :store="store" prop="NextNumber" :label="_t('Next Number')" />
    <OBooleanField :store="store" prop="IsActive" :label="_t('Active')" />
  </OListView>
</template>

<script setup lang="ts">
import type { WebModelStore } from '@/web/web/stores/modelStore';
import type Sequence from '@/base/service/models/sequence';
import { useRouter } from 'vue-router';
import OListView, { type RowEventPayload } from '@/web/web/components/view/OListView.vue';
import OVColumn from '@/web/web/components/vtable/OVColumn.vue';
import OVarCharField from '@/web/web/components/field/OVarCharField.vue';
import OIntField from '@/web/web/components/field/OIntField.vue';
import OBigintField from '@/web/web/components/field/OBigintField.vue';
import OBooleanField from '@/web/web/components/field/OBooleanField.vue';
import OManyToOneField from '@/web/web/components/field/OManyToOneField.vue';
import OSearchView from '@/web/web/components/view/OSearchView.vue';
import { useListViewExpose } from '@/web/web/composables/useListView';
import { defineModelActions } from '@/core/web/resource';
import { usePermission } from '@/auth/web/composables/usePermission';
import { createTranslate } from '@/web/web/i18n';

defineOptions({ name: 'SequenceListView', inheritAttrs: true });
const { _t } = createTranslate('base', { scope: 'web/views/SequenceListView' });
const props = defineProps<{ store: WebModelStore<Sequence> }>();
const sequenceActions = defineModelActions('base.Sequence', { entityTitle: 'Sequence' });
const { hasAction } = usePermission();
const router = useRouter();
function onRowClick(payload: RowEventPayload<Sequence>) {
  router.push(`/base/sequences/${payload.row.Id}`);
}
const { listRef, expose } = useListViewExpose<Sequence>();
defineExpose(expose);
</script>
