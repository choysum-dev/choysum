// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * Partner-style IMD extension contract.
 * Keep in sync with cutover partner_bank expr and partner detail tab panels.
 */
export const PARTNER_DETAIL_TAB_PANELS_ANCHOR = 'partner.detail.tab-panels';

/** Preferred public xpath for inserting ChoyTab panes into partner detail. */
export const PARTNER_DETAIL_TAB_PANELS_XPATH = `//*[@data-anchor='${PARTNER_DETAIL_TAB_PANELS_ANCHOR}']`;
