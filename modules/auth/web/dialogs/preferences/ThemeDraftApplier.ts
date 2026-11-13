// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { defineComponent, watch } from 'vue';
import { resolveThemeMode, type ChoyThemeMode } from '@/auth/web/stores/auth/user_appearance';

/**
 * Live-apply a preferences Theme draft to the shell via watch (not render).
 * Parent owns lastAppliedTheme / cancel restore; this only emits resolved modes.
 */
export const ThemeDraftApplier = defineComponent({
  name: 'ThemeDraftApplier',
  props: {
    theme: { default: undefined as unknown },
    active: { type: Boolean, default: true },
  },
  emits: {
    apply: (_mode: ChoyThemeMode) => true,
  },
  setup(props, { emit }) {
    watch(
      () => resolveThemeMode(props.theme),
      mode => {
        if (!props.active) return;
        emit('apply', mode);
      },
      // Post-flush so parent @apply listeners are attached before the first emit.
      { immediate: true, flush: 'post' },
    );
    return () => null;
  },
});
