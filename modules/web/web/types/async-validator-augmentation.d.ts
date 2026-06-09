// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

// Augment async-validator RuleItem with the trigger field used by Element Plus.
import 'async-validator';

declare module 'async-validator' {
  interface RuleItem {
    /**
     * Trigger mode used by the Element Plus extension.
     * Accepts 'blur', 'change', or an array of those values.
     */
    trigger?: 'blur' | 'change' | Array<'blur' | 'change'>;
  }
}
