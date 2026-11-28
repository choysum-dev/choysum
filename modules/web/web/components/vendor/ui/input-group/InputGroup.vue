<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<script setup lang="ts">
import type { HTMLAttributes } from "vue"
import { cn } from "../../../../lib/utils"

const props = defineProps<{
  class?: HTMLAttributes["class"]
}>()
</script>

<template>
  <div
    data-slot="input-group"
    role="group"
    :class="cn(
      'group/input-group border-input dark:bg-input/30 relative flex w-full items-center rounded-md border shadow-sm transition-[color,box-shadow] outline-none',
      'h-control min-w-0 has-[>textarea]:h-auto',

      // Variants based on alignment.
      'has-[>[data-align=inline-start]]:[&>input]:ps-2',
      'has-[>[data-align=inline-end]]:[&>input]:pe-2',
      'has-[>[data-align=block-start]]:h-auto has-[>[data-align=block-start]]:flex-col has-[>[data-align=block-start]]:[&>input]:pb-3',
      'has-[>[data-align=block-end]]:h-auto has-[>[data-align=block-end]]:flex-col has-[>[data-align=block-end]]:[&>input]:pt-3',

      // Focus state.
      'has-[[data-slot=input-group-control]:focus-visible]:border-ring has-[[data-slot=input-group-control]:focus-visible]:ring-ring/50 has-[[data-slot=input-group-control]:focus-visible]:ring-3',

      // Error state: host aria-invalid (reliable) plus nested has- for child controls.
      'aria-invalid:border-destructive aria-invalid:ring-destructive/20 dark:aria-invalid:ring-destructive/40',
      'has-[[data-slot][aria-invalid=true]]:ring-destructive/20 has-[[data-slot][aria-invalid=true]]:border-destructive dark:has-[[data-slot][aria-invalid=true]]:ring-destructive/40',

      props.class,
    )"
  >
    <slot />
  </div>
</template>

<!--
  Tighten control padding next to inline addons. Kit lists has-[>[data-align]]:[&>input]:p*
  utilities, but nested has- variants are not always emitted — without this, Input keeps
  px-3 on both sides and the word-limit counter sits too far from the text (and RTL breaks
  if we used physical padding-right).
-->
<style>
  /* !important: beat layered Input px-3 utilities. Logical for LTR/RTL. */
  [data-slot='input-group']:has(>[data-align='inline-end']) > [data-slot='input-group-control'] {
    padding-inline-end: 0.25rem !important;
  }
  /* Word-limit (end addon only): start inset matches addon outer pe-3 (12px). */
  [data-slot='input-group']:has(>[data-align='inline-end']):not(:has(>[data-align='inline-start']))
    > [data-slot='input-group-control'] {
    padding-inline-start: 0.75rem !important;
  }
  [data-slot='input-group']:has(>[data-align='inline-start']) > [data-slot='input-group-control'] {
    padding-inline-start: 0.25rem !important;
  }
</style>
