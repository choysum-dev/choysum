<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <Message
    class="choy-chatter-field-change"
    data-anchor="choy.chatter.field-change"
    align="start"
  >
    <MessageAvatar aria-hidden="true">
      <Avatar class="size-8">
        <AvatarFallback class="text-xs font-medium">{{ initials }}</AvatarFallback>
      </Avatar>
    </MessageAvatar>
    <div class="flex min-w-0 flex-1 flex-col gap-1">
      <MessageHeader class="justify-between gap-2 px-0">
        <span class="font-semibold text-foreground">{{ authorLabel }}</span>
        <span>{{ timeLabel }}</span>
      </MessageHeader>
      <Bubble variant="outline" align="start" class="border-l-[3px] border-l-info bg-muted/40">
        <BubbleContent class="whitespace-pre-wrap break-words text-foreground/90">
          {{ summary }}
        </BubbleContent>
      </Bubble>
    </div>
  </Message>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import type { ChatterFieldChangeEntry } from './chatterTypes';
import { formatChoyUtcIso, formatFieldChangeSummary, resolveChoyChatterInitials } from './chatterHelpers';
import Avatar from '../vendor/ui/avatar/Avatar.vue';
import AvatarFallback from '../vendor/ui/avatar/AvatarFallback.vue';
import Bubble from '../vendor/ui/bubble/Bubble.vue';
import BubbleContent from '../vendor/ui/bubble/BubbleContent.vue';
import Message from '../vendor/ui/message/Message.vue';
import MessageAvatar from '../vendor/ui/message/MessageAvatar.vue';
import MessageHeader from '../vendor/ui/message/MessageHeader.vue';

const props = withDefaults(
  defineProps<{
    entry: ChatterFieldChangeEntry;
    authorLabel: string;
    labels?: {
      created?: string;
      unlinked?: string;
      changed?: (field: string, oldValue: string, newValue: string) => string;
      action?: (name: string) => string;
      fieldFallback?: string;
    };
  }>(),
  {
    labels: () => ({}),
  },
);

const timeLabel = computed(() => formatChoyUtcIso(props.entry.at));

const summary = computed(() =>
  formatFieldChangeSummary(props.entry, {
    created: props.labels.created || 'Record created',
    unlinked: props.labels.unlinked || 'Record removed',
    changed:
      props.labels.changed ||
      ((field, oldValue, newValue) => `${field} changed from ${oldValue} to ${newValue}`),
    action: props.labels.action || (name => `Action: ${name}`),
    fieldFallback: props.labels.fieldFallback || 'Field',
  }),
);

const initials = computed(() => resolveChoyChatterInitials(props.authorLabel));
</script>
