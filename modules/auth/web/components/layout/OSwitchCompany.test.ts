// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { syncCompanyDraftsFromJwt } from './o_switch_company_draft';

test('OSwitchCompany draft guard: ignores JWT sync while panel is open', () => {
  const applied: Array<{ active: string; enabled: string[] }> = [];
  syncCompanyDraftsFromJwt({
    panelVisible: true,
    activeCompanyId: 'c2',
    enabledCompanyIds: ['c2'],
    apply: (active, enabled) => applied.push({ active, enabled }),
  });
  expect(applied.length).toBe(0);
});

test('OSwitchCompany draft guard: syncs drafts from JWT when panel is closed', () => {
  const applied: Array<{ active: string; enabled: string[] }> = [];
  syncCompanyDraftsFromJwt({
    panelVisible: false,
    activeCompanyId: 'c2',
    enabledCompanyIds: ['c1', 'c2'],
    apply: (active, enabled) => applied.push({ active, enabled }),
  });
  expect(applied).toEqual([{ active: 'c2', enabled: ['c1', 'c2'] }]);
});
