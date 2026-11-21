<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <div
    data-slot="view-body-host"
    :class="hostClass"
    v-bind="hostAttrs"
  >
    <slot />
    <!-- Live region must stay outside the inert host subtree so AT can announce it. -->
    <Teleport to="body">
      <div
        v-if="busy"
        role="status"
        aria-live="polite"
        class="sr-only"
        data-slot="view-busy-status"
      >
        {{ busyLabel }}
      </div>
    </Teleport>
    <!-- stale: light fade via host opacity only; no blocking overlay. -->
    <Transition name="busy-overlay">
      <div
        v-if="showOverlay"
        data-slot="view-busy-overlay"
        :data-mode="mode"
        class="absolute inset-0 z-10"
        :class="
          mode === 'skeleton'
            ? 'flex flex-col gap-2 bg-background/80 p-4'
            : 'flex items-center justify-center bg-background/50'
        "
        aria-hidden="true"
      >
        <template v-if="mode === 'skeleton'">
          <ChoySkeleton class="h-8 w-1/3" />
          <ChoySkeleton class="h-10 w-full" />
          <ChoySkeleton class="h-10 w-full" />
          <ChoySkeleton class="h-10 w-2/3" />
        </template>
        <ChoySpinner v-else class="size-6 text-primary" />
      </div>
    </Transition>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import ChoySkeleton from '../layout/ChoySkeleton.vue';
import ChoySpinner from '../layout/ChoySpinner.vue';
import type { ClassValue } from '../../lib/utils';
import { createTranslate } from '@/web/web/i18n';
import { useDelayedVisualBusy } from '@/web/web/composables/useDelayedVisualBusy';
import { viewBusyHostAttrs, viewBusyHostClass, type ViewBusyMode } from './viewBusy';

defineOptions({ name: 'ChoyViewBusyHost' });

const { _t } = createTranslate('web', { scope: 'web/components/view/ViewBusyHost' });

const props = withDefaults(
  defineProps<{
    busy?: boolean;
    /** stale | skeleton | block; default block keeps prior lock+spinner behavior. */
    mode?: ViewBusyMode;
    class?: ClassValue;
    /** Polite live-region text announced while busy (overlay stays aria-hidden). */
    busyLabel?: string;
  }>(),
  { busy: false, mode: 'block' },
);

const busy = computed(() => !!props.busy);
const mode = computed<ViewBusyMode>(() => props.mode || 'block');
const busyLabel = computed(() => props.busyLabel || _t('Loading'));
/** Overlay only — aria-busy / inert stay on logical `busy`. Stale never uses an overlay. */
const overlayCandidate = computed(() => busy.value && mode.value !== 'stale');
const showOverlay = useDelayedVisualBusy(overlayCandidate);
const hostClass = computed(() => viewBusyHostClass(busy.value, mode.value, props.class));
const hostAttrs = computed(() => viewBusyHostAttrs(busy.value, mode.value));
</script>
