// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

export {
  wrapRepositoryValidationError,
  throwRepositorySqlWriteError,
  recordRepositoryPlatformCreateWhitelistAudit,
  resolveRepositoryPlatformCreateWriteWhitelist,
  resolveRepositoryPlatformRejectUnknownFields,
  validateRepositoryWrite,
} from './bridge';
export type { ValidationMode, KernelValidationRule, KernelIssueCode } from './kernel';
export {
  KernelValidationError,
  validateIntFields,
  validateSelectionFields,
  validateRequiredFields,
  validateRelationShapeFields,
  validateDecimalFields,
  validateFields,
} from './kernel';
