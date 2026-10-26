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
    :orientation="orientation"
    :vColumnProps="vColumnProps"
    :toView="toView"
    :fromView="fromView"
    :required="required"
    :readonly="readonly"
    :visible="visible"
    :cellVisible="cellVisible"
    :renderMode="renderMode"
    :showInlineError="showInlineError"
    :label-for-control="false"
    v-bind="$attrs"
  >
    <template #edit="{ fieldValue, inputId, inputName, labelId, describedBy, invalid, required }">
      <ChoyHtmlEditorCell
        :field-value="fieldValue"
        :control-id="inputId"
        :control-name="inputName"
        :label-id="labelId"
        :described-by="describedBy || undefined"
        :invalid="!!invalid"
        :required="!!required"
        :placeholder="editorPlaceholder"
      />
    </template>
    <template #display="{ fieldValue }">
      <span v-if="plaintext" class="choy-field-display-text">{{ htmlToPlaintext(fieldValue().value) }}</span>
      <div
        v-else-if="isEmptyHtml(fieldValue().value)"
        class="choy-html-field__empty text-sm text-muted-foreground"
      >
        {{ emptyDisplayText }}
      </div>
      <div v-else class="prose-choy" v-html="sanitizeHtmlForClient(fieldValue().value)" />
    </template>
  </ChoyFieldBase>
</template>

<script setup lang="ts" generic="T extends BaseModel, P extends FieldPath<T, string | null | undefined>, V = FieldPathType<T, P>">
import type { RuleItem } from 'async-validator';
import type { Editor } from '@tiptap/vue-3';
import type { BaseModel, FieldPath, FieldPathType } from '@/core/rpc';
import type { WebModelStore } from '@/web/web/stores/modelStore';
import { computed, defineComponent, h, onBeforeUnmount, ref, watch, type Component } from 'vue';
import { EditorContent, useEditor } from '@tiptap/vue-3';
import Link from '@tiptap/extension-link';
import StarterKit from '@tiptap/starter-kit';
import {
  Bold,
  Italic,
  Strikethrough,
  Heading2,
  Heading3,
  List,
  ListOrdered,
  Quote,
  Code,
  Link2,
  Undo2,
  Redo2,
} from 'lucide-vue-next';
import { useField } from '@/web/web/composables/useField';
import type { NarrowAggProp, NonNumericAggFns, UseField } from '@/web/web/composables/useField';
import ChoyFieldBase, { type FieldStateExpr, type FormItemProps } from './ChoyFieldBase.vue';
import { htmlToPlaintext, normalizeHtmlForStore, resolveChoyHtmlLinkHref, sanitizeHtmlForClient } from './htmlHelpers';
import { htmlEditorChain, htmlEditorIsActive, htmlEditorSetContent } from './tiptapHtmlCommands';
import { createTranslate } from '@/web/web/i18n';

const { _t } = createTranslate('web', { scope: 'web/components/field/ChoyHtmlField' });

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
    orientation?: 'horizontal' | 'vertical';
    vColumnProps?: Record<string, unknown>;

    required?: FieldStateExpr<T, V>;
    readonly?: FieldStateExpr<T, V>;
    visible?: FieldStateExpr<T, V>;
    cellVisible?: FieldStateExpr<T, V>;

    renderMode?: 'auto' | 'form' | 'table' | 'inline';
    showInlineError?: boolean;
    /** When true, display mode shows plaintext instead of sanitized HTML. */
    plaintext?: boolean;
    /** Display-mode placeholder when the HTML value is empty. */
    emptyText?: string;
    /** Edit-mode placeholder when the document is empty. */
    placeholder?: string;
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
    placeholder: '',
  },
);

const binding = (props.binding ??
  useField<T, P, V>({ store: props.store as WebModelStore<T>, prop: props.prop as P, agg: props.agg })) as UseField<T, V>;

const toView = (raw: unknown): ViewType => (raw == null ? null : String(raw));
const fromView = (v: ViewType) => v as unknown as V;

function isEmptyHtml(raw: unknown): boolean {
  return normalizeHtmlForStore(raw == null ? null : String(raw)) == null;
}

const emptyDisplayText = computed(() => {
  if (props.emptyText != null) return String(props.emptyText);
  return '';
});

const editorPlaceholder = computed(() => {
  const custom = props.placeholder != null ? String(props.placeholder).trim() : '';
  return custom || _t('Write rich text…');
});

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
    labelId: { type: String, default: '' },
    describedBy: { type: String, default: '' },
    invalid: { type: Boolean, default: false },
    required: { type: Boolean, default: false },
    placeholder: { type: String, default: '' },
  },
  setup(cellProps) {
    const read = () => (cellProps.fieldValue as () => { value: ViewType })().value;
    const write = (next: ViewType) => {
      (cellProps.fieldValue as () => { value: ViewType })().value = next;
    };

    const linkOpen = ref(false);
    const linkHref = ref('https://');
    const linkError = ref<string | null>(null);

    function toolbarButton(icon: Component, tooltip: string, active: boolean, onClick: () => void) {
      return h(
        'button',
        {
          type: 'button',
          class: [
            'inline-flex h-7 w-7 items-center justify-center rounded text-muted-foreground hover:bg-muted hover:text-foreground',
            active ? 'bg-muted text-foreground' : '',
          ],
          title: tooltip,
          'aria-label': tooltip,
          'aria-pressed': active ? 'true' : 'false',
          onClick: (e: Event) => {
            e.preventDefault();
            onClick();
          },
        },
        [h(icon, { class: 'size-3.5', 'aria-hidden': 'true' })],
      );
    }

    function toolbarSep() {
      return h('span', {
        class: 'mx-0.5 h-4 w-px shrink-0 bg-border',
        'aria-hidden': 'true',
      });
    }

    function openLinkPopover(ed: Editor): void {
      if (ed.isActive('link')) {
        htmlEditorChain(ed).focus().unsetLink().run();
        linkOpen.value = false;
        linkError.value = null;
        return;
      }
      const prev = ed.getAttributes('link').href as string | undefined;
      linkHref.value = prev || 'https://';
      linkError.value = null;
      linkOpen.value = true;
    }

    function applyLink(ed: Editor): void {
      const resolved = resolveChoyHtmlLinkHref(String(linkHref.value || '').trim());
      if (resolved === null) {
        htmlEditorChain(ed).focus().unsetLink().run();
        linkOpen.value = false;
        linkError.value = null;
        return;
      }
      if (resolved === false) {
        linkError.value = _t('Invalid URL');
        return;
      }
      htmlEditorChain(ed).focus().extendMarkRange('link').setLink({ href: resolved }).run();
      linkOpen.value = false;
      linkError.value = null;
    }

    function controlAttributes(): Record<string, string> {
      const next: Record<string, string> = {
        class: 'choy-html-field__prose prose-choy min-h-[8rem] px-3 py-2 text-sm outline-none',
      };
      if (cellProps.controlId) next.id = cellProps.controlId;
      if (cellProps.controlName) next.name = cellProps.controlName;
      if (cellProps.labelId) next['aria-labelledby'] = cellProps.labelId;
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
      for (const key of [
        'id',
        'name',
        'aria-labelledby',
        'aria-invalid',
        'aria-required',
        'aria-describedby',
      ] as const) {
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
          cellProps.placeholder,
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
      const toolbar = ed
        ? h(
            'div',
            {
              class: 'flex flex-wrap items-center gap-0.5 border-b border-border bg-muted/30 px-1.5 py-1',
              role: 'toolbar',
            },
            [
              toolbarButton(Bold, _t('Bold'), htmlEditorIsActive(ed, 'bold'), () =>
                htmlEditorChain(ed).focus().toggleBold().run(),
              ),
              toolbarButton(Italic, _t('Italic'), htmlEditorIsActive(ed, 'italic'), () =>
                htmlEditorChain(ed).focus().toggleItalic().run(),
              ),
              toolbarButton(Strikethrough, _t('Strikethrough'), htmlEditorIsActive(ed, 'strike'), () =>
                htmlEditorChain(ed).focus().toggleStrike().run(),
              ),
              toolbarButton(Code, _t('Inline code'), htmlEditorIsActive(ed, 'code'), () =>
                htmlEditorChain(ed).focus().toggleCode().run(),
              ),
              toolbarSep(),
              toolbarButton(Heading2, _t('Heading 2'), htmlEditorIsActive(ed, 'heading', { level: 2 }), () =>
                htmlEditorChain(ed).focus().toggleHeading({ level: 2 }).run(),
              ),
              toolbarButton(Heading3, _t('Heading 3'), htmlEditorIsActive(ed, 'heading', { level: 3 }), () =>
                htmlEditorChain(ed).focus().toggleHeading({ level: 3 }).run(),
              ),
              toolbarButton(Quote, _t('Blockquote'), htmlEditorIsActive(ed, 'blockquote'), () =>
                htmlEditorChain(ed).focus().toggleBlockquote().run(),
              ),
              toolbarSep(),
              toolbarButton(List, _t('Bullet list'), htmlEditorIsActive(ed, 'bulletList'), () =>
                htmlEditorChain(ed).focus().toggleBulletList().run(),
              ),
              toolbarButton(ListOrdered, _t('Ordered list'), htmlEditorIsActive(ed, 'orderedList'), () =>
                htmlEditorChain(ed).focus().toggleOrderedList().run(),
              ),
              toolbarButton(Link2, _t('Link'), htmlEditorIsActive(ed, 'link'), () => openLinkPopover(ed)),
              toolbarSep(),
              toolbarButton(Undo2, _t('Undo'), false, () => htmlEditorChain(ed).focus().undo().run()),
              toolbarButton(Redo2, _t('Redo'), false, () => htmlEditorChain(ed).focus().redo().run()),
            ],
          )
        : null;

      const linkPanel =
        ed && linkOpen.value
          ? h(
              'div',
              {
                class:
                  'choy-html-field__link-popover flex flex-wrap items-center gap-2 border-b border-border bg-background px-2 py-1.5',
              },
              [
                h('input', {
                  class:
                    'choy-html-field__link-input min-w-[12rem] flex-1 rounded-md border border-border bg-background px-2 py-1 text-sm',
                  type: 'url',
                  value: linkHref.value,
                  placeholder: _t('URL'),
                  'aria-label': _t('URL'),
                  onInput: (e: Event) => {
                    linkHref.value = (e.target as HTMLInputElement).value;
                    linkError.value = null;
                  },
                  onKeydown: (e: KeyboardEvent) => {
                    if (e.key === 'Enter') {
                      e.preventDefault();
                      applyLink(ed);
                    } else if (e.key === 'Escape') {
                      e.preventDefault();
                      linkOpen.value = false;
                      linkError.value = null;
                    }
                  },
                }),
                h(
                  'button',
                  {
                    type: 'button',
                    class: 'rounded px-2 py-0.5 text-xs bg-primary text-primary-foreground',
                    onClick: (e: Event) => {
                      e.preventDefault();
                      applyLink(ed);
                    },
                  },
                  _t('Apply'),
                ),
                h(
                  'button',
                  {
                    type: 'button',
                    class: 'rounded px-2 py-0.5 text-xs hover:bg-muted',
                    onClick: (e: Event) => {
                      e.preventDefault();
                      linkOpen.value = false;
                      linkError.value = null;
                    },
                  },
                  _t('Cancel'),
                ),
                linkError.value
                  ? h('span', { class: 'basis-full text-xs text-destructive', role: 'alert' }, linkError.value)
                  : null,
              ],
            )
          : null;

      const empty = !ed || !!(ed as { isEmpty?: boolean }).isEmpty;
      const body = h('div', { class: 'relative' }, [
        empty && cellProps.placeholder
          ? h(
              'div',
              {
                class:
                  'pointer-events-none absolute inset-x-3 top-2 z-[1] text-sm text-muted-foreground',
                'aria-hidden': 'true',
              },
              cellProps.placeholder,
            )
          : null,
        h(EditorContent, { editor: ed }),
      ]);

      return h(
        'div',
        { class: 'choy-html-field__edit overflow-hidden rounded-md border border-border bg-background' },
        [toolbar, linkPanel, body],
      );
    };
  },
});
</script>
