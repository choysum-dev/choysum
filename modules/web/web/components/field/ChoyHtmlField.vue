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
import StarterKit from '@tiptap/starter-kit';
import Link from '@tiptap/extension-link';
import { Bold, Italic, Link2, List, ListOrdered } from 'lucide-vue-next';
import { useField } from '@/web/web/composables/useField';
import type { NarrowAggProp, NonNumericAggFns, UseField } from '@/web/web/composables/useField';
import ChoyFieldBase, { type FieldStateExpr, type FormItemProps } from './ChoyFieldBase.vue';
import { htmlToPlaintext, normalizeHtmlForStore, resolveChoyHtmlLinkHref, sanitizeHtmlForClient } from './htmlHelpers';
import { htmlEditorChain, htmlEditorSetContent } from './tiptapHtmlCommands';
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

function isEmptyHtml(raw: unknown): boolean {
  return normalizeHtmlForStore(raw == null ? null : String(raw)) == null;
}

const emptyDisplayText = computed(() => {
  const custom = props.emptyText != null ? String(props.emptyText).trim() : '';
  return custom || _t('No content');
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
    describedBy: { type: String, default: '' },
    invalid: { type: Boolean, default: false },
    required: { type: Boolean, default: false },
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
      const toolbar = ed
        ? h(
            'div',
            {
              class: 'flex flex-wrap gap-1 border-b border-border bg-muted/30 px-2 py-1',
              role: 'toolbar',
            },
            [
              toolbarButton(Bold, _t('Bold'), ed.isActive('bold'), () =>
                htmlEditorChain(ed).focus().toggleBold().run(),
              ),
              toolbarButton(Italic, _t('Italic'), ed.isActive('italic'), () =>
                htmlEditorChain(ed).focus().toggleItalic().run(),
              ),
              toolbarButton(List, _t('Bullet list'), ed.isActive('bulletList'), () =>
                htmlEditorChain(ed).focus().toggleBulletList().run(),
              ),
              toolbarButton(ListOrdered, _t('Ordered list'), ed.isActive('orderedList'), () =>
                htmlEditorChain(ed).focus().toggleOrderedList().run(),
              ),
              toolbarButton(Link2, _t('Link'), ed.isActive('link'), () => openLinkPopover(ed)),
            ],
          )
        : null;

      const linkPanel =
        ed && linkOpen.value
          ? h('div', { class: 'choy-html-field__link-popover flex flex-wrap items-center gap-2 border-b border-border bg-background px-2 py-1.5' }, [
              h('input', {
                class: 'choy-html-field__link-input min-w-[12rem] flex-1 rounded-md border border-border bg-background px-2 py-1 text-sm',
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
            ])
          : null;

      return h(
        'div',
        { class: 'choy-html-field__edit overflow-hidden rounded-md border border-border bg-background' },
        [toolbar, linkPanel, h(EditorContent, { editor: ed })],
      );
    };
  },
});
</script>
