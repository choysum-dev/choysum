// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { Comment, Fragment, Text, type VNode } from 'vue';

/** Resolve the displayed stat value (D7): explicit `value` wins, else relation array length, else empty. */
export function resolveStatDisplayValue(opts: {
  value?: number | string | null;
  relationValue?: unknown;
  emptyValue?: number | string;
}): number | string {
  if (opts.value !== undefined && opts.value !== null) return opts.value;
  if (Array.isArray(opts.relationValue)) return opts.relationValue.length;
  return opts.emptyValue ?? '—';
}

/** True when a default-slot VNode tree has at least one meaningful child (D3 / D9). */
export function slotHasContent(nodes: unknown[] | undefined | null): boolean {
  if (!nodes || nodes.length === 0) return false;
  return nodes.some(isMeaningfulVNode);
}

function isMeaningfulVNode(node: unknown): boolean {
  if (node == null || typeof node !== 'object') return false;
  const vnode = node as VNode;
  const type = vnode.type;
  if (type === Comment) return false;
  if (type === Text) {
    return String(vnode.children ?? '').trim().length > 0;
  }
  if (type === Fragment) {
    return Array.isArray(vnode.children) && slotHasContent(vnode.children as unknown[]);
  }
  return true;
}
