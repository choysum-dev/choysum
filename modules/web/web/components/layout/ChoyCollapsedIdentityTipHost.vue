<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <!--
    Tip wraps a layout host, not the menu/popover trigger. Putting Tooltip on
    SidebarMenuButton (via :tooltip) makes Dropdown/Popover as-child anchor to
    TooltipRoot and parks the panel at the viewport origin.
  -->
  <Tooltip>
    <TooltipTrigger as-child>
      <div class="w-full min-w-0">
        <slot />
      </div>
    </TooltipTrigger>
    <TooltipContent
      side="right"
      align="center"
      :hidden="state !== 'collapsed' || isMobile || !label"
    >
      {{ label }}
    </TooltipContent>
  </Tooltip>
</template>

<script lang="ts" setup>
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from '../vendor/ui/tooltip';
import { useSidebar } from '../vendor/ui/sidebar/index';

defineOptions({ name: 'ChoyCollapsedIdentityTipHost' });

defineProps<{
  /** Collapsed-rail tip text; omit while the panel/menu is open. */
  label?: string;
}>();

const { state, isMobile } = useSidebar();
</script>
