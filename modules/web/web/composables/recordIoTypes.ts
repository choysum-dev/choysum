// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

export type PageIoMenuItem = {
  key: 'import' | 'export' | string;
  label: string;
  disabled?: boolean;
  hidden?: boolean;
  onClick: () => void;
};

/** List/kanban page IO capability declaration. */
export type RecordIoConfig = {
  model: string;
  import?: {
    enabled: boolean;
    uploadHint?: string;
    columnMapping?: Record<string, string>;
  };
  export?: {
    enabled: boolean;
  };
};
