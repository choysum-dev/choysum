// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

/**
 * Browser / QuickJS-friendly TypeScript System stubs for language-core codegen.
 * Avoids Node fs APIs; createParsedCommandLineByJson only needs a few methods.
 */

export type MinimalSys = {
  args: string[];
  newLine: string;
  useCaseSensitiveFileNames: boolean;
  write: (s: string) => void;
  writeOutputIsTTY?: () => boolean;
  readFile: (path: string) => string | undefined;
  writeFile: (path: string, data: string) => void;
  resolvePath: (path: string) => string;
  fileExists: (path: string) => boolean;
  directoryExists: (path: string) => boolean;
  createDirectory: (path: string) => void;
  getExecutingFilePath: () => string;
  getCurrentDirectory: () => string;
  getDirectories: (path: string) => string[];
  readDirectory: (
    path: string,
    extensions?: readonly string[],
    exclude?: readonly string[],
    include?: readonly string[],
    depth?: number,
  ) => string[];
  exit: (exitCode?: number) => void;
};

/** Fixed minimal System used for Vue language-core service-script codegen. */
export function createMinimalSys(currentDirectory = "/"): MinimalSys {
  const cwd = currentDirectory || "/";
  return {
    args: [],
    newLine: "\n",
    useCaseSensitiveFileNames: true,
    write() {},
    writeOutputIsTTY: () => false,
    readFile: () => undefined,
    writeFile() {},
    resolvePath: (p) => p,
    fileExists: () => false,
    directoryExists: () => true,
    createDirectory() {},
    getExecutingFilePath: () => "/typescript.js",
    getCurrentDirectory: () => cwd,
    getDirectories: () => [],
    readDirectory: () => [],
    exit() {},
  };
}
