// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

export type CompilerFs = {
  fileExists(path: string): boolean;
  readFile(path: string): string | undefined;
};

export function compilerFs(): CompilerFs {
  return (globalThis as unknown as { compilerFs: CompilerFs }).compilerFs;
}
