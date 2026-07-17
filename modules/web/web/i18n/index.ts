// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

export {
  createTranslate,
  getGlobalComposer,
  notifyComposerMessagesChanged,
  trackComposerMessageRevision,
  translateTerm,
} from './translate';
export type { ComposerLike, TranslateOptions, TextSource } from './translate';
export type { TextDescriptor } from '../../../core/service/i18n';
