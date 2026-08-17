// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { createTranslate } from '@/core/service/i18n';

/** Audit service terminology binder (module owner = audit). */
const translate = createTranslate('audit');
export const _t = translate._t;
export const _lt = translate._lt;
