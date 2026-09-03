// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * Partner service model exports.
 *
 * Layout exemplar: field/constraint/compute logic on the model class; at most one
 * `_partner_bridge.ts` bypass. Prefer this shape for new domain models.
 */
export { default as Partner } from './partner';
export { default as PartnerContact } from './partner_contact';
