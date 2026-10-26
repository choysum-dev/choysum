<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<!--
  Form / detail partition on the shared white sheet — not a nested card.
  Built on shadcn FieldSet + FieldLegend (+ FieldDescription) for a11y grouping.
-->
<template>
  <div
    data-anchor="choy.section"
    data-testid="choy-section"
    :data-divided="divided ? '' : undefined"
    :class="cn('flex min-w-0 flex-col gap-4', divided ? 'mt-8' : undefined, props.class)"
  >
    <!--
      Visible title is a plain div (not <legend>). Divided sections append a
      trailing hairline with gap-4 so text never sits flush against the rule.
      <legend> stays sr-only inside FieldSet for a11y only.
    -->
    <div
      v-if="title && !$slots.header"
      data-slot="choy-section-title"
      class="flex min-w-0 items-center gap-4"
    >
      <span class="shrink-0 text-base font-semibold text-foreground">{{ title }}</span>
      <span
        v-if="divided"
        data-slot="choy-section-rule"
        class="h-px min-w-0 flex-1 bg-border"
        aria-hidden="true"
      />
    </div>
    <!-- Divided + #header / no title: no title row, so keep a full-width rule. -->
    <div
      v-else-if="divided"
      data-slot="choy-section-rule"
      class="h-px w-full bg-border"
      aria-hidden="true"
    />
    <FieldSet class="min-w-0 gap-4">
      <FieldLegend
        v-if="title && !$slots.header"
        variant="legend"
        class="sr-only"
      >
        {{ title }}
      </FieldLegend>
      <div
        v-if="description || $slots.header"
        data-slot="choy-section-header"
        class="flex flex-col gap-1"
      >
        <slot name="header">
          <FieldDescription v-if="description">
            {{ description }}
          </FieldDescription>
        </slot>
      </div>
      <div v-if="$slots.default" data-slot="choy-section-content" class="flex min-w-0 flex-col gap-4">
        <slot />
      </div>
    </FieldSet>
  </div>
</template>

<script setup lang="ts">
import FieldDescription from '../vendor/ui/field/FieldDescription.vue';
import FieldLegend from '../vendor/ui/field/FieldLegend.vue';
import FieldSet from '../vendor/ui/field/FieldSet.vue';
import { cn, type ClassValue } from '../../lib/utils';

defineOptions({ name: 'ChoySection' });

/**
 * Sheet section for form/detail partitions. Title/description props, or #header
 * + default content. Optional trailing hairline after the title for phase breaks.
 * No border, radius, or bg-card — those belong on ChoyCard.
 */
const props = withDefaults(
  defineProps<{
    class?: ClassValue;
    title?: string;
    description?: string;
    /**
     * Draw a hairline for phase breaks: trailing after the title when present,
     * otherwise a full-width rule (#header / untitled).
     */
    divided?: boolean;
  }>(),
  {
    divided: false,
  },
);
</script>
