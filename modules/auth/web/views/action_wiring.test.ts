// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it } from 'vitest';
import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';

function viewSource(fileName: string): string {
  return readFileSync(resolve(__dirname, fileName), 'utf8');
}

describe('auth view action wiring', () => {
  it('wires User list/form views to permission-aware action ids', () => {
    const list = viewSource('UserListView.vue');
    const form = viewSource('UserFormView.vue');

    expect(list).toContain("createTranslate('auth'");
    expect(list).toContain("defineModelActions('auth.User', {");
    expect(list).toContain("entityTitle: _lt('User')");
    expect(list).toContain(':action-ids="{ create: userActions.create, delete: userActions.delete }"');
    expect(list).toContain(':has-action="hasAction"');
    expect(list).toContain('usePermission');

    expect(form).toContain("createTranslate('auth'");
    expect(form).toContain("defineModelActions('auth.User', {");
    expect(form).toContain("entityTitle: _lt('User')");
    expect(form).toContain(':action-ids="{ create: userActions.create, edit: userActions.edit, copy: userActions.copy, delete: userActions.delete }"');
    expect(form).toContain(':has-action="hasAction"');
    expect(form).toContain('usePermission');
  });

  it('wires Role list/form views to permission-aware action ids', () => {
    const list = viewSource('RoleListView.vue');
    const form = viewSource('RoleFormView.vue');

    expect(list).toContain("createTranslate('auth'");
    expect(list).toContain("defineModelActions('auth.Role', {");
    expect(list).toContain("entityTitle: _lt('Role')");
    expect(list).toContain(':action-ids="{ create: roleActions.create, delete: roleActions.delete }"');
    expect(list).toContain(':has-action="hasAction"');
    expect(list).toContain('usePermission');

    expect(form).toContain("createTranslate('auth'");
    expect(form).toContain("defineModelActions('auth.Role', {");
    expect(form).toContain("entityTitle: _lt('Role')");
    expect(form).toContain(':action-ids="{ create: roleActions.create, edit: roleActions.edit, copy: roleActions.copy, delete: roleActions.delete }"');
    expect(form).toContain(':has-action="hasAction"');
    expect(form).toContain('prop="AccessUiResourceIds"');
    expect(form).toContain('OManyToManyRefTreeField');
    expect(form).not.toContain('OAuthUiResourceTreeField');
    expect(form).toContain(':label="_t(\'Advanced Mode\')"');
    expect(form).toContain('prop="UiResources"');
    expect(form).toContain('UiResources.Mode');
    expect(form).toContain('UiResources.IrApplicationId');
    expect(form).toContain('UiResources.IrUiResourceId');
    expect(form).toContain('RecordRules.Kind');
    expect(form).toContain('MethodAccesses.IrApplicationId');
    expect(form).not.toContain('Manual Maintenance');
    expect(form).toContain('usePermission');
  });

  it('wires Session list/form views to permission-aware action ids', () => {
    const list = viewSource('SessionListView.vue');
    const form = viewSource('SessionFormView.vue');

    expect(list).toContain("createTranslate('auth'");
    expect(list).toContain("defineModelActions('auth.Session', {");
    expect(list).toContain("entityTitle: _lt('Session')");
    expect(list).toContain(':action-ids="{ create: sessionActions.create, delete: sessionActions.delete }"');
    expect(list).toContain(':has-action="hasAction"');
    expect(list).toContain('usePermission');

    expect(form).toContain("createTranslate('auth'");
    expect(form).toContain("defineModelActions('auth.Session', {");
    expect(form).toContain("entityTitle: _lt('Session')");
    expect(form).toContain(
      ':action-ids="{ create: sessionActions.create, edit: sessionActions.edit, copy: sessionActions.copy, delete: sessionActions.delete }"'
    );
    expect(form).toContain(':has-action="hasAction"');
    expect(form).toContain('usePermission');
  });

  it('wires Token list/form views to permission-aware action ids', () => {
    const list = viewSource('TokenListView.vue');
    const form = viewSource('TokenFormView.vue');

    expect(list).toContain("createTranslate('auth'");
    expect(list).toContain("defineModelActions('auth.Token', {");
    expect(list).toContain("entityTitle: _lt('Token')");
    expect(list).toContain(':action-ids="{ create: tokenActions.create, delete: tokenActions.delete }"');
    expect(list).toContain(':has-action="hasAction"');
    expect(list).toContain("v-action=\"['auth.action.token_edit', 'auth.action.token_copy']\"");
    expect(list).toContain("v-action.disable.and=\"['auth.action.token_edit', 'auth.action.token_delete']\"");
    expect(list).toContain('usePermission');

    expect(form).toContain("createTranslate('auth'");
    expect(form).toContain("defineModelActions('auth.Token', {");
    expect(form).toContain("entityTitle: _lt('Token')");
    expect(form).toContain(':action-ids="{ create: tokenActions.create, edit: tokenActions.edit, copy: tokenActions.copy, delete: tokenActions.delete }"');
    expect(form).toContain(':has-action="hasAction"');
    expect(form).toContain('usePermission');
  });
});
