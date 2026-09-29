<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <ChoyFormView
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
    <ChoyCard :title="_t('UI Resource Grant')" class="mb-3.5">
      <div
        class=" mb-3 rounded-md border border-info/40 bg-info/10 px-3 py-2 text-sm"
        role="alert"
      >
        <p class="font-medium text-foreground">{{ _t('Cross-role UI resource grant editor') }}</p>
        <p class="mt-1 text-foreground/80">
          {{
            _t(
              'Role is required. Prefer the Role form UI tree for day-to-day grants; use this page for cross-role browse and manual bypass rows. Empty Application and UI Resource means global allow/deny for that role.'
            )
          }}
        </p>
      </div>
      <ChoyGrid :cols="12">
        <ChoyCol :span="6">
          <ChoyManyToOneField
            :store="store"
            prop="RoleId"
            :search-view="RoleListView"
            :search-view-title="_t('Select Role')"
            @value-click="onRoleValueClick"
          />
        </ChoyCol>
        <ChoyCol :span="6">
          <ChoySelectionField :store="store" prop="Mode" />
        </ChoyCol>
        <ChoyCol :span="6">
          <ChoyManyToOneRefField :store="store" prop="MetaApplicationId" />
        </ChoyCol>
        <ChoyCol :span="6">
          <ChoyManyToOneRefField :store="store" prop="MetaUiResourceId" />
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
import type RoleUiResource from '@/auth/service/models/role_ui_resource';
import type Role from '@/auth/service/models/role';
import type { ValueClickPayload as ManyToOneValueClickPayload } from '@/web/web/components/field/manyToOneTypes';
import RoleListView from '@/auth/web/views/RoleListView.vue';
import { defineModelActions } from '@/core/web/resource';
import { usePermission } from '@/auth/web/composables/usePermission';
import { resolvePageStore } from '@/web/web/composables/usePageContext';
import { ChoyCard, ChoyCol, ChoyDatetimeField, ChoyFormView, ChoyGrid, ChoyManyToOneField, ChoySelectionField, ChoyManyToOneRefField} from '@/web';
import type { ChoyViewMode as ViewMode } from '@/web';
import { createTranslate } from '@/web/web/i18n';
import { roleIdFromValueClick } from '@/auth/web/views/role_value_click';

defineOptions({ name: 'RoleUiResourceFormView', inheritAttrs: true });
const { _t, _lt } = createTranslate('auth', { scope: 'web/views/RoleUiResourceFormView' });

const props = withDefaults(
  defineProps<{
    store?: WebModelStore<RoleUiResource>;
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

const store = resolvePageStore(props.store, 'RoleUiResourceFormView');
const { recordId, viewMode, showHeader, createAction } = props;
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

