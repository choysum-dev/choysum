<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <ChoyDropdownMenu v-model:open="menuOpen">
    <ChoyDropdownMenuTrigger as-child>
      <ChoyButton
        type="button"
        size="default"
        variant="outline"
        :disabled="!application || downloading"
        :aria-label="_t('Export PO')"
        data-testid="terminology-export-trigger"
      >
        {{ _t('Export PO') }}
        <ChevronDown class="size-4" aria-hidden="true" />
      </ChoyButton>
    </ChoyDropdownMenuTrigger>
    <ChoyDropdownMenuContent align="start" class="min-w-[12rem]" data-testid="terminology-export-menu">
      <ChoyDropdownMenuLabel>{{ _t('Module') }}</ChoyDropdownMenuLabel>
      <div v-if="loadingModules" class="px-2 py-1.5 text-sm text-muted-foreground">
        {{ _t('Loading modules…') }}
      </div>
      <div v-else-if="!modules.length" class="px-2 py-1.5 text-sm text-muted-foreground">
        {{ _t('No modules') }}
      </div>
      <ChoyDropdownMenuItem
        v-for="mod in modules"
        :key="mod"
        role="menuitemradio"
        :aria-checked="mod === selectedModule"
        :disabled="downloading"
        class="gap-2 text-sm font-normal"
        :data-testid="`terminology-export-module-${mod}`"
        @select="onSelectModule($event, mod)"
      >
        <!--
          Global index.css forces svg/lucide { color: inherit !important; stroke: currentColor }.
          Wrap so the icon inherits text-primary from this span, not the focused item foreground.
        -->
        <span
          class="inline-flex text-primary"
          :class="mod === selectedModule ? 'opacity-100' : 'opacity-0'"
          aria-hidden="true"
        >
          <Check class="size-4" />
        </span>
        {{ mod }}
      </ChoyDropdownMenuItem>
      <ChoyDropdownMenuSeparator />
      <ChoyDropdownMenuItem
        :disabled="!canDownload || downloading"
        class="text-sm font-normal text-primary focus:text-primary data-[disabled]:text-primary"
        data-testid="terminology-export-download"
        @select="onDownloadPo"
      >
        {{ downloading ? _t('Downloading…') : _t('Download PO') }}
      </ChoyDropdownMenuItem>
    </ChoyDropdownMenuContent>
  </ChoyDropdownMenu>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue';
import { Check, ChevronDown } from 'lucide-vue-next';
import {
  ChoyButton,
  ChoyDropdownMenu,
  ChoyDropdownMenuContent,
  ChoyDropdownMenuItem,
  ChoyDropdownMenuLabel,
  ChoyDropdownMenuSeparator,
  ChoyDropdownMenuTrigger,
} from '@/web';
import { createTranslate } from '@/web/web/i18n';
import { useI18nStore } from '@/web/web/stores/i18nStore';
import { downloadTerminologyPo } from '@/web/web/stores/i18nStore/poDownload';
import { useAuthStore } from '@/auth/web/stores/auth';
import { ChoyMessage } from '@/web/web/composables/useChoyMessage';
import { createStoreByModel } from '@/web/web/stores/registry';
import {
  collectModuleNamesFromMetaModuleRows,
  installedModulesSearchCondition,
} from './terminologyExportModules';

defineOptions({ name: 'TerminologyExportMenu' });

const props = defineProps<{
  application: string;
}>();

const { _t } = createTranslate('web', { scope: 'web/components/terminology/TerminologyExportMenu' });
const i18nStore = useI18nStore();
const authStore = useAuthStore();

const menuOpen = ref(false);
const modules = ref<string[]>([]);
const selectedModule = ref('');
const loadingModules = ref(false);
const modulesLoaded = ref(false);
const downloading = ref(false);
let loadInflight: Promise<void> | null = null;

const application = computed(() => String(props.application || '').trim());
const canDownload = computed(
  () => Boolean(application.value && selectedModule.value.trim() && i18nStore.terminologyLang)
);

watch(
  application,
  (app) => {
    selectedModule.value = app;
    modules.value = app ? [app] : [];
    modulesLoaded.value = false;
    loadInflight = null;
  },
  { immediate: true }
);

watch(menuOpen, (open) => {
  if (open && !modulesLoaded.value) void loadModules();
});

async function loadModules() {
  const app = application.value;
  if (!app) return;
  if (loadInflight) {
    await loadInflight;
    return;
  }
  loadingModules.value = true;
  const pending = (async () => {
    try {
      const metaModuleStore = createStoreByModel('meta.MetaModule');
      const rows = await metaModuleStore.Search(installedModulesSearchCondition(app) as any, {
        fields: ['Name'],
        orderBy: [{ field: 'Name', direction: 'asc' }],
      } as any);
      if (application.value !== app) return;
      const fromMeta = collectModuleNamesFromMetaModuleRows(rows);
      const next = fromMeta.length ? fromMeta : [app];
      modules.value = next;
      if (!next.includes(selectedModule.value)) {
        selectedModule.value = next.includes(app) ? app : next[0] || '';
      }
    } catch {
      if (application.value !== app) return;
      modules.value = [app];
      if (!selectedModule.value) selectedModule.value = app;
    } finally {
      if (application.value === app) {
        modulesLoaded.value = true;
        loadingModules.value = false;
      }
    }
  })();
  loadInflight = pending;
  try {
    await pending;
  } finally {
    if (loadInflight === pending) loadInflight = null;
  }
}

function onSelectModule(event: Event, mod: string) {
  event.preventDefault();
  selectedModule.value = mod;
}

async function onDownloadPo() {
  if (!canDownload.value || downloading.value) return;
  const moduleName = selectedModule.value.trim();
  const lang = String(i18nStore.terminologyLang || '').trim();
  downloading.value = true;
  try {
    const blob = await downloadTerminologyPo({
      lang,
      application: application.value,
      module: moduleName,
      accessToken: authStore.tokens?.accessToken,
    });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = `${moduleName}-${lang}.po`;
    a.rel = 'noopener';
    document.body.appendChild(a);
    a.click();
    a.remove();
    URL.revokeObjectURL(url);
  } catch (err: any) {
    ChoyMessage.error(err?.message || _t('PO download failed'));
  } finally {
    downloading.value = false;
  }
}
</script>
