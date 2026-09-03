// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { createTranslate } from '@/core/service/i18n';
import { createDomainNormalizationBridge } from '@/core/service/utils/domain_normalization_bridge';

const { _t } = createTranslate('partner_bank');
const bridge = createDomainNormalizationBridge('partner_bank', _t);

/** Throw a partner-bank-domain InvalidArgument error. */
export const fail = bridge.fail;

/** Map a domain-agnostic normalization failure into partner-bank-domain InvalidArgument. */
export const mapNormalizationToPartnerBank = bridge.mapNormalizationError;

export const normalizeOptionalText = bridge.normalizeOptionalText;
export const assertRequiredText = bridge.normalizeRequiredText;
