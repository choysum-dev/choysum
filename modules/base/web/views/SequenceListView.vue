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
    <OVarCharField :store="store" prop="Name" />
    <OVarCharField :store="store" prop="Code" />
    <OManyToOneField :store="store" prop="CompanyId"><OVarCharField :store="store" prop="CompanyId.Name" /></OManyToOneField>
    <OVarCharField :store="store" prop="Prefix" />
    <OVarCharField :store="store" prop="Suffix" />
    <OIntField :store="store" prop="Padding" />
    <OBigintField :store="store" prop="NextNumber" />
    <OBooleanField :store="store" prop="IsActive" />
  </OListView>
</template>

<script setup lang="ts">
import type { WebModelStore } from '@/web/web/stores/modelStore';
import type Sequence from '@/base/service/models/sequence';
import { useRouter } from 'vue-router';
import OListView from '@/web/web/components/view/OListView.vue';
import type { RowEventPayload } from '@/web/web/components/view/listViewTypes';
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
const { _t, _lt } = createTranslate('base', { scope: 'web/views/SequenceListView' });
const props = defineProps<{ store: WebModelStore<Sequence> }>();
const sequenceActions = defineModelActions('base.Sequence', { entityTitle: _lt('Sequence') });
const { hasAction } = usePermission();
const router = useRouter();
function onRowClick(payload: RowEventPayload<Sequence>) {
  router.push(`/base/sequences/${payload.row.Id}`);
}
const { listRef, expose } = useListViewExpose<Sequence>();
defineExpose(expose);
</script>
