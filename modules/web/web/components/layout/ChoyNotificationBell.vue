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
        <button type="button" class="relative inline-flex cursor-pointer items-center justify-center border-0 bg-transparent p-1.5 text-foreground" :aria-label="_t('Notifications')">
          <Bell class="size-5" />
          <span v-if="inboxUnreadCount > 0" class="absolute right-0 top-0 min-w-4 rounded-full bg-danger px-1 text-center text-[10px] leading-4 text-white">{{ inboxUnreadCount }}</span>
        </button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end" class="choy-notification-bell__menu">
        <div class="flex items-center justify-between gap-2 border-b border-border px-3 py-2 text-sm text-muted-foreground">
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
        <div v-if="loading" class="p-3 text-sm text-muted-foreground">{{ _t('Loading...') }}</div>
        <div v-else-if="error" class="p-3 text-sm text-danger">{{ error }}</div>
        <div v-else-if="rows.length === 0" class="p-3 text-sm text-muted-foreground">{{ _t('No notifications') }}</div>
        <template v-else>
          <DropdownMenuItem
            v-for="row in rows"
            :key="String(row.Id)"
            :class="row.IsRead !== true ? 'is-unread bg-primary/10' : undefined"
            @select="() => handleItemClick(row)"
          >
            <div class="flex min-w-64 flex-col gap-1">
              <div class="text-sm text-foreground">
                {{ formatNotificationTitle(row) }}
              </div>
              <div class="text-xs text-muted-foreground">
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

