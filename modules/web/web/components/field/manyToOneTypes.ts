// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

export type ValueClickPayload<T = any> = {
  id: string;
  item: Partial<T> | null;
  label: string;
  source: 'display';
  event: MouseEvent | KeyboardEvent;
};
