// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

/**
 * FE unit stub for `lucide-vue-next`.
 * Icons are decorative in unit mounts; render a named empty SVG.
 */
import { defineComponent, h } from 'vue';

function icon(name) {
  return defineComponent({
    name,
    inheritAttrs: false,
    setup(_, { attrs }) {
      return () => h('svg', { 'data-lucide': name, ...attrs });
    },
  });
}

export const Bell = icon('Bell');
export const Bookmark = icon('Bookmark');
export const Building2 = icon('Building2');
export const Check = icon('Check');
export const ChevronDown = icon('ChevronDown');
export const ChevronLeft = icon('ChevronLeft');
export const ChevronRight = icon('ChevronRight');
export const CircleAlert = icon('CircleAlert');
export const CircleHelp = icon('CircleHelp');
export const Copy = icon('Copy');
export const FileText = icon('FileText');
export const House = icon('House');
export const Image = icon('Image');
export const Languages = icon('Languages');
export const Loader2 = icon('Loader2');
export const Pencil = icon('Pencil');
export const Plus = icon('Plus');
export const RefreshCw = icon('RefreshCw');
export const RotateCcw = icon('RotateCcw');
export const Search = icon('Search');
export const Settings = icon('Settings');
export const Trash2 = icon('Trash2');
export const Upload = icon('Upload');
export const X = icon('X');
