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
          <p class="rfv-advanced__hint">
            {{
              _t(
                'Primary path: check resources in this tree. Checking a resource uniformly derives Method allow for its Requires (UI-Option-A; no read/write split). Advanced → UI Resource Details is a manual bypass only.'
              )
            }}
          </p>
          <p class="rfv-advanced__hint rfv-advanced__hint--tight">
            {{ _t('Click a node label to inspect Requires → derived RPCs (checkbox still controls the grant).') }}
          </p>
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
            :fields="['Type', 'Requires']"
            :default-expand-all="true"
            :check-strictly="false"
          >
            <template #node="{ row, label }">
              <span
                class="rfv-ui-resource-node"
                :class="{ 'is-inspected': inspectedUiResourceId === String(row?.Id ?? '') }"
                @click.stop="inspectUiResource(row)"
              >
                <el-icon class="rfv-ui-resource-node__icon">
                  <component :is="resolveUiResourceTypeIcon(row?.Type)" />
                </el-icon>
                <span class="rfv-ui-resource-node__label">{{ resolveUiResourceLabel(row, label) }}</span>
              </span>
            </template>
          </OManyToManyRefTreeField>
          <div v-if="inspectedUiResource" class="rfv-ui-requires">
            <div class="rfv-ui-requires__title">
              {{ _t('Requires → derived Method RPCs') }}
              <span class="rfv-ui-requires__resource">{{ inspectedUiResourceLabel }}</span>
            </div>
            <p class="rfv-advanced__hint rfv-advanced__hint--tight">
              {{
                _t(
                  'Under UI-Option-A, these RPCs are uniformly Method-allow when this resource is granted (unless a manual Method deny brakes them). Record/Field rules are not derived from UI.'
                )
              }}
            </p>
            <ul v-if="inspectedRequires.length > 0" class="rfv-ui-requires__list">
              <li v-for="req in inspectedRequires" :key="req">
                <code>{{ req }}</code>
              </li>
            </ul>
            <p v-else class="rfv-ui-requires__empty">
              {{ _t('No Requires on this resource — granting it does not derive Method access.') }}
            </p>
          </div>
        </el-tab-pane>

        <el-tab-pane :label="_t('Advanced Mode')" name="advanced">
          <p class="rfv-advanced__hint">
            {{
              _t(
                'Configure record/field/RPC grants under deny-default. The UI resource tree does not derive Record or Field rules; Advanced is the main place for data and method access.'
              )
            }}
          </p>
          <el-collapse v-model="advancedPanels" class="rfv-advanced" accordion>
            <el-collapse-item name="record_rules" :title="_t('Record Rules')">
              <el-alert
                class="rfv-rr-alert"
                type="info"
                :closable="false"
                show-icon
                :title="_t('This form only edits rules for this role')"
                :description="
                  _t(
                    'OneToMany rows are always bound to the current role. All-users or cross-role Record/Field/Method/UI rules belong in dedicated menus, not here. Model/Application empty means scope-global (all models), which is not the same as all-users audience.'
                  )
                "
              />
              <p class="rfv-advanced__hint rfv-advanced__hint--tight">
                {{ _t('Without a matching grant, records are invisible or not writable (deny-default).') }}
              </p>
              <OOneToManyField :store="store" prop="RecordRules" label="" :default-record="defaultRecordRule">
                <OSelectionField :store="store" prop="RecordRules.Kind" />
                <OManyToOneRefField :store="store" prop="RecordRules.IrApplicationId" />
                <OManyToOneRefField :store="store" prop="RecordRules.IrModelId" />
                <OJsonobjectField :store="store" prop="RecordRules.Condition" :allow-array="true" />
                <OBooleanField :store="store" prop="RecordRules.PermRead" />
                <OBooleanField :store="store" prop="RecordRules.PermWrite" />
                <OBooleanField :store="store" prop="RecordRules.PermCreate" />
                <OBooleanField :store="store" prop="RecordRules.PermDelete" />
              </OOneToManyField>
            </el-collapse-item>

            <el-collapse-item name="field_rules" :title="_t('Field Rules')">
              <p class="rfv-advanced__hint rfv-advanced__hint--tight">
                {{ _t('Field visibility under deny-default. Leave Application/Model/Field empty for wider scopes.') }}
              </p>
              <OOneToManyField :store="store" prop="FieldRules" label="">
                <OManyToOneRefField :store="store" prop="FieldRules.IrApplicationId" />
                <OManyToOneRefField :store="store" prop="FieldRules.IrModelId" />
                <OManyToOneRefField :store="store" prop="FieldRules.IrFieldId" />
                <OSelectionField :store="store" prop="FieldRules.PermRead" />
                <OSelectionField :store="store" prop="FieldRules.PermWrite" />
              </OOneToManyField>
            </el-collapse-item>

            <el-collapse-item name="method_accesses" :title="_t('Method Access')">
              <p class="rfv-advanced__hint rfv-advanced__hint--tight">
                {{ _t('RPC allow/deny under deny-default. New rows default to allow; use deny as an explicit brake.') }}
              </p>
              <OOneToManyField :store="store" prop="MethodAccesses" label="" :default-record="defaultMethodAccess">
                <OManyToOneRefField :store="store" prop="MethodAccesses.IrApplicationId" />
                <OManyToOneRefField :store="store" prop="MethodAccesses.IrModelId" />
                <OManyToOneRefField :store="store" prop="MethodAccesses.IrServiceId" />
                <OSelectionField :store="store" prop="MethodAccesses.Mode" />
              </OOneToManyField>
            </el-collapse-item>

            <el-collapse-item name="ui_resources" :title="_t('UI Resource Details (manual bypass)')">
              <p class="rfv-advanced__hint rfv-advanced__hint--tight">
                {{ _t('Secondary to the UI Resource Access tree above. Prefer the tree for day-to-day grants.') }}
              </p>
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

import { ElCard, ElRow, ElCol, ElTabs, ElTabPane, ElCollapse, ElCollapseItem, ElIcon, ElAlert } from 'element-plus';
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
import { normalizeUiResourceRequires } from '@/auth/web/views/role_ui_requires_explain';

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

const inspectedUiResource = ref<Record<string, any> | null>(null);

const inspectedUiResourceId = computed(() => String(inspectedUiResource.value?.Id ?? '').trim());

const inspectedUiResourceLabel = computed(() => resolveUiResourceLabel(inspectedUiResource.value as UiResourceRow | undefined));

const inspectedRequires = computed(() =>
  normalizeUiResourceRequires(inspectedUiResource.value?.Requires ?? inspectedUiResource.value?.requires)
);

function inspectUiResource(row: any) {
  if (!row || typeof row !== 'object') {
    inspectedUiResource.value = null;
    return;
  }
  inspectedUiResource.value = row;
}

/** New RecordRule rows default to grant (RoleId is always this role via O2M inverse). */
const defaultRecordRule: Record<string, any> = { Kind: 'grant' };

/** New MethodAccess rows default to allow (deny is an explicit brake). */
const defaultMethodAccess: Record<string, any> = { Mode: 'allow' };

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

.rfv-advanced__hint {
  margin: 0 0 12px;
  color: var(--el-text-color-secondary);
  font-size: 13px;
  line-height: 1.5;
}

.rfv-advanced__hint--tight {
  margin-bottom: 8px;
}

.rfv-rr-alert {
  margin-bottom: 10px;
}

.rfv-ui-resource-node {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  cursor: pointer;
  border-radius: 4px;
  padding: 0 4px;
}

.rfv-ui-resource-node.is-inspected {
  background: var(--el-color-primary-light-9);
  color: var(--el-color-primary);
}

.rfv-ui-resource-node__icon {
  color: var(--el-text-color-secondary);
  font-size: 14px;
}

.rfv-ui-requires {
  margin-top: 12px;
  padding: 10px 12px;
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 6px;
  background: var(--el-fill-color-blank);
}

.rfv-ui-requires__title {
  font-weight: 600;
  font-size: 13px;
  margin-bottom: 6px;
}

.rfv-ui-requires__resource {
  margin-left: 8px;
  font-weight: 500;
  color: var(--el-text-color-regular);
}

.rfv-ui-requires__list {
  margin: 0;
  padding-left: 18px;
  font-size: 13px;
  line-height: 1.6;
}

.rfv-ui-requires__empty {
  margin: 0;
  font-size: 13px;
  color: var(--el-text-color-secondary);
}
</style>
