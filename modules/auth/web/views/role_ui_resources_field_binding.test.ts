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
    expect(source).toContain(':label="_t(\'Accessible UI Resources\')"');
    expect(source).toContain(':lazy="false"');
    expect(source).toContain(':max-depth="0"');
    expect(source).toContain('children-field="Childs"');
    expect(source).toContain(':root-condition="{');
    expect(source).toContain("['Type', '=', 'MENU']");
    expect(source).toContain("['ParentId', 'is', null]");
    expect(source).toContain(":fields=\"['Type', 'Requires']\"");
    expect(source).toContain(':check-strictly="false"');
    expect(source).toContain('<template #node="{ row, label }">');
    expect(source).toContain('resolveUiResourceTypeIcon(row?.Type)');
    expect(source).toContain('inspectUiResource(row)');
    expect(source).toContain('Requires → derived Method RPCs');
    expect(source).toContain('UI-Option-A');
    expect(source).toContain('Primary path: check resources in this tree');
    expect(source).toContain('normalizeUiResourceRequires');
    expect(source).not.toContain('OAuthUiResourceTreeField');
  });

  it('keeps Advanced Mode as this-role data/RPC grant surface with Kind and app scope', () => {
    const source = roleFormSource();

    expect(source).toContain(':label="_t(\'Advanced Mode\')"');
    expect(source).toContain("Record Rules'");
    expect(source).toContain("Field Rules'");
    expect(source).toContain("Method Access'");
    expect(source).toContain("UI Resource Details (manual bypass)'");
    expect(source).not.toContain('Manual Maintenance');

    expect(source).toContain('RecordRules.Kind');
    expect(source).not.toContain('RecordRules.RoleId');
    expect(source).not.toContain('Applies to Role (empty = all users)');
    expect(source).toContain('This form only edits rules for this role');
    expect(source).toContain('dedicated menus');
    expect(source).toContain('RecordRules.IrApplicationId');
    expect(source).toContain('FieldRules.IrApplicationId');
    expect(source).toContain('MethodAccesses.IrApplicationId');
    expect(source).toContain(':default-record="defaultRecordRule"');
    expect(source).toContain(':default-record="defaultMethodAccess"');
    expect(source).toContain("Mode: 'allow'");
    expect(source).toContain("Kind: 'grant'");
    expect(source).toContain('scope-global');

    expect(source).toContain('prop="UiResources"');
    expect(source).toContain('UiResources.Mode');
    expect(source).toContain('UiResources.IrApplicationId');
    expect(source).toContain('UiResources.IrUiResourceId');
  });
});
