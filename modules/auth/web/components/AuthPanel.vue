<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <div
    class="auth-panel mx-auto flex min-h-screen w-full max-w-md flex-col items-center justify-center gap-4 px-4 py-8"
    data-testid="auth-panel"
  >
    <div class="w-full">
      <slot />
    </div>
    <p
      class="auth-panel__attrib m-0 max-w-full text-center text-xs leading-snug text-muted-foreground"
      data-testid="auth-panel-attrib"
    >
      {{ attribLine }}
    </p>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import { useI18n } from 'vue-i18n';

/**
 * Fullscreen Auth canvas wrapper: centered narrow panel slot + short open-source
 * attribution under the panel (copyright / Powered by / version).
 */
defineOptions({ name: 'AuthPanel' });

let tLayout: (key: string, values?: Record<string, unknown>) => string = (key) => key;
try {
  const i18n = useI18n({ useScope: 'global' });
  tLayout = (key, values) => String(i18n.t(key, values as any));
} catch {
  tLayout = (key) => key;
}

const appVersion = computed(
  () => String((import.meta as ImportMeta).env?.CHOYSUM_APP_VERSION || '').trim() || 'dev',
);
const year = computed(() => new Date().getFullYear());
const attribLine = computed(
  () =>
    `${tLayout('layout.footer.copyright', { year: year.value })} · ${tLayout('layout.footer.powered')} · ${tLayout('layout.footer.version', { version: appVersion.value })}`,
);
</script>
