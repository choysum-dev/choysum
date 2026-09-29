<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <div
    v-if="chromeMode"
    data-anchor="choy.notification-bell"
    :class="cn('choy-notification-bell relative inline-flex', props.class)"
  >
    <Button
      variant="ghost"
      size="icon"
      type="button"
      :aria-label="ariaLabel"
      @click="emit('click')"
    >
      <Bell class="h-4 w-4" aria-hidden="true" />
    </Button>
    <Badge
      v-if="badgeText"
      aria-hidden="true"
      class="pointer-events-none absolute -right-1 -top-1 min-w-5 justify-center px-1 text-[10px]"
    >
      {{ badgeText }}
    </Badge>
    <span
      role="status"
      class="absolute h-px w-px overflow-hidden whitespace-nowrap opacity-0"
    >
      {{ unreadCount ? `${unreadCount} unread notifications` : '' }}
    </span>
  </div>
  <div v-else data-testid="inbox-engine" :class="props.class">
    <DropdownMenu v-if="isAuthenticated" @update:open="handleVisibleChange">
      <DropdownMenuTrigger as-child>
        <button type="button" class="choy-notification-bell__button" :aria-label="_t('Notifications')">
          <Bell class="size-5" />
          <span v-if="inboxUnreadCount > 0" class="choy-notification-bell__badge">{{ inboxUnreadCount }}</span>
        </button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end" class="choy-notification-bell__menu">
        <div class="choy-notification-bell__toolbar">
          <span>{{ _t('Notifications') }}</span>
          <ChoyButton
            v-if="inboxUnreadCount > 0"
            size="sm"
            variant="link"
            data-test="notification-mark-all-read"
            @click.stop="markAllRead"
          >
            {{ _t('Mark all read') }}
          </ChoyButton>
        </div>
        <div v-if="loading" class="choy-notification-bell__empty">{{ _t('Loading...') }}</div>
        <div v-else-if="error" class="choy-notification-bell__empty choy-notification-bell__empty--error">{{ error }}</div>
        <div v-else-if="rows.length === 0" class="choy-notification-bell__empty">{{ _t('No notifications') }}</div>
        <template v-else>
          <DropdownMenuItem
            v-for="row in rows"
            :key="String(row.Id)"
            @select="() => handleItemClick(row)"
          >
            <div class="choy-notification-bell__item">
              <div class="choy-notification-bell__item-title">
                {{ formatNotificationTitle(row) }}
              </div>
              <div class="choy-notification-bell__item-meta">
                {{ formatUtcIso(row.CreatedAt, 'YYYY-MM-DD HH:mm') || '' }}
              </div>
            </div>
          </DropdownMenuItem>
        </template>
      </DropdownMenuContent>
    </DropdownMenu>
  </div>
</template>

<script setup lang="ts">
import { Bell } from 'lucide-vue-next';
import { computed, onMounted, onUnmounted, ref } from 'vue';
import Badge from '../vendor/ui/badge/Badge.vue';
import Button from '../vendor/ui/button/Button.vue';
import DropdownMenu from '../vendor/ui/dropdown-menu/DropdownMenu.vue';
import DropdownMenuContent from '../vendor/ui/dropdown-menu/DropdownMenuContent.vue';
import DropdownMenuItem from '../vendor/ui/dropdown-menu/DropdownMenuItem.vue';
import DropdownMenuTrigger from '../vendor/ui/dropdown-menu/DropdownMenuTrigger.vue';
import ChoyButton from './ChoyButton.vue';
import { cn, type ClassValue } from '../../lib/utils';
import {
  choyNotificationAriaLabel,
  choyNotificationBadgeText,
  choyNotificationUnreadCount
} from './choyNotificationBellChrome';
import { isChoyNotificationInboxMode } from './choyNotificationBellMode';
import {
  useInjectedNotificationInbox,
  type InboxNotificationRow
} from '@/web/web/composables/chatter/useNotificationInbox';
import { formatUtcIso } from '@/web/web/utils/datetime';
import { createTranslate } from '@/web/web/i18n';

defineOptions({ name: 'ChoyNotificationBell', inheritAttrs: false });

const { _t } = createTranslate('web', { scope: 'web/components/layout/ChoyNotificationBell' });

const props = defineProps<{
  class?: ClassValue;
  count?: number;
  label?: string;
}>();

const emit = defineEmits<{
  click: [];
}>();

const chromeMode = computed(() => !isChoyNotificationInboxMode(props.count));
const unreadCount = computed(() => choyNotificationUnreadCount(props.count));
const badgeText = computed(() => choyNotificationBadgeText(unreadCount.value));
const ariaLabel = computed(() => choyNotificationAriaLabel(props.label, unreadCount.value));

const isAuthenticated = ref(false);
let stopAuthSubscribe: (() => void) | undefined;
let disposed = false;

const inbox = useInjectedNotificationInbox(() => isAuthenticated.value);
const { rows, loading, error, unreadCount: inboxUnreadCount, refresh, markRead, markAllRead, activate, deactivate } =
  inbox;

onMounted(async () => {
  if (chromeMode.value) return;
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
  if (!chromeMode.value) {
    deactivate();
  }
});

function handleVisibleChange(open: boolean) {
  if (open && isAuthenticated.value) {
    void refresh();
  }
}

function formatNotificationTitle(row: InboxNotificationRow): string {
  const model = typeof row.Model === 'string' ? row.Model.trim() : '';
  const resId = row.ResId != null ? String(row.ResId).trim() : '';
  if (model && resId) {
    return _t('Update on %s (%s)', model, resId);
  }
  return _t('New notification');
}

async function handleItemClick(row: InboxNotificationRow) {
  if (row.Id == null || String(row.Id).trim() === '') return;
  if (row.IsRead !== true) {
    await markRead(String(row.Id));
  }
}
</script>

<style scoped>
.choy-notification-bell__button {
  position: relative;display: inline-flex;align-items: center;justify-content: center;padding: 0.375rem;border: 0;background: transparent;color: var(--choy-color-foreground, inherit);cursor: pointer;
}
.choy-notification-bell__badge {
  position: absolute;top: 0;right: 0;min-width: 1rem;padding: 0 0.25rem;border-radius: 999px;background: var(--choy-color-danger, #dc2626);color: #fff;font-size: 10px;line-height: 1rem;text-align: center;
}
.choy-notification-bell__toolbar {
  display: flex;align-items: center;justify-content: space-between;gap: 0.5rem;padding: 0.5rem 0.75rem;border-bottom: 1px solid var(--choy-color-border, #e5e7eb);color: var(--choy-color-muted-foreground, #6b7280);font-size: 0.875rem;
}
.choy-notification-bell__empty {
  padding: 0.75rem;color: var(--choy-color-muted-foreground, #6b7280);font-size: 0.875rem;
}
.choy-notification-bell__empty--error {
  color: var(--choy-color-danger, #dc2626);
}
.choy-notification-bell__item {
  display: flex;flex-direction: column;gap: 0.25rem;min-width: 16rem;
}
.choy-notification-bell__item-title {
  color: var(--choy-color-foreground, inherit);font-size: 0.875rem;
}
.choy-notification-bell__item-meta {
  color: var(--choy-color-muted-foreground, #6b7280);font-size: 0.75rem;
}
:global(.choy-notification-bell__menu .is-unread) {
  background: color-mix(in oklab, var(--choy-color-primary, #2563eb) 12%, transparent);
}
</style>
