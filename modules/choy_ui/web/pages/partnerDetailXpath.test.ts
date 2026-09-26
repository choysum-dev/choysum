// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import {
  PARTNER_DETAIL_TAB_PANELS_ANCHOR,
  PARTNER_DETAIL_TAB_PANELS_XPATH,
} from './partnerDetailXpath';

test('partner detail tab-panels xpath uses pointed data-anchor', () => {
  expect(PARTNER_DETAIL_TAB_PANELS_ANCHOR).toBe('partner.detail.tab-panels');
  expect(PARTNER_DETAIL_TAB_PANELS_XPATH).toBe(
    "//*[@data-anchor='partner.detail.tab-panels']",
  );
  // Must not target legacy el-tabs / data-slot contracts.
  expect(PARTNER_DETAIL_TAB_PANELS_XPATH.includes('el-tabs')).toBe(false);
  expect(PARTNER_DETAIL_TAB_PANELS_XPATH.includes('data-slot')).toBe(false);
});
