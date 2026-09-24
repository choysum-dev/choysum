<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<script setup lang="ts">
import { computed } from 'vue';
import { CircleHelp } from 'lucide-vue-next';
import { cn, type ClassValue } from '../../lib/utils';
import Tooltip from '../vendor/ui/tooltip/Tooltip.vue';
import TooltipContent from '../vendor/ui/tooltip/TooltipContent.vue';
import TooltipProvider from '../vendor/ui/tooltip/TooltipProvider.vue';
import TooltipTrigger from '../vendor/ui/tooltip/TooltipTrigger.vue';
import {
  choyFieldChromeDefaults,
  resolveChoyFieldVisible,
  type ChoyFieldChromeProps,
} from './fieldHelpers';

/**
 * Field chrome only: label, help tooltip, required mark, and error text.
 * Control widgets live in the default slot.
 */
const props = withDefaults(
  defineProps<
    ChoyFieldChromeProps & {
      class?: ClassValue;
    }
  >(),
  { ...choyFieldChromeDefaults },
);

const isVisible = computed(() => resolveChoyFieldVisible(props.visible));
</script>

<template>
  <div
    v-if="isVisible"
    data-anchor="choy.field-base"
    :class="cn('choy-field-base flex w-full flex-col gap-1.5', props.class)"
  >
    <div
      v-if="label || help || required"
      class="choy-field-base__label-row flex items-center gap-1.5"
    >
      <label
        v-if="label"
        class="text-sm font-medium text-foreground"
        :for="name || undefined"
      >
        {{ label }}
        <span v-if="required" class="text-danger" aria-hidden="true">*</span>
      </label>
      <span
        v-else-if="required"
        class="text-sm text-danger"
        aria-hidden="true"
      >*</span>
      <TooltipProvider v-if="help">
        <Tooltip>
          <TooltipTrigger as-child>
            <button
              type="button"
              class="inline-flex size-4 items-center justify-center text-foreground/50 hover:text-foreground"
              :aria-label="`Help: ${label || name || 'field'}`"
            >
              <CircleHelp class="size-3.5" />
            </button>
          </TooltipTrigger>
          <TooltipContent>
            <p class="max-w-xs whitespace-pre-wrap">{{ help }}</p>
          </TooltipContent>
        </Tooltip>
      </TooltipProvider>
    </div>
    <div class="choy-field-base__control min-w-0">
      <slot />
    </div>
    <p
      v-if="error"
      class="choy-field-base__error text-sm text-danger"
      role="alert"
    >
      {{ error }}
    </p>
  </div>
</template>
