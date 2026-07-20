// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { createTranslate } from '@/core/service/i18n';

/** Base service terminology binder (module owner = base). */
const translate = createTranslate('base');
export const _t = translate._t;
export const _lt = translate._lt;
