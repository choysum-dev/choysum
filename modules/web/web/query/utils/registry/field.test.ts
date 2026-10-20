// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import {
  clearFieldsByStore,
  exportFieldSelection,
  pathsToFieldSelection,
  registerFieldPath,
  unregisterFieldPath,
} from './field';

test('exportFieldSelection keeps bare CurrencyId when nested currency companions are registered', () => {
  const storeId = 'sandbox.SandboxRecord:currency-sel';
  clearFieldsByStore(storeId);

  // Monetary field registers CurrencyId + CurrencyId.DecimalDigits / DisplayName / …
  // ManyToOneRef field also registers CurrencyId and CurrencyId.DisplayName.
  for (const path of [
    'CurrencyId',
    'CurrencyId.DisplayName',
    'CurrencyId.DecimalDigits',
    'CurrencyId.Symbol',
    'CurrencyId.Code',
    'Amount',
  ]) {
    registerFieldPath(storeId, path);
  }

  const paths = exportFieldSelection(storeId).filter(p => p !== 'Id').sort();
  expect(paths).toEqual(
    [
      'Amount',
      'CurrencyId',
      'CurrencyId.Code',
      'CurrencyId.DecimalDigits',
      'CurrencyId.DisplayName',
      'CurrencyId.Symbol',
    ].sort()
  );

  const sel = pathsToFieldSelection(paths)!;
  expect(sel).toContain('CurrencyId');
  expect(sel).toContain('Amount');
  expect(sel.some(x => x && typeof x === 'object' && 'CurrencyId' in (x as object))).toBe(true);

  clearFieldsByStore(storeId);
});

test('exportFieldSelection unregister of nested path does not drop bare parent registration', () => {
  const storeId = 'sandbox.SandboxRecord:currency-unreg';
  clearFieldsByStore(storeId);

  registerFieldPath(storeId, 'CurrencyId');
  registerFieldPath(storeId, 'CurrencyId.DisplayName');
  unregisterFieldPath(storeId, 'CurrencyId.DisplayName');

  expect(exportFieldSelection(storeId).filter(p => p !== 'Id')).toEqual(['CurrencyId']);

  clearFieldsByStore(storeId);
});
