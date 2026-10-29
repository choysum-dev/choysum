// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import type { MenuItem } from '@/core/web/menu';
import { defineMenu } from '@/core/web/resource';
import { FlaskConical } from 'lucide-vue-next';
import {
  sandboxConfigurationMenuTitle,
  sandboxRecordPropertiesMenuTitle,
  sandboxRecordsMenuTitle,
  sandboxRootMenuTitle,
} from './titles';

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
      // Pathless folder: app configuration (properties schema, future settings).
      defineMenu('sandbox.menu.configuration', {
        title: sandboxConfigurationMenuTitle,
        sequence: 90,
        children: [
          defineMenu('sandbox.menu.record_properties', {
            title: sandboxRecordPropertiesMenuTitle,
            path: '/sandbox/configuration/record-properties',
            sequence: 10,
          }),
        ],
      }),
    ],
  }),
];
