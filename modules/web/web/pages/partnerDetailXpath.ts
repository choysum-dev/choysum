// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * Partner-style IMD extension contract (PR7 dogfood).
 * Keep in sync with DogfoodPartnerFormXpath.vue and cutover partner_bank expr.
 */
export const PARTNER_DETAIL_TAB_PANELS_ANCHOR = 'partner.detail.tab-panels';

/** Preferred public xpath for inserting ChoyTab panes into partner detail. */
export const PARTNER_DETAIL_TAB_PANELS_XPATH = `//*[@data-anchor='${PARTNER_DETAIL_TAB_PANELS_ANCHOR}']`;
