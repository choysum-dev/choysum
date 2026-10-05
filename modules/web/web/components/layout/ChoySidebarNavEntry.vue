<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<script setup lang="ts">
import { computed } from 'vue'
import { useRouter } from 'vue-router'
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
 * Renders item.icon only when the menu declares one.
 */
const props = defineProps<{
  item: MenuItem
  depth: number
  labelOf: (item: MenuItem) => string
  leafActive: (item: MenuItem) => boolean
  groupExpanded: (item: MenuItem) => boolean
  onGroupOpenChange: (item: MenuItem, open: boolean) => void
  onLeafClick: (item: MenuItem) => void
}>()

let router: ReturnType<typeof useRouter> | null = null
try {
  router = useRouter()
} catch {
  router = null
}

const visibleChildren = computed(() => (props.item.children || []).filter((c) => !c.hidden))
const isGroup = computed(() => visibleChildren.value.length > 0)
const Wrapper = computed(() => (props.depth === 0 ? SidebarMenuItem : SidebarMenuSubItem))
const Button = computed(() => (props.depth === 0 ? SidebarMenuButton : SidebarMenuSubButton))

/**
 * Resolves the leaf href, including the router base for in-app paths.
 */
function leafHref(item: MenuItem): string | undefined {
  if (!item.path) return undefined
  if (item.externalLink) return item.path
  if (!router) return item.path
  try {
    return router.resolve(item.path).href || item.path
  } catch {
    return item.path
  }
}

/**
 * Maps menu openMode to an <a> target for external leaves.
 */
function leafTarget(item: MenuItem): string | undefined {
  if (!item.externalLink) return undefined
  switch (item.openMode) {
    case 'window':
      return '_blank'
    case 'parent':
      return '_parent'
    case 'top':
      return '_top'
    default:
      return '_self'
  }
}

/**
 * Left-click in-app leaves go through navigateTo; modified/external clicks stay native.
 */
function onLeafActivate(item: MenuItem, e: MouseEvent) {
  if (item.disabled) {
    e.preventDefault()
    return
  }
  if (e.metaKey || e.ctrlKey || e.shiftKey || e.altKey || e.button !== 0) return
  if (item.externalLink) return
  e.preventDefault()
  props.onLeafClick(item)
}
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
          class="text-foreground hover:bg-accent/30"
        >
          <component :is="item.icon" v-if="item.icon" />
          <span>{{ labelOf(item) }}</span>
          <ChevronRight
            v-if="depth === 0"
            class="ms-auto text-muted-foreground/70 transition-transform group-data-[state=open]/collapsible:rotate-90"
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
      :as="item.path ? 'a' : 'button'"
      :href="item.disabled ? undefined : leafHref(item)"
      :target="leafTarget(item)"
      :rel="leafTarget(item) === '_blank' ? 'noopener noreferrer' : undefined"
      :type="item.path ? undefined : 'button'"
      :is-active="leafActive(item)"
      :tooltip="depth === 0 ? labelOf(item) : undefined"
      :aria-disabled="item.disabled || undefined"
      :tabindex="item.disabled ? -1 : undefined"
      :aria-current="leafActive(item) ? 'page' : undefined"
      data-testid="choy-sidebar-nav-leaf"
      @click="onLeafActivate(item, $event)"
    >
      <component :is="item.icon" v-if="item.icon" />
      <span>{{ labelOf(item) }}</span>
    </component>
  </component>
</template>
