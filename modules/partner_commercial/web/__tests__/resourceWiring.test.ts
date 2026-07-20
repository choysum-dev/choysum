// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it } from 'vitest';
import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';

/**
 * Loads a partner commercial web source file relative to the current test directory.
 */
function source(relativePath: string): string {
  return readFileSync(resolve(__dirname, '..', relativePath), 'utf8');
}

describe('partner_commercial resource wiring', () => {
  it('wires partner_commercial form extension with local action declarations', () => {
    const form = source('views/PartnerFormView.vue');

    expect(form).toContain("defineModelActions('partner.PartnerIdentifier', {");
    expect(form).toContain("entityTitle: _lt('Identifier')");
    expect(form).toContain('<OOneToManyKanbanField');
    expect(form).toContain(':editable="canEditIdentifiers()"');
    expect(form).toContain(':form-view="PartnerIdentifierFormView"');
  });
});
