// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { REL_ALIAS_PREFIX } from '../../infra/database/plugin/constants';

const RELATION_ALIAS_CANDIDATES_CACHE = new Map<string, string[]>();

function toSnakeCase(value: string): string {
  return value
    .replace(/([A-Z])/g, '_$1')
    .replace(/^_+/, '')
    .toLowerCase();
}

export { REL_ALIAS_PREFIX };

export function buildRelationAliasCandidates(fieldName: string): string[] {
  const cached = RELATION_ALIAS_CANDIDATES_CACHE.get(fieldName);
  if (cached) return cached;

  const lowerCamel = fieldName ? fieldName.charAt(0).toLowerCase() + fieldName.slice(1) : fieldName;
  const snakeCase = toSnakeCase(fieldName);
  const candidates = [
    `${REL_ALIAS_PREFIX}${fieldName}`,
    `${REL_ALIAS_PREFIX}${lowerCamel}`,
    `${REL_ALIAS_PREFIX}${snakeCase}`,
    `${REL_ALIAS_PREFIX}_${snakeCase}`,
  ];

  RELATION_ALIAS_CANDIDATES_CACHE.set(fieldName, candidates);
  return candidates;
}