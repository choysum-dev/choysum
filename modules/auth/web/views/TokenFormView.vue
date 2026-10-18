<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <ChoyFormView
    v-bind="{ store, recordId, viewMode, showHeader, createAction }"
    :action-ids="{ create: tokenActions.create, edit: tokenActions.edit, copy: tokenActions.copy, delete: tokenActions.delete }"
    :has-action="hasAction"
    v-on="$attrs"
  >
    <!-- Form sections on the shared view-body sheet (no nested Card / bg-card). -->
    <div data-anchor="choy.form.sheet" class="min-h-full">
      <section class="space-y-4">
        <h3 class="text-base font-semibold tracking-tight text-foreground">{{ _t('Token Information') }}</h3>
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
            <ChoyVarcharField :store="store" prop="DisplayName" />
          </ChoyCol>
          <ChoyCol :span="6">
            <ChoyVarcharField :store="store" prop="TokenType" />
          </ChoyCol>
          <ChoyCol :span="6">
            <ChoyVarcharField :store="store" prop="TokenId" />
          </ChoyCol>
          <ChoyCol :span="6">
            <ChoyDatetimeField :store="store" prop="ExpiresAt" />
          </ChoyCol>
          <ChoyCol :span="6">
            <ChoyBooleanField :store="store" prop="Revoked" widget="checkbox" />
          </ChoyCol>
          <ChoyCol :span="12">
            <ChoyVarcharField :store="store" prop="RevocationReason" />
          </ChoyCol>
        </ChoyGrid>
      </section>

      <section class="mt-8 space-y-4 border-t border-border pt-6">
        <h3 class="text-base font-semibold tracking-tight text-foreground">{{ _t('System Information') }}</h3>
        <ChoyGrid :cols="12">
          <ChoyCol :span="6">
            <ChoyDatetimeField :store="store" prop="RevokedAt" />
          </ChoyCol>
          <ChoyCol :span="6">
            <ChoyDatetimeField :store="store" prop="CreatedAt" />
          </ChoyCol>
          <ChoyCol :span="6">
            <ChoyDatetimeField :store="store" prop="UpdatedAt" />
          </ChoyCol>
        </ChoyGrid>
      </section>
    </div>
  </ChoyFormView>
</template>

<script setup lang="ts">
import { useRouter } from 'vue-router';
import type { RouteLocationRaw } from 'vue-router';
import type { WebModelStore } from '@/web/web/stores/modelStore';

import type Token from '@/auth/service/models/token';

import type { ValueClickPayload as ManyToOneValueClickPayload } from '@/web/web/components/field/manyToOneTypes';
import UserListView from '@/auth/web/views/UserListView.vue';
import type User from '@/auth/service/models/user/user';
import { defineModelActions } from '@/core/web/resource';
import { usePermission } from '@/auth/web/composables/usePermission';
import { resolvePageStore } from '@/web/web/composables/usePageContext';
import { ChoyBooleanField, ChoyCol, ChoyDatetimeField, ChoyFormView, ChoyGrid, ChoyManyToOneField, ChoyVarcharField } from '@/web';
import type { ChoyViewMode as ViewMode } from '@/web';
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
