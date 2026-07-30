<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <OFieldBase
    :binding="binding"
    :label="label"
    :rules="mergedRules"
    :formItemProps="formItemProps"
    :vColumnProps="vColumnProps"
    :toView="toView"
    :fromView="fromView"
    :required="required"
    :readonly="readonly"
    :visible="visible"
    :cellVisible="cellVisible"
    :renderMode="renderMode"
    :showInlineError="showInlineError"
    v-bind="$attrs"
  >
    <template #edit="{ fieldValue }">
      <div class="o-htmlfield-edit">
        <div v-if="editor" class="o-htmlfield-toolbar">
          <button type="button" class="o-htmlfield-btn" :class="{ active: editor.isActive('bold') }" @click.prevent="htmlEditorChain(editor).focus().toggleBold().run()">
            B
          </button>
          <button type="button" class="o-htmlfield-btn" :class="{ active: editor.isActive('italic') }" @click.prevent="htmlEditorChain(editor).focus().toggleItalic().run()">
            I
          </button>
          <button type="button" class="o-htmlfield-btn" :class="{ active: editor.isActive('bulletList') }" @click.prevent="htmlEditorChain(editor).focus().toggleBulletList().run()">
            •
          </button>
          <button type="button" class="o-htmlfield-btn" :class="{ active: editor.isActive('orderedList') }" @click.prevent="htmlEditorChain(editor).focus().toggleOrderedList().run()">
            1.
          </button>
          <button type="button" class="o-htmlfield-btn" :class="{ active: editor.isActive('link') }" @click.prevent="toggleLink">
            Link
          </button>
        </div>
        <EditorContent :editor="editor" class="o-htmlfield-editor" />
        <OHtmlCommitBridge :field-value="fieldValue" :get-html="getEditorHtml" :set-html="setEditorHtml" />
      </div>
    </template>

    <template #display="{ fieldValue }">
      <div v-if="isTableLike" class="o-htmlfield-plaintext">{{ toPlaintext(fieldValue().value) }}</div>
      <div v-else class="o-htmlfield-display" v-html="toSafeHtml(fieldValue().value)"></div>
    </template>
  </OFieldBase>
</template>

<script setup lang="ts" generic="T extends BaseModel, P extends FieldPath<T, string | null | undefined>, V = FieldPathType<T, P>">
import type { RuleItem } from 'async-validator';
import type { BaseModel, FieldPath, FieldPathType } from '@/core/rpc';
import type { WebModelStore } from '@/web/web/stores/modelStore';
import type { FormItemProps } from 'element-plus';
import { computed, defineComponent, onBeforeUnmount, watch } from 'vue';
import { EditorContent, useEditor } from '@tiptap/vue-3';
import StarterKit from '@tiptap/starter-kit';
import Link from '@tiptap/extension-link';
import { htmlEditorChain, htmlEditorSetContent } from './tiptap_html_commands';
import { useField } from '@/web/web/composables/useField';
import type { UseField } from '@/web/web/composables/useField';
import type { NarrowAggProp, NonNumericAggFns } from '@/web/web/composables/useField';
import OFieldBase, { type FieldStateExpr } from './OFieldBase.vue';
import { createTranslate } from '@/web/web/i18n';
import { htmlToPlaintext, normalizeHtmlForStore, sanitizeHtmlForClient } from './ohtml_helpers';

const { _t } = createTranslate('web', { scope: 'web/components/field/OHtmlField' });

defineOptions({ name: 'OHtmlField', inheritAttrs: false });

type IsAny<T> = 0 extends 1 & T ? true : false;
type ViewType = string | null;

const props = withDefaults(
  defineProps<{
    store?: WebModelStore<T>;
    prop?: P | (IsAny<T> extends true ? string : never);
    binding?: UseField<T, V>;
    label?: string;
    rules?: RuleItem[];
    required?: FieldStateExpr<T, V>;
    readonly?: FieldStateExpr<T, V>;
    visible?: FieldStateExpr<T, V>;
    cellVisible?: FieldStateExpr<T, V>;
    formItemProps?: Partial<FormItemProps>;
    vColumnProps?: {
      width?: number | string;
      minWidth?: number;
      align?: 'left' | 'center' | 'right';
      fixed?: 'left' | 'right';
      sortable?: boolean;
    };
    agg?: NarrowAggProp<NonNumericAggFns>;
    renderMode?: 'auto' | 'form' | 'table' | 'inline';
    showInlineError?: boolean;
  }>(),
  {
    rules: () => [],
    required: false,
    readonly: false,
    visible: true,
    cellVisible: true,
    formItemProps: () => ({}),
    vColumnProps: () => ({}),
    renderMode: 'auto',
    showInlineError: false,
  }
);

const binding = (props.binding ?? useField<T, P, V>({ store: props.store as WebModelStore<T>, prop: props.prop as P, agg: props.agg })) as UseField<T, V>;

const isTableLike = computed(() => {
  const mode = props.renderMode;
  if (mode === 'table' || mode === 'inline') return true;
  if (mode === 'form') return false;
  return binding.env?.isForm === false;
});

const toView = (raw: any): ViewType => (raw == null ? null : String(raw));
const fromView = (v: ViewType) => v as unknown as V;
const toSafeHtml = (v: ViewType) => sanitizeHtmlForClient(v);
const toPlaintext = (v: ViewType) => htmlToPlaintext(v);

const editor = useEditor({
  extensions: [
    StarterKit,
    Link.configure({
      openOnClick: false,
      autolink: true,
      defaultProtocol: 'https',
      protocols: ['http', 'https', 'mailto'],
    }),
  ],
  content: '',
  editorProps: {
    attributes: {
      class: 'o-htmlfield-prose',
    },
  },
});

function getEditorHtml(): string {
  return editor.value!.getHTML();
}

function setEditorHtml(html: string | null) {
  if (!editor.value) return;
  const next = html == null || html === '' ? '' : sanitizeHtmlForClient(html);
  const current = editor.value.getHTML();
  if (current === next) return;
  htmlEditorSetContent(editor.value, next || '', false);
}

function toggleLink() {
  // Toolbar is rendered only when editor exists.
  const ed = editor.value!;
  if (ed.isActive('link')) {
    htmlEditorChain(ed).focus().unsetLink().run();
    return;
  }
  const prev = ed.getAttributes('link').href as string | undefined;
  const href = window.prompt(_t('URL'), prev || 'https://');
  if (href == null) return;
  const trimmed = href.trim();
  if (!trimmed) {
    htmlEditorChain(ed).focus().unsetLink().run();
    return;
  }
  htmlEditorChain(ed).focus().extendMarkRange('link').setLink({ href: trimmed }).run();
}

/** Sync TipTap HTML ↔ fieldValue without nesting watchers in the parent setup. */
const OHtmlCommitBridge = defineComponent({
  name: 'OHtmlCommitBridge',
  props: {
    fieldValue: { type: Function, required: true },
    getHtml: { type: Function, required: true },
    setHtml: { type: Function, required: true },
  },
  setup(p) {
    let applyingFromStore = false;
    const pushStoreToEditor = () => {
      applyingFromStore = true;
      try {
        const next = (p.fieldValue as any)().value;
        (p.setHtml as (v: string | null) => void)(next == null ? null : String(next));
      } finally {
        applyingFromStore = false;
      }
    };
    watch(
      () => (p.fieldValue as any)().value,
      () => {
        pushStoreToEditor();
      },
      { immediate: true }
    );
    watch(
      () => editor.value,
      (ed, _old, onCleanup) => {
        if (!ed) return;
        // TipTap creates the editor in onMounted; re-apply store HTML once it exists.
        pushStoreToEditor();
        const handleUpdate = () => {
          if (applyingFromStore) return;
          const cleaned = normalizeHtmlForStore((p.getHtml as () => string)());
          const model = (p.fieldValue as any)();
          if (model.value !== cleaned) model.value = cleaned;
        };
        ed.on('update', handleUpdate);
        onCleanup(() => {
          ed.off('update', handleUpdate);
        });
      },
      { immediate: true }
    );
    return () => null;
  },
});

onBeforeUnmount(() => {
  editor.value?.destroy();
});

const internalRule = {
  type: 'string',
  validator: (_r: unknown, value: unknown, cb: (error?: Error) => void) => {
    if (value == null || value === '') return cb();
    if (typeof value !== 'string') return cb(new Error(_t('Must be a string')));
    cb();
  },
} as RuleItem;
const mergedRules = computed<RuleItem[]>(() => [...props.rules, internalRule]);
</script>

<style lang="scss" scoped>
.o-htmlfield-edit {
  width: 100%;
  border: 1px solid var(--el-border-color);
  border-radius: var(--el-border-radius-base);
  background: var(--el-fill-color-blank);
}
.o-htmlfield-toolbar {
  display: flex;
  flex-wrap: wrap;
  gap: 4px;
  padding: 6px 8px;
  border-bottom: 1px solid var(--el-border-color-lighter);
}
.o-htmlfield-btn {
  min-width: 28px;
  height: 28px;
  padding: 0 8px;
  border: 1px solid transparent;
  border-radius: 4px;
  background: transparent;
  color: var(--el-text-color-regular);
  cursor: pointer;
  font: inherit;
  &.active,
  &:hover {
    background: var(--el-fill-color-light);
    border-color: var(--el-border-color);
  }
}
.o-htmlfield-editor {
  min-height: 120px;
  padding: 8px 11px;
  :deep(.o-htmlfield-prose) {
    outline: none;
    min-height: 100px;
  }
  :deep(.ProseMirror p) {
    margin: 0 0 0.5em;
  }
}
.o-htmlfield-display {
  line-height: 1.5;
  color: var(--el-text-color-primary);
  padding: 0 11px;
  word-break: break-word;
  :deep(p) {
    margin: 0 0 0.5em;
  }
}
.o-htmlfield-plaintext {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  color: var(--el-text-color-primary);
  padding: 0 11px;
}
</style>
