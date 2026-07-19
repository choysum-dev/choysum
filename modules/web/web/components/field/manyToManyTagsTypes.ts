// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

export type TagClickPayload<T = any> = {
  id: string;
  item: Partial<T>;
  label: string;
  source: 'display';
  event: MouseEvent | KeyboardEvent;
};
