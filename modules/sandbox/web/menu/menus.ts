// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import type { MenuItem } from '@/core/web/menu';
import { defineMenu } from '@/core/web/resource';
import { FlaskConical } from 'lucide-vue-next';
import { sandboxRecordsMenuTitle, sandboxRootMenuTitle } from './titles';

export const sandboxMenus: MenuItem[] = [
  defineMenu('sandbox.menu.root', {
    title: sandboxRootMenuTitle,
    sequence: 90,
    // Lab flask: field playground / kitchen-sink experiments (distinct from meta's Boxes).
    icon: FlaskConical,
    children: [
      defineMenu('sandbox.menu.records', {
        title: sandboxRecordsMenuTitle,
        path: '/sandbox/records',
        sequence: 10,
      }),
    ],
  }),
];
