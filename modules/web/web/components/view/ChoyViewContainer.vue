<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<!--
  Region spacing (header ↔ body ↔ footer) is flex gap on the root. Shell canvas
  is flush. When sheetSurface is on, view-body is the floating paper
  (choy.view.sheet) and view-header shares the same gutter so Action Bar tracks it.
-->
<template>
  <div
    data-slot="view-container"
    :class="
      cn(
        'flex h-full min-h-0 w-full min-w-0 flex-1 flex-col',
        !hasGapUtility(props.class) ? 'gap-3' : undefined,
        props.class,
      )
    "
  >
    <!--
      flex + gap owns Action Bar ↔ header-extra rhythm (Form statusbar /
      Chart controls). Children should not add vertical padding for that gap.
    -->
    <div
      v-if="regionHasContent('header')"
      data-slot="view-header"
      :class="headerClass"
    >
      <slot name="header" />
    </div>
    <div
      data-slot="view-body"
      data-anchor="choy.view.sheet"
      :data-sheet-surface="sheetSurface ? 'true' : 'false'"
      :class="bodyClass"
    >
      <slot />
    </div>
    <div v-if="regionHasContent('footer')" data-slot="view-footer" class="shrink-0">
      <slot name="footer" />
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, useSlots } from 'vue';
import { cn, type ClassValue } from '../../lib/utils';
import { slotHasContent } from '../button/statInfoHelpers';

defineOptions({ name: 'ChoyViewContainer' });

const props = withDefaults(
  defineProps<{
    class?: ClassValue;
    /**
     * Floating paper on view-body (bg / radius / shadow + gutter mx).
     * Off for embedded / dialog hosts so they do not double-frame.
     */
    sheetSurface?: boolean;
  }>(),
  {
    sheetSurface: true,
  },
);

const slots = useSlots();

const sheetSurface = computed(() => props.sheetSurface !== false);

/** Shared inline gutter when the paper is on (Action Bar tracks the sheet). */
const sheetGutterClass = 'mx-[var(--choy-view-sheet-gutter)]';

const headerClass = computed(() =>
  cn('flex shrink-0 flex-col gap-2', sheetSurface.value ? sheetGutterClass : undefined),
);

const bodyClass = computed(() =>
  cn(
    'flex min-h-0 min-w-0 flex-1 flex-col overflow-hidden',
    sheetSurface.value
      ? [
          sheetGutterClass,
          // rounded-md == --choy-view-sheet-radius / nav-pane; hairline keeps
          // the edge readable when pane ≈ card white.
          'rounded-md border border-border bg-card shadow-xs',
        ]
      : undefined,
  ),
);

/** Skip empty / comment-only header|footer so flex gap does not leave blank bands. */
function regionHasContent(name: 'header' | 'footer'): boolean {
  const raw = slots[name]?.();
  if (raw == null) return false;
  return slotHasContent(Array.isArray(raw) ? raw : [raw]);
}

/** True when a class token is a full (both-axis) Tailwind `gap-*` utility. */
function isFullGapToken(token: string): boolean {
  const base = token.includes(':') ? token.slice(token.lastIndexOf(':') + 1) : token;
  // Axis-specific gap-x-*/gap-y-* must not suppress the container's default gap-3.
  return /^(?:!)?gap-(?!x-|y-)/.test(base);
}

/** True when `class` already sets a Tailwind gap-* (so the default gap-3 is skipped). */
function hasGapUtility(value: ClassValue | undefined): boolean {
  if (value == null || value === false || value === '') return false;
  if (typeof value === 'string') return value.split(/\s+/).some((part) => isFullGapToken(part));
  if (typeof value === 'number') return false;
  if (Array.isArray(value)) return value.some((item) => hasGapUtility(item as ClassValue));
  if (typeof value === 'object') {
    return Object.entries(value).some(([key, on]) => !!on && isFullGapToken(key));
  }
  return false;
}
</script>
