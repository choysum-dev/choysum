// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

export type PageIoMenuItem = {
  key: 'import' | 'export' | string;
  label: string;
  disabled?: boolean;
  hidden?: boolean;
  onClick: () => void;
};

/** List/kanban page IO capability declaration. Model comes from the page/list store. */
export type RecordIoConfig = {
  import?: {
    enabled: boolean;
    uploadHint?: string;
    columnMapping?: Record<string, string>;
  };
  export?: {
    enabled: boolean;
  };
};
