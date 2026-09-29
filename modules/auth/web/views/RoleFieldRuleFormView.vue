<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <ChoyFormView
    v-bind="{ store, recordId, viewMode, showHeader, createAction }"
    :action-ids="{
      create: fieldRuleActions.create,
      edit: fieldRuleActions.edit,
      copy: fieldRuleActions.copy,
      delete: fieldRuleActions.delete,
    }"
    :has-action="hasAction"
    v-on="$attrs"
  >
    <ChoyCard :title="_t('Field Rule')" class="frfv-card">
      <div
        class="frfv-card__hint mb-3 rounded-md border border-info/40 bg-info/10 px-3 py-2 text-sm"
        role="alert"
      >
        <p class="font-medium text-foreground">{{ _t('Cross-role field rule editor') }}</p>
        <p class="mt-1 text-foreground/80">
          {{
            _t(
              'Role is required. Pick exactly one scope: Field (+ Model), Model, Application, Logical Model (all host apps / all business fields on that short name), or leave all empty for Global.'
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
          <ChoyManyToOneRefField :store="store" prop="MetaApplicationId" />
        </ChoyCol>
        <ChoyCol :span="6">
          <ChoyManyToOneRefField :store="store" prop="MetaModelId" />
        </ChoyCol>
        <ChoyCol :span="6">
          <ChoyManyToOneRefField :store="store" prop="MetaFieldId" />
        </ChoyCol>
        <ChoyCol :span="6">
          <ChoySelectionField :store="store" prop="LogicalModelName" />
        </ChoyCol>
        <ChoyCol :span="6">
          <ChoySelectionField :store="store" prop="PermRead" />
        </ChoyCol>
        <ChoyCol :span="6">
          <ChoySelectionField :store="store" prop="PermWrite" />
        </ChoyCol>
      </ChoyGrid>
    </ChoyCard>

    <ChoyCard :title="_t('System Information')" class="frfv-card">
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
import type RoleFieldRule from '@/auth/service/models/role_field_rule';
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

defineOptions({ name: 'RoleFieldRuleFormView', inheritAttrs: true });
const { _t, _lt } = createTranslate('auth', { scope: 'web/views/RoleFieldRuleFormView' });

const props = withDefaults(
  defineProps<{
    store?: WebModelStore<RoleFieldRule>;
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

const store = resolvePageStore(props.store, 'RoleFieldRuleFormView');
const { recordId, viewMode, showHeader, createAction } = props;
const fieldRuleActions = defineModelActions('auth.RoleFieldRule', { entityTitle: _lt('Field Rule') });
const { hasAction } = usePermission();
const router = useRouter();

/**
 * Open the referenced role from the field-rule form.
 */
function onRoleValueClick(payload: ManyToOneValueClickPayload<Role>) {
  const id = roleIdFromValueClick(payload);
  if (!id) return;
  void router.push(`/auth/roles/${id}`);
}
</script>

<style scoped>
.frfv-card {
  margin-bottom: 14px;
}
.frfv-card__header {
  font-weight: 600;
  color: var(--choy-foreground, inherit);
}
.frfv-card__hint {
  margin-bottom: 12px;
}
</style>
