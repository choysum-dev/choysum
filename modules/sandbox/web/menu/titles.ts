// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { createTranslate } from '@/web/web/i18n';

const { _lt } = createTranslate('sandbox', { scope: 'web/menu/titles' });

export const sandboxRootMenuTitle = _lt('Sandbox');
export const sandboxRecordsMenuTitle = _lt('Field Kitchen Sink');
export const sandboxConfigurationMenuTitle = _lt('Configuration');
export const sandboxRecordPropertiesMenuTitle = _lt('Record Properties');
