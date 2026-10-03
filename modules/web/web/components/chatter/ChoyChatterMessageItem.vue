<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <Message
    class="choy-chatter-message"
    data-anchor="choy.chatter.message"
    align="start"
  >
    <MessageAvatar aria-hidden="true">
      <Avatar class="size-8">
        <AvatarFallback class="text-xs font-medium">{{ initials }}</AvatarFallback>
      </Avatar>
    </MessageAvatar>
    <MessageContent class="flex-1 gap-1">
      <MessageHeader class="justify-between gap-2 px-0">
        <span class="font-semibold text-foreground">{{ authorLabel }}</span>
        <span>{{ timeLabel }}</span>
      </MessageHeader>
      <Bubble variant="outline" align="start">
        <BubbleContent class="whitespace-pre-wrap break-words">
          {{ entry.body }}
        </BubbleContent>
      </Bubble>
    </MessageContent>
  </Message>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import type { ChatterMessageEntry } from './chatterTypes';
import { formatChoyUtcIso, resolveChoyChatterInitials } from './chatterHelpers';
import Avatar from '../vendor/ui/avatar/Avatar.vue';
import AvatarFallback from '../vendor/ui/avatar/AvatarFallback.vue';
import Bubble from '../vendor/ui/bubble/Bubble.vue';
import BubbleContent from '../vendor/ui/bubble/BubbleContent.vue';
import Message from '../vendor/ui/message/Message.vue';
import MessageAvatar from '../vendor/ui/message/MessageAvatar.vue';
import MessageContent from '../vendor/ui/message/MessageContent.vue';
import MessageHeader from '../vendor/ui/message/MessageHeader.vue';

const props = defineProps<{
  entry: ChatterMessageEntry;
  authorLabel: string;
}>();

const timeLabel = computed(() => formatChoyUtcIso(props.entry.at));
const initials = computed(() => resolveChoyChatterInitials(props.authorLabel));
</script>
