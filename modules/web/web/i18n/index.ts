// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

export { createTranslate } from '@/core/service/i18n';
export type { TermReference } from '@/core/service/i18n';
export { installBrowserI18nBridge } from './browser_bridge';
export {
  getGlobalComposer,
  notifyComposerMessagesChanged,
  trackComposerMessageRevision,
  translateTerm,
} from './translate';
export type {
  ComposerLike,
  CreateTranslateOptions,
  TranslateOptions,
  TextSource,
} from './translate';
