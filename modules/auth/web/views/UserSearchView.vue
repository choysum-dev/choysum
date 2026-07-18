<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <OSearchView
    :store="store"
    :default-filters="[
      {
        name: _t('Active'),
        query: ['IsActive', '=', true],
        selected: true,
      },
    ]"
    :default-groups="defaultGroups"
    :initial-emit="true"
    @query-update="onQueryUpdate"
  />
</template>

<script setup lang="ts">
import type User from '@/auth/service/models/user';
import type { WebModelStore } from '@/web/web/stores/modelStore';
import type { GroupBySpec, QueryUpdatePayload } from '@/web/web/query/types';
import OSearchView from '@/web/web/components/view/OSearchView.vue';
import { createTranslate } from '@/web/web/i18n';

const { _t } = createTranslate('auth', { scope: 'web/views/UserSearchView' });

// Seed the initial group only once, then let store.queryState drive subsequent changes.
const defaultGroups = ['IsActive'] as GroupBySpec<User>[];

const props = defineProps<{ store: WebModelStore<User>; test: string }>();
const { store } = props;

defineOptions({ name: 'UserSearchView' });

const emit = defineEmits<{ (e: 'query-update', payload: QueryUpdatePayload<User>): void }>();

/**
 * Forward the inner search view query event to parent list containers.
 */
function onQueryUpdate(payload: QueryUpdatePayload<User>) {
  emit('query-update', payload);
}
</script>
