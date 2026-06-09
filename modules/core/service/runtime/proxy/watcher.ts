// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { Dep } from './dep';

type WatcherStackItem = {
  ref: unknown;
  key?: string;
};

/**
 * Watcher lazily evaluates computed model fields and tracks dependency invalidation.
 */
export class Watcher<T = unknown> {
  private value: unknown;
  private dirty = true;
  private error: Error | null = null;
  private lastComputeTime = 0;
  private initialized = false; // Indicates whether the watcher has been initialized.

  /**
   * Tracks the current watcher evaluation stack to detect circular dependencies.
   */
  static evalStack: WatcherStackItem[] = [];

  constructor(
    private target: T,
    private getter: (options: { self: T }) => unknown,
    private key?: string,
    private onError?: (error: Error) => void
  ) {
    // Lazily initialize so the first access performs the computation.
  }

  /**
   * Returns the computed value, evaluating it on demand when dirty.
   */
  get(): unknown {
    if (this.dirty) {
      if (Watcher.evalStack.some(item => item.ref === this)) {
        const stack = [...Watcher.evalStack, { ref: this, key: this.key }].map(w => w.key).join(' -> ');
        throw new Error(`Detected circular computed property dependency: ${stack}`);
      }

      Watcher.evalStack.push({ ref: this, key: this.key });
      const startTime = Date.now();

      try {
        Dep.target = this;
        this.value = this.getter({ self: this.target });
        this.dirty = false;
        this.error = null;
        this.initialized = true;
      } catch (e: unknown) {
        console.error(e);
        const error = e instanceof Error ? e : new Error(String(e));
        this.error = error;
        this.onError?.(error);
        throw error;
      } finally {
        this.lastComputeTime = Date.now() - startTime;
        Watcher.evalStack.pop();
        Dep.target = null;
      }
    }
    return this.value;
  }

  /**
   * Returns the last computation error, if any.
   */
  getError(): Error | null {
    return this.error;
  }

  /**
   * Returns the duration of the most recent evaluation in milliseconds.
   */
  getComputeTime(): number {
    return this.lastComputeTime;
  }

  /**
   * Marks the watcher dirty so the next read recomputes its value.
   */
  update() {
    this.dirty = true;
  }

  /**
   * Reports whether the watcher has completed at least one evaluation.
   */
  isInitialized(): boolean {
    return this.initialized;
  }
}
