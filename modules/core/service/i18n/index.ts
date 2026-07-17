// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

export {
  createTranslate,
  withI18nScope,
  resolveI18nScope,
  formatScope,
  resolveRequestLang,
  createTermIdentity,
  createTextDescriptor,
  createTextDescriptorKey,
  isTextDescriptor,
  TEXT_DESCRIPTOR_NAMESPACE,
} from './translate';
export type {
  TranslateFn,
  LazyTranslate,
  TermIdentity,
  TextDescriptor,
  TranslateOptions,
} from './translate';
export type { ResolveI18nScopeOptions } from './scope';
export type { ResolveRequestLangFallbacks } from './request_lang';
