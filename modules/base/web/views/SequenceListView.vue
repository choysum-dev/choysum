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
    :action-ids="{ create: sequenceActions.create, delete: sequenceActions.delete }"
    :has-action="hasAction"
    detail-path="/base/sequences/:id"
  >
    <ChoyTableColumn type="selection" :vColumnProps="{ align: 'center' }" />
    <ChoyTableColumn type="index" :vColumnProps="{ align: 'right' }" />
    <ChoyVarcharField :store="store" prop="Name" />
    <ChoyVarcharField :store="store" prop="Code" />
    <ChoyManyToOneField :store="store" prop="CompanyId"><ChoyVarcharField :store="store" prop="CompanyId.Name" /></ChoyManyToOneField>
    <ChoyVarcharField :store="store" prop="Prefix" />
    <ChoyVarcharField :store="store" prop="Suffix" />
    <ChoyIntField :store="store" prop="Padding" />
    <ChoyBigintField :store="store" prop="NextNumber" />
    <ChoyBooleanField :store="store" prop="IsActive" />
  </ChoyListView>
</template>

<script setup lang="ts">
import type { ModelStore } from '@/web/web/stores/modelStore';
import type Sequence from '@/base/service/models/sequence';
import { useListViewExpose } from '@/web/web/composables/useListView';
import { resolvePageStore } from '@/web/web/composables/usePageContext';
import { defineModelActions } from '@/core/web/resource';
import { usePermission } from '@/auth/web/composables/usePermission';
import { createTranslate } from '@/web/web/i18n';
import { ChoyBooleanField, ChoyListView, ChoyManyToOneField, ChoyNumberField, ChoySearchView, ChoyTableColumn, ChoyVarcharField, ChoyIntField, ChoyBigintField} from '@/web';

defineOptions({ name: 'SequenceListView', inheritAttrs: true });
const { _t, _lt } = createTranslate('base', { scope: 'web/views/SequenceListView' });
const props = defineProps<{ store?: ModelStore<Sequence> }>();
const store = resolvePageStore(props.store, 'SequenceListView');
const sequenceActions = defineModelActions('base.Sequence', { entityTitle: _lt('Sequence') });
const { hasAction } = usePermission();
const { listRef, expose } = useListViewExpose<Sequence>();
defineExpose(expose);
</script>
