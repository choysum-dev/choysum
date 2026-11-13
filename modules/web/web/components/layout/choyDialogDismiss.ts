// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/** Lock Reka dismiss when dismissible is false. */
export function lockDialogDismiss(event: Event, dismissible: boolean): void {
  if (!dismissible) event.preventDefault();
}
