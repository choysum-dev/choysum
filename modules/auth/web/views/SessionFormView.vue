<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <OFormView
    v-bind="{ store, recordId, viewMode, showHeader, createAction }"
    :action-ids="{ create: sessionActions.create, edit: sessionActions.edit, copy: sessionActions.copy, delete: sessionActions.delete }"
    :has-action="hasAction"
    v-on="$attrs"
  >
    <el-card shadow="never" class="sfv-card">
      <template #header
        ><div class="sfv-card__header"><span>Session Information</span></div></template
      >
      <el-row :gutter="12">
        <el-col :xs="24" :sm="12" :md="8" :lg="6" :xl="6">
          <OManyToOneField
            :store="store"
            prop="UserId"
            label="User"
            :search-view="UserListView"
            search-view-title="Select User"
            @value-click="onUserValueClick"
          />
        </el-col>
        <el-col :xs="24" :sm="12" :md="8" :lg="6" :xl="6">
          <OVarCharField :store="store" prop="AccessTokenId" label="Access Token ID" />
        </el-col>
        <el-col :span="24">
          <OTextField :store="store" prop="DeviceInfo" label="Device Info" />
        </el-col>
        <el-col :xs="24" :sm="12" :md="8" :lg="6" :xl="6">
          <OVarCharField :store="store" prop="IpAddress" label="IP Address" />
        </el-col>
        <el-col :xs="24" :sm="12" :md="8" :lg="6" :xl="6">
          <ODateTimeField :store="store" prop="ExpiresAt" label="Expires At" />
        </el-col>
        <el-col :xs="24" :sm="12" :md="8" :lg="6" :xl="6">
          <ODateTimeField :store="store" prop="LastActivityAt" label="Last Activity" />
        </el-col>
        <el-col :xs="24" :sm="12" :md="8" :lg="6" :xl="6">
          <OVarCharField :store="store" prop="Status" label="Status" />
        </el-col>
      </el-row>
    </el-card>

    <el-card shadow="never" class="sfv-card">
      <template #header
        ><div class="sfv-card__header"><span>System Information</span></div></template
      >
      <el-row :gutter="12">
        <el-col :xs="24" :sm="12" :md="8" :lg="6" :xl="6">
          <ODateTimeField :store="store" prop="CreatedAt" label="Created At" />
        </el-col>
        <el-col :xs="24" :sm="12" :md="8" :lg="6" :xl="6">
          <ODateTimeField :store="store" prop="UpdatedAt" label="Updated At" />
        </el-col>
      </el-row>
    </el-card>
  </OFormView>
</template>

<script setup lang="ts">
import { useRouter } from 'vue-router';
import type { RouteLocationRaw } from 'vue-router';
import type { WebModelStore } from '@/web/web/stores/modelStore';
import type Session from '@/auth/service/models/session';
import type User from '@/auth/service/models/user';
import { ElCard, ElRow, ElCol } from 'element-plus';

import OFormView from '@/web/web/components/view/OFormView.vue';
import OVarCharField from '@/web/web/components/field/OVarCharField.vue';
import OTextField from '@/web/web/components/field/OTextField.vue';
import ODateTimeField from '@/web/web/components/field/ODatetimeField.vue';
import OManyToOneField from '@/web/web/components/field/OManyToOneField.vue';
import type { ValueClickPayload as ManyToOneValueClickPayload } from '@/web/web/components/field/OManyToOneField.vue';
import UserListView from '@/auth/web/views/UserListView.vue';
import type { ViewMode } from '@/web/web/components/view/OViewScope.vue';
import { defineModelActions } from '@/core/web/resource';
import { usePermission } from '@/auth/web/composables/usePermission';

defineOptions({ name: 'SessionFormView', inheritAttrs: true });

const props = withDefaults(
  defineProps<{
    store: WebModelStore<Session>;
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
const sessionActions = defineModelActions('auth.Session', { entityTitle: 'Session' });
const { hasAction } = usePermission();
const router = useRouter();

/**
 * Open the referenced user record from the session form.
 */
function onUserValueClick(payload: ManyToOneValueClickPayload<User>) {
  const id = String(payload?.id || '').trim();
  if (!id) return;
  void router.push(`/auth/users/${id}`);
}
</script>

<style scoped>
.sfv-card {
  margin-bottom: 14px;
}
.sfv-card__header {
  font-weight: 600;
  color: var(--el-text-color-primary);
}
</style>
