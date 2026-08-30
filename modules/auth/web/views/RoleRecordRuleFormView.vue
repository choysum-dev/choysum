<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <OFormView
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
    <el-card shadow="never" class="rrfv-card">
      <template #header
        ><div class="rrfv-card__header"><span>{{ _t('Record Rule') }}</span></div></template
      >
      <RoleRecordRuleAudienceHints />
      <el-row :gutter="12">
        <el-col :xs="24" :sm="12" :md="8" :lg="6" :xl="6">
          <OManyToOneField
            :store="store"
            prop="RoleId"
            :search-view="RoleListView"
            :search-view-title="_t('Select Role')"
            @value-click="onRoleValueClick"
          />
        </el-col>
        <el-col :xs="24" :sm="12" :md="8" :lg="6" :xl="6">
          <OSelectionField :store="store" prop="Kind" />
        </el-col>
        <el-col :xs="24" :sm="12" :md="8" :lg="6" :xl="6">
          <OManyToOneRefField :store="store" prop="MetaApplicationId" />
        </el-col>
        <el-col :xs="24" :sm="12" :md="8" :lg="6" :xl="6">
          <OManyToOneRefField :store="store" prop="MetaModelId" />
        </el-col>
        <el-col :span="24">
          <OJsonobjectField :store="store" prop="Condition" :allow-array="true" />
        </el-col>
        <el-col :xs="24" :sm="12" :md="8" :lg="6" :xl="6">
          <OBooleanField :store="store" prop="PermRead" />
        </el-col>
        <el-col :xs="24" :sm="12" :md="8" :lg="6" :xl="6">
          <OBooleanField :store="store" prop="PermWrite" />
        </el-col>
        <el-col :xs="24" :sm="12" :md="8" :lg="6" :xl="6">
          <OBooleanField :store="store" prop="PermCreate" />
        </el-col>
        <el-col :xs="24" :sm="12" :md="8" :lg="6" :xl="6">
          <OBooleanField :store="store" prop="PermDelete" />
        </el-col>
      </el-row>
    </el-card>

    <el-card shadow="never" class="rrfv-card">
      <template #header
        ><div class="rrfv-card__header"><span>{{ _t('System Information') }}</span></div></template
      >
      <el-row :gutter="12">
        <el-col :xs="24" :sm="12" :md="8" :lg="6" :xl="6">
          <ODateTimeField :store="store" prop="CreatedAt" />
        </el-col>
        <el-col :xs="24" :sm="12" :md="8" :lg="6" :xl="6">
          <ODateTimeField :store="store" prop="UpdatedAt" />
        </el-col>
      </el-row>
    </el-card>
  </OFormView>
</template>

<script setup lang="ts">
import { useRouter } from 'vue-router';
import type { RouteLocationRaw } from 'vue-router';
import type { WebModelStore } from '@/web/web/stores/modelStore';
import type RoleRecordRule from '@/auth/service/models/role_record_rule';
import type Role from '@/auth/service/models/role';
import { ElCard, ElRow, ElCol } from 'element-plus';

import OFormView from '@/web/web/components/view/OFormView.vue';
import OBooleanField from '@/web/web/components/field/OBooleanField.vue';
import ODateTimeField from '@/web/web/components/field/ODatetimeField.vue';
import OJsonobjectField from '@/web/web/components/field/OJsonobjectField.vue';
import OSelectionField from '@/web/web/components/field/OSelectionField.vue';
import OManyToOneField from '@/web/web/components/field/OManyToOneField.vue';
import OManyToOneRefField from '@/web/web/components/field/OManyToOneRefField.vue';
import type { ValueClickPayload as ManyToOneValueClickPayload } from '@/web/web/components/field/manyToOneTypes';
import RoleListView from '@/auth/web/views/RoleListView.vue';
import RoleRecordRuleAudienceHints from '@/auth/web/views/RoleRecordRuleAudienceHints.vue';
import type { ViewMode } from '@/web/web/components/view/OViewScope.vue';
import { defineModelActions } from '@/core/web/resource';
import { usePermission } from '@/auth/web/composables/usePermission';
import { resolvePageStore } from '@/web/web/composables/usePageContext';
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
  color: var(--el-text-color-primary);
}
</style>
