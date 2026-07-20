// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it } from 'vitest';
import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';

/**
 * Loads a partner web source file relative to the current test directory.
 */
function source(relativePath: string): string {
  return readFileSync(resolve(__dirname, '..', relativePath), 'utf8');
}

describe('partner resource wiring', () => {
  it('declares route actions for all partner routes', () => {
    const routes = source('route/routes.ts');

    expect(routes).toContain('actions: [');
    expect(routes).toContain("'partner.action.partner_create'");
    expect(routes).toContain("'partner.action.partner_edit'");
    expect(routes).toContain("'partner.action.partner_delete'");
    expect(routes).toContain("'partner.action.partner_copy'");
    expect(routes).toContain("'partner.action.partner_open_detail'");
  });

  it('wires list/form views with local action declarations', () => {
    const list = source('views/PartnerListView.vue');
    const form = source('views/PartnerFormView.vue');

    expect(list).toContain("defineAction('partner.action.partner_open_detail'");
    expect(list).toContain("defineModelActions('partner.Partner', {");
    expect(list).toContain("entityTitle: _lt('Partner')");
    expect(list).toContain(':action-ids="{ create: partnerActions.create, delete: partnerActions.delete }"');
    expect(list).toContain(':has-action="hasAction"');
    expect(list).toContain('hasAction(partnerOpenDetailAction)');

    expect(form).toContain("defineModelActions('partner.Partner', {");
    expect(form).toContain("entityTitle: _lt('Partner')");
    expect(form).toContain('create: partnerActions.create');
    expect(form).toContain('edit: partnerActions.edit');
    expect(form).toContain('copy: partnerActions.copy');
    expect(form).toContain('delete: partnerActions.delete');
    expect(form).toContain(':has-action="hasAction"');
  });
});
