<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <OFormView
    v-bind="{ store, recordId, viewMode, showHeader, createAction }"
    :action-ids="{ create: userActions.create, edit: userActions.edit, copy: userActions.copy, delete: userActions.delete }"
    :has-action="hasAction"
    v-on="$attrs"
  >
    <el-card shadow="never" class="ufv-card">
      <template #header
        ><div class="ufv-card__header"><span>{{ _t('Account Information') }}</span></div></template
      >
      <el-row :gutter="12">
        <el-col :xs="24" :sm="12" :md="8" :lg="6" :xl="6">
          <OImageField :store="store" prop="Avatar" />
        </el-col>
        <el-col :xs="24" :sm="12" :md="8" :lg="6" :xl="6">
          <OVarCharField :store="store" prop="Username" :rules="requiredRules" />
        </el-col>
        <el-col :xs="24" :sm="12" :md="8" :lg="6" :xl="6">
          <OVarCharField :store="store" prop="Email" />
        </el-col>
        <el-col :xs="24" :sm="12" :md="8" :lg="6" :xl="6">
          <OVarCharField :store="store" prop="Phone" />
        </el-col>
        <el-col :xs="24" :sm="12" :md="8" :lg="6" :xl="6">
          <OBooleanField :store="store" prop="IsActive" widget="switch" />
        </el-col>
        <el-col :xs="24" :sm="12" :md="8" :lg="6" :xl="6">
          <OVarCharField :store="store" prop="FirstName" />
        </el-col>
        <el-col :xs="24" :sm="12" :md="8" :lg="6" :xl="6">
          <OVarCharField :store="store" prop="LastName" />
        </el-col>
        <el-col :xs="24" :sm="12" :md="8" :lg="6" :xl="6">
          <OVarCharField :store="store" prop="FullName" />
        </el-col>
        <el-col :xs="24" :sm="12" :md="8" :lg="6" :xl="6">
          <OVarCharField :store="store" prop="Language" />
        </el-col>
        <el-col :xs="24" :sm="12" :md="8" :lg="6" :xl="6">
          <OSelectionField
            :store="store"
            prop="Timezone"
            :placeholder="_t('Select a time zone')"
            :select-props="{ filterable: true, allowCreate: false }"
          />
        </el-col>
        <el-col :xs="24" :sm="12" :md="8" :lg="6" :xl="6">
          <OManyToOneRefField :store="store" prop="CompanyId" @value-click="onCompanyValueClick" />
        </el-col>
        <el-col :xs="24" :sm="12" :md="8" :lg="6" :xl="6">
          <OJsonobjectField :store="store" prop="Preferences" />
        </el-col>
      </el-row>

      <el-row :gutter="12">
        <el-col :xs="24" :sm="12" :md="8" :lg="6" :xl="6">
          <ODateTimeField :store="store" prop="LastLogin" />
        </el-col>
        <el-col :xs="24" :sm="12" :md="8" :lg="6" :xl="6">
          <ODateTimeField :store="store" prop="CreatedAt" />
        </el-col>
        <el-col :xs="24" :sm="12" :md="8" :lg="6" :xl="6">
          <ODateTimeField :store="store" prop="UpdatedAt" />
        </el-col>
      </el-row>

      <el-row :gutter="12">
        <el-col :xs="24" :sm="24" :md="24" :lg="24" :xl="24">
          <OManyToManyRefTagsField
            :store="store"
            prop="CompanyIds"
            :label="_t('Accessible Companies')"
            :search-list="CompanyListView"
            :search-view-title="_t('Select Company')"
            :tag-label-field="['Name', 'DisplayName', 'Code', 'Id']"
            @tag-click="onCompanyTagClick"
          />
        </el-col>
      </el-row>
    </el-card>

    <el-tabs model-value="roles" type="card" class="ufv-tabs">
      <el-tab-pane :label="_t('Roles')" name="roles">
        <OManyToManyField :store="store" prop="Roles" label="" :search-list="RoleListView" :search-view-title="_t('Select Role')">
          <OCharField :store="store" prop="Roles.Id" />
          <OVarCharField :store="store" prop="Roles.Name" />
          <ODateTimeField :store="store" prop="Roles.CreatedAt" />
        </OManyToManyField>
      </el-tab-pane>

      <el-tab-pane :label="_t('Sessions')" name="sessions">
        <OOneToManyField :store="store" prop="Sessions" label="">
          <OCharField :store="store" prop="Sessions.Id" />
          <OVarCharField :store="store" prop="Sessions.Status" />
          <ODateTimeField :store="store" prop="Sessions.LastActivityAt" />
          <ODateTimeField :store="store" prop="Sessions.CreatedAt" />
        </OOneToManyField>
      </el-tab-pane>

      <el-tab-pane :label="_t('Tokens')" name="tokens">
        <OOneToManyField :store="store" prop="Tokens" label="">
          <OCharField :store="store" prop="Tokens.Id" />
          <OVarCharField :store="store" prop="Tokens.TokenType" />
          <ODateTimeField :store="store" prop="Tokens.CreatedAt" />
        </OOneToManyField>
      </el-tab-pane>
    </el-tabs>
  </OFormView>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import { useRouter } from 'vue-router';
import type { RouteLocationRaw } from 'vue-router';
import type { ClientModel, BaseModel } from '@/core/rpc';
import type { WebModelStore } from '@/web/web/stores/modelStore';

import type User from '@/auth/service/models/user';
import type Company from '@/base/service/models/company';
import { ElCard, ElRow, ElCol, ElTabs, ElTabPane } from 'element-plus';

import OFormView from '@/web/web/components/view/OFormView.vue';
import OBooleanField from '@/web/web/components/field/OBooleanField.vue';
import OCharField from '@/web/web/components/field/OCharField.vue';
import OVarCharField from '@/web/web/components/field/OVarCharField.vue';
import OSelectionField from '@/web/web/components/field/OSelectionField.vue';
import OImageField from '@/web/web/components/field/OImageField.vue';
import ODateTimeField from '@/web/web/components/field/ODatetimeField.vue';
import OOneToManyField from '@/web/web/components/field/OOneToManyField.vue';
import OManyToManyField from '@/web/web/components/field/OManyToManyField.vue';
import RoleListView from '@/auth/web/views/RoleListView.vue';
import CompanyListView from '@/base/web/views/CompanyListView.vue';
import OJsonobjectField from '@/web/web/components/field/OJsonobjectField.vue';
import OManyToOneRefField from '@/web/web/components/field/OManyToOneRefField.vue';
import type { ValueClickPayload as ManyToOneRefValueClickPayload } from '@/web/web/components/field/manyToOneTypes';
import OManyToManyRefTagsField from '@/web/web/components/field/OManyToManyRefTagsField.vue';
import type { TagClickPayload as RefTagClickPayload } from '@/web/web/components/field/manyToManyTagsTypes';
import type { ViewMode } from '@/web/web/components/view/OViewScope.vue';
import { defineModelActions } from '@/core/web/resource';
import { usePermission } from '@/auth/web/composables/usePermission';
import { createTranslate } from '@/web/web/i18n';

defineOptions({ name: 'UserFormView', inheritAttrs: true });
const { _t, _lt } = createTranslate('auth', { scope: 'web/views/UserFormView' });
const requiredRules = computed(() => [{ required: true, message: _t('Required') }]);

const props = withDefaults(
  defineProps<{
    store: WebModelStore<User>;
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
const userActions = defineModelActions('auth.User', { entityTitle: _lt('User') });
const { hasAction } = usePermission();

const router = useRouter();

/**
 * Open the primary company record from the user form.
 */
function onCompanyValueClick(payload: ManyToOneRefValueClickPayload<Company>) {
  const id = String(payload?.id || '').trim();
  if (!id) return;
  void router.push({ name: 'CompanyDetail', params: { id } });
}

/**
 * Open a company record from the accessible-company tag list.
 */
function onCompanyTagClick(payload: RefTagClickPayload<Company>) {
  const id = String(payload?.id || '').trim();
  if (!id) return;
  void router.push({ name: 'CompanyDetail', params: { id } });
}
</script>

<style scoped>
.ufv-card {
  margin-bottom: 14px;
}
.ufv-card__header {
  font-weight: 600;
  color: var(--el-text-color-primary);
}
.ufv-tabs {
  --el-tabs-header-height: 42px;
}
</style>
