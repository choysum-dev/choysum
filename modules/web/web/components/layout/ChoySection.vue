<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<!--
  Form / detail partition on the shared white sheet — not a nested card.
  Built on shadcn FieldSet + FieldLegend (+ FieldDescription) for a11y grouping.
-->
<template>
  <FieldSet
    data-anchor="choy.section"
    data-testid="choy-section"
    :data-divided="divided ? '' : undefined"
    :class="
      cn(
        // FieldSet defaults to gap-6; ERP form sections match Token/Company (gap-4).
        'gap-4',
        divided ? 'mt-8 border-t border-border pt-6' : undefined,
        props.class,
      )
    "
  >
    <div
      v-if="title || description || $slots.header"
      data-slot="choy-section-header"
      class="flex flex-col gap-1"
    >
      <slot name="header">
        <FieldLegend
          v-if="title"
          variant="legend"
          class="mb-0 font-semibold tracking-tight text-foreground"
        >
          {{ title }}
        </FieldLegend>
        <FieldDescription v-if="description">
          {{ description }}
        </FieldDescription>
      </slot>
    </div>
    <div v-if="$slots.default" data-slot="choy-section-content" class="flex min-w-0 flex-col gap-4">
      <slot />
    </div>
  </FieldSet>
</template>

<script setup lang="ts">
import FieldDescription from '../vendor/ui/field/FieldDescription.vue';
import FieldLegend from '../vendor/ui/field/FieldLegend.vue';
import FieldSet from '../vendor/ui/field/FieldSet.vue';
import { cn, type ClassValue } from '../../lib/utils';

defineOptions({ name: 'ChoySection' });

/**
 * Sheet section for form/detail partitions. Title/description props, or #header
 * + default content. Optional top rule for a following block (e.g. system info).
 * No border, radius, or bg-card — those belong on ChoyCard.
 */
const props = withDefaults(
  defineProps<{
    class?: ClassValue;
    title?: string;
    description?: string;
    /** Draw a top hairline and extra block spacing before this section. */
    divided?: boolean;
  }>(),
  {
    divided: false,
  },
);
</script>
