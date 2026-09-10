// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

/** Runtime JSON injected by the e2e host (fields match Go runtimeInfo). */
export type E2ERuntime = {
  pid?: number;
  port?: number;
  baseURL: string;
  configPath?: string;
  dbPath?: string;
  module?: string;
  scenario?: string;
  specsDir?: string;
  fixtures?: string[];
};

export type WaitUntilState = 'load' | 'domcontentloaded';

export type GotoOptions = {
  waitUntil?: WaitUntilState;
};

export type TimeoutOptions = {
  timeout?: number;
};

export type ResponseMatch = {
  urlIncludes?: string;
  method?: string;
  contentTypePrefix?: string;
};

export type E2EResponse = {
  status(): number;
  headers(): Record<string, string>;
  url(): string;
  body(): Promise<Uint8Array>;
};

export type Locator = {
  __choysum_e2e_locator__?: true;
  click(): Promise<void>;
  fill(value: string): Promise<void>;
  first(): Locator;
};

/**
 * Dual-run page surface shared by the QJS host and Playwright.
 * Locator-returning methods stay typed; async CDP/PW seams stay loose so
 * Playwright's Page remains assignable for shared utils (login/grpcweb).
 */
export type Page = {
  /** Set by the QJS host page proxy; absent on Playwright pages. */
  __choysum_e2e_page__?: boolean;
  goto(...args: any[]): Promise<any>;
  // Rest args stay loose so Playwright Page (options object, roles, etc.) stays assignable.
  locator(...args: any[]): Locator;
  getByPlaceholder(...args: any[]): Locator;
  getByText(...args: any[]): Locator;
  click(...args: any[]): Promise<any>;
  fill(...args: any[]): Promise<any>;
  evaluate(...args: any[]): Promise<any>;
  waitForFunction(...args: any[]): Promise<any>;
  waitForResponse(...args: any[]): Promise<any>;
  url(...args: any[]): any;
  screenshot(...args: any[]): Promise<any>;
};

export type E2EExpect = {
  toBeVisible(opts?: TimeoutOptions): Promise<void>;
  toBeEnabled(opts?: TimeoutOptions): Promise<void>;
  toHaveCount(n: number, opts?: TimeoutOptions): Promise<void>;
  toHaveURL(reOrString: RegExp | string, opts?: TimeoutOptions): Promise<void>;
};

/** Value assertions used by shared utils (and PW shim → @playwright/test). */
export type ValueExpect = {
  toBe(expected: unknown): void;
  toEqual(expected: unknown): void;
  toBeTruthy(): void;
  toBeFalsy(): void;
  toBeNull(): void;
  toBeUndefined(): void;
  toBeDefined(): void;
  toContain(expected: unknown): void;
  toMatch(expected: string | RegExp): void;
  toHaveLength(expected: number): void;
};

export declare const test: typeof globalThis extends { test: infer T } ? T : (name: string, fn: () => unknown) => void;
export declare function expect(target: Locator, message?: string): E2EExpect;
export declare function expect(target: Page, message?: string): E2EExpect;
export declare function expect(target: unknown, message?: string): ValueExpect;
export declare const page: Page;
export declare const runtime: E2ERuntime;
export declare function randomUUID(): string;
