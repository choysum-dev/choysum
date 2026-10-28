<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <div
    data-anchor="choy.properties-definition-editor"
    class="choy-properties-definition-editor flex w-full flex-col gap-3"
    data-testid="choy-properties-definition-editor"
  >
    <div class="flex flex-wrap items-center gap-2">
      <ChoyButton
        size="sm"
        :disabled="readonly || saving || loading || !canSave"
        data-testid="choy-properties-definition-save"
        @click="onSave"
      >
        {{ _t('Save') }}
      </ChoyButton>
      <ChoyButton
        size="sm"
        variant="outline"
        :disabled="readonly || saving || loading"
        data-testid="choy-properties-definition-add"
        @click="onAdd"
      >
        {{ _t('Add property') }}
      </ChoyButton>
      <span
        v-if="loadError"
        class="text-sm text-destructive"
        data-testid="choy-properties-definition-error"
      >
        {{ loadError }}
      </span>
      <span
        v-else-if="saveError"
        class="text-sm text-destructive"
        data-testid="choy-properties-definition-save-error"
      >
        {{ saveError }}
      </span>
    </div>

    <div
      v-if="!drafts.length"
      class="text-sm text-muted-foreground"
      data-testid="choy-properties-definition-empty"
    >
      {{ _t('No properties defined') }}
    </div>

    <div
      v-for="(item, index) in drafts"
      :key="index"
      class="grid gap-2 rounded-md border border-border p-3 sm:grid-cols-2"
      :data-index="index"
    >
      <Input
        v-model="item.name"
        class="w-full"
        :disabled="readonly"
        :placeholder="_t('Name')"
        data-testid="choy-properties-definition-name"
      />
      <Select v-model="item.type" :disabled="readonly">
        <SelectTrigger
          class="w-full"
          :placeholder="_t('Type')"
          data-testid="choy-properties-definition-type"
        />
        <SelectContent>
          <SelectItem v-for="t in typeOptions" :key="t" :value="t">
            {{ t }}
          </SelectItem>
        </SelectContent>
      </Select>
      <div class="flex min-w-0 items-center gap-2">
        <Input
          v-model="item.string"
          class="min-w-0 flex-1"
          :disabled="readonly"
          :placeholder="_t('Label')"
          data-testid="choy-properties-definition-string"
        />
        <ChoyButton
          size="sm"
          variant="outline"
          :disabled="readonly"
          data-testid="choy-properties-definition-translate-string"
          @click="openStringTranslate(index)"
        >
          {{ _t('Translate') }}
        </ChoyButton>
      </div>
      <Input
        v-model="item.default"
        class="w-full"
        :disabled="readonly"
        :placeholder="_t('Default')"
        data-testid="choy-properties-definition-default"
      />
      <label class="flex items-center gap-2 text-sm sm:col-span-2">
        <Checkbox
          :model-value="item.readonly"
          :disabled="readonly"
          data-testid="choy-properties-definition-readonly"
          @update:model-value="item.readonly = $event === true"
        />
        {{ _t('Readonly') }}
      </label>
      <Textarea
        v-if="item.type === 'selection'"
        v-model="item.selectionText"
        class="font-mono text-xs sm:col-span-2"
        :rows="3"
        :disabled="readonly"
        :placeholder="_t('Selection JSON')"
        data-testid="choy-properties-definition-selection"
      />
      <div v-if="item.type === 'selection'" class="flex flex-wrap gap-2 sm:col-span-2">
        <ChoyButton
          v-for="opt in selectionOptionValues(item)"
          :key="opt"
          size="sm"
          variant="outline"
          :disabled="readonly"
          :data-testid="`choy-properties-definition-translate-option-${opt}`"
          @click="openSelectionTranslate(index, opt)"
        >
          {{ _t('Translate %s', opt) }}
        </ChoyButton>
      </div>
      <div class="sm:col-span-2">
        <ChoyButton
          size="sm"
          variant="destructive"
          :disabled="readonly"
          data-testid="choy-properties-definition-remove"
          @click="onRemove(index)"
        >
          {{ _t('Remove') }}
        </ChoyButton>
      </div>
    </div>

    <ChoyPropertyLabelTranslationsDialog
      v-model="translateOpen"
      :label-title="translateTitle"
      :translations="translateMap"
      @saved="onTranslateSaved"
    />
  </div>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue';
import { getActivePinia } from 'pinia';
import type { PropertyItemDefinition } from '@/core/service/orm/model/properties_types';
import { TRANSLATED_BASE_LANG } from '@/core/service/orm/repository/projection/translated_field_codec';
import { createTranslate } from '@/web/web/i18n';
import { createStoreByModel } from '@/web/web/stores/registry';
import { useI18nStore } from '@/web/web/stores/i18nStore';
import type { WebModelStore } from '@/web/web/stores/modelStore';
import ChoyButton from '@/web/web/components/layout/ChoyButton.vue';
import Checkbox from '../vendor/ui/checkbox/Checkbox.vue';
import Input from '../vendor/ui/input/Input.vue';
import Textarea from '../vendor/ui/textarea/Textarea.vue';
import Select from '../vendor/ui/select/Select.vue';
import SelectContent from '../vendor/ui/select/SelectContent.vue';
import SelectItem from '../vendor/ui/select/SelectItem.vue';
import SelectTrigger from '../vendor/ui/select/SelectTrigger.vue';
import ChoyPropertyLabelTranslationsDialog from './ChoyPropertyLabelTranslationsDialog.vue';
import { normalizeSelectionOptions } from './propertiesHelpers';
import {
  PROPERTY_DEFINITION_V1_TYPE_OPTIONS,
  applyDraftCurrentLangToMaps,
  buildDefinitionScopeCondition,
  definitionItemsToDrafts,
  draftsToDefinitionItems,
  emptyDraftItem,
  syncDraftLabelsForLang,
  type DefinitionEditorDraftItem,
} from './propertiesDefinitionHelpers';

defineOptions({ name: 'ChoyPropertiesDefinitionEditor' });

const { _t } = createTranslate('web', { scope: 'web/components/field/ChoyPropertiesDefinitionEditor' });

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
  },
);

const emit = defineEmits<{
  saved: [payload: { id: string; definition: unknown[] }];
}>();

const typeOptions = PROPERTY_DEFINITION_V1_TYPE_OPTIONS;
const drafts = ref<DefinitionEditorDraftItem[]>([]);
const previousItems = ref<PropertyItemDefinition[]>([]);
const definitionId = ref<string | null>(null);
const loading = ref(false);
const saving = ref(false);
const loadError = ref('');
const saveError = ref('');
let reloadSeq = 0;

const translateOpen = ref(false);
const translateTitle = ref('');
const translateMap = ref<Record<string, string>>({});
type TranslateTarget =
  | { kind: 'string'; index: number }
  | { kind: 'selection'; index: number; value: string };
const translateTarget = ref<TranslateTarget | null>(null);

const canSave = computed(
  () =>
    Boolean(String(props.application || '').trim()) &&
    Boolean(String(props.targetModel || '').trim()) &&
    Boolean(String(props.propertiesField || '').trim()),
);

function editorLang(): string {
  if (!getActivePinia()) return TRANSLATED_BASE_LANG;
  try {
    const lang = String(useI18nStore().terminologyLang || '').trim();
    return lang || TRANSLATED_BASE_LANG;
  } catch {
    return TRANSLATED_BASE_LANG;
  }
}

function selectionOptionValues(item: DefinitionEditorDraftItem): string[] {
  try {
    const parsed = item.selectionText ? JSON.parse(item.selectionText) : [];
    return normalizeSelectionOptions(parsed).map(o => o.value);
  } catch {
    return Object.keys(item.selectionLabelMaps || {});
  }
}

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
  const seq = ++reloadSeq;
  loadError.value = '';
  saveError.value = '';
  definitionId.value = null;
  drafts.value = [];
  previousItems.value = [];
  if (!canSave.value) return;

  const store = resolveStore();
  if (!store || typeof (store as any).Search !== 'function') {
    if (seq === reloadSeq && !loadError.value) {
      loadError.value = _t('PropertyDefinition store is unavailable');
    }
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
      {
        fields: ['Id', 'Definition', 'TargetModel', 'PropertiesField', 'ContainerModel', 'ContainerId'],
        limit: 1,
      },
    );
    if (seq !== reloadSeq) return;
    const row = Array.isArray(rows) && rows[0] ? rows[0] : null;
    if (row) {
      definitionId.value = String(row.Id || '').trim() || null;
      const raw = Array.isArray(row.Definition) ? (row.Definition as PropertyItemDefinition[]) : [];
      previousItems.value = raw;
      drafts.value = definitionItemsToDrafts(raw, editorLang());
    } else {
      definitionId.value = null;
      previousItems.value = [];
      drafts.value = [];
    }
  } catch (e) {
    if (seq !== reloadSeq) return;
    loadError.value = e instanceof Error ? e.message : String(e);
    console.warn('[ChoyPropertiesDefinitionEditor] load failed', e);
  } finally {
    if (seq === reloadSeq) {
      loading.value = false;
    }
  }
}

function onAdd() {
  if (loading.value || saving.value || props.readonly) return;
  drafts.value = [...drafts.value, emptyDraftItem()];
}

function onRemove(index: number) {
  drafts.value = drafts.value.filter((_, i) => i !== index);
}

function openStringTranslate(index: number) {
  const item = drafts.value[index];
  if (!item || props.readonly) return;
  applyDraftCurrentLangToMaps(item, editorLang());
  translateTarget.value = { kind: 'string', index };
  translateTitle.value = item.name || _t('Label');
  translateMap.value = { ...item.stringMap };
  translateOpen.value = true;
}

function openSelectionTranslate(index: number, value: string) {
  const item = drafts.value[index];
  if (!item || props.readonly) return;
  applyDraftCurrentLangToMaps(item, editorLang());
  translateTarget.value = { kind: 'selection', index, value };
  translateTitle.value = `${item.name || _t('Label')}.${value}`;
  translateMap.value = { ...(item.selectionLabelMaps[value] || {}) };
  translateOpen.value = true;
}

function onTranslateSaved(next: Record<string, string>) {
  const target = translateTarget.value;
  if (!target) return;
  const item = drafts.value[target.index];
  if (!item) return;
  if (target.kind === 'string') {
    item.stringMap = { ...next };
    syncDraftLabelsForLang(item, editorLang());
  } else {
    item.selectionLabelMaps = {
      ...item.selectionLabelMaps,
      [target.value]: { ...next },
    };
    syncDraftLabelsForLang(item, editorLang());
  }
  translateTarget.value = null;
}

async function onSave() {
  saveError.value = '';
  if (props.readonly || loading.value || saving.value || !canSave.value) return;
  const store = resolveStore();
  if (!store) {
    saveError.value = _t('PropertyDefinition store is unavailable');
    return;
  }

  let definition: unknown[];
  try {
    definition = draftsToDefinitionItems(drafts.value, editorLang(), previousItems.value);
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
    console.warn('[ChoyPropertiesDefinitionEditor] save failed', e);
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
    editorLang(),
  ],
  () => {
    void reload();
  },
  { immediate: true },
);

defineExpose({ reload, drafts, definitionId, previousItems });
</script>
