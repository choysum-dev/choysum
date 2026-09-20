// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later
/**
 * Flat ESLint config for modules/*/service (BT-8d / HC10).
 *
 * Production TypeScript under modules/<app>/service forbids explicit `any`.
 * Test files keep `any` allowed (no zero-KPI for tests). Remaining `: any`
 * parameters are separate cleanup debt; the CI DoD for assertions is
 * count_as_any.py (not this ESLint config — typecheck jobs do not run ESLint).
 *
 * Local run (Node 22+; installs eslint + typescript-eslint packages ephemerally):
 *   npx --yes -p eslint@9 -p @typescript-eslint/parser -p @typescript-eslint/eslint-plugin \\
 *     eslint --config eslint.config.mjs 'modules/*/service/**/*.{ts,tsx}'
 *
 * CI typecheck jobs do not run ESLint (no Node on that path). Production
 * as-any DoD is gated by: python3 scripts/dev/count_as_any.py --max 324
 */
import tseslint from '@typescript-eslint/eslint-plugin';
import tsparser from '@typescript-eslint/parser';

/** @type {import('eslint').Linter.Config[]} */
export default [
  {
    files: ['modules/*/service/**/*.{ts,tsx}'],
    ignores: ['modules/*/service/**/*.test.{ts,tsx}', 'modules/*/service/**/tests/**'],
    languageOptions: {
      parser: tsparser,
      parserOptions: {
        ecmaVersion: 'latest',
        sourceType: 'module',
      },
    },
    plugins: {
      '@typescript-eslint': tseslint,
    },
    rules: {
      '@typescript-eslint/no-explicit-any': 'error',
    },
  },
  {
    files: ['modules/*/service/**/*.test.{ts,tsx}', 'modules/*/service/**/tests/**/*.{ts,tsx}'],
    languageOptions: {
      parser: tsparser,
      parserOptions: {
        ecmaVersion: 'latest',
        sourceType: 'module',
      },
    },
    plugins: {
      '@typescript-eslint': tseslint,
    },
    rules: {
      '@typescript-eslint/no-explicit-any': 'off',
    },
  },
];
