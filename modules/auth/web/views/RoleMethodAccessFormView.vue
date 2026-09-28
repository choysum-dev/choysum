<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <ChoyFormView
    v-bind="{ store, recordId, viewMode, showHeader, createAction }"
    :action-ids="{
      create: methodAccessActions.create,
      edit: methodAccessActions.edit,
      copy: methodAccessActions.copy,
      delete: methodAccessActions.delete,
    }"
    :has-action="hasAction"
    v-on="$attrs"
  >
    <ChoyCard :title="_t('Method Access')" class="mafv-card">
      <div
        class="mafv-card__hint mb-3 rounded-md border border-info/40 bg-info/10 px-3 py-2 text-sm"
        role="alert"
      >
        <p class="font-medium text-foreground">{{ _t('Cross-role method access editor') }}</p>
        <p class="mt-1 text-foreground/80">
          {{
            _t(
              'Role is required. Pick exactly one scope: Service, Model, Application, Logical Model (all host apps sharing that short name), or leave all empty for Global. New rows default to Mode=deny; prefer allow for grants. Source is always manual under UI-Option-A.'
            )
          }}
        </p>
      </div>
      <ChoyGrid :cols="12">
        <ChoyCol :span="6">
          <ChoyManyToOneField value-mode="record"
            :store="store"
            prop="RoleId"
            :search-view="RoleListView"
            :search-view-title="_t('Select Role')"
            @value-click="onRoleValueClick"
          />
        </ChoyCol>
        <ChoyCol :span="6">
          <ChoyManyToOneField :store="store" prop="MetaApplicationId" />
        </ChoyCol>
        <ChoyCol :span="6">
          <ChoyManyToOneField :store="store" prop="MetaModelId" />
        </ChoyCol>
        <ChoyCol :span="6">
          <ChoyManyToOneField :store="store" prop="MetaServiceId" />
        </ChoyCol>
        <ChoyCol :span="6">
          <ChoySelectionField :store="store" prop="LogicalModelName" />
        </ChoyCol>
        <ChoyCol :span="6">
          <ChoyJsonField :store="store" prop="LogicalMethods" :allow-array="true" />
        </ChoyCol>
        <ChoyCol :span="6">
          <ChoySelectionField :store="store" prop="Mode" />
        </ChoyCol>
        <ChoyCol :span="6">
          <ChoySelectionField :store="store" prop="Source" />
        </ChoyCol>
      </ChoyGrid>
    </ChoyCard>

    <ChoyCard :title="_t('System Information')" class="mafv-card">
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
import type RoleMethodAccess from '@/auth/service/models/role_method_access';
import type Role from '@/auth/service/models/role';
import type { ValueClickPayload as ManyToOneValueClickPayload } from '@/web/web/components/field/manyToOneTypes';
import RoleListView from '@/auth/web/views/RoleListView.vue';
import { defineModelActions } from '@/core/web/resource';
import { usePermission } from '@/auth/web/composables/usePermission';
import { resolvePageStore } from '@/web/web/composables/usePageContext';
import { ChoyCard, ChoyCol, ChoyDatetimeField, ChoyFormView, ChoyGrid, ChoyJsonField, ChoyManyToOneField, ChoySelectionField } from '@/web';
import type { ChoyViewMode as ViewMode } from '@/web';
import { createTranslate } from '@/web/web/i18n';
import { roleIdFromValueClick } from '@/auth/web/views/role_value_click';

defineOptions({ name: 'RoleMethodAccessFormView', inheritAttrs: true });
const { _t, _lt } = createTranslate('auth', { scope: 'web/views/RoleMethodAccessFormView' });

const props = withDefaults(
  defineProps<{
    store?: WebModelStore<RoleMethodAccess>;
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

const store = resolvePageStore(props.store, 'RoleMethodAccessFormView');
const { recordId, viewMode, showHeader, createAction } = props;
const methodAccessActions = defineModelActions('auth.RoleMethodAccess', { entityTitle: _lt('Method Access') });
const { hasAction } = usePermission();
const router = useRouter();

/**
 * Open the referenced role from the method-access form.
 */
function onRoleValueClick(payload: ManyToOneValueClickPayload<Role>) {
  const id = roleIdFromValueClick(payload);
  if (!id) return;
  void router.push(`/auth/roles/${id}`);
}
</script>

<style scoped>
.mafv-card {
  margin-bottom: 14px;
}
.mafv-card__header {
  font-weight: 600;
  color: var(--choy-foreground, inherit);
}
.mafv-card__hint {
  margin-bottom: 12px;
}
</style>
