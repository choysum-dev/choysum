// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import type { ModelMetadata } from '../../metadata';
import { ValidationPipelineError, type ConstraintMode, type ValidationIssue } from '../../metadata';
import { asObjectRecord } from '@/core/utils/object';
import { wrapRepositoryValidationError } from './error_helpers';

function normalizeSqlColumns(raw: string): string[] {
  return Array.from(
    new Set(
      raw
        .split(',')
        .map(col => col.trim())
        .filter(Boolean)
        .map(col => {
          const parts = col
            .split('.')
            .map(p => p.trim())
            .filter(Boolean);
          return parts[parts.length - 1] || col;
        })
    )
  );
}

function parseSqlValidationIssue(error: unknown): ValidationIssue | undefined {
  const collectTexts = (err: unknown, depth: number = 0): string[] => {
    if (!err || depth > 4) return [];

    const errRecord = asObjectRecord(err);
    const texts: string[] = [];
    if (err instanceof Error) {
      const msg = String(err.message || '').trim();
      if (msg) texts.push(msg);

      const str = String(err.toString?.() || '').trim();
      if (str) texts.push(str);
    } else {
      const str = String(err).trim();
      if (str) texts.push(str);
    }

    const cause = errRecord?.cause;
    if (cause) texts.push(...collectTexts(cause, depth + 1));

    const metadata = asObjectRecord(errRecord?.metadata);
    if (metadata) {
      for (const value of Object.values(metadata)) {
        const s = String(value || '').trim();
        if (s) texts.push(s);
      }
    }

    return texts;
  };

  const candidates = Array.from(new Set(collectTexts(error))).filter(Boolean);
  if (candidates.length === 0) return undefined;

  const raw = candidates.join('\n');

  const sqliteUnique = /UNIQUE\s+constraint\s+failed:\s*([^\n]+)/i.exec(raw);
  if (sqliteUnique?.[1]) {
    const columns = normalizeSqlColumns(sqliteUnique[1]);
    return {
      scope: 'sql',
      field: columns[0],
      code: 'sql_unique_violation',
      message: raw,
      severity: 'error',
      meta: {
        sqlColumns: columns,
        sqlField: columns[0],
      },
    };
  }

  const pgUnique = /duplicate\s+key\s+value\s+violates\s+unique\s+constraint\s+"([^"]+)"/i.exec(raw);
  if (pgUnique?.[1]) {
    return {
      scope: 'sql',
      code: 'sql_unique_violation',
      message: raw,
      severity: 'error',
      meta: {
        sqlConstraint: pgUnique[1],
      },
    };
  }

  const sqliteCheck = /CHECK\s+constraint\s+failed(?::\s*([^\n]+))?/i.exec(raw);
  if (sqliteCheck) {
    const constraint = String(sqliteCheck[1] || '').trim();
    return {
      scope: 'sql',
      code: 'sql_check_violation',
      message: raw,
      severity: 'error',
      meta: constraint
        ? {
            sqlConstraint: constraint,
          }
        : undefined,
    };
  }

  const pgCheck = /violates\s+check\s+constraint\s+"([^"]+)"/i.exec(raw);
  if (pgCheck?.[1]) {
    return {
      scope: 'sql',
      code: 'sql_check_violation',
      message: raw,
      severity: 'error',
      meta: {
        sqlConstraint: pgCheck[1],
      },
    };
  }

  if (/FOREIGN\s+KEY\s+constraint\s+failed/i.test(raw)) {
    return {
      scope: 'sql',
      code: 'sql_fk_violation',
      message: raw,
      severity: 'error',
    };
  }

  const pgFk = /violates\s+foreign\s+key\s+constraint\s+"([^"]+)"/i.exec(raw);
  if (pgFk?.[1]) {
    return {
      scope: 'sql',
      code: 'sql_fk_violation',
      message: raw,
      severity: 'error',
      meta: {
        sqlConstraint: pgFk[1],
      },
    };
  }

  return undefined;
}

export function throwRepositorySqlWriteError(meta: ModelMetadata, error: unknown, mode: ConstraintMode): never {
  const issue = parseSqlValidationIssue(error);
  if (!issue) {
    throw error;
  }

  const pipelineError = new ValidationPipelineError('sql constraint failed', [issue]);
  throw wrapRepositoryValidationError(meta, pipelineError, mode);
}
