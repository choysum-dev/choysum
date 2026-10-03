// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { ChoyDialog, ChoyDialogContent, ChoyDialogTitle } from './choyDialog';
import { ChoySheet, ChoySheetContent } from './choySheet';

test('choyDialog barrel re-exports vendor Dialog SFC constructors', () => {
  expect(ChoyDialog).toBeTruthy();
  expect(ChoyDialogContent).toBeTruthy();
  expect(ChoyDialogTitle).toBeTruthy();
});

test('choySheet barrel re-exports vendor Sheet SFC constructors', () => {
  expect(ChoySheet).toBeTruthy();
  expect(ChoySheetContent).toBeTruthy();
});
