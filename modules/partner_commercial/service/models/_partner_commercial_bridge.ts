// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { createTranslate } from '@/core/service/i18n';
import { createDomainNormalizationBridge } from '@/core/service/utils/domain_normalization_bridge';

const { _t } = createTranslate('partner_commercial');
const bridge = createDomainNormalizationBridge('partner_commercial', _t);

/** Throw a partner-commercial-domain InvalidArgument error. */
export const fail = bridge.fail;

/** Map a domain-agnostic normalization failure into partner-commercial-domain InvalidArgument. */
export const mapNormalizationToPartnerCommercial = bridge.mapNormalizationError;

export const normalizeOptionalRefId = bridge.normalizeOptionalRefId;
export const normalizeOptionalText = bridge.normalizeOptionalText;
export const normalizeOptionalTranslatedText = bridge.normalizeOptionalTranslatedText;
export const assertRequiredText = bridge.assertRequiredText;
export const assertDateOrUndefined = bridge.assertDateOrUndefined;
