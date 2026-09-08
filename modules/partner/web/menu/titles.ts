// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { createTranslate } from '@/web/web/i18n';

const { _lt } = createTranslate('partner', { scope: 'web/menu/menus' });

/** Root and leaf menu TermReferences shared by menus.ts and FE unit tests. */
export const partnerRootMenuTitle = _lt('Partner Management');
export const partnerListMenuTitle = _lt('Partner List');
