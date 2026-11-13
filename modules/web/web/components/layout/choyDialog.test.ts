// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import {
  ChoyDialog,
  ChoyDialogContent,
  ChoyDialogDescription,
  ChoyDialogFooter,
  ChoyDialogHeader,
  ChoyDialogTitle,
} from './choyDialog';
import { lockDialogDismiss } from './choyDialogDismiss.ts';
import {
  ChoySheet,
  ChoySheetClose,
  ChoySheetContent,
  ChoySheetDescription,
  ChoySheetHeader,
  ChoySheetTitle,
  ChoySheetTrigger,
} from './choySheet';

test('choyDialog barrel exports Dialog family constructors', () => {
  expect(ChoyDialog).toBeTruthy();
  expect(ChoyDialogContent).toBeTruthy();
  expect(ChoyDialogDescription).toBeTruthy();
  expect(ChoyDialogFooter).toBeTruthy();
  expect(ChoyDialogHeader).toBeTruthy();
  expect(ChoyDialogTitle).toBeTruthy();
});

test('lockDialogDismiss: prevents default only when not dismissible', () => {
  const blocked = { defaultPrevented: false, preventDefault() { this.defaultPrevented = true; } };
  lockDialogDismiss(blocked as unknown as Event, false);
  expect(blocked.defaultPrevented).toBe(true);

  const allowed = { defaultPrevented: false, preventDefault() { this.defaultPrevented = true; } };
  lockDialogDismiss(allowed as unknown as Event, true);
  expect(allowed.defaultPrevented).toBe(false);
});

test('choySheet barrel re-exports vendor Sheet SFC constructors', () => {
  expect(ChoySheet).toBeTruthy();
  expect(ChoySheetContent).toBeTruthy();
  expect(ChoySheetDescription).toBeTruthy();
  expect(ChoySheetHeader).toBeTruthy();
  expect(ChoySheetTitle).toBeTruthy();
  expect(ChoySheetTrigger).toBeTruthy();
  expect(ChoySheetClose).toBeTruthy();
});
