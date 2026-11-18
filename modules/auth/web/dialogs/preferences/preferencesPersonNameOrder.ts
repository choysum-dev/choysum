// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import {
  personNameFieldOrder,
  type PersonNameFieldProp,
} from '@/core/service/utils/person_name';

/**
 * Preferences name-field order from the active UI Language.PersonNameFormat.
 * Falls back to FirstName | LastName when the template is missing/invalid.
 */
export function preferencesPersonNameFieldOrder(template?: string | null): PersonNameFieldProp[] {
  return personNameFieldOrder(template);
}
