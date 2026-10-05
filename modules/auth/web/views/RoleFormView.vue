<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <ChoyFormView
    v-bind="{ store, recordId, viewMode, showHeader, createAction }"
    :action-ids="{ create: roleActions.create, edit: roleActions.edit, copy: roleActions.copy, delete: roleActions.delete }"
    :has-action="hasAction"
    v-on="$attrs"
  >
    <template #button-box>
      <ChoyButtonBox>
        <ChoyStatInfo :store="store" prop="Users" :label="_t('Users')" :icon="User" @click="activeTab = 'users'" />
        <ChoyStatInfo
          :store="store"
          prop="ImpliedRoles"
          :label="_t('Included Roles')"
          :icon="GitBranch"
          @click="activeTab = 'implied_roles'"
        />
        <ChoyStatInfo :store="store" prop="RecordRules" :label="_t('Record Rules')" :icon="Settings" @click="openRecordRules" />
      </ChoyButtonBox>
    </template>

    <ChoyCard :title="_t('Basic Information')" class="mb-3.5">
      <ChoyGrid :cols="12">
        <ChoyCol :span="6">
          <ChoyVarcharField :store="store" prop="Name" :rules="requiredRules" />
        </ChoyCol>
        <ChoyCol :span="6">
          <ChoyVarcharField :store="store" prop="DisplayName" />
        </ChoyCol>
        <ChoyCol :span="6">
          <ChoyVarcharField :store="store" prop="Code" :rules="requiredRules" />
        </ChoyCol>
        <ChoyCol :span="6">
          <ChoyBooleanField :store="store" prop="IsActive" widget="checkbox" />
        </ChoyCol>
        <ChoyCol :span="6">
          <ChoyBooleanField :store="store" prop="IsSystem" widget="checkbox" />
        </ChoyCol>
        <ChoyCol :span="12">
          <ChoyVarcharField :store="store" prop="Description" />
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

    <ChoyCard :title="_t('Related Data')" class="mb-3.5">
      <ChoyTabs v-model="activeTab">
        <ChoyTab :label="_t('Users')" value="users">
          <ChoyManyToManyField :store="store" prop="Users" label="" :search-list="UserListView" :search-view-title="_t('Select User')">
            <ChoyVarcharField :store="store" prop="Users.Id" />
            <ChoyVarcharField :store="store" prop="Users.Username" />
            <ChoyVarcharField :store="store" prop="Users.FullName" />
          </ChoyManyToManyField>
        </ChoyTab>
        <ChoyTab :label="_t('Included Roles')" value="implied_roles">
          <ChoyManyToManyField :store="store" prop="ImpliedRoles" label="" :search-list="RoleListView" :search-view-title="_t('Select Role')">
            <ChoyVarcharField :store="store" prop="ImpliedRoles.Name" />
            <ChoyVarcharField :store="store" prop="ImpliedRoles.Code" />
          </ChoyManyToManyField>
        </ChoyTab>
        <ChoyTab :label="_t('UI Resource Access')" value="ui_permissions">
          <p class="mb-3 text-sm leading-relaxed text-foreground/70">
            {{
              _t(
                'Primary path: check resources in this tree. Checking a resource uniformly derives Method allow for its Requires (UI-Option-A; no read/write split). Advanced → UI Resource Details is a manual bypass only.'
              )
            }}
          </p>
          <p class="mb-2 text-sm leading-relaxed text-foreground/70">
            {{ _t('Click a node label to inspect Requires → derived RPCs (checkbox still controls the grant).') }}
          </p>
          <ChoyManyToManyRefTreeField
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
              <button
                type="button"
                class="rfv-ui-resource-node inline-flex items-center gap-1.5 cursor-pointer rounded px-1 m-0 border-0 bg-transparent font-[inherit] text-inherit leading-[inherit] text-left"
                :class="{ 'is-inspected bg-primary/10 text-primary': isInspectedUiResourceRow(inspectedUiResourceId, row) }"
                @click.stop="inspectUiResource(row)"
              >
                <component :is="resolveUiResourceTypeIcon(row?.Type)" class="size-3.5 shrink-0 text-foreground/60" aria-hidden="true" />
                <span class="rfv-ui-resource-node__label">{{ resolveUiResourceLabel(row, label) }}</span>
              </button>
            </template>
          </ChoyManyToManyRefTreeField>
          <div v-if="inspectedUiResource" class="rfv-ui-requires mt-3 rounded-md border border-border bg-background px-3 py-2.5">
            <div class="rfv-ui-requires__title mb-1.5 text-[13px] font-semibold">
              {{ _t('Requires → derived Method RPCs') }}
              <span class="rfv-ui-requires__resource ml-2 font-medium text-foreground">{{ inspectedUiResourceLabel }}</span>
            </div>
            <p class="mb-2 text-sm leading-relaxed text-foreground/70">
              {{
                _t(
                  'Under UI-Option-A, these RPCs are uniformly Method-allow when this resource is granted (unless a manual Method deny brakes them). Record/Field rules are not derived from UI.'
                )
              }}
            </p>
            <ul v-if="inspectedRequires.length > 0" class="rfv-ui-requires__list m-0 list-disc pl-[18px] text-[13px] leading-relaxed">
              <li v-for="req in inspectedRequires" :key="req">
                <code>{{ req }}</code>
              </li>
            </ul>
            <p v-else class="rfv-ui-requires__empty m-0 text-[13px] text-foreground/70">
              {{ _t('No Requires on this resource — granting it does not derive Method access.') }}
            </p>
          </div>
        </ChoyTab>

        <ChoyTab :label="_t('Advanced Mode')" value="advanced">
          <p class="mb-3 text-sm leading-relaxed text-foreground/70">
            {{
              _t(
                'Configure record/field/RPC grants under deny-default. The UI resource tree does not derive Record or Field rules; Advanced is the main place for data and method access.'
              )
            }}
          </p>
          <div class="rfv-advanced mt-1 space-y-2">
            <details class="rounded-md border border-border" :open="advancedPanels === 'record_rules'">
              <summary
                class="cursor-pointer select-none px-3 py-2 text-sm font-medium"
                @click.prevent="toggleAdvancedPanel('record_rules')"
              >
                {{ _t('Record Rules') }}
              </summary>
              <div class="border-t border-border px-3 py-3">
                <div
                  class="mb-2.5 rounded-md border border-info/40 bg-info/10 px-3 py-2 text-sm"
                  role="alert"
                >
                  <p class="font-medium text-foreground">{{ _t('This form only edits rules for this role') }}</p>
                  <p class="mt-1 text-foreground/80">
                    {{
                      _t(
                        'OneToMany rows are always bound to the current role. All-users or cross-role Record/Field/Method/UI rules belong under Access Control → Access Rules, not here. Model/Application empty means scope-global (all models), which is not the same as all-users audience.'
                      )
                    }}
                  </p>
                </div>
                <p class="mb-2 text-sm leading-relaxed text-foreground/70">
                  {{ _t('Without a matching grant, records are invisible or not writable (deny-default).') }}
                </p>
                <ChoyOneToManyField :store="store" prop="RecordRules" label="" :default-record="defaultRecordRule">
                  <ChoySelectionField :store="store" prop="RecordRules.Kind" />
                  <ChoyManyToOneRefField :store="store" prop="RecordRules.MetaApplicationId" />
                  <ChoyManyToOneRefField :store="store" prop="RecordRules.MetaModelId" />
                  <ChoyJsonField :store="store" prop="RecordRules.Condition" :allow-array="true" />
                  <ChoyBooleanField :store="store" prop="RecordRules.PermRead" />
                  <ChoyBooleanField :store="store" prop="RecordRules.PermWrite" />
                  <ChoyBooleanField :store="store" prop="RecordRules.PermCreate" />
                  <ChoyBooleanField :store="store" prop="RecordRules.PermDelete" />
                </ChoyOneToManyField>
              </div>
            </details>

            <details class="rounded-md border border-border" :open="advancedPanels === 'field_rules'">
              <summary
                class="cursor-pointer select-none px-3 py-2 text-sm font-medium"
                @click.prevent="toggleAdvancedPanel('field_rules')"
              >
                {{ _t('Field Rules') }}
              </summary>
              <div class="border-t border-border px-3 py-3">
                <p class="mb-2 text-sm leading-relaxed text-foreground/70">
                  {{ _t('Field visibility under deny-default. Leave Application/Model/Field empty for wider scopes.') }}
                </p>
                <ChoyOneToManyField :store="store" prop="FieldRules" label="">
                  <ChoyManyToOneRefField :store="store" prop="FieldRules.MetaApplicationId" />
                  <ChoyManyToOneRefField :store="store" prop="FieldRules.MetaModelId" />
                  <ChoyManyToOneRefField :store="store" prop="FieldRules.MetaFieldId" />
                  <ChoySelectionField :store="store" prop="FieldRules.PermRead" />
                  <ChoySelectionField :store="store" prop="FieldRules.PermWrite" />
                </ChoyOneToManyField>
              </div>
            </details>

            <details class="rounded-md border border-border" :open="advancedPanels === 'method_accesses'">
              <summary
                class="cursor-pointer select-none px-3 py-2 text-sm font-medium"
                @click.prevent="toggleAdvancedPanel('method_accesses')"
              >
                {{ _t('Method Access') }}
              </summary>
              <div class="border-t border-border px-3 py-3">
                <p class="mb-2 text-sm leading-relaxed text-foreground/70">
                  {{ _t('RPC allow/deny under deny-default. New rows default to allow; use deny as an explicit brake.') }}
                </p>
                <ChoyOneToManyField :store="store" prop="MethodAccesses" label="" :default-record="defaultMethodAccess">
                  <ChoyManyToOneRefField :store="store" prop="MethodAccesses.MetaApplicationId" />
                  <ChoyManyToOneRefField :store="store" prop="MethodAccesses.MetaModelId" />
                  <ChoyManyToOneRefField :store="store" prop="MethodAccesses.MetaServiceId" />
                  <ChoySelectionField :store="store" prop="MethodAccesses.Mode" />
                </ChoyOneToManyField>
              </div>
            </details>

            <details class="rounded-md border border-border" :open="advancedPanels === 'ui_resources'">
              <summary
                class="cursor-pointer select-none px-3 py-2 text-sm font-medium"
                @click.prevent="toggleAdvancedPanel('ui_resources')"
              >
                {{ _t('UI Resource Details (manual bypass)') }}
              </summary>
              <div class="border-t border-border px-3 py-3">
                <p class="mb-2 text-sm leading-relaxed text-foreground/70">
                  {{ _t('Secondary to the UI Resource Access tree above. Prefer the tree for day-to-day grants.') }}
                </p>
                <ChoyOneToManyField :store="store" prop="UiResources" label="">
                  <ChoySelectionField :store="store" prop="UiResources.Mode" />
                  <ChoyManyToOneRefField :store="store" prop="UiResources.MetaApplicationId" />
                  <ChoyManyToOneRefField :store="store" prop="UiResources.MetaUiResourceId" />
                </ChoyOneToManyField>
              </div>
            </details>
          </div>
        </ChoyTab>
      </ChoyTabs>
    </ChoyCard>
  </ChoyFormView>
</template>

<script setup lang="ts">
import { computed, defineAsyncComponent, ref } from 'vue';
import type { RouteLocationRaw } from 'vue-router';
import type { WebModelStore } from '@/web/web/stores/modelStore';
import type Role from '@/auth/service/models/role';

import { CircleHelp, GitBranch, Menu, Settings, User } from 'lucide-vue-next';

const UserListView = defineAsyncComponent(() => import('./UserListView.vue'));
const RoleListView = defineAsyncComponent(() => import('./RoleListView.vue'));
import { defineModelActions } from '@/core/web/resource';
import { usePermission } from '@/auth/web/composables/usePermission';
import { resolvePageStore } from '@/web/web/composables/usePageContext';
import { useI18n } from 'vue-i18n';
import { ChoyBooleanField, ChoyButtonBox, ChoyCard, ChoyCol, ChoyDatetimeField, ChoyFormView, ChoyGrid, ChoyJsonField, ChoyManyToManyField, ChoyManyToOneField, ChoyOneToManyField, ChoySelectionField, ChoyStatInfo, ChoyTab, ChoyTabs, ChoyVarcharField, ChoyManyToOneRefField, ChoyManyToManyRefTreeField} from '@/web';
import type { ChoyViewMode as ViewMode } from '@/web';
import { createTranslate, translateTerm } from '@/web/web/i18n';
import type { TermReference } from '@/core/service/i18n';
import { selectInspectedUiResource, getInspectedUiResourceId, getInspectedUiResourceRequires, isInspectedUiResourceRow } from '@/auth/web/views/role_ui_requires_explain';

defineOptions({ name: 'RoleFormView', inheritAttrs: true });
const { _t, _lt } = createTranslate('auth', { scope: 'web/views/RoleFormView' });
const requiredRules = computed(() => [{ required: true, message: _t('Required') }]);

const props = withDefaults(
  defineProps<{
    store?: WebModelStore<Role>;
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

const store = resolvePageStore(props.store, 'RoleFormView');
const { recordId, viewMode, showHeader, createAction } = props;
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
      return Menu;
    case 'ROUTE':
      return GitBranch;
    case 'ACTION':
      return Settings;
    default:
      return CircleHelp;
  }
}

const inspectedUiResource = ref<Record<string, any> | null>(null);

const inspectedUiResourceId = computed(() => getInspectedUiResourceId(inspectedUiResource.value));

const inspectedUiResourceLabel = computed(() => resolveUiResourceLabel(inspectedUiResource.value as UiResourceRow | undefined));

const inspectedRequires = computed(() => getInspectedUiResourceRequires(inspectedUiResource.value));

function inspectUiResource(row: any) {
  inspectedUiResource.value = selectInspectedUiResource(row);
}

/** New RecordRule rows default to grant (RoleId is always this role via O2M inverse). */
const defaultRecordRule: Record<string, any> = { Kind: 'grant' };

/** New MethodAccess rows default to allow (deny is an explicit brake). */
const defaultMethodAccess: Record<string, any> = { Mode: 'allow' };

const activeTab = ref('users');
const advancedPanels = ref('');

function toggleAdvancedPanel(name: string) {
  advancedPanels.value = advancedPanels.value === name ? '' : name;
}

function openRecordRules() {
  activeTab.value = 'advanced';
  advancedPanels.value = 'record_rules';
}

defineExpose({
  inspectUiResource,
  inspectedUiResource,
  inspectedUiResourceId,
  inspectedUiResourceLabel,
  inspectedRequires,
  activeTab,
  advancedPanels,
  openRecordRules,
  toggleAdvancedPanel,
  resolveUiResourceTypeIcon,
  resolveUiResourceLabel,
});
</script>

