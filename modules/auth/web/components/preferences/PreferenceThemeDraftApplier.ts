// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { defineComponent, watch } from 'vue';
import { resolvePreferencesThemeMode, type ChoyThemeMode } from './preferences_theme';

/**
 * Live-apply a preferences Theme draft to the shell via watch (not render).
 * Parent owns lastAppliedTheme / cancel restore; this only emits resolved modes.
 */
export const PreferenceThemeDraftApplier = defineComponent({
  name: 'PreferenceThemeDraftApplier',
  props: {
    theme: { default: undefined as unknown },
    active: { type: Boolean, default: true },
  },
  emits: {
    apply: (_mode: ChoyThemeMode) => true,
  },
  setup(props, { emit }) {
    watch(
      () => resolvePreferencesThemeMode(props.theme),
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
