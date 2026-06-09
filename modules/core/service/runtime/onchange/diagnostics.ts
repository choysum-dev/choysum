// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { ENABLE_DIAGNOSTICS, DIAGNOSTICS_LEVEL } from './constants';
import type { OnchangeDiagnostics, OnchangeDiagnosticMessage, OnchangeMessage, Timer } from './types';

function now(): number {
  return typeof performance !== 'undefined' && performance.now ? performance.now() : Date.now();
}

/**
 * Starts an onchange timer handle.
 */
export function startTimer(): Timer {
  return { start: () => {}, stop: () => 0, elapsed: () => 0 };
}

/**
 * Stops a timer handle and returns elapsed milliseconds.
 */
export function endTimer(t: Timer): number {
  return t.stop();
}

/**
 * DiagnosticsBuilder incrementally collects onchange diagnostic fields.
 */
export class DiagnosticsBuilder {
  private missingCount = 0;
  private prefetchTimeMs = 0;
  private pathDepthMax = 1;
  private computeRecomputed: string[] = [];
  private readsRoots: string[] = [];
  private changedSeeds: string[] = [];
  private iterations = 0;
  private loopThreshold?: number;
  private cachedPlanUsed?: boolean;
  private messages: OnchangeDiagnosticMessage[] = [];

  /**
   * Records whether the path-plan cache was used.
   */
  enablePlanCache(flag: boolean) {
    this.cachedPlanUsed = flag;
    return this;
  }

  /**
   * Records the missing-field count.
   */
  setMissingCount(n: number) {
    this.missingCount = n;
    return this;
  }

  /**
   * Records prefetch latency in milliseconds.
   */
  setPrefetchTime(ms: number) {
    this.prefetchTimeMs = ms;
    return this;
  }

  /**
   * Records the maximum path depth touched by the run.
   */
  setPathDepthMax(d: number) {
    this.pathDepthMax = d;
    return this;
  }

  /**
   * Records recomputed compute-field names.
   */
  setComputeRecomputed(list: Iterable<string>) {
    this.computeRecomputed = Array.from(list);
    return this;
  }

  /**
   * Records the root fields that were read.
   */
  setReadsRoots(list: Iterable<string>) {
    this.readsRoots = Array.from(list);
    return this;
  }

  /**
   * Records the changed seed fields.
   */
  setChangedSeeds(list: Iterable<string>) {
    this.changedSeeds = Array.from(list);
    return this;
  }

  /**
   * Records the executed iteration count.
   */
  setIterations(i: number) {
    this.iterations = i;
    return this;
  }

  /**
   * Records the loop threshold used by the run.
   */
  setLoopThreshold(t?: number) {
    this.loopThreshold = t;
    return this;
  }

  /**
   * Appends one diagnostic message.
   */
  pushMessage(msg: OnchangeDiagnosticMessage) {
    this.messages.push(msg);
    return this;
  }

  /**
   * Builds the final diagnostics object.
   */
  build(): OnchangeDiagnostics {
    return {
      missingCount: this.missingCount,
      prefetchTimeMs: this.prefetchTimeMs,
      pathDepthMax: this.pathDepthMax,
      computeRecomputed: this.computeRecomputed,
      readsRoots: this.readsRoots,
      changedSeeds: this.changedSeeds,
      iterations: this.iterations,
      loopThreshold: this.loopThreshold,
      cachedPlanUsed: this.cachedPlanUsed,
      messages: this.messages,
    };
  }
}

/**
 * Attaches diagnostics to an onchange result payload.
 */
export function attachDiagnostics(result: { messages?: OnchangeMessage[]; diagnostics?: OnchangeDiagnostics }, diag: OnchangeDiagnostics) {
  if (!ENABLE_DIAGNOSTICS) return;

  // Ensure the messages array exists for defensive handling.
  if (!result.messages) {
    result.messages = [];
  }

  result.diagnostics = {
    missingCount: diag.missingCount,
    prefetchTimeMs: diag.prefetchTimeMs,
    pathDepthMax: diag.pathDepthMax,
    computeRecomputed: diag.computeRecomputed,
    readsRoots: diag.readsRoots,
    changedSeeds: diag.changedSeeds,
    iterations: diag.iterations,
    loopThreshold: diag.loopThreshold,
    cachedPlanUsed: diag.cachedPlanUsed,
    messages: DIAGNOSTICS_LEVEL === 'debug' ? diag.messages : [],
  };
}

/**
 * Builds a normalized diagnostics object from partial values.
 */
export function buildDiagnostics(params: Partial<OnchangeDiagnostics>): OnchangeDiagnostics {
  return {
    missingCount: params.missingCount ?? 0,
    prefetchTimeMs: params.prefetchTimeMs ?? 0,
    pathDepthMax: params.pathDepthMax ?? 1,
    computeRecomputed: params.computeRecomputed ?? [],
    readsRoots: params.readsRoots ?? [],
    changedSeeds: params.changedSeeds ?? [],
    iterations: params.iterations ?? 0,
    loopThreshold: params.loopThreshold,
    cachedPlanUsed: params.cachedPlanUsed,
    messages: params.messages ?? [],
  };
}
