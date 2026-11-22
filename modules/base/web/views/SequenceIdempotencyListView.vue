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
    :action-ids="{ create: sequenceIdempotencyActions.create, delete: sequenceIdempotencyActions.delete }"
    :has-action="hasAction"
    detail-path="/base/sequence-idempotencies/:id"
  >
    <ChoyTableColumn type="selection" :vColumnProps="{ align: 'center' }" />
    <ChoyTableColumn type="index" :vColumnProps="{ align: 'right' }" />
    <ChoyManyToOneField :store="store" prop="SequenceId"
      ><ChoyVarcharField :store="store" prop="SequenceId.Name"
    /></ChoyManyToOneField>
    <ChoyVarcharField :store="store" prop="IdempotencyKey" />
    <ChoyIntField :store="store" prop="Count" />
    <ChoyBooleanField :store="store" prop="DryRun" />
    <ChoyBigintField :store="store" prop="RangeStart" />
    <ChoyBigintField :store="store" prop="RangeEnd" />
    <ChoyDatetimeField :store="store" prop="ExpiresAt" />
  </ChoyListView>
</template>

<script setup lang="ts">
import type { ModelStore } from '@/web/web/stores/modelStore';
import type SequenceIdempotency from '@/base/service/models/sequence_idempotency';
import { useListViewExpose } from '@/web/web/composables/useListView';
import { resolvePageStore } from '@/web/web/composables/usePageContext';
import { defineModelActions } from '@/core/web/resource';
import { usePermission } from '@/auth/web/composables/usePermission';
import { createTranslate } from '@/web/web/i18n';
import { ChoyBooleanField, ChoyDatetimeField, ChoyListView, ChoyManyToOneField, ChoyNumberField, ChoySearchView, ChoyTableColumn, ChoyVarcharField, ChoyIntField, ChoyBigintField} from '@/web';

defineOptions({ name: 'SequenceIdempotencyListView', inheritAttrs: true });
const { _t, _lt } = createTranslate('base', { scope: 'web/views/SequenceIdempotencyListView' });
const props = defineProps<{ store?: ModelStore<SequenceIdempotency> }>();
const store = resolvePageStore(props.store, 'SequenceIdempotencyListView');
const sequenceIdempotencyActions = defineModelActions('base.SequenceIdempotency', { entityTitle: _lt('Sequence Idempotency Record') });
const { hasAction } = usePermission();
const { listRef, expose } = useListViewExpose<SequenceIdempotency>();
defineExpose(expose);
</script>
