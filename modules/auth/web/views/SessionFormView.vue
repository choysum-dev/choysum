<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <ChoyFormView
    v-bind="{ store, recordId, viewMode, showHeader, createAction }"
    :action-ids="{ create: sessionActions.create, edit: sessionActions.edit, copy: sessionActions.copy, delete: sessionActions.delete }"
    :has-action="hasAction"
    v-on="$attrs"
  >
    <ChoyCard :title="_t('Session Information')" class="mb-3.5">
      <ChoyGrid :cols="12">
        <ChoyCol :span="6">
          <ChoyManyToOneField
            :store="store"
            prop="UserId"
            :search-view="UserListView"
            :search-view-title="_t('Select User')"
            @value-click="onUserValueClick"
          />
        </ChoyCol>
        <ChoyCol :span="6">
          <ChoyVarcharField :store="store" prop="AccessTokenId" />
        </ChoyCol>
        <ChoyCol :span="12">
          <ChoyTextField :store="store" prop="DeviceInfo" />
        </ChoyCol>
        <ChoyCol :span="6">
          <ChoyVarcharField :store="store" prop="IpAddress" />
        </ChoyCol>
        <ChoyCol :span="6">
          <ChoyDatetimeField :store="store" prop="ExpiresAt" />
        </ChoyCol>
        <ChoyCol :span="6">
          <ChoyDatetimeField :store="store" prop="LastActivityAt" />
        </ChoyCol>
        <ChoyCol :span="6">
          <ChoyVarcharField :store="store" prop="Status" />
        </ChoyCol>
      </ChoyGrid>
    </ChoyCard>

    <ChoyCard :title="_t('System Information')" class="mb-3.5">
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
import type Session from '@/auth/service/models/session';
import type User from '@/auth/service/models/user/user';

import type { ValueClickPayload as ManyToOneValueClickPayload } from '@/web/web/components/field/manyToOneTypes';
import UserListView from '@/auth/web/views/UserListView.vue';
import { defineModelActions } from '@/core/web/resource';
import { usePermission } from '@/auth/web/composables/usePermission';
import { resolvePageStore } from '@/web/web/composables/usePageContext';
import { ChoyCard, ChoyCol, ChoyDatetimeField, ChoyFormView, ChoyGrid, ChoyManyToOneField, ChoyTextField, ChoyVarcharField } from '@/web';
import type { ChoyViewMode as ViewMode } from '@/web';
import { createTranslate } from '@/web/web/i18n';

defineOptions({ name: 'SessionFormView', inheritAttrs: true });
const { _t, _lt } = createTranslate('auth', { scope: 'web/views/SessionFormView' });

const props = withDefaults(
  defineProps<{
    store?: WebModelStore<Session>;
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

const store = resolvePageStore(props.store, 'SessionFormView');
const { recordId, viewMode, showHeader, createAction } = props;
const sessionActions = defineModelActions('auth.Session', { entityTitle: _lt('Session') });
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

