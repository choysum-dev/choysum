// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

// Augment async-validator RuleItem with trigger used by Choy form validation.
import 'async-validator';

declare module 'async-validator' {
  interface RuleItem {
    /**
     * When to run this rule: on blur, on change, or both.
     * Matches ChoyField / form item trigger convention.
     */
    trigger?: 'blur' | 'change' | Array<'blur' | 'change'>;
  }
}
