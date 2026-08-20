<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <el-dropdown v-if="isAuthenticated" trigger="click" placement="bottom-end" @visible-change="handleVisibleChange">
    <el-badge :value="unreadCount || undefined" :hidden="unreadCount === 0" class="o-notification-bell">
      <el-button text class="o-notification-bell__button" :aria-label="_t('Notifications')">
        <el-icon :size="20">
          <Bell />
        </el-icon>
      </el-button>
    </el-badge>
    <template #dropdown>
      <el-dropdown-menu class="o-notification-bell__menu">
        <div class="o-notification-bell__toolbar">
          <span>{{ _t('Notifications') }}</span>
          <el-button v-if="unreadCount > 0" link type="primary" size="small" @click.stop="markAllRead">
            {{ _t('Mark all read') }}
          </el-button>
        </div>
        <div v-if="loading" class="o-notification-bell__empty">{{ _t('Loading...') }}</div>
        <div v-else-if="error" class="o-notification-bell__empty o-notification-bell__empty--error">{{ error }}</div>
        <div v-else-if="rows.length === 0" class="o-notification-bell__empty">{{ _t('No notifications') }}</div>
        <template v-else>
          <el-dropdown-item
            v-for="row in rows"
            :key="String(row.Id)"
            :class="{ 'is-unread': row.IsRead !== true }"
            @click="handleItemClick(row)"
          >
            <div class="o-notification-bell__item">
              <div class="o-notification-bell__item-title">
                {{ formatNotificationTitle(row) }}
              </div>
              <div class="o-notification-bell__item-meta">
                {{ formatUtcIso(row.CreatedAt, 'YYYY-MM-DD HH:mm') || '' }}
              </div>
            </div>
          </el-dropdown-item>
        </template>
      </el-dropdown-menu>
    </template>
  </el-dropdown>
</template>

<script setup lang="ts">
import { onMounted, onUnmounted, ref } from 'vue';
import { Bell } from '@element-plus/icons-vue';
import { ElBadge, ElButton, ElDropdown, ElDropdownItem, ElDropdownMenu, ElIcon } from 'element-plus';
import { useNotificationInbox, type InboxNotificationRow } from '@/web/web/composables/chatter/useNotificationInbox';
import { formatUtcIso } from '@/web/web/utils/datetime';
import { createTranslate } from '@/web/web/i18n';

const { _t } = createTranslate('web', { scope: 'web/components/layout/ONotificationBell' });
const isAuthenticated = ref(false);
let stopAuthSubscribe: (() => void) | undefined;
let disposed = false;

const inbox = useNotificationInbox(() => isAuthenticated.value);
const { rows, loading, error, unreadCount, markRead, markAllRead, activate, deactivate } = inbox;

onMounted(async () => {
  try {
    const { useAuthStore } = await import('@/auth/web/stores/auth');
    if (disposed) return;
    const authStore = useAuthStore();
    isAuthenticated.value = !!authStore.isAuthenticated;
    stopAuthSubscribe = (authStore as any).$subscribe?.(() => {
      if (disposed) return;
      const next = !!authStore.isAuthenticated;
      if (next === isAuthenticated.value) return;
      isAuthenticated.value = next;
      if (next) {
        void activate();
      } else {
        deactivate();
      }
    });
    if (isAuthenticated.value) {
      await activate();
    }
    if (disposed) {
      stopAuthSubscribe?.();
      stopAuthSubscribe = undefined;
      deactivate();
    }
  } catch {
    if (!disposed) {
      isAuthenticated.value = false;
    }
  }
});

onUnmounted(() => {
  disposed = true;
  stopAuthSubscribe?.();
  stopAuthSubscribe = undefined;
  deactivate();
});

async function handleVisibleChange(visible: boolean): Promise<void> {
  if (visible && isAuthenticated.value) {
    await inbox.refresh();
  }
}

function formatNotificationTitle(row: InboxNotificationRow): string {
  const model = String(row.Model || '').trim();
  const resId = String(row.ResId || '').trim();
  if (model && resId) {
    return _t('Update on %s (%s)', model, resId);
  }
  return _t('New notification');
}

async function handleItemClick(row: InboxNotificationRow): Promise<void> {
  const id = String(row.Id || '').trim();
  if (!id || row.IsRead === true) return;
  await markRead(id);
}
</script>

<style scoped>
.o-notification-bell__button {
  height: 36px;
  width: 36px;
  padding: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--el-text-color-regular);
}

.o-notification-bell__menu {
  min-width: 320px;
  max-width: 360px;
}

.o-notification-bell__toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  padding: 8px 12px;
  font-weight: 600;
  border-bottom: 1px solid var(--el-border-color-lighter);
}

.o-notification-bell__empty {
  padding: 16px 12px;
  text-align: center;
  color: var(--el-text-color-secondary);
  font-size: 13px;
}

.o-notification-bell__empty--error {
  color: var(--el-color-danger);
}

.o-notification-bell__item {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.o-notification-bell__item-title {
  font-size: 13px;
  color: var(--el-text-color-primary);
}

.o-notification-bell__item-meta {
  font-size: 12px;
  color: var(--el-text-color-secondary);
}

:global(.o-notification-bell__menu .el-dropdown-menu__item.is-unread) {
  background: var(--el-color-primary-light-9);
}
</style>
