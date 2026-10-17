<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <div
    data-slot="view-action-bar"
    :data-layout="props.layout"
    :class="
      cn(
        // Region gap (ActionBar ↔ body) belongs on ChoyViewContainer.
        // pb-* here is only air between controls and the hairline when bordered.
        'items-center gap-3 border-b border-border min-h-control',
        !isBorderless(props.class) ? 'pb-2' : undefined,
        props.layout === 'thirds'
          ? 'grid grid-cols-[auto_1fr_auto] max-md:grid-cols-1'
          : 'flex justify-between max-md:flex-col max-md:items-stretch',
        props.class,
      )
    "
  >
    <!-- Keep the first thirds-grid column even when actions are empty so center/right do not shift. -->
    <div
      v-if="$slots['system-actions'] || $slots['user-actions'] || props.layout === 'thirds'"
      :class="
        cn(
          'flex min-w-0 items-center gap-3 max-md:flex-col max-md:items-stretch',
          props.layout === 'split' ? 'flex-1' : '',
        )
      "
    >
      <ChoyActionTray v-if="$slots['system-actions']" :aria-label="props.systemActionsLabel">
        <slot name="system-actions" />
      </ChoyActionTray>
      <template v-if="$slots['system-actions'] && $slots['user-actions']">
        <Separator
          orientation="vertical"
          class="mx-0 !h-4 w-px shrink-0 max-md:hidden"
        />
        <div class="hidden max-md:block w-full border-t border-border" aria-hidden="true" />
      </template>
      <ChoyActionTray
        v-if="$slots['user-actions']"
        :aria-label="props.userActionsLabel"
        class="max-md:pt-0"
      >
        <slot name="user-actions" />
      </ChoyActionTray>
    </div>
    <div v-if="$slots.center" class="min-w-0 flex-1">
      <slot name="center" />
    </div>
    <div
      v-if="$slots['header-right']"
      class="flex items-center justify-end gap-3 max-md:justify-center"
    >
      <slot name="header-right" />
    </div>
  </div>
</template>

<script setup lang="ts">
import ChoyActionTray from '../layout/ChoyActionTray.vue';
import { Separator } from '../vendor/ui/separator';
import { cn, type ClassValue } from '../../lib/utils';

defineOptions({ name: 'ChoyViewActionBar' });

const props = withDefaults(
  defineProps<{
    class?: ClassValue;
    /** split = flex toolbar; thirds = grid with a center column. */
    layout?: 'split' | 'thirds';
    systemActionsLabel?: string;
    userActionsLabel?: string;
  }>(),
  {
    layout: 'split',
    systemActionsLabel: 'System actions',
    userActionsLabel: 'User actions',
  },
);

/** True when the consumer clears the bottom hairline (no pb air needed). */
function isBorderless(value: ClassValue | undefined): boolean {
  if (value == null || value === false || value === '') return false;
  if (typeof value === 'string') {
    return /(?:^|\s)(?:!)?border-b-0(?:\s|$)/.test(value) || /\[border-bottom-width:0px\]/.test(value);
  }
  if (typeof value === 'number') return false;
  if (Array.isArray(value)) return value.some((item) => isBorderless(item as ClassValue));
  if (typeof value === 'object') {
    return Object.entries(value).some(
      ([key, on]) =>
        !!on &&
        (/(?:^|:)(?:!)?border-b-0(?:$|:)/.test(key) || key.includes('[border-bottom-width:0px]')),
    );
  }
  return false;
}
</script>
