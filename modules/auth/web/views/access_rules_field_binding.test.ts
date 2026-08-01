// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it } from 'vitest';
import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';

function viewSource(fileName: string): string {
  return readFileSync(resolve(__dirname, fileName), 'utf8');
}

describe('Access Rules admin field binding (PR-C-5)', () => {
  it('exposes nullable RoleId and Kind on standalone RecordRule form', () => {
    const form = viewSource('RoleRecordRuleFormView.vue');
    const hints = viewSource('RoleRecordRuleAudienceHints.vue');

    expect(form).toContain('prop="RoleId"');
    expect(form).toContain('prop="Kind"');
    expect(form).toContain('prop="MetaApplicationId"');
    expect(form).toContain('prop="MetaModelId"');
    expect(form).toContain('prop="Condition"');
    expect(form).toContain('RoleRecordRuleAudienceHints');
    expect(hints).toContain('Wide-open grant for all users');
    expect(hints).toContain('Audience and scope are separate');
    expect(hints).toContain('isGrantEveryoneWarning');
  });

  it('requires RoleId on Field / Method / UI grant forms', () => {
    for (const file of ['RoleFieldRuleFormView.vue', 'RoleMethodAccessFormView.vue', 'RoleUiResourceFormView.vue']) {
      const form = viewSource(file);
      expect(form).toContain('prop="RoleId"');
      expect(form).toContain('RoleListView');
      expect(form).toContain('Select Role');
    }
  });
});
