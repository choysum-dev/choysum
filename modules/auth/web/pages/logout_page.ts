// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { ChoysumError } from '../error';

/** Map a logout failure to the card header message. */
export function formatLogoutError(err: unknown, unknownMessage: string): string {
  if (err instanceof ChoysumError) return err.message;
  if (err instanceof Error) return err.message;
  return unknownMessage;
}

/** Clear a logout auto-redirect interval when one is running. */
export function stopLogoutRedirectTimer(
  timer: ReturnType<typeof setInterval> | undefined
): undefined {
  if (timer) clearInterval(timer);
  return undefined;
}

/** Advance the signed-out countdown; `done` means redirect now. */
export function nextLogoutCountdown(countdown: number): { countdown: number; done: boolean } {
  const next = countdown - 1;
  return { countdown: next, done: next <= 0 };
}
