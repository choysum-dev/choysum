// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { createTranslate } from '@/core/service/i18n';
import { createDomainNormalizationBridge } from '@/core/service/utils/domain_normalization_bridge';

const { _t } = createTranslate('partner');
const bridge = createDomainNormalizationBridge('partner', _t);

/** Throw a partner-domain InvalidArgument error. */
export const fail = bridge.fail;

/** Map a domain-agnostic normalization failure into partner-domain InvalidArgument. */
export const mapNormalizationToPartner = bridge.mapNormalizationError;

export const normalizeOptionalText = bridge.normalizeOptionalText;
export const assertRequiredText = bridge.normalizeRequiredText;
export const assertRequiredTranslatedText = bridge.normalizeRequiredTranslatedText;
export const normalizeOptionalTranslatedText = bridge.normalizeOptionalTranslatedText;
export const translatedTextHasValue = bridge.translatedTextHasValue;
export const assertNonNegativeInt = bridge.normalizeNonNegativeInt;
export const normalizeSequenceInt = bridge.normalizeSequenceInt;
