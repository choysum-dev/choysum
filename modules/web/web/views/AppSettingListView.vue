<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <ChoyListView
    ref="listRef"
    v-bind="$attrs"
    :store="store"
    :searchView="ChoySearchView"
    :create-action="createPath"
    :action-ids="{
      create: appSettingActions.create,
      delete: appSettingActions.delete,
    }"
    :has-action="hasAction"
    @row-click="onRowClick"
  >
    <template v-if="showSettingsSwitcher" #header-right>
      <ChoyViewSwitcher
        :aria-label="_t('View switcher')"
        :active="activeView"
        :options="viewSwitcherOptions"
        @select="onViewSelect"
      />
    </template>
    <ChoyTableColumn type="selection" :vColumnProps="{ align: 'center' }" />
    <ChoyTableColumn type="index" :vColumnProps="{ align: 'end', width: 44 }" />
    <ChoyVarcharField
      :store="store"
      prop="Key"
      :label="_t('Key')"
      :vColumnProps="{ minWidth: 180 }"
    />
    <ChoyTextField
      :store="store"
      prop="Value"
      :label="_t('Value')"
      :vColumnProps="{ minWidth: 220 }"
    />
  </ChoyListView>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import type { WebModelStore } from '@/web/web/stores/modelStore';
import { useListViewExpose } from '@/web/web/composables/useListView';
import { resolvePageStore } from '@/web/web/composables/usePageContext';
import { usePermission } from '@/auth/web/composables/usePermission';
import { createTranslate } from '@/web/web/i18n';
import {
  ChoyListView,
  ChoySearchView,
  ChoyTableColumn,
  ChoyTextField,
  ChoyVarcharField,
  ChoyViewSwitcher,
  resolveChoyViewRoutePath,
  useChoyViewRouteSwitch,
  type ChoyViewSwitcherOption,
  type RowEventPayload,
} from '@/web';
import { appSettingActions } from './app_setting_actions';

defineOptions({ name: 'AppSettingListView', inheritAttrs: true });
const { _t } = createTranslate('web', { scope: 'web/views/AppSettingListView' });

const router = useRouter();
const route = useRoute();

const props = defineProps<{
  store?: WebModelStore<any>;
  application: string;
}>();

const application = computed(() => String(props.application || '').trim());
const store = resolvePageStore(props.store, 'AppSettingListView');
const createPath = computed(() => {
  const app = application.value;
  return app ? `/web/app-settings/${app}/new` : '';
});

const { hasAction } = usePermission();
const { active: activeView, selectView: onViewSelect } = useChoyViewRouteSwitch();

const showSettingsSwitcher = computed(
  () => resolveChoyViewRoutePath(route.meta, 'settings') !== '',
);

const viewSwitcherOptions = computed<ChoyViewSwitcherOption[]>(() => [
  { value: 'settings', label: _t('Settings View') },
  { value: 'list', label: _t('Parameters View') },
]);

function onRowClick(payload: RowEventPayload) {
  const id = String(payload.row?.Id ?? '').trim();
  const app = application.value;
  if (id && app) router.push(`/web/app-settings/${app}/${id}`);
}

const { listRef, expose } = useListViewExpose<any>();
defineExpose(expose);
</script>
