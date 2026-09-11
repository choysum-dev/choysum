// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

/** Runtime JSON injected by the e2e host (fields match Go runtimeInfo). */
export type E2ERuntime = {
  pid?: number;
  port?: number;
  baseURL: string;
  configPath?: string;
  dbPath?: string;
  runDir?: string;
  module?: string;
  scenario?: string;
  specsDir?: string;
  fixtures?: string[];
  ci?: boolean;
};

export type WaitUntilState = 'load' | 'domcontentloaded';

export type GotoOptions = {
  waitUntil?: WaitUntilState;
};

export type ReloadOptions = {
  waitUntil?: WaitUntilState;
};

export type TimeoutOptions = {
  timeout?: number;
};

export type WaitForSelectorOptions = TimeoutOptions & {
  state?: 'attached' | 'visible' | 'hidden';
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

export type LocatorOptions = {
  hasText?: string | RegExp;
};

export type GetByRoleOptions = {
  name?: string | RegExp;
  exact?: boolean;
};

export type Locator = {
  __choysum_e2e_locator__?: true;
  click(): Promise<void>;
  fill(value: string): Promise<void>;
  press(key: string): Promise<void>;
  waitFor(opts?: WaitForSelectorOptions): Promise<void>;
  first(): Locator;
  last(): Locator;
  nth(index: number): Locator;
  locator(selector: string, options?: LocatorOptions): Locator;
  getByRole(...args: any[]): Locator;
  count(): Promise<number>;
  textContent(): Promise<string | null>;
  innerText(): Promise<string | null>;
  getAttribute(name: string): Promise<string | null>;
  isEnabled(): Promise<boolean>;
  isVisible(): Promise<boolean>;
  allTextContents(): Promise<string[]>;
};

/**
 * Page surface used by the QuickJS e2e host (`@choysum/e2e`).
 * Locator-returning methods stay typed; async CDP seams stay loose for shared utils.
 */
export type RouteFulfillOptions = {
  status?: number;
  contentType?: string;
  body?: string;
  headers?: Record<string, string>;
};

export type Route = {
  request(): { url(): string; method(): string };
  fulfill(options?: RouteFulfillOptions): Promise<void>;
  continue(): Promise<void>;
  abort(): Promise<void>;
};

export type Page = {
  /** Set by the QJS host page proxy. */
  __choysum_e2e_page__?: boolean;
  goto(...args: any[]): Promise<any>;
  reload(...args: any[]): Promise<any>;
  locator(...args: any[]): Locator;
  getByPlaceholder(...args: any[]): Locator;
  getByText(...args: any[]): Locator;
  getByTestId(...args: any[]): Locator;
  getByRole(...args: any[]): Locator;
  click(...args: any[]): Promise<any>;
  fill(...args: any[]): Promise<any>;
  evaluate(...args: any[]): Promise<any>;
  waitForFunction(...args: any[]): Promise<any>;
  waitForResponse(...args: any[]): Promise<any>;
  waitForTimeout(...args: any[]): Promise<any>;
  waitForURL(...args: any[]): Promise<any>;
  waitForSelector(...args: any[]): Promise<any>;
  route(url: string, handler: (route: Route) => unknown | Promise<unknown>): Promise<void>;
  unroute(url?: string, handler?: (route: Route) => unknown | Promise<unknown>): Promise<void>;
  url(...args: any[]): any;
  screenshot(...args: any[]): Promise<any>;
};

export type E2EExpect = {
  toBeVisible(opts?: TimeoutOptions): Promise<void>;
  toBeEnabled(opts?: TimeoutOptions): Promise<void>;
  toBeChecked(opts?: TimeoutOptions): Promise<void>;
  toHaveCount(n: number, opts?: TimeoutOptions): Promise<void>;
  toHaveURL(reOrString: RegExp | string, opts?: TimeoutOptions): Promise<void>;
  toHaveText(expected: string | RegExp, opts?: TimeoutOptions): Promise<void>;
  toHaveClass(expected: string | RegExp, opts?: TimeoutOptions): Promise<void>;
  not: {
    toHaveText(expected: string | RegExp, opts?: TimeoutOptions): Promise<void>;
    toHaveClass(expected: string | RegExp, opts?: TimeoutOptions): Promise<void>;
  };
};

export type PollMatchers = {
  toBe(expected: unknown): Promise<void>;
  toBeGreaterThan(expected: number): Promise<void>;
  toBeGreaterThanOrEqual(expected: number): Promise<void>;
  toMatch(expected: string | RegExp): Promise<void>;
  not: Omit<PollMatchers, 'not'>;
};

export type PollOptions = {
  timeout?: number;
};

/** Value assertions used by shared utils. */
export type ValueExpect = {
  readonly not: ValueExpect;
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

type E2ETestFn = {
  (name: string, fn: () => unknown): void;
  setTimeout(ms: number): void;
  skip(condition: unknown, message?: string): void;
};

export declare const test: E2ETestFn;
export declare function expect(target: Locator, message?: string): E2EExpect;
export declare function expect(target: Page, message?: string): E2EExpect;
export declare function expect(target: unknown, message?: string): ValueExpect;
export declare namespace expect {
  function poll(fn: () => unknown | Promise<unknown>, opts?: PollOptions): PollMatchers;
}
export declare const page: Page;
export declare const runtime: E2ERuntime;
export declare function randomUUID(): string;
export declare function readTextFile(path: string): Promise<string>;
