// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { describe, it, expect } from 'vitest';
import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';

function roleFormSource(): string {
  return readFileSync(resolve(__dirname, 'RoleFormView.vue'), 'utf8');
}

describe('Role UiResources field binding', () => {
  it('declares AccessUiResourceIds as primary UI resource editor', () => {
    const source = roleFormSource();

    expect(source).toContain('OManyToManyRefTreeField');
    expect(source).toContain('prop="AccessUiResourceIds"');
    expect(source).toContain('label="Accessible UI Resources"');
    expect(source).toContain(':lazy="false"');
    expect(source).toContain(':max-depth="0"');
    expect(source).toContain('children-field="Childs"');
    expect(source).toContain(':root-condition="{');
    expect(source).toContain("['Type', '=', 'MENU']");
    expect(source).toContain("['ParentId', 'is', null]");
    expect(source).toContain(':fields="[\'Type\']"');
    expect(source).toContain(':check-strictly="false"');
    expect(source).toContain('<template #node="{ row, label }">');
    expect(source).toContain('resolveUiResourceTypeIcon(row?.Type)');
    expect(source).not.toContain('OAuthUiResourceTreeField');
  });

  it('keeps UiResources and rule tables in advanced manual mode', () => {
    const source = roleFormSource();

    expect(source).toContain('label="Advanced Mode"');
    expect(source).toContain('Record Rules (Manual Maintenance)');
    expect(source).toContain('Field Rules (Manual Maintenance)');
    expect(source).toContain('Method Access (Manual Maintenance)');
    expect(source).toContain('UI Resource Details (Manual Maintenance)');

    expect(source).toContain('prop="UiResources"');
    expect(source).toContain('UiResources.Mode');
    expect(source).toContain('UiResources.IrApplicationId');
    expect(source).toContain('UiResources.IrUiResourceId');
  });
});
