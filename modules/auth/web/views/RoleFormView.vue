<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <OFormView
    v-bind="{ store, recordId, viewMode, showHeader, createAction }"
    :action-ids="{ create: roleActions.create, edit: roleActions.edit, copy: roleActions.copy, delete: roleActions.delete }"
    :has-action="hasAction"
    v-on="$attrs"
  >
    <el-card shadow="never" class="rfv-card">
      <template #header
        ><div class="rfv-card__header"><span>{{ _t('Basic Information') }}</span></div></template
      >
      <el-row :gutter="12">
        <el-col :xs="24" :sm="12" :md="8" :lg="6" :xl="6">
          <OVarCharField :store="store" prop="Name" :rules="requiredRules" />
        </el-col>
        <el-col :xs="24" :sm="12" :md="8" :lg="6" :xl="6">
          <OVarCharField :store="store" prop="DisplayName" />
        </el-col>
        <el-col :xs="24" :sm="12" :md="8" :lg="6" :xl="6">
          <OVarCharField :store="store" prop="Code" :rules="requiredRules" />
        </el-col>
        <el-col :xs="24" :sm="12" :md="8" :lg="6" :xl="6">
          <OBooleanField :store="store" prop="IsActive" widget="checkbox" />
        </el-col>
        <el-col :xs="24" :sm="12" :md="8" :lg="6" :xl="6">
          <OBooleanField :store="store" prop="IsSystem" widget="checkbox" />
        </el-col>
        <el-col :span="24">
          <OVarCharField :store="store" prop="Description" />
        </el-col>
      </el-row>
    </el-card>

    <el-card shadow="never" class="rfv-card">
      <template #header
        ><div class="rfv-card__header"><span>{{ _t('System Information') }}</span></div></template
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

    <el-card shadow="never" class="rfv-card">
      <template #header
        ><div class="rfv-card__header"><span>{{ _t('Related Data') }}</span></div></template
      >
      <el-tabs v-model="activeTab" type="card" class="rfv-tabs">
        <el-tab-pane :label="_t('Users')" name="users">
          <OManyToManyField :store="store" prop="Users" label="" :search-list="UserListView" :search-view-title="_t('Select User')">
            <OCharField :store="store" prop="Users.Id" />
            <OVarCharField :store="store" prop="Users.Username" />
            <OVarCharField :store="store" prop="Users.FullName" />
          </OManyToManyField>
        </el-tab-pane>
        <el-tab-pane :label="_t('Included Roles')" name="implied_roles">
          <OManyToManyField :store="store" prop="ImpliedRoles" label="" :search-list="RoleListView" :search-view-title="_t('Select Role')">
            <OVarCharField :store="store" prop="ImpliedRoles.Name" />
            <OVarCharField :store="store" prop="ImpliedRoles.Code" />
          </OManyToManyField>
        </el-tab-pane>
        <el-tab-pane :label="_t('UI Resource Access')" name="ui_permissions">
          <OManyToManyRefTreeField
            :store="store"
            prop="AccessUiResourceIds"
            :label="_t('Accessible UI Resources')"
            :lazy="false"
            :max-depth="0"
            children-field="Childs"
            :root-condition="{
              And: [
                ['Type', '=', 'MENU'],
                ['ParentId', 'is', null],
              ],
            }"
            :fields="['Type']"
            :default-expand-all="true"
            :check-strictly="false"
          >
            <template #node="{ row, label }">
              <span class="rfv-ui-resource-node">
                <el-icon class="rfv-ui-resource-node__icon">
                  <component :is="resolveUiResourceTypeIcon(row?.Type)" />
                </el-icon>
                <span class="rfv-ui-resource-node__label">{{ resolveUiResourceLabel(row, label) }}</span>
              </span>
            </template>
          </OManyToManyRefTreeField>
        </el-tab-pane>

        <el-tab-pane :label="_t('Advanced Mode')" name="advanced">
          <el-collapse v-model="advancedPanels" class="rfv-advanced" accordion>
            <el-collapse-item name="record_rules" :title="_t('Record Rules (Manual Maintenance)')">
              <OOneToManyField :store="store" prop="RecordRules" label="">
                <OManyToOneRefField :store="store" prop="RecordRules.IrModelId" />
                <OJsonobjectField :store="store" prop="RecordRules.Condition" :allow-array="true" />
                <OBooleanField :store="store" prop="RecordRules.PermRead" />
                <OBooleanField :store="store" prop="RecordRules.PermWrite" />
                <OBooleanField :store="store" prop="RecordRules.PermCreate" />
                <OBooleanField :store="store" prop="RecordRules.PermDelete" />
              </OOneToManyField>
            </el-collapse-item>

            <el-collapse-item name="field_rules" :title="_t('Field Rules (Manual Maintenance)')">
              <OOneToManyField :store="store" prop="FieldRules" label="">
                <OManyToOneRefField :store="store" prop="FieldRules.IrModelId" />
                <OManyToOneRefField :store="store" prop="FieldRules.IrFieldId" />
                <OSelectionField :store="store" prop="FieldRules.PermRead" />
                <OSelectionField :store="store" prop="FieldRules.PermWrite" />
              </OOneToManyField>
            </el-collapse-item>

            <el-collapse-item name="method_accesses" :title="_t('Method Access (Manual Maintenance)')">
              <OOneToManyField :store="store" prop="MethodAccesses" label="">
                <OManyToOneRefField :store="store" prop="MethodAccesses.IrModelId" />
                <OManyToOneRefField :store="store" prop="MethodAccesses.IrServiceId" />
                <OSelectionField :store="store" prop="MethodAccesses.Mode" />
              </OOneToManyField>
            </el-collapse-item>

            <el-collapse-item name="ui_resources" :title="_t('UI Resource Details (Manual Maintenance)')">
              <OOneToManyField :store="store" prop="UiResources" label="">
                <OSelectionField :store="store" prop="UiResources.Mode" />
                <OManyToOneRefField :store="store" prop="UiResources.IrApplicationId" />
                <OManyToOneRefField :store="store" prop="UiResources.IrUiResourceId" />
              </OOneToManyField>
            </el-collapse-item>
          </el-collapse>
        </el-tab-pane>
      </el-tabs>
    </el-card>
  </OFormView>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue';
import type { RouteLocationRaw } from 'vue-router';
import type { WebModelStore } from '@/web/web/stores/modelStore';
import type Role from '@/auth/service/models/role';

import { ElCard, ElRow, ElCol, ElTabs, ElTabPane, ElCollapse, ElCollapseItem, ElIcon } from 'element-plus';
import { Menu as MenuIcon, Connection, Operation, QuestionFilled } from '@element-plus/icons-vue';

import OFormView from '@/web/web/components/view/OFormView.vue';
import OCharField from '@/web/web/components/field/OCharField.vue';
import OVarCharField from '@/web/web/components/field/OVarCharField.vue';
import OBooleanField from '@/web/web/components/field/OBooleanField.vue';
import ODateTimeField from '@/web/web/components/field/ODatetimeField.vue';
import OManyToManyField from '@/web/web/components/field/OManyToManyField.vue';
import OManyToManyRefTreeField from '@/web/web/components/field/OManyToManyRefTreeField.vue';
import OOneToManyField from '@/web/web/components/field/OOneToManyField.vue';
import OJsonobjectField from '@/web/web/components/field/OJsonobjectField.vue';
import OSelectionField from '@/web/web/components/field/OSelectionField.vue';
import OManyToOneRefField from '@/web/web/components/field/OManyToOneRefField.vue';
import UserListView from './UserListView.vue';
import RoleListView from './RoleListView.vue';
import type { ViewMode } from '@/web/web/components/view/OViewScope.vue';
import { defineModelActions } from '@/core/web/resource';
import { usePermission } from '@/auth/web/composables/usePermission';
import { useI18n } from 'vue-i18n';
import { createTranslate, translateTerm } from '@/web/web/i18n';
import type { TermReference } from '@/core/service/i18n';

defineOptions({ name: 'RoleFormView', inheritAttrs: true });
const { _t, _lt } = createTranslate('auth', { scope: 'web/views/RoleFormView' });
const requiredRules = computed(() => [{ required: true, message: _t('Required') }]);

const props = withDefaults(
  defineProps<{
    store: WebModelStore<Role>;
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
const roleActions = defineModelActions('auth.Role', { entityTitle: _lt('Role') });
const { hasAction } = usePermission();
const composer = useI18n({ useScope: 'global' });

type UiResourceRow = {
  Title?: string;
  TitleText?: TermReference | null;
  Name?: string;
  Id?: string;
};

function resolveUiResourceLabel(row?: UiResourceRow, label?: string) {
  const fallback = String(label || row?.Title || row?.Name || row?.Id || '');
  return translateTerm(composer, row?.TitleText ?? undefined, fallback);
}

/**
 * Resolve the icon used for a UI resource node.
 */
function resolveUiResourceTypeIcon(type?: string) {
  switch (type) {
    case 'MENU':
      return MenuIcon;
    case 'ROUTE':
      return Connection;
    case 'ACTION':
      return Operation;
    default:
      return QuestionFilled;
  }
}

const activeTab = ref('users');
const advancedPanels = ref('');
</script>

<style scoped>
.rfv-card {
  margin-bottom: 14px;
}
.rfv-card__header {
  font-weight: 600;
  color: var(--el-text-color-primary);
}
.rfv-tabs {
  --el-tabs-header-height: 42px;
}

.rfv-advanced {
  margin-top: 4px;
}

.rfv-ui-resource-node {
  display: inline-flex;
  align-items: center;
  gap: 6px;
}

.rfv-ui-resource-node__icon {
  color: var(--el-text-color-secondary);
  font-size: 14px;
}
</style>
