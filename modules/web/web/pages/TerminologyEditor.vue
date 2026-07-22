<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <OPage>
    <div class="terminology-editor">
      <div class="terminology-editor__filters">
        <el-select v-model="lang" placeholder="Language" style="width: 140px" @change="onFilterChange">
          <el-option v-for="code in localeOptions" :key="code" :label="code" :value="uiKeyToLang(code)" />
        </el-select>
        <el-select
          v-model="application"
          clearable
          placeholder="Application (required without search)"
          style="width: 200px"
          @change="onFilterChange"
        >
          <el-option label="All (search only)" value="" />
          <el-option v-for="app in applicationOptions" :key="app" :label="app" :value="app" />
        </el-select>
        <el-input
          v-model="moduleFilter"
          clearable
          placeholder="Module"
          style="width: 160px"
          @change="onFilterChange"
        />
        <el-input
          v-model="q"
          clearable
          placeholder="Search src / value / scope"
          style="width: 260px"
          @keyup.enter="loadTerms"
        />
        <el-button type="primary" :loading="loading" @click="loadTerms">Search</el-button>
        <el-button type="success" :disabled="!dirtyRows.length" :loading="saving" @click="saveDirty">
          Save
        </el-button>
        <el-button :disabled="!application.trim()" :loading="downloading" @click="downloadPo">
          Download PO
        </el-button>
      </div>

      <el-alert
        v-if="!q.trim() && !application"
        type="info"
        :closable="false"
        show-icon
        title="Select an Application, or enter a search query to search across apps."
        class="terminology-editor__hint"
      />
      <el-alert
        v-if="errorMessage"
        type="error"
        :closable="true"
        show-icon
        :title="errorMessage"
        class="terminology-editor__hint"
        @close="errorMessage = ''"
      />
      <el-alert
        v-if="truncated"
        type="warning"
        :closable="false"
        show-icon
        title="Results truncated (cross-app search is capped)."
        class="terminology-editor__hint"
      />

      <el-table :data="rows" v-loading="loading" border stripe height="calc(100vh - 260px)">
        <el-table-column prop="application" label="Application" width="110" />
        <el-table-column prop="module" label="Module" width="110" />
        <el-table-column label="Component" width="180" show-overflow-tooltip>
          <template #default="{ row }">
            {{ componentHintFromScope(row.scope) }}
          </template>
        </el-table-column>
        <el-table-column prop="scope" label="Scope" min-width="200" show-overflow-tooltip />
        <el-table-column prop="src" label="Src" min-width="160" show-overflow-tooltip />
        <el-table-column label="Value" min-width="200">
          <template #default="{ row }">
            <el-input v-model="row.value" @input="markDirty(row)" />
          </template>
        </el-table-column>
        <el-table-column prop="source" label="Source" width="100" />
        <el-table-column prop="status" label="Status" width="100" />
      </el-table>

      <div v-if="application" class="terminology-editor__pager">
        <el-pagination
          background
          layout="total, prev, pager, next"
          :total="total"
          :page-size="limit"
          :current-page="page"
          @current-change="onPageChange"
        />
      </div>
    </div>
  </OPage>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue';
import { ElMessage } from 'element-plus';
import OPage from '@/web/web/components/page/OPage.vue';
import {
  useI18nStore,
  uiKeyToLang,
  fetchTerms,
  patchTerms,
  downloadTerminologyPo,
  componentHintFromScope,
  type TermItem,
} from '@/web/web/stores/i18nStore';

defineOptions({ name: 'TerminologyEditorPage' });

type EditableTerm = TermItem & { _key: string; _dirty?: boolean };

const i18nStore = useI18nStore();
const loading = ref(false);
const saving = ref(false);
const downloading = ref(false);
const errorMessage = ref('');
const truncated = ref(false);
const rows = ref<EditableTerm[]>([]);
const total = ref(0);
const limit = 50;
const page = ref(1);
const q = ref('');
const moduleFilter = ref('');
const application = ref('web');
const lang = ref(i18nStore.terminologyLang || 'en_US');

const localeOptions = computed(() => i18nStore.activeUiKeys?.length
  ? i18nStore.activeUiKeys
  : i18nStore.supportedLocales);

const applicationOptions = ['auth', 'web', 'base', 'meta', 'partner'];

const dirtyRows = computed(() => rows.value.filter(r => r._dirty));

function rowKey(item: TermItem): string {
  return [item.application, item.module, item.scope, item.src, item.kind || 'literal'].join('\u001f');
}

function markDirty(row: EditableTerm) {
  row._dirty = true;
}

function onFilterChange() {
  page.value = 1;
  loadTerms();
}

function onPageChange(next: number) {
  page.value = next;
  loadTerms();
}

async function loadTerms() {
  errorMessage.value = '';
  truncated.value = false;
  const queryQ = q.value.trim();
  const app = application.value.trim();
  if (!queryQ && !app) {
    rows.value = [];
    total.value = 0;
    return;
  }

  loading.value = true;
  try {
    const res = await fetchTerms({
      lang: lang.value,
      application: app || undefined,
      module: moduleFilter.value.trim() || undefined,
      q: queryQ || undefined,
      limit: app ? limit : 100,
      offset: app ? (page.value - 1) * limit : 0,
    });
    rows.value = (res.items || []).map(item => reactive({
      ...item,
      kind: item.kind || 'literal',
      _key: rowKey(item),
      _dirty: false,
    }));
    total.value = Number(res.total || 0);
    truncated.value = res.truncated === true;
  } catch (err: any) {
    errorMessage.value = String(err?.message || err);
    rows.value = [];
    total.value = 0;
  } finally {
    loading.value = false;
  }
}

async function saveDirty() {
  const dirty = dirtyRows.value;
  if (!dirty.length) {
    return;
  }
  saving.value = true;
  errorMessage.value = '';
  try {
    await patchTerms({
      lang: lang.value,
      items: dirty.map(row => ({
        application: row.application,
        module: row.module,
        scope: row.scope,
        src: row.src,
        value: row.value,
        kind: row.kind || 'literal',
      })),
    });
    for (const row of dirty) {
      row._dirty = false;
      row.source = 'override';
    }
    await i18nStore.reloadTerminology();
    ElMessage.success('Terminology saved and reloaded');
    await loadTerms();
  } catch (err: any) {
    errorMessage.value = String(err?.message || err);
  } finally {
    saving.value = false;
  }
}

async function downloadPo() {
  const app = application.value.trim();
  if (!app) {
    ElMessage.warning('Select an Application before downloading PO');
    return;
  }
  downloading.value = true;
  errorMessage.value = '';
  try {
    const { filename, blob } = await downloadTerminologyPo({
      lang: lang.value,
      application: app,
      module: moduleFilter.value.trim() || undefined,
    });
    const href = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = href;
    a.download = filename;
    a.click();
    URL.revokeObjectURL(href);
    ElMessage.success(`Downloaded ${filename}`);
  } catch (err: any) {
    errorMessage.value = String(err?.message || err);
  } finally {
    downloading.value = false;
  }
}

onMounted(() => {
  loadTerms();
});
</script>

<style scoped>
.terminology-editor__filters {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  margin-bottom: 12px;
  align-items: center;
}
.terminology-editor__hint {
  margin-bottom: 12px;
}
.terminology-editor__pager {
  margin-top: 12px;
  display: flex;
  justify-content: flex-end;
}
</style>
