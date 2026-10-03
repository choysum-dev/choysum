<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <div
    class="auth-panel mx-auto flex min-h-svh w-full max-w-sm flex-col items-center justify-center gap-6 px-4 py-8"
    data-testid="auth-panel"
  >
    <a
      href="/"
      class="flex items-center gap-2 self-center font-medium text-foreground no-underline"
      data-testid="auth-panel-brand"
      @click.prevent="onBrandClick"
    >
      <img :src="logoUrl" alt="" class="size-6 shrink-0" width="24" height="24" />
      Choysum
    </a>
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
import { useRouter } from 'vue-router';
import logoUrl from '@/web/web/assets/logo-32.png';
import { resolveRuntimeDefaultLandPath } from '@/web/web/router/resolveRuntimeDefaultLandPath';

/**
 * Auth canvas column: brand lockup, form slot, attribution.
 */
defineOptions({ name: 'AuthPanel' });

let tLayout: (key: string, values?: Record<string, unknown>) => string = (key) => key;
try {
  const i18n = useI18n({ useScope: 'global' });
  tLayout = (key, values) => String(i18n.t(key, values as any));
} catch {
  tLayout = (key) => key;
}

let onBrandClick = () => {};
try {
  const router = useRouter();
  if (router) {
    onBrandClick = () => {
      void router.push(resolveRuntimeDefaultLandPath());
    };
  }
} catch {
  onBrandClick = () => {};
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
