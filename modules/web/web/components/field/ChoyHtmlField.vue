<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <ChoyFieldBase
    :binding="binding"
    :label="label"
    :rules="rules"
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
    <template #edit="{ fieldValue, inputId, inputName, describedBy, invalid, required }">
      <ChoyHtmlEditorCell
        :field-value="fieldValue"
        :control-id="inputId"
        :control-name="inputName"
        :described-by="describedBy || undefined"
        :invalid="!!invalid"
        :required="!!required"
      />
    </template>
    <template #display="{ fieldValue }">
      <span v-if="plaintext" class="choy-field-display-text">{{ htmlToPlaintext(fieldValue().value) }}</span>
      <div v-else class="prose-choy" v-html="sanitizeHtmlForClient(fieldValue().value)" />
    </template>
  </ChoyFieldBase>
</template>

<script setup lang="ts" generic="T extends BaseModel, P extends FieldPath<T, string | null | undefined>, V = FieldPathType<T, P>">
import type { RuleItem } from 'async-validator';
import type { Editor } from '@tiptap/vue-3';
import type { BaseModel, FieldPath, FieldPathType } from '@/core/rpc';
import type { WebModelStore } from '@/web/web/stores/modelStore';
import { defineComponent, h, onBeforeUnmount, watch } from 'vue';
import { EditorContent, useEditor } from '@tiptap/vue-3';
import StarterKit from '@tiptap/starter-kit';
import Link from '@tiptap/extension-link';
import { useField } from '@/web/web/composables/useField';
import type { NarrowAggProp, NonNumericAggFns, UseField } from '@/web/web/composables/useField';
import ChoyFieldBase, { type FieldStateExpr, type FormItemProps } from './ChoyFieldBase.vue';
import { htmlToPlaintext, normalizeHtmlForStore, resolveChoyHtmlLinkHref, sanitizeHtmlForClient } from './htmlHelpers';
import { htmlEditorChain, htmlEditorSetContent } from './tiptapHtmlCommands';

defineOptions({ name: 'ChoyHtmlField', inheritAttrs: false });

type IsAny<T> = 0 extends 1 & T ? true : false;
type ViewType = string | null;

const props = withDefaults(
  defineProps<{
    store?: WebModelStore<T>;
    prop?: P | (IsAny<T> extends true ? string : never);
    agg?: NarrowAggProp<NonNumericAggFns>;
    binding?: UseField<T, V>;

    label?: string;
    rules?: RuleItem[];
    formItemProps?: Partial<FormItemProps>;
    vColumnProps?: Record<string, unknown>;

    required?: FieldStateExpr<T, V>;
    readonly?: FieldStateExpr<T, V>;
    visible?: FieldStateExpr<T, V>;
    cellVisible?: FieldStateExpr<T, V>;

    renderMode?: 'auto' | 'form' | 'table' | 'inline';
    showInlineError?: boolean;
    /** When true, display mode shows plaintext instead of sanitized HTML. */
    plaintext?: boolean;
  }>(),
  {
    rules: () => [],
    formItemProps: () => ({}),
    vColumnProps: () => ({}),
    renderMode: 'auto',
    showInlineError: false,
    plaintext: false,
    visible: true,
    cellVisible: true,
    readonly: false,
    required: false,
  },
);

const binding = (props.binding ??
  useField<T, P, V>({ store: props.store as WebModelStore<T>, prop: props.prop as P, agg: props.agg })) as UseField<T, V>;

const toView = (raw: unknown): ViewType => (raw == null ? null : String(raw));
const fromView = (v: ViewType) => v as unknown as V;

function toolbarButton(label: string, active: boolean, onClick: () => void) {
  return h(
    'button',
    {
      type: 'button',
      class: ['rounded px-2 py-0.5 text-xs hover:bg-muted', active ? 'bg-muted' : ''],
      onClick: (e: Event) => {
        e.preventDefault();
        onClick();
      },
    },
    label,
  );
}

function toggleLink(ed: Editor): void {
  if (ed.isActive('link')) {
    htmlEditorChain(ed).focus().unsetLink().run();
    return;
  }
  const prev = ed.getAttributes('link').href as string | undefined;
  const href = typeof window !== 'undefined' ? window.prompt('URL', prev || 'https://') : null;
  if (href == null) return;
  const resolved = resolveChoyHtmlLinkHref(href.trim());
  if (resolved === null) {
    htmlEditorChain(ed).focus().unsetLink().run();
    return;
  }
  if (resolved === false) return;
  htmlEditorChain(ed).focus().extendMarkRange('link').setLink({ href: resolved }).run();
}

/**
 * TipTap editor bound to the ChoyFieldBase slot value. Mounted only in edit mode.
 */
const ChoyHtmlEditorCell = defineComponent({
  name: 'ChoyHtmlEditorCell',
  inheritAttrs: false,
  props: {
    fieldValue: { type: Function, required: true },
    controlId: { type: String, default: '' },
    controlName: { type: String, default: '' },
    describedBy: { type: String, default: '' },
    invalid: { type: Boolean, default: false },
    required: { type: Boolean, default: false },
  },
  setup(cellProps) {
    const read = () => (cellProps.fieldValue as () => { value: ViewType })().value;
    const write = (next: ViewType) => {
      (cellProps.fieldValue as () => { value: ViewType })().value = next;
    };

    function controlAttributes(): Record<string, string> {
      const next: Record<string, string> = {
        class: 'choy-html-field__prose min-h-[6rem] px-3 py-2 text-sm outline-none',
      };
      if (cellProps.controlId) next.id = cellProps.controlId;
      if (cellProps.controlName) next.name = cellProps.controlName;
      if (cellProps.invalid) next['aria-invalid'] = 'true';
      if (cellProps.required) next['aria-required'] = 'true';
      if (cellProps.describedBy) next['aria-describedby'] = cellProps.describedBy;
      return next;
    }

    function applyControlAttributes(ed: unknown): void {
      const dom = (ed as { view?: { dom?: HTMLElement } } | null | undefined)?.view?.dom;
      if (!dom) return;
      const next = controlAttributes();
      // Class stays on editorProps.attributes; rewriting it here would drop TipTap/ProseMirror roots.
      for (const key of ['id', 'name', 'aria-invalid', 'aria-required', 'aria-describedby'] as const) {
        if (next[key]) dom.setAttribute(key, next[key]!);
        else dom.removeAttribute(key);
      }
    }

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
        attributes: controlAttributes(),
      },
      onUpdate: ({ editor: ed }) => {
        write(normalizeHtmlForStore(ed.getHTML()));
      },
    });

    watch(
      () => [read(), editor.value] as const,
      ([next]) => {
        const ed = editor.value;
        if (!ed) return;
        const sanitized = next == null || next === '' ? '' : sanitizeHtmlForClient(next);
        const current = ed.getHTML();
        if (current === sanitized) return;
        // Empty docs serialize as '<p></p>' while the store normalizes to null.
        if (!sanitized && current === '<p></p>') return;
        htmlEditorSetContent(ed, sanitized, false);
      },
      { immediate: true },
    );

    watch(
      () =>
        [
          cellProps.controlId,
          cellProps.controlName,
          cellProps.invalid,
          cellProps.required,
          cellProps.describedBy,
          editor.value,
        ] as const,
      () => applyControlAttributes(editor.value),
      { immediate: true },
    );

    onBeforeUnmount(() => {
      editor.value?.destroy();
    });

    return () => {
      const ed = editor.value;
      return h(
        'div',
        { class: 'choy-html-field__edit overflow-hidden rounded-md border border-border bg-background' },
        [
          ed
            ? h(
                'div',
                {
                  class: 'flex flex-wrap gap-1 border-b border-border bg-muted/30 px-2 py-1',
                  role: 'toolbar',
                },
                [
                  toolbarButton('B', ed.isActive('bold'), () => htmlEditorChain(ed).focus().toggleBold().run()),
                  toolbarButton('I', ed.isActive('italic'), () => htmlEditorChain(ed).focus().toggleItalic().run()),
                  toolbarButton('•', ed.isActive('bulletList'), () => htmlEditorChain(ed).focus().toggleBulletList().run()),
                  toolbarButton('1.', ed.isActive('orderedList'), () => htmlEditorChain(ed).focus().toggleOrderedList().run()),
                  toolbarButton('Link', ed.isActive('link'), () => toggleLink(ed)),
                ],
              )
            : null,
          h(EditorContent, { editor: ed }),
        ],
      );
    };
  },
});
</script>
