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

export const BarChart3 = icon('BarChart3');
export const Bell = icon('Bell');
export const Bookmark = icon('Bookmark');
export const Boxes = icon('Boxes');
export const Building2 = icon('Building2');
export const Check = icon('Check');
export const CheckCircle2 = icon('CheckCircle2');
export const ChevronDown = icon('ChevronDown');
export const ChevronLeft = icon('ChevronLeft');
export const ChevronRight = icon('ChevronRight');
export const CircleAlert = icon('CircleAlert');
export const CircleCheck = icon('CircleCheck');
export const CircleHelp = icon('CircleHelp');
export const Copy = icon('Copy');
export const Database = icon('Database');
export const FileText = icon('FileText');
export const GitBranch = icon('GitBranch');
export const History = icon('History');
export const House = icon('House');
export const Image = icon('Image');
export const Info = icon('Info');
export const Languages = icon('Languages');
export const LayoutGrid = icon('LayoutGrid');
export const List = icon('List');
export const Loader2 = icon('Loader2');
export const Menu = icon('Menu');
export const Monitor = icon('Monitor');
export const Moon = icon('Moon');
export const OctagonX = icon('OctagonX');
export const PanelLeft = icon('PanelLeft');
export const MoreHorizontal = icon('MoreHorizontal');
export const Pencil = icon('Pencil');
export const Plus = icon('Plus');
export const RefreshCw = icon('RefreshCw');
export const RotateCcw = icon('RotateCcw');
export const School = icon('School');
export const Search = icon('Search');
export const Settings = icon('Settings');
export const Shield = icon('Shield');
export const Sun = icon('Sun');
export const Trash2 = icon('Trash2');
export const TriangleAlert = icon('TriangleAlert');
export const Upload = icon('Upload');
export const User = icon('User');
export const UserRound = icon('UserRound');
export const Users = icon('Users');
export const X = icon('X');
export const XCircle = icon('XCircle');
