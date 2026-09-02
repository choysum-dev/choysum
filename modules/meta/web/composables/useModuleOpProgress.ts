// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { onTips, subscribeModuleOp } from '@/core/web/tip';

/** Snapshot returned by MetaModule.GetOpStatus (authoritative). */
export type ModuleOpStatusSnapshot = {
  status: string;
  summary?: unknown;
  resultStatus?: 'SUCCEEDED' | 'FAILED';
  failureKind?: 'RETRYABLE' | 'NON_RETRYABLE' | 'NONE';
  reload_triggered?: boolean;
  reload_failed?: boolean;
  reload_web?: boolean;
  retryAfterMs?: number;
  errorDomain?: string;
  errorCode?: string;
};

export type ModuleOpProgressHooks = {
  fetchStatus: (jobId: string) => Promise<ModuleOpStatusSnapshot>;
  /** False when the progress dialog/session is closed. */
  isActive: () => boolean;
  onStatus: (status: ModuleOpStatusSnapshot) => void;
  onTerminal: (status: ModuleOpStatusSnapshot) => void;
  onTimeout: () => void;
  onTransientNetworkError?: () => void;
  onHardError?: (message: string) => void;
};

const TERMINAL = new Set(['succeeded', 'failed', 'cancelled']);
const MAX_DURATION_MS = 10 * 60 * 1000;
const TIP_DEBOUNCE_MS = 80;
const POLL_INITIAL_MS = 1000;
const POLL_MAX_MS = 5000;
const POLL_STEP_MS = 500;

function isTerminalStatus(status?: string): boolean {
  return TERMINAL.has(String(status || '').toLowerCase());
}

function isTransientNetworkError(message: string): boolean {
  return (
    message.includes('Failed to fetch') ||
    message.includes('NetworkError') ||
    message.includes('ERR_CONNECTION_REFUSED') ||
    message.includes('Load failed')
  );
}

function clampRetryDelay(retryAfterMs: number | undefined, fallbackMs: number): number {
  if (retryAfterMs == null || !Number.isFinite(retryAfterMs)) {
    return fallbackMs;
  }
  return Math.min(POLL_MAX_MS, Math.max(POLL_INITIAL_MS, retryAfterMs));
}

function errorMessage(error: unknown): string {
  return String((error as { message?: string })?.message || error || '').trim();
}

/**
 * C1 Meta module-op progress session: boot Unary + tip-driven refresh +
 * short-backoff poll only after tip stream ends/errors while still non-terminal.
 */
export function createModuleOpProgressSession(hooks: ModuleOpProgressHooks) {
  let tipController: AbortController | null = null;
  let pollTimer: ReturnType<typeof setTimeout> | undefined;
  let deadlineTimer: ReturnType<typeof setTimeout> | undefined;
  let sessionGeneration = 0;
  let reachedTerminal = false;
  let transientNotified = false;
  let timedOut = false;

  function clearPoll(): void {
    if (pollTimer != null) {
      clearTimeout(pollTimer);
      pollTimer = undefined;
    }
  }

  function clearDeadline(): void {
    if (deadlineTimer != null) {
      clearTimeout(deadlineTimer);
      deadlineTimer = undefined;
    }
  }

  function stop(): void {
    sessionGeneration += 1;
    tipController?.abort();
    tipController = null;
    clearPoll();
    clearDeadline();
  }

  function notifyTransient(): void {
    if (transientNotified) return;
    transientNotified = true;
    hooks.onTransientNetworkError?.();
  }

  function notifyHardError(error: unknown): void {
    hooks.onHardError?.(errorMessage(error) || 'Failed to get status');
  }

  function fireTimeout(generation: number): void {
    if (generation !== sessionGeneration || !hooks.isActive() || reachedTerminal || timedOut) {
      return;
    }
    timedOut = true;
    tipController?.abort();
    clearPoll();
    clearDeadline();
    hooks.onTimeout();
  }

  async function applyFetched(
    jobId: string,
    generation: number
  ): Promise<ModuleOpStatusSnapshot | null> {
    if (generation !== sessionGeneration || !hooks.isActive() || timedOut) {
      return null;
    }
    const status = await hooks.fetchStatus(jobId);
    if (generation !== sessionGeneration || !hooks.isActive() || timedOut) {
      return null;
    }
    hooks.onStatus(status);
    if (isTerminalStatus(status.status)) {
      reachedTerminal = true;
      tipController?.abort();
      clearPoll();
      clearDeadline();
      hooks.onTerminal(status);
      if (status.reload_web && typeof window !== 'undefined') {
        window.location.reload();
      }
    }
    return status;
  }

  function startPollFallback(jobId: string, generation: number, startAt: number): void {
    clearPoll();
    let intervalMs = POLL_INITIAL_MS;

    const tick = async () => {
      if (generation !== sessionGeneration || !hooks.isActive() || reachedTerminal || timedOut) {
        return;
      }
      let status: ModuleOpStatusSnapshot | null = null;
      try {
        status = await applyFetched(jobId, generation);
        if (status && isTerminalStatus(status.status)) {
          return;
        }
      } catch (error) {
        if (generation !== sessionGeneration || !hooks.isActive() || timedOut) {
          return;
        }
        try {
          const message = errorMessage(error);
          if (isTransientNetworkError(message)) {
            notifyTransient();
          } else {
            notifyHardError(error);
          }
        } catch {
          // Progress hooks must not break fallback polling.
        }
      }

      if (generation !== sessionGeneration || !hooks.isActive() || reachedTerminal || timedOut) {
        return;
      }
      if (Date.now() - startAt > MAX_DURATION_MS) {
        fireTimeout(generation);
        return;
      }

      const nextDelay = clampRetryDelay(status?.retryAfterMs, intervalMs);
      intervalMs = Math.min(POLL_MAX_MS, intervalMs + POLL_STEP_MS);
      pollTimer = setTimeout(() => {
        void tick();
      }, nextDelay);
    };

    void tick();
  }

  /**
   * Watches one job until terminal status, timeout, or stop().
   * Remains pending for the whole tip/fallback lifecycle (callers such as
   * ModuleKanbanView.submitOperation intentionally await that full session).
   */
  async function watch(jobId: string): Promise<void> {
    const id = String(jobId || '').trim();
    stop();
    if (!id) return;

    const generation = sessionGeneration;
    reachedTerminal = false;
    transientNotified = false;
    timedOut = false;
    const startAt = Date.now();

    try {
      const boot = await applyFetched(id, generation);
      if (generation !== sessionGeneration || !hooks.isActive() || timedOut) {
        return;
      }
      if (!boot || isTerminalStatus(boot.status)) {
        return;
      }
    } catch (error) {
      if (generation !== sessionGeneration || !hooks.isActive() || timedOut) {
        return;
      }
      try {
        const message = errorMessage(error);
        if (isTransientNetworkError(message)) {
          notifyTransient();
        } else {
          notifyHardError(error);
        }
      } catch {
        // Progress hooks must not break boot handling.
      }
      // Still try tip/fallback — boot failure alone is not terminal.
    }

    tipController = new AbortController();
    const signal = tipController.signal;
    let debounceTimer: ReturnType<typeof setTimeout> | undefined;
    let refreshChain: Promise<void> = Promise.resolve();

    deadlineTimer = setTimeout(() => {
      fireTimeout(generation);
    }, Math.max(0, MAX_DURATION_MS - (Date.now() - startAt)));

    const scheduleTipRefresh = () => {
      if (debounceTimer != null) {
        clearTimeout(debounceTimer);
      }
      debounceTimer = setTimeout(() => {
        refreshChain = refreshChain
          .then(async () => {
            if (signal.aborted || generation !== sessionGeneration || reachedTerminal || timedOut) {
              return;
            }
            try {
              await applyFetched(id, generation);
            } catch (error) {
              if (signal.aborted || generation !== sessionGeneration || timedOut) {
                return;
              }
              try {
                const message = errorMessage(error);
                if (isTransientNetworkError(message)) {
                  notifyTransient();
                } else {
                  notifyHardError(error);
                }
              } catch {
                // Progress hooks must not break tip refresh chaining.
              }
            }
          });
      }, TIP_DEBOUNCE_MS);
    };

    try {
      await onTips(
        subscribeModuleOp(id, signal),
        async () => {
          scheduleTipRefresh();
        },
        signal
      );
    } catch {
      // Stream error; fall through to short-backoff poll when still active.
    } finally {
      clearDeadline();
      if (debounceTimer != null) {
        clearTimeout(debounceTimer);
      }
      await refreshChain;
    }

    if (
      generation === sessionGeneration &&
      !timedOut &&
      !signal.aborted &&
      hooks.isActive() &&
      !reachedTerminal
    ) {
      startPollFallback(id, generation, startAt);
    }
  }

  return { watch, stop };
}
