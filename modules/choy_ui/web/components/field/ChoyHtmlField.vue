<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<script setup lang="ts">
import { onBeforeUnmount, watch } from 'vue';
import { EditorContent, useEditor } from '@tiptap/vue-3';
import StarterKit from '@tiptap/starter-kit';
import Link from '@tiptap/extension-link';
import type { ClassValue } from '../../lib/utils';
import ChoyFieldBase from './ChoyFieldBase.vue';
import {
  choyFieldChromeDefaults,
  type ChoyFieldChromeProps,
} from './fieldHelpers';
import { htmlToPlaintext, normalizeHtmlForStore, sanitizeHtmlForClient } from './htmlHelpers';
import { htmlEditorChain, htmlEditorSetContent } from './tiptapHtmlCommands';

/**
 * Rich-text field (TipTap + sanitize). Isolation: defineModel string | null.
 */
const props = withDefaults(
  defineProps<
    ChoyFieldChromeProps & {
      class?: ClassValue;
      /** When true, show plaintext instead of sanitized HTML. */
      plaintext?: boolean;
    }
  >(),
  {
    ...choyFieldChromeDefaults,
    plaintext: false,
  },
);

const model = defineModel<string | null>({ default: null });

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
  editable: !(props.disabled || props.readonly),
  editorProps: {
    attributes: {
      class: 'choy-html-field__prose min-h-[6rem] px-3 py-2 text-sm outline-none',
    },
  },
  onUpdate: ({ editor: ed }) => {
    model.value = normalizeHtmlForStore(ed.getHTML());
  },
});

watch(
  () => [model.value, editor.value] as const,
  ([next]) => {
    if (!editor.value) return;
    const sanitized = next == null || next === '' ? '' : sanitizeHtmlForClient(next);
    const current = editor.value.getHTML();
    if (current === sanitized) return;
    htmlEditorSetContent(editor.value, sanitized || '', false);
  },
  { immediate: true },
);

watch(
  () => [props.disabled, props.readonly] as const,
  ([disabled, readonly]) => {
    editor.value?.setEditable(!(disabled || readonly));
  },
);

onBeforeUnmount(() => {
  editor.value?.destroy();
});

function toggleLink(): void {
  const ed = editor.value;
  if (!ed) return;
  if (ed.isActive('link')) {
    htmlEditorChain(ed).focus().unsetLink().run();
    return;
  }
  const prev = ed.getAttributes('link').href as string | undefined;
  const href = typeof window !== 'undefined' ? window.prompt('URL', prev || 'https://') : null;
  if (href == null) return;
  const trimmed = href.trim();
  if (!trimmed) {
    htmlEditorChain(ed).focus().unsetLink().run();
    return;
  }
  htmlEditorChain(ed).focus().extendMarkRange('link').setLink({ href: trimmed }).run();
}

const displayPlain = () => htmlToPlaintext(model.value);
const displayHtml = () => sanitizeHtmlForClient(model.value);
</script>

<template>
  <ChoyFieldBase
    data-anchor="choy.html-field"
    :class="props.class"
    :label="label"
    :help="help"
    :required="required"
    :readonly="readonly"
    :disabled="disabled"
    :error="error"
    :name="name"
    :visible="visible"
  >
    <template #default="{ controlId, ariaInvalid, ariaRequired, ariaDescribedby }">
      <div
        v-if="readonly || disabled"
        :id="controlId"
        class="choy-html-field__display rounded-md border border-border bg-muted/20 px-3 py-2 text-sm"
        :aria-invalid="ariaInvalid"
        :aria-required="ariaRequired"
        :aria-describedby="ariaDescribedby"
      >
        <span v-if="plaintext">{{ displayPlain() }}</span>
        <div v-else class="prose-choy" v-html="displayHtml()" />
      </div>
      <div
        v-else
        class="choy-html-field__edit overflow-hidden rounded-md border border-border bg-background"
      >
        <div
          v-if="editor"
          class="flex flex-wrap gap-1 border-b border-border bg-muted/30 px-2 py-1"
          role="toolbar"
        >
          <button
            type="button"
            class="rounded px-2 py-0.5 text-xs font-semibold hover:bg-muted"
            :class="{ 'bg-muted': editor.isActive('bold') }"
            @click.prevent="htmlEditorChain(editor).focus().toggleBold().run()"
          >
            B
          </button>
          <button
            type="button"
            class="rounded px-2 py-0.5 text-xs italic hover:bg-muted"
            :class="{ 'bg-muted': editor.isActive('italic') }"
            @click.prevent="htmlEditorChain(editor).focus().toggleItalic().run()"
          >
            I
          </button>
          <button
            type="button"
            class="rounded px-2 py-0.5 text-xs hover:bg-muted"
            :class="{ 'bg-muted': editor.isActive('bulletList') }"
            @click.prevent="htmlEditorChain(editor).focus().toggleBulletList().run()"
          >
            •
          </button>
          <button
            type="button"
            class="rounded px-2 py-0.5 text-xs hover:bg-muted"
            :class="{ 'bg-muted': editor.isActive('orderedList') }"
            @click.prevent="htmlEditorChain(editor).focus().toggleOrderedList().run()"
          >
            1.
          </button>
          <button
            type="button"
            class="rounded px-2 py-0.5 text-xs hover:bg-muted"
            :class="{ 'bg-muted': editor.isActive('link') }"
            @click.prevent="toggleLink"
          >
            Link
          </button>
        </div>
        <EditorContent
          :editor="editor"
          :id="controlId"
          :aria-invalid="ariaInvalid"
          :aria-required="ariaRequired"
          :aria-describedby="ariaDescribedby"
        />
      </div>
    </template>
  </ChoyFieldBase>
</template>
