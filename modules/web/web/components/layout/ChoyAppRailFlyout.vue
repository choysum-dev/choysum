<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<script setup lang="ts">
import { computed } from 'vue'
import type { MenuItem } from '@/core/web/menu'
import {
  DropdownMenuGroup,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuSub,
  DropdownMenuSubContent,
  DropdownMenuSubTrigger,
} from '../vendor/ui/dropdown-menu'
import { buildAppNavGroups, visibleMenuChildren, type AppNavGroup } from './buildAppNavGroups'

defineOptions({ name: 'ChoyAppRailFlyout' })

const props = defineProps<{
  /** App root: group like ChoySidebarNav. Nested nodes pass `mode="flat"`. */
  root: MenuItem
  labelOf: (item: MenuItem) => string
  onLeafSelect: (item: MenuItem) => void
  /** grouped = top-level app flyout; flat = submenu of a branch item's children. */
  mode?: 'grouped' | 'flat'
}>()

const mode = computed(() => props.mode || 'grouped')

const groups = computed((): AppNavGroup[] => {
  if (mode.value !== 'grouped') return []
  return buildAppNavGroups(props.root, props.labelOf)
})

const flatItems = computed((): MenuItem[] => {
  if (mode.value !== 'flat') return []
  return visibleMenuChildren(props.root)
})

function itemKey(item: MenuItem): string {
  return item.id || item.path || item.title || ''
}

function isBranch(item: MenuItem): boolean {
  return visibleMenuChildren(item).length > 0
}
</script>

<template>
  <template v-if="mode === 'grouped'">
    <template v-for="(group, gi) in groups" :key="group.key">
      <DropdownMenuSeparator v-if="gi > 0" />
      <DropdownMenuGroup>
        <DropdownMenuLabel v-if="group.label" data-testid="choy-shell-app-rail-flyout-group">
          {{ group.label }}
        </DropdownMenuLabel>
        <template v-for="item in group.items" :key="itemKey(item)">
          <DropdownMenuSub v-if="isBranch(item)">
            <DropdownMenuSubTrigger data-testid="choy-shell-app-rail-flyout-sub">
              {{ labelOf(item) }}
            </DropdownMenuSubTrigger>
            <DropdownMenuSubContent>
              <ChoyAppRailFlyout
                :root="item"
                mode="flat"
                :label-of="labelOf"
                :on-leaf-select="onLeafSelect"
              />
            </DropdownMenuSubContent>
          </DropdownMenuSub>
          <DropdownMenuItem
            v-else
            :disabled="!!item.disabled"
            data-testid="choy-shell-app-rail-flyout-leaf"
            @select="onLeafSelect(item)"
          >
            {{ labelOf(item) }}
          </DropdownMenuItem>
        </template>
      </DropdownMenuGroup>
    </template>
  </template>
  <template v-else>
    <template v-for="item in flatItems" :key="itemKey(item)">
      <DropdownMenuSub v-if="isBranch(item)">
        <DropdownMenuSubTrigger data-testid="choy-shell-app-rail-flyout-sub">
          {{ labelOf(item) }}
        </DropdownMenuSubTrigger>
        <DropdownMenuSubContent>
          <ChoyAppRailFlyout
            :root="item"
            mode="flat"
            :label-of="labelOf"
            :on-leaf-select="onLeafSelect"
          />
        </DropdownMenuSubContent>
      </DropdownMenuSub>
      <DropdownMenuItem
        v-else
        :disabled="!!item.disabled"
        data-testid="choy-shell-app-rail-flyout-leaf"
        @select="onLeafSelect(item)"
      >
        {{ labelOf(item) }}
      </DropdownMenuItem>
    </template>
  </template>
</template>
