<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <ChoyFormView
    v-bind="{ store, recordId, viewMode, showHeader, createAction }"
    :action-ids="{
      create: recordRuleActions.create,
      edit: recordRuleActions.edit,
      copy: recordRuleActions.copy,
      delete: recordRuleActions.delete,
    }"
    :has-action="hasAction"
    v-on="$attrs"
  >
    <ChoyCard :title="_t('Record Rule')" class="rrfv-card">
      <RoleRecordRuleAudienceHints />
      <ChoyGrid :cols="12">
        <ChoyCol :span="6">
          <ChoyManyToOneField
            :store="store"
            prop="RoleId"
            :search-view="RoleListView"
            :search-view-title="_t('Select Role')"
            @value-click="onRoleValueClick"
          />
        </ChoyCol>
        <ChoyCol :span="6">
          <ChoySelectionField :store="store" prop="Kind" />
        </ChoyCol>
        <ChoyCol :span="6">
          <ChoyManyToOneRefField :store="store" prop="MetaApplicationId" />
        </ChoyCol>
        <ChoyCol :span="6">
          <ChoyManyToOneRefField :store="store" prop="MetaModelId" />
        </ChoyCol>
        <ChoyCol :span="12">
          <ChoyJsonField :store="store" prop="Condition" :allow-array="true" />
        </ChoyCol>
        <ChoyCol :span="6">
          <ChoyBooleanField :store="store" prop="PermRead" />
        </ChoyCol>
        <ChoyCol :span="6">
          <ChoyBooleanField :store="store" prop="PermWrite" />
        </ChoyCol>
        <ChoyCol :span="6">
          <ChoyBooleanField :store="store" prop="PermCreate" />
        </ChoyCol>
        <ChoyCol :span="6">
          <ChoyBooleanField :store="store" prop="PermDelete" />
        </ChoyCol>
      </ChoyGrid>
    </ChoyCard>

    <ChoyCard :title="_t('System Information')" class="rrfv-card">
      <ChoyGrid :cols="12">
        <ChoyCol :span="6">
          <ChoyDatetimeField :store="store" prop="CreatedAt" />
        </ChoyCol>
        <ChoyCol :span="6">
          <ChoyDatetimeField :store="store" prop="UpdatedAt" />
        </ChoyCol>
      </ChoyGrid>
    </ChoyCard>
  </ChoyFormView>
</template>

<script setup lang="ts">
import { useRouter } from 'vue-router';
import type { RouteLocationRaw } from 'vue-router';
import type { WebModelStore } from '@/web/web/stores/modelStore';
import type RoleRecordRule from '@/auth/service/models/role_record_rule';
import type Role from '@/auth/service/models/role';

import type { ValueClickPayload as ManyToOneValueClickPayload } from '@/web/web/components/field/manyToOneTypes';
import RoleListView from '@/auth/web/views/RoleListView.vue';
import RoleRecordRuleAudienceHints from '@/auth/web/views/RoleRecordRuleAudienceHints.vue';
import { defineModelActions } from '@/core/web/resource';
import { usePermission } from '@/auth/web/composables/usePermission';
import { resolvePageStore } from '@/web/web/composables/usePageContext';
import { ChoyBooleanField, ChoyCard, ChoyCol, ChoyDatetimeField, ChoyFormView, ChoyGrid, ChoyJsonField, ChoyManyToOneField, ChoySelectionField, ChoyManyToOneRefField} from '@/web';
import type { ChoyViewMode as ViewMode } from '@/web';
import { createTranslate } from '@/web/web/i18n';
import { roleIdFromValueClick } from '@/auth/web/views/role_value_click';

defineOptions({ name: 'RoleRecordRuleFormView', inheritAttrs: true });
const { _t, _lt } = createTranslate('auth', { scope: 'web/views/RoleRecordRuleFormView' });

const props = withDefaults(
  defineProps<{
    store?: WebModelStore<RoleRecordRule>;
    recordId?: string;
    viewMode?: ViewMode;
    showHeader?: boolean;
    createAction?: string | RouteLocationRaw;
  }>(),
  {
    showHeader: true,
    createAction: undefined,
  }
);

const store = resolvePageStore(props.store, 'RoleRecordRuleFormView');
const { recordId, viewMode, showHeader, createAction } = props;
const recordRuleActions = defineModelActions('auth.RoleRecordRule', { entityTitle: _lt('Record Rule') });
const { hasAction } = usePermission();
const router = useRouter();

/**
 * Open the referenced role from the record-rule form.
 */
function onRoleValueClick(payload: ManyToOneValueClickPayload<Role>) {
  const id = roleIdFromValueClick(payload);
  if (!id) return;
  void router.push(`/auth/roles/${id}`);
}
</script>

<style scoped>
.rrfv-card {
  margin-bottom: 14px;
}
.rrfv-card__header {
  font-weight: 600;
  color: var(--choy-foreground, inherit);
}
</style>
