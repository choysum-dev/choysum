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
    :action-ids="{ create: sequenceIdempotencyActions.create, delete: sequenceIdempotencyActions.delete }"
    :has-action="hasAction"
    @row-click="onRowClick"
  >
    <OVColumn type="selection" :vColumnProps="{ align: 'center' }" />
    <OVColumn type="index" :vColumnProps="{ align: 'right' }" />
    <OManyToOneField :store="store" prop="SequenceId"
      ><OVarCharField :store="store" prop="SequenceId.Name"
    /></OManyToOneField>
    <OVarCharField :store="store" prop="IdempotencyKey" />
    <OIntField :store="store" prop="Count" />
    <OBooleanField :store="store" prop="DryRun" />
    <OBigintField :store="store" prop="RangeStart" />
    <OBigintField :store="store" prop="RangeEnd" />
    <ODateTimeField :store="store" prop="ExpiresAt" />
  </OListView>
</template>

<script setup lang="ts">
import type { WebModelStore } from '@/web/web/stores/modelStore';
import type SequenceIdempotency from '@/base/service/models/sequence_idempotency';
import { useRouter } from 'vue-router';
import OListView from '@/web/web/components/view/OListView.vue';
import type { RowEventPayload } from '@/web/web/components/view/listViewTypes';
import OVColumn from '@/web/web/components/vtable/OVColumn.vue';
import OVarCharField from '@/web/web/components/field/OVarCharField.vue';
import OIntField from '@/web/web/components/field/OIntField.vue';
import OBigintField from '@/web/web/components/field/OBigintField.vue';
import OBooleanField from '@/web/web/components/field/OBooleanField.vue';
import ODateTimeField from '@/web/web/components/field/ODatetimeField.vue';
import OManyToOneField from '@/web/web/components/field/OManyToOneField.vue';
import OSearchView from '@/web/web/components/view/OSearchView.vue';
import { useListViewExpose } from '@/web/web/composables/useListView';
import { defineModelActions } from '@/core/web/resource';
import { usePermission } from '@/auth/web/composables/usePermission';
import { createTranslate } from '@/web/web/i18n';

defineOptions({ name: 'SequenceIdempotencyListView', inheritAttrs: true });
const { _t, _lt } = createTranslate('base', { scope: 'web/views/SequenceIdempotencyListView' });
const props = defineProps<{ store: WebModelStore<SequenceIdempotency> }>();
const sequenceIdempotencyActions = defineModelActions('base.SequenceIdempotency', { entityTitle: _lt('Sequence Idempotency Record') });
const { hasAction } = usePermission();
const router = useRouter();
function onRowClick(payload: RowEventPayload<SequenceIdempotency>) {
  router.push(`/base/sequence-idempotencies/${payload.row.Id}`);
}
const { listRef, expose } = useListViewExpose<SequenceIdempotency>();
defineExpose(expose);
</script>
