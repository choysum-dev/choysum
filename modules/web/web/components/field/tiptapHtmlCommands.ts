// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import type { Editor } from '@tiptap/vue-3';

/**
 * TipTap command typings bridge: keep call sites typed without relying on
 * per-file `declare module` merges that vue-tsc may not apply in SFCs.
 */
type HtmlEditorChain = {
  focus: (position?: unknown, options?: { scrollIntoView?: boolean }) => HtmlEditorChain;
  toggleBold: () => HtmlEditorChain;
  toggleItalic: () => HtmlEditorChain;
  toggleBulletList: () => HtmlEditorChain;
  toggleOrderedList: () => HtmlEditorChain;
  unsetLink: () => HtmlEditorChain;
  extendMarkRange: (typeOrName: string) => HtmlEditorChain;
  setLink: (attributes: { href: string }) => HtmlEditorChain;
  run: () => boolean;
};

export function htmlEditorChain(editor: Editor): HtmlEditorChain {
  return editor.chain() as unknown as HtmlEditorChain;
}

export function htmlEditorSetContent(editor: Editor, content: string, emitUpdate = false): boolean {
  return (
    editor.commands as unknown as { setContent: (c: string, emit?: boolean) => boolean }
  ).setContent(content, emitUpdate);
}
