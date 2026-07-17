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
    <OManyToOneField :store="store" prop="SequenceId" :label="_t('Sequence')"
      ><OVarCharField :store="store" prop="SequenceId.Name" :label="_t('Sequence')"
    /></OManyToOneField>
    <OVarCharField :store="store" prop="IdempotencyKey" :label="_t('Idempotency Key')" />
    <OIntField :store="store" prop="Count" :label="_t('Count')" />
    <OBooleanField :store="store" prop="DryRun" :label="_t('Dry Run')" />
    <OBigintField :store="store" prop="RangeStart" :label="_t('Start')" />
    <OBigintField :store="store" prop="RangeEnd" :label="_t('End')" />
    <ODateTimeField :store="store" prop="ExpiresAt" :label="_t('Expires At')" />
  </OListView>
</template>

<script setup lang="ts">
import type { WebModelStore } from '@/web/web/stores/modelStore';
import type SequenceIdempotency from '@/base/service/models/sequence_idempotency';
import { useRouter } from 'vue-router';
import OListView, { type RowEventPayload } from '@/web/web/components/view/OListView.vue';
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
const { _t } = createTranslate('base', { scope: 'web/views/SequenceIdempotencyListView' });
const props = defineProps<{ store: WebModelStore<SequenceIdempotency> }>();
const sequenceIdempotencyActions = defineModelActions('base.SequenceIdempotency', { entityTitle: 'Sequence Idempotency Record' });
const { hasAction } = usePermission();
const router = useRouter();
function onRowClick(payload: RowEventPayload<SequenceIdempotency>) {
  router.push(`/base/sequence-idempotencies/${payload.row.Id}`);
}
const { listRef, expose } = useListViewExpose<SequenceIdempotency>();
defineExpose(expose);
</script>
