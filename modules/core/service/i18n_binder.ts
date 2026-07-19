// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { createTranslate } from '@/core/service/i18n';

/**
 * Core module terminology binder (catalog owner = core).
 *
 * Kept separate from `./i18n/` (the shared createTranslate platform package)
 * so module-owned `_t('…')` call sites extract into modules/core/i18n/core.pot.
 */
const translate = createTranslate('core');
export const _t = translate._t;
