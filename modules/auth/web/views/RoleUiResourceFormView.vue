<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <OFormView
    v-bind="{ store, recordId, viewMode, showHeader, createAction }"
    :action-ids="{
      create: uiResourceGrantActions.create,
      edit: uiResourceGrantActions.edit,
      copy: uiResourceGrantActions.copy,
      delete: uiResourceGrantActions.delete,
    }"
    :has-action="hasAction"
    v-on="$attrs"
  >
    <el-card shadow="never" class="urgfv-card">
      <template #header
        ><div class="urgfv-card__header"><span>{{ _t('UI Resource Grant') }}</span></div></template
      >
      <el-alert
        class="urgfv-card__hint"
        type="info"
        :closable="false"
        show-icon
        :title="_t('Cross-role UI resource grant editor')"
        :description="
          _t(
            'Role is required. Prefer the Role form UI tree for day-to-day grants; use this page for cross-role browse and manual bypass rows. Empty Application and UI Resource means global allow/deny for that role.'
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
          <OSelectionField :store="store" prop="Mode" />
        </el-col>
        <el-col :xs="24" :sm="12" :md="8" :lg="6" :xl="6">
          <OManyToOneRefField :store="store" prop="MetaApplicationId" />
        </el-col>
        <el-col :xs="24" :sm="12" :md="8" :lg="6" :xl="6">
          <OManyToOneRefField :store="store" prop="MetaUiResourceId" />
        </el-col>
      </el-row>
    </el-card>

    <el-card shadow="never" class="urgfv-card">
      <template #header
        ><div class="urgfv-card__header"><span>{{ _t('System Information') }}</span></div></template
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
import type RoleUiResource from '@/auth/service/models/role_ui_resource';
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

defineOptions({ name: 'RoleUiResourceFormView', inheritAttrs: true });
const { _t, _lt } = createTranslate('auth', { scope: 'web/views/RoleUiResourceFormView' });

const props = withDefaults(
  defineProps<{
    store: WebModelStore<RoleUiResource>;
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
const uiResourceGrantActions = defineModelActions('auth.RoleUiResource', { entityTitle: _lt('UI Resource Grant') });
const { hasAction } = usePermission();
const router = useRouter();

/**
 * Open the referenced role from the UI-resource grant form.
 */
function onRoleValueClick(payload: ManyToOneValueClickPayload<Role>) {
  const id = roleIdFromValueClick(payload);
  if (!id) return;
  void router.push(`/auth/roles/${id}`);
}
</script>

<style scoped>
.urgfv-card {
  margin-bottom: 14px;
}
.urgfv-card__header {
  font-weight: 600;
  color: var(--el-text-color-primary);
}
.urgfv-card__hint {
  margin-bottom: 12px;
}
</style>
