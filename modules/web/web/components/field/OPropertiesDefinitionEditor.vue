<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <div class="o-properties-definition-editor" data-testid="o-properties-definition-editor">
    <div class="o-properties-definition-editor__toolbar">
      <el-button
        type="primary"
        size="small"
        :disabled="readonly || saving || !canSave"
        :loading="saving"
        data-testid="o-properties-definition-save"
        @click="onSave"
      >
        {{ _t('Save') }}
      </el-button>
      <el-button size="small" :disabled="readonly || saving" data-testid="o-properties-definition-add" @click="onAdd">
        {{ _t('Add property') }}
      </el-button>
      <span v-if="loadError" class="o-properties-definition-editor__error" data-testid="o-properties-definition-error">
        {{ loadError }}
      </span>
      <span v-else-if="saveError" class="o-properties-definition-editor__error" data-testid="o-properties-definition-save-error">
        {{ saveError }}
      </span>
    </div>

    <div v-if="!drafts.length" class="o-properties-definition-editor__empty" data-testid="o-properties-definition-empty">
      {{ _t('No properties defined') }}
    </div>

    <div
      v-for="(item, index) in drafts"
      :key="index"
      class="o-properties-definition-editor__row"
      :data-index="index"
    >
      <el-input
        v-model="item.name"
        size="small"
        :disabled="readonly"
        :placeholder="_t('Name')"
        data-testid="o-properties-definition-name"
      />
      <el-select v-model="item.type" size="small" :disabled="readonly" data-testid="o-properties-definition-type">
        <el-option v-for="t in typeOptions" :key="t" :label="t" :value="t" />
      </el-select>
      <el-input
        v-model="item.string"
        size="small"
        :disabled="readonly"
        :placeholder="_t('Label')"
        data-testid="o-properties-definition-string"
      />
      <el-input
        v-model="item.default"
        size="small"
        :disabled="readonly"
        :placeholder="_t('Default')"
        data-testid="o-properties-definition-default"
      />
      <el-switch
        v-model="item.readonly"
        size="small"
        :disabled="readonly"
        data-testid="o-properties-definition-readonly"
      />
      <el-input
        v-if="item.type === 'selection'"
        v-model="item.selectionText"
        size="small"
        type="textarea"
        :autosize="{ minRows: 1, maxRows: 4 }"
        :disabled="readonly"
        :placeholder="_t('Selection JSON')"
        data-testid="o-properties-definition-selection"
      />
      <el-button
        size="small"
        type="danger"
        plain
        :disabled="readonly"
        data-testid="o-properties-definition-remove"
        @click="onRemove(index)"
      >
        {{ _t('Remove') }}
      </el-button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue';
import { ElButton, ElInput, ElOption, ElSelect, ElSwitch } from 'element-plus';
import { createTranslate } from '@/web/web/i18n';
import { createStoreByModel } from '@/web/web/stores/registry';
import type { WebModelStore } from '@/web/web/stores/modelStore';
import {
  PROPERTY_DEFINITION_V1_TYPE_OPTIONS,
  buildDefinitionScopeCondition,
  definitionItemsToDrafts,
  draftsToDefinitionItems,
  emptyDraftItem,
  type DefinitionEditorDraftItem,
} from './oproperties_definition_helpers';

defineOptions({ name: 'OPropertiesDefinitionEditor' });

const { _t } = createTranslate('web', { scope: 'web/components/field/OPropertiesDefinitionEditor' });

const props = withDefaults(
  defineProps<{
    application: string;
    targetModel: string;
    propertiesField: string;
    containerModel?: string | null;
    containerId?: string | null;
    readonly?: boolean;
    /** Optional pre-built store (tests). */
    store?: WebModelStore<any> | null;
  }>(),
  {
    containerModel: null,
    containerId: null,
    readonly: false,
    store: null,
  }
);

const emit = defineEmits<{
  saved: [payload: { id: string; definition: unknown[] }];
}>();

const typeOptions = PROPERTY_DEFINITION_V1_TYPE_OPTIONS;
const drafts = ref<DefinitionEditorDraftItem[]>([]);
const definitionId = ref<string | null>(null);
const loading = ref(false);
const saving = ref(false);
const loadError = ref('');
const saveError = ref('');

const canSave = computed(
  () =>
    Boolean(String(props.application || '').trim()) &&
    Boolean(String(props.targetModel || '').trim()) &&
    Boolean(String(props.propertiesField || '').trim())
);

function resolveStore(): WebModelStore<any> | null {
  if (props.store) return props.store;
  const app = String(props.application || '').trim();
  if (!app) return null;
  try {
    return createStoreByModel(`${app}.PropertyDefinition`);
  } catch (e) {
    loadError.value = e instanceof Error ? e.message : String(e);
    return null;
  }
}

async function reload() {
  loadError.value = '';
  saveError.value = '';
  definitionId.value = null;
  drafts.value = [];
  if (!canSave.value) return;

  const store = resolveStore();
  if (!store || typeof (store as any).Search !== 'function') {
    loadError.value = _t('PropertyDefinition store is unavailable');
    return;
  }

  loading.value = true;
  try {
    const And = buildDefinitionScopeCondition({
      targetModel: props.targetModel,
      propertiesField: props.propertiesField,
      containerModel: props.containerModel,
      containerId: props.containerId,
    });
    const rows = await (store as any).Search(
      { And },
      { fields: ['Id', 'Definition', 'TargetModel', 'PropertiesField', 'ContainerModel', 'ContainerId'], limit: 1 }
    );
    const row = Array.isArray(rows) && rows[0] ? rows[0] : null;
    if (row) {
      definitionId.value = String(row.Id || '').trim() || null;
      drafts.value = definitionItemsToDrafts(row.Definition);
    } else {
      definitionId.value = null;
      drafts.value = [];
    }
  } catch (e) {
    loadError.value = e instanceof Error ? e.message : String(e);
    console.warn('[OPropertiesDefinitionEditor] load failed', e);
  } finally {
    loading.value = false;
  }
}

function onAdd() {
  drafts.value = [...drafts.value, emptyDraftItem()];
}

function onRemove(index: number) {
  drafts.value = drafts.value.filter((_, i) => i !== index);
}

async function onSave() {
  saveError.value = '';
  if (props.readonly || !canSave.value) return;
  const store = resolveStore();
  if (!store) {
    saveError.value = _t('PropertyDefinition store is unavailable');
    return;
  }

  let definition: unknown[];
  try {
    definition = draftsToDefinitionItems(drafts.value);
  } catch (e) {
    saveError.value = e instanceof Error ? e.message : String(e);
    return;
  }

  const payload: Record<string, unknown> = {
    TargetModel: String(props.targetModel).trim(),
    PropertiesField: String(props.propertiesField).trim(),
    ContainerModel:
      props.containerModel == null || props.containerModel === '' ? null : String(props.containerModel),
    ContainerId: props.containerId == null || props.containerId === '' ? null : String(props.containerId),
    Definition: definition,
  };

  saving.value = true;
  try {
    if (definitionId.value) {
      await (store as any).UpdateById(definitionId.value, { Definition: definition });
      emit('saved', { id: definitionId.value, definition });
    } else {
      const created = await (store as any).Create(payload);
      const id = String(created?.Id || '').trim();
      definitionId.value = id || null;
      emit('saved', { id, definition });
    }
    await reload();
  } catch (e) {
    saveError.value = e instanceof Error ? e.message : String(e);
    console.warn('[OPropertiesDefinitionEditor] save failed', e);
  } finally {
    saving.value = false;
  }
}

watch(
  () => [
    props.application,
    props.targetModel,
    props.propertiesField,
    props.containerModel,
    props.containerId,
    props.store,
  ],
  () => {
    void reload();
  },
  { immediate: true }
);

defineExpose({ reload, drafts, definitionId });
</script>

<style scoped lang="scss">
.o-properties-definition-editor {
  display: flex;
  flex-direction: column;
  gap: 8px;
  width: 100%;
}
.o-properties-definition-editor__toolbar {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  align-items: center;
}
.o-properties-definition-editor__row {
  display: grid;
  grid-template-columns: minmax(80px, 1fr) 110px minmax(80px, 1fr) minmax(80px, 1fr) auto minmax(120px, 1.4fr) auto;
  gap: 6px;
  align-items: start;
}
.o-properties-definition-editor__empty {
  color: var(--el-text-color-secondary);
  font-size: 13px;
}
.o-properties-definition-editor__error {
  color: var(--el-color-danger);
  font-size: 13px;
}
</style>
