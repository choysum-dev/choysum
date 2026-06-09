// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

// Backend TS test runtime (QuickJS) globals injected by `choysum test`.
// See pkg/jsengine/scripts/choysumtest/choysumtest.js
declare global {
  type ChoysumTestOptions = {
    pattern?: string;
    failFast?: boolean;
  };

  type ChoysumTestErrorInfo = {
    name?: string;
    message: string;
    stack?: string;
  };

  type ChoysumTestCaseResult = {
    name: string;
    ok: boolean;
    durationMs: number;
    error: ChoysumTestErrorInfo | null;
  };

  type ChoysumTestReport = {
    total: number;
    passed: number;
    failed: number;
    cases: ChoysumTestCaseResult[];
  };

  function test(name: string, fn: () => void | Promise<void>): void;

  type ChoysumPropertyPath = string | number | Array<string | number>;

  interface ChoysumExpectation<T> {
    readonly not: ChoysumExpectation<T>;
    toBe(expected: T): void;
    toEqual(expected: unknown): void;
    toMatchObject(expected: unknown): void;
    toBeTruthy(): void;
    toBeFalsy(): void;
    toBeUndefined(): void;
    toBeDefined(): void;
    toBeNull(): void;
    toMatch(expected: string | RegExp): void;
    toHaveLength(expected: number): void;
    toContain(expected: unknown): void;
    toHaveProperty(path: ChoysumPropertyPath, expected?: unknown): void;
    toBeGreaterThan(expected: number): void;
    toBeGreaterThanOrEqual(expected: number): void;
    toBeLessThan(expected: number): void;
    toBeLessThanOrEqual(expected: number): void;
    toThrow(expected?: string | RegExp): void;
  }

  function expect<T = unknown>(received: T): ChoysumExpectation<T>;
  function expectRejects(received: Promise<unknown> | (() => Promise<unknown>), expected?: string | RegExp): Promise<void>;

  // Runner is injected by the init script and used by the generated tests entry.
  // eslint-disable-next-line @typescript-eslint/naming-convention
  var __choysum_test_run__: (options?: ChoysumTestOptions) => Promise<ChoysumTestReport>;
}

export {};
