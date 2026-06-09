// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import type { ConstraintContext, ValidationIssue } from '../orm/metadata/constraint';
import type { KernelValidationRule } from '../orm/repository/validation';
import { ValidationEngine } from './validation';

/**
 * RuntimeValidationOptions configures which validation stages are executed.
 */
export type RuntimeValidationOptions = {
  includeKernel?: boolean;
  includePlatform?: boolean;
  includeConstraints?: boolean;
  kernelRules?: KernelValidationRule[];
  platformCreateWriteWhitelist?: string[];
  platformRejectUnknownFields?: boolean;
  onPlatformCreateWhitelistHit?: (fields: string[]) => void;
};

/**
 * Returns validation issues gathered from the configured runtime validation stages.
 */
export async function validateRuntimeIssues(ctx: ConstraintContext, options?: RuntimeValidationOptions): Promise<ValidationIssue[]> {
  return await ValidationEngine.validate(ctx, options);
}

/**
 * Runs runtime validation and throws when any error-level issue is detected.
 */
export async function validateRuntimeOrThrow(ctx: ConstraintContext, options?: RuntimeValidationOptions): Promise<void> {
  await ValidationEngine.validateOrThrow(ctx, options);
}
