<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <TooltipProvider v-if="resolved.length > 1" :delay-duration="delayDuration">
    <ChoyButtonGroup
      data-testid="choy-view-switcher"
      :aria-label="ariaLabel"
      :class="props.class"
    >
      <Tooltip v-for="opt in resolved" :key="opt.value">
        <TooltipTrigger as-child>
          <ChoyButton
            type="button"
            size="icon"
            variant="outline"
            :disabled="opt.disabled"
            :aria-pressed="opt.value === active"
            :aria-label="opt.label"
            :data-testid="`choy-view-switcher-${opt.value}`"
            :class="opt.value === active ? 'z-[1] bg-muted text-foreground hover:bg-muted' : undefined"
            @click="onSelect(opt)"
          >
            <component :is="opt.icon" class="size-4" aria-hidden="true" />
          </ChoyButton>
        </TooltipTrigger>
        <TooltipContent side="bottom" :side-offset="6">
          {{ opt.label }}
        </TooltipContent>
      </Tooltip>
    </ChoyButtonGroup>
  </TooltipProvider>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import ChoyButton from '../layout/ChoyButton.vue';
import ChoyButtonGroup from '../layout/ChoyButtonGroup.vue';
import Tooltip from '../vendor/ui/tooltip/Tooltip.vue';
import TooltipContent from '../vendor/ui/tooltip/TooltipContent.vue';
import TooltipProvider from '../vendor/ui/tooltip/TooltipProvider.vue';
import TooltipTrigger from '../vendor/ui/tooltip/TooltipTrigger.vue';
import type { ClassValue } from '../../lib/utils';
import {
  resolveChoyViewSwitcherOptions,
  type ChoyViewSwitcherOption,
  type ChoyViewSwitcherResolvedOption,
} from './choyViewSwitcherPresets';

defineOptions({ name: 'ChoyViewSwitcher' });

export type { ChoyViewSwitcherOption } from './choyViewSwitcherPresets';

const props = withDefaults(
  defineProps<{
    class?: ClassValue;
    /** Currently selected view value (must match an option `value`). */
    active: string;
    /**
     * Ordered view options (list / kanban / chart / …).
     * Icons resolve from builtins when omitted; unknown kinds need `icon`.
     */
    options: ChoyViewSwitcherOption[];
    ariaLabel?: string;
    /** Hover delay before the tip opens (ms). */
    delayDuration?: number;
  }>(),
  {
    ariaLabel: 'View switcher',
    delayDuration: 200,
  },
);

const emit = defineEmits<{
  select: [value: string];
}>();

const resolved = computed(() => resolveChoyViewSwitcherOptions(props.options));

function onSelect(opt: ChoyViewSwitcherResolvedOption) {
  if (opt.disabled) return;
  emit('select', opt.value);
}
</script>
