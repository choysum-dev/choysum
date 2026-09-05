// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

/** Stable virtual helper paths used by Go overlay VFS. */
export const VUE_HELPER_TEMPLATE =
  "/choysum-vue-virtual/types/template-helpers.d.ts";
export const VUE_HELPER_PROPS =
  "/choysum-vue-virtual/types/props-fallback.d.ts";

export type RewriteEdit = { index: number; delta: number };

/**
 * Rewrite language-core helper /// <reference types="…"> to stable virtual paths.
 * Returns rewritten content and per-edit deltas (original indices) for mapping shifts.
 */
export function rewriteVueHelperReferences(content: string): {
  content: string;
  edits: RewriteEdit[];
} {
  const replacements: { re: RegExp; to: string }[] = [
    {
      re: /\/\/\/\s*<reference\s+types="[^"]*template-helpers\.d\.ts"\s*\/>/g,
      to: `/// <reference path="${VUE_HELPER_TEMPLATE}" />`,
    },
    {
      re: /\/\/\/\s*<reference\s+types="[^"]*props-fallback\.d\.ts"\s*\/>/g,
      to: `/// <reference path="${VUE_HELPER_PROPS}" />`,
    },
  ];

  type Match = { index: number; from: string; to: string };
  const matches: Match[] = [];
  for (const { re, to } of replacements) {
    re.lastIndex = 0;
    let m: RegExpExecArray | null;
    while ((m = re.exec(content)) !== null) {
      matches.push({ index: m.index, from: m[0], to });
    }
  }
  matches.sort((a, b) => a.index - b.index);

  const edits: RewriteEdit[] = [];
  let shift = 0;
  let out = content;
  for (const m of matches) {
    const at = m.index + shift;
    out = out.slice(0, at) + m.to + out.slice(at + m.from.length);
    const delta = m.to.length - m.from.length;
    edits.push({ index: m.index, delta });
    shift += delta;
  }
  return { content: out, edits };
}

/** Sum of length deltas for edits whose original index is at or before pos. */
export function cumulativeDeltaAfter(edits: RewriteEdit[], pos: number): number {
  let delta = 0;
  for (const e of edits) {
    if (e.index <= pos) {
      delta += e.delta;
    }
  }
  return delta;
}
