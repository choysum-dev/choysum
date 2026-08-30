<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <OFormView
    v-bind="{ store, recordId, viewMode, showHeader, createAction }"
    :action-ids="{ create: tokenActions.create, edit: tokenActions.edit, copy: tokenActions.copy, delete: tokenActions.delete }"
    :has-action="hasAction"
    v-on="$attrs"
  >
    <el-card shadow="never" class="tfv-card">
      <template #header
        ><div class="tfv-card__header"><span>{{ _t('Token Information') }}</span></div></template
      >
      <el-row :gutter="12">
        <el-col :xs="24" :sm="12" :md="8" :lg="6" :xl="6">
          <OManyToOneField
            :store="store"
            prop="UserId"
            :search-view="UserListView"
            :search-view-title="_t('Select User')"
            @value-click="onUserValueClick"
          />
        </el-col>
        <el-col :xs="24" :sm="12" :md="8" :lg="6" :xl="6">
          <OVarCharField :store="store" prop="DisplayName" />
        </el-col>
        <el-col :xs="24" :sm="12" :md="8" :lg="6" :xl="6">
          <OVarCharField :store="store" prop="TokenType" />
        </el-col>
        <el-col :xs="24" :sm="12" :md="8" :lg="6" :xl="6">
          <OVarCharField :store="store" prop="TokenId" />
        </el-col>
        <el-col :xs="24" :sm="12" :md="8" :lg="6" :xl="6">
          <ODateTimeField :store="store" prop="ExpiresAt" />
        </el-col>
        <el-col :xs="24" :sm="12" :md="8" :lg="6" :xl="6">
          <OBooleanField :store="store" prop="Revoked" widget="checkbox" />
        </el-col>
        <el-col :span="24">
          <OVarCharField :store="store" prop="RevocationReason" />
        </el-col>
      </el-row>
    </el-card>

    <el-card shadow="never" class="tfv-card">
      <template #header
        ><div class="tfv-card__header"><span>{{ _t('System Information') }}</span></div></template
      >
      <el-row :gutter="12">
        <el-col :xs="24" :sm="12" :md="8" :lg="6" :xl="6">
          <ODateTimeField :store="store" prop="RevokedAt" />
        </el-col>
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
import type { ClientModel, BaseModel } from '@/core/rpc';
import type { WebModelStore } from '@/web/web/stores/modelStore';

import type Token from '@/auth/service/models/token';
import { ElCard, ElRow, ElCol } from 'element-plus';

import OFormView from '@/web/web/components/view/OFormView.vue';
import OVarCharField from '@/web/web/components/field/OVarCharField.vue';
import OTextField from '@/web/web/components/field/OTextField.vue';
import ODateTimeField from '@/web/web/components/field/ODatetimeField.vue';
import OBooleanField from '@/web/web/components/field/OBooleanField.vue';
import OManyToOneField from '@/web/web/components/field/OManyToOneField.vue';
import type { ValueClickPayload as ManyToOneValueClickPayload } from '@/web/web/components/field/manyToOneTypes';
import UserListView from '@/auth/web/views/UserListView.vue';
import type User from '@/auth/service/models/user';
import type { ViewMode } from '@/web/web/components/view/OViewScope.vue';
import { defineModelActions } from '@/core/web/resource';
import { usePermission } from '@/auth/web/composables/usePermission';
import { resolvePageStore } from '@/web/web/composables/usePageContext';
import { createTranslate } from '@/web/web/i18n';

defineOptions({ name: 'TokenFormView', inheritAttrs: true });
const { _t, _lt } = createTranslate('auth', { scope: 'web/views/TokenFormView' });

const props = withDefaults(
  defineProps<{
    store?: WebModelStore<Token>;
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

const store = resolvePageStore(props.store, 'TokenFormView');
const { recordId, viewMode, showHeader, createAction } = props;
const tokenActions = defineModelActions('auth.Token', { entityTitle: _lt('Token') });
const { hasAction } = usePermission();
const router = useRouter();

/**
 * Open the referenced user record from the token form.
 */
function onUserValueClick(payload: ManyToOneValueClickPayload<User>) {
  const id = String(payload?.id || '').trim();
  if (!id) return;
  void router.push({ name: 'UserDetail', params: { id } });
}
</script>

<style scoped>
.tfv-card {
  margin-bottom: 14px;
}
.tfv-card__header {
  font-weight: 600;
  color: var(--el-text-color-primary);
}
</style>
