<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <ChoyFormView
    v-bind="{ store, recordId, viewMode, showHeader, createAction }"
    :action-ids="{ create: userActions.create, edit: userActions.edit, copy: userActions.copy, delete: userActions.delete }"
    :has-action="hasAction"
    v-on="$attrs"
  >
    <ChoyCard :title="_t('Account Information')" class="mb-3.5">
      <ChoyGrid :cols="12">
        <ChoyCol :span="6">
          <ChoyImageField :store="store" prop="Avatar" />
        </ChoyCol>
        <ChoyCol :span="6">
          <ChoyVarcharField :store="store" prop="Username" :rules="requiredRules" />
        </ChoyCol>
        <ChoyCol :span="6">
          <ChoyVarcharField :store="store" prop="Email" />
        </ChoyCol>
        <ChoyCol :span="6">
          <ChoyVarcharField :store="store" prop="Phone" />
        </ChoyCol>
        <ChoyCol :span="6">
          <ChoyBooleanField :store="store" prop="IsActive" widget="switch" />
        </ChoyCol>
        <ChoyCol :span="6">
          <ChoyVarcharField :store="store" prop="FirstName" />
        </ChoyCol>
        <ChoyCol :span="6">
          <ChoyVarcharField :store="store" prop="LastName" />
        </ChoyCol>
        <ChoyCol :span="6">
          <ChoyVarcharField :store="store" prop="FullName" />
        </ChoyCol>
        <ChoyCol :span="6">
          <ChoyManyToOneRefField :store="store" prop="LanguageId" />
        </ChoyCol>
        <ChoyCol :span="6">
          <ChoySelectionField
            :store="store"
            prop="Timezone"
            :placeholder="_t('Select a time zone')"
            :select-props="{ filterable: true, allowCreate: false }"
          />
        </ChoyCol>
        <ChoyCol :span="6">
          <ChoyManyToOneRefField :store="store" prop="CompanyId" @value-click="onCompanyValueClick" />
        </ChoyCol>
        <ChoyCol :span="6">
          <ChoyJsonField :store="store" prop="Preferences" />
        </ChoyCol>
      </ChoyGrid>

      <ChoyGrid :cols="12">
        <ChoyCol :span="6">
          <ChoyDatetimeField :store="store" prop="LastLogin" />
        </ChoyCol>
        <ChoyCol :span="6">
          <ChoyDatetimeField :store="store" prop="CreatedAt" />
        </ChoyCol>
        <ChoyCol :span="6">
          <ChoyDatetimeField :store="store" prop="UpdatedAt" />
        </ChoyCol>
      </ChoyGrid>

      <ChoyGrid :cols="12">
        <ChoyCol :span="12">
          <ChoyManyToManyRefTagsField
            :store="store"
            prop="CompanyIds"
            :label="_t('Accessible Companies')"
            :search-list="CompanyListView"
            :search-view-title="_t('Select Company')"
            :tag-label-field="['Name', 'DisplayName', 'Code', 'Id']"
            @tag-click="onCompanyTagClick as any"
          />
        </ChoyCol>
      </ChoyGrid>
    </ChoyCard>

    <ChoyTabs v-model="userDetailTab">
      <ChoyTab :label="_t('Roles')" value="roles">
        <ChoyManyToManyField :store="store" prop="Roles" label="" :search-list="RoleListView" :search-view-title="_t('Select Role')">
          <ChoyVarcharField :store="store" prop="Roles.Id" />
          <ChoyVarcharField :store="store" prop="Roles.Name" />
          <ChoyDatetimeField :store="store" prop="Roles.CreatedAt" />
        </ChoyManyToManyField>
      </ChoyTab>

      <ChoyTab :label="_t('Sessions')" value="sessions">
        <ChoyOneToManyField :store="store" prop="Sessions" label="">
          <ChoyVarcharField :store="store" prop="Sessions.Id" />
          <ChoyVarcharField :store="store" prop="Sessions.Status" />
          <ChoyDatetimeField :store="store" prop="Sessions.LastActivityAt" />
          <ChoyDatetimeField :store="store" prop="Sessions.CreatedAt" />
        </ChoyOneToManyField>
      </ChoyTab>

      <ChoyTab :label="_t('Tokens')" value="tokens">
        <ChoyOneToManyField :store="store" prop="Tokens" label="">
          <ChoyVarcharField :store="store" prop="Tokens.Id" />
          <ChoyVarcharField :store="store" prop="Tokens.TokenType" />
          <ChoyDatetimeField :store="store" prop="Tokens.CreatedAt" />
        </ChoyOneToManyField>
      </ChoyTab>
    </ChoyTabs>

    <ChoyChatter v-if="recordId" :key="recordId" bind-store model="auth.User" :res-id="recordId" />
  </ChoyFormView>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue';
import { useRouter } from 'vue-router';
import type { RouteLocationRaw } from 'vue-router';
import type { ClientModel, BaseModel } from '@/core/rpc';
import type { WebModelStore } from '@/web/web/stores/modelStore';

import type User from '@/auth/service/models/user/user';
import type Company from '@/base/service/models/company';

import RoleListView from '@/auth/web/views/RoleListView.vue';
import CompanyListView from '@/base/web/views/CompanyListView.vue';
import type { ValueClickPayload as ManyToOneRefValueClickPayload } from '@/web/web/components/field/manyToOneTypes';
import type { TagClickPayload as RefTagClickPayload } from '@/web/web/components/field/manyToManyTagsTypes';
import { defineModelActions } from '@/core/web/resource';
import { usePermission } from '@/auth/web/composables/usePermission';
import { resolvePageStore } from '@/web/web/composables/usePageContext';
import { ChoyBooleanField, ChoyCard, ChoyChatter, ChoyCol, ChoyDatetimeField, ChoyFormView, ChoyGrid, ChoyImageField, ChoyJsonField, ChoyManyToManyField, ChoyManyToOneField, ChoyOneToManyField, ChoySelectionField, ChoyTab, ChoyTabs, ChoyVarcharField, ChoyManyToOneRefField, ChoyManyToManyRefTagsField} from '@/web';
import type { ChoyViewMode as ViewMode } from '@/web';
import { createTranslate } from '@/web/web/i18n';

defineOptions({ name: 'UserFormView', inheritAttrs: true });
const { _t, _lt } = createTranslate('auth', { scope: 'web/views/UserFormView' });
const userDetailTab = ref('roles');
const requiredRules = computed(() => [{ required: true, message: _t('Required') }]);

const props = withDefaults(
  defineProps<{
    store?: WebModelStore<User>;
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

const store = resolvePageStore(props.store, 'UserFormView');
const { recordId, viewMode, showHeader, createAction } = props;
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

