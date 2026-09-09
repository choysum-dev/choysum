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

export type Page = {
  __choysum_e2e_page__?: true;
  goto(url: string, opts?: GotoOptions): Promise<void>;
  locator(selector: string): Locator;
  getByPlaceholder(reOrString: RegExp | string): Locator;
  getByText(text: string): Locator;
  click(selector: string): Promise<void>;
  fill(selector: string, value: string): Promise<void>;
  evaluate(fnOrSource: Function | string, arg?: unknown): Promise<unknown>;
  waitForFunction(fnOrSource: Function | string, arg?: unknown, opts?: TimeoutOptions): Promise<void>;
  waitForResponse(match: ResponseMatch | string | ((r: any) => boolean), opts?: TimeoutOptions): Promise<E2EResponse>;
  url(): Promise<string> | string;
  screenshot(opts?: { path?: string } | string): Promise<void>;
};

export type E2EExpect = {
  toBeVisible(opts?: TimeoutOptions): Promise<void>;
  toBeEnabled(opts?: TimeoutOptions): Promise<void>;
  toHaveCount(n: number, opts?: TimeoutOptions): Promise<void>;
  toHaveURL(reOrString: RegExp | string, opts?: TimeoutOptions): Promise<void>;
};

export declare const test: typeof globalThis extends { test: infer T } ? T : (name: string, fn: () => unknown) => void;
export declare function expect(target: Locator | Page | unknown): E2EExpect | unknown;
export declare const page: Page;
export declare const runtime: E2ERuntime;
export declare function randomUUID(): string;
