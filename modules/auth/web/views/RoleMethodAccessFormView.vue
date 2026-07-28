<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <OFormView
    v-bind="{ store, recordId, viewMode, showHeader, createAction }"
    :action-ids="{
      create: methodAccessActions.create,
      edit: methodAccessActions.edit,
      copy: methodAccessActions.copy,
      delete: methodAccessActions.delete,
    }"
    :has-action="hasAction"
    v-on="$attrs"
  >
    <el-card shadow="never" class="mafv-card">
      <template #header
        ><div class="mafv-card__header"><span>{{ _t('Method Access') }}</span></div></template
      >
      <el-alert
        class="mafv-card__hint"
        type="info"
        :closable="false"
        show-icon
        :title="_t('Cross-role method access editor')"
        :description="
          _t(
            'Role is required. New rows default to Mode=deny on the model; prefer allow for grants and deny as an explicit brake. Source is always manual under UI-Option-A (UI grants live in RoleUiResource; do not materialize Method).'
          )
        "
      />
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
          <OManyToOneRefField :store="store" prop="IrApplicationId" />
        </el-col>
        <el-col :xs="24" :sm="12" :md="8" :lg="6" :xl="6">
          <OManyToOneRefField :store="store" prop="IrModelId" />
        </el-col>
        <el-col :xs="24" :sm="12" :md="8" :lg="6" :xl="6">
          <OManyToOneRefField :store="store" prop="IrServiceId" />
        </el-col>
        <el-col :xs="24" :sm="12" :md="8" :lg="6" :xl="6">
          <OSelectionField :store="store" prop="Mode" />
        </el-col>
        <el-col :xs="24" :sm="12" :md="8" :lg="6" :xl="6">
          <OSelectionField :store="store" prop="Source" />
        </el-col>
      </el-row>
    </el-card>

    <el-card shadow="never" class="mafv-card">
      <template #header
        ><div class="mafv-card__header"><span>{{ _t('System Information') }}</span></div></template
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
import type RoleMethodAccess from '@/auth/service/models/role_method_access';
import type Role from '@/auth/service/models/role';
import { ElCard, ElRow, ElCol, ElAlert } from 'element-plus';

import OFormView from '@/web/web/components/view/OFormView.vue';
import ODateTimeField from '@/web/web/components/field/ODatetimeField.vue';
import OSelectionField from '@/web/web/components/field/OSelectionField.vue';
import OManyToOneField from '@/web/web/components/field/OManyToOneField.vue';
import OManyToOneRefField from '@/web/web/components/field/OManyToOneRefField.vue';
import type { ValueClickPayload as ManyToOneValueClickPayload } from '@/web/web/components/field/manyToOneTypes';
import RoleListView from '@/auth/web/views/RoleListView.vue';
import type { ViewMode } from '@/web/web/components/view/OViewScope.vue';
import { defineModelActions } from '@/core/web/resource';
import { usePermission } from '@/auth/web/composables/usePermission';
import { createTranslate } from '@/web/web/i18n';
import { roleIdFromValueClick } from '@/auth/web/views/role_value_click';

defineOptions({ name: 'RoleMethodAccessFormView', inheritAttrs: true });
const { _t, _lt } = createTranslate('auth', { scope: 'web/views/RoleMethodAccessFormView' });

const props = withDefaults(
  defineProps<{
    store: WebModelStore<RoleMethodAccess>;
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

const { store, recordId, viewMode, showHeader, createAction } = props;
const methodAccessActions = defineModelActions('auth.RoleMethodAccess', { entityTitle: _lt('Method Access') });
const { hasAction } = usePermission();
const router = useRouter();

/**
 * Open the referenced role from the method-access form.
 */
function onRoleValueClick(payload: ManyToOneValueClickPayload<Role>) {
  const id = roleIdFromValueClick(payload);
  if (!id) return;
  void router.push(`/auth/roles/${id}`);
}
</script>

<style scoped>
.mafv-card {
  margin-bottom: 14px;
}
.mafv-card__header {
  font-weight: 600;
  color: var(--el-text-color-primary);
}
.mafv-card__hint {
  margin-bottom: 12px;
}
</style>
