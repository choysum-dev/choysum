// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

export type LocalTokenDeps = {
  randomUUID?: () => string;
  now?: () => number;
};

let localTokenCounter = 0;

export function nextLocalToken(prefix: string, deps: LocalTokenDeps = {}): string {
  const randomUUID =
    Object.prototype.hasOwnProperty.call(deps, 'randomUUID')
      ? deps.randomUUID
      : globalThis.crypto?.randomUUID?.bind(globalThis.crypto);
  if (randomUUID) {
    return `${prefix}:${randomUUID()}`;
  }

  const now = (deps.now ?? Date.now)().toString(36);
  localTokenCounter += 1;
  return `${prefix}:${now}:${localTokenCounter.toString(36)}`;
}

export function resetLocalTokenCounterForTest(): void {
  localTokenCounter = 0;
}
