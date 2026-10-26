// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * CodeMirror 6 host for ChoyJsonField edit mode.
 * FE unit stubs replace codemirror / @codemirror/* packages.
 */

import { basicSetup } from 'codemirror';
import { EditorView } from '@codemirror/view';
import { EditorState } from '@codemirror/state';
import { json } from '@codemirror/lang-json';
import { linter, lintGutter, type Diagnostic } from '@codemirror/lint';
import {
  localizeJsonParseError,
  parseJsonLintOffset,
  type JsonLintTranslate,
} from './jsonLintMessages';

export type JsonCodeMirrorHandle = {
  getDoc: () => string;
  setDoc: (text: string) => void;
  focus: () => void;
  destroy: () => void;
  dom: HTMLElement;
};

export type MountJsonCodeMirrorOpts = {
  parent: HTMLElement;
  doc: string;
  onUpdate?: (text: string) => void;
  onBlur?: () => void;
  /** Applied to the contenteditable / textarea for a11y. */
  ariaDescribedBy?: string;
  ariaInvalid?: boolean;
  id?: string;
  /** Localize JSON.parse / lint tip strings (defaults to identity). */
  translate?: JsonLintTranslate;
};

/** Product-light CM theme using CSS variables (avoid dark one-dark skin). */
const choyJsonCmTheme = EditorView.theme({
  '&': {
    fontSize: '13px',
    backgroundColor: 'transparent',
  },
  '.cm-content': {
    fontFamily: 'ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace',
    lineHeight: '1.375',
    padding: '0.5rem 0.75rem',
    minHeight: '8rem',
    caretColor: 'var(--choy-color-foreground)',
    color: 'var(--choy-color-foreground)',
  },
  '.cm-gutters': {
    backgroundColor: 'color-mix(in oklab, var(--choy-color-muted) 55%, transparent)',
    color: 'var(--choy-color-muted-foreground)',
    border: 'none',
    borderInlineEnd: '1px solid var(--choy-color-border)',
  },
  '.cm-activeLineGutter': {
    backgroundColor: 'transparent',
  },
  '.cm-activeLine': {
    backgroundColor: 'color-mix(in oklab, var(--choy-color-muted) 35%, transparent)',
  },
  '&.cm-focused': {
    outline: 'none',
  },
  '.cm-selectionBackground, &.cm-focused .cm-selectionBackground': {
    backgroundColor: 'color-mix(in oklab, var(--choy-color-primary) 22%, transparent) !important',
  },
});

/** Lint source: JSON.parse + localized tip (no CM locale pack). */
export function choyJsonParseLinter(translate: JsonLintTranslate) {
  return (view: EditorView): Diagnostic[] => {
    const text = view.state.doc.toString();
    if (!String(text).trim()) return [];
    try {
      JSON.parse(text);
      return [];
    } catch (err) {
      const raw = err instanceof Error ? err.message : String(err);
      const from = parseJsonLintOffset(raw, view.state.doc.length);
      return [
        {
          from,
          to: from,
          severity: 'error',
          message: localizeJsonParseError(raw, translate),
        },
      ];
    }
  };
}

export function mountJsonCodeMirror(opts: MountJsonCodeMirrorOpts): JsonCodeMirrorHandle {
  const translate: JsonLintTranslate = opts.translate ?? ((msgid, ...args) => {
    let out = msgid;
    for (const arg of args) {
      out = out.replace('%s', String(arg));
    }
    return out;
  });

  const extensions: unknown[] = [
    basicSetup,
    json(),
    lintGutter(),
    linter(choyJsonParseLinter(translate) as any),
    choyJsonCmTheme,
    EditorView.updateListener.of((update: { docChanged: boolean; state: { doc: { toString: () => string } } }) => {
      if (update.docChanged) {
        opts.onUpdate?.(update.state.doc.toString());
      }
    }),
    EditorView.domEventHandlers({
      blur: () => {
        opts.onBlur?.();
        return false;
      },
    }),
  ];

  if (opts.id || opts.ariaDescribedBy || opts.ariaInvalid) {
    extensions.push(
      EditorView.contentAttributes.of({
        ...(opts.id ? { id: opts.id } : {}),
        ...(opts.ariaDescribedBy ? { 'aria-describedby': opts.ariaDescribedBy } : {}),
        ...(opts.ariaInvalid ? { 'aria-invalid': 'true' } : {}),
      }),
    );
  }

  const state = EditorState.create({
    doc: opts.doc ?? '',
    extensions: extensions as any,
  });
  const view = new EditorView({ state, parent: opts.parent });

  return {
    getDoc: () => view.state.doc.toString(),
    setDoc: (text: string) => {
      const next = text ?? '';
      const cur = view.state.doc.toString();
      if (cur === next) return;
      view.dispatch({
        changes: { from: 0, to: view.state.doc.length, insert: next },
      });
    },
    focus: () => view.focus(),
    destroy: () => view.destroy(),
    dom: view.dom,
  };
}
