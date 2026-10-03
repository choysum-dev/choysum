<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<script setup lang="ts">
import { computed } from 'vue'
import { ChevronRight } from 'lucide-vue-next'
import type { MenuItem } from '@/core/web/menu'
import Collapsible from '../vendor/ui/collapsible/Collapsible.vue'
import CollapsibleContent from '../vendor/ui/collapsible/CollapsibleContent.vue'
import CollapsibleTrigger from '../vendor/ui/collapsible/CollapsibleTrigger.vue'
import {
  SidebarMenuButton,
  SidebarMenuItem,
  SidebarMenuSub,
  SidebarMenuSubButton,
  SidebarMenuSubItem,
} from '../vendor/ui/sidebar/index'

defineOptions({ name: 'ChoySidebarNavEntry' })

/**
 * One sidebar nav node (group or leaf). Recurses for nested children at any depth.
 */
const props = defineProps<{
  item: MenuItem
  depth: number
  labelOf: (item: MenuItem) => string
  iconOf: (item: MenuItem) => unknown
  leafActive: (item: MenuItem) => boolean
  groupExpanded: (item: MenuItem) => boolean
  onGroupOpenChange: (item: MenuItem, open: boolean) => void
  onLeafClick: (item: MenuItem) => void
}>()

const visibleChildren = computed(() => (props.item.children || []).filter((c) => !c.hidden))
const isGroup = computed(() => visibleChildren.value.length > 0)
const Wrapper = computed(() => (props.depth === 0 ? SidebarMenuItem : SidebarMenuSubItem))
const Button = computed(() => (props.depth === 0 ? SidebarMenuButton : SidebarMenuSubButton))
</script>

<template>
  <component :is="Wrapper">
    <Collapsible
      v-if="isGroup"
      :open="groupExpanded(item)"
      class="group/collapsible"
      @update:open="(v) => onGroupOpenChange(item, !!v)"
    >
      <CollapsibleTrigger as-child>
        <component
          :is="Button"
          as="button"
          type="button"
          :tooltip="depth === 0 ? labelOf(item) : undefined"
          :disabled="item.disabled || undefined"
          :aria-expanded="groupExpanded(item) ? 'true' : 'false'"
          data-testid="choy-sidebar-nav-group"
        >
          <component :is="iconOf(item)" />
          <span>{{ labelOf(item) }}</span>
          <ChevronRight
            v-if="depth === 0"
            class="ms-auto transition-transform group-data-[state=open]/collapsible:rotate-90"
          />
        </component>
      </CollapsibleTrigger>
      <CollapsibleContent>
        <SidebarMenuSub>
          <ChoySidebarNavEntry
            v-for="child in visibleChildren"
            :key="child.id || child.path || child.title"
            :item="child"
            :depth="depth + 1"
            :label-of="labelOf"
            :icon-of="iconOf"
            :leaf-active="leafActive"
            :group-expanded="groupExpanded"
            :on-group-open-change="onGroupOpenChange"
            :on-leaf-click="onLeafClick"
          />
        </SidebarMenuSub>
      </CollapsibleContent>
    </Collapsible>
    <component
      v-else
      :is="Button"
      as="button"
      type="button"
      :is-active="leafActive(item)"
      :tooltip="depth === 0 ? labelOf(item) : undefined"
      :disabled="item.disabled || undefined"
      data-testid="choy-sidebar-nav-leaf"
      @click="onLeafClick(item)"
    >
      <component :is="iconOf(item)" />
      <span>{{ labelOf(item) }}</span>
    </component>
  </component>
</template>
