// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

/**
 * Loose ambient for legacy Playwright e2e specs still partitioned onto Node.
 * Shaped so `@playwright/test`.Page stays assignable to `@choysum/e2e`.Page
 * (shared login/grpcweb helpers) under strict noImplicitAny.
 */
declare module '@playwright/test' {
  // Keep this structurally compatible with `@choysum/e2e` Locator (required
  // click/fill/first) while allowing extra Playwright locator APIs used in specs.
  export type Locator = {
    __choysum_e2e_locator__?: true;
    click(...args: any[]): Promise<any>;
    fill(value: string, ...args: any[]): Promise<any>;
    first(): Locator;
    locator(...args: any[]): Locator;
    getByRole(...args: any[]): Locator;
    getByText(...args: any[]): Locator;
    getByPlaceholder(...args: any[]): Locator;
    getByTestId(...args: any[]): Locator;
    filter(...args: any[]): Locator;
    nth(...args: any[]): Locator;
    count(...args: any[]): Promise<number>;
    allTextContents(): Promise<string[]>;
    [key: string]: any;
  };

  // Required methods mirror `@choysum/e2e`.Page; index signature covers PW-only APIs.
  export type Page = {
    __choysum_e2e_page__?: boolean;
    goto(...args: any[]): Promise<any>;
    locator(...args: any[]): Locator;
    getByPlaceholder(...args: any[]): Locator;
    getByText(...args: any[]): Locator;
    getByRole(...args: any[]): Locator;
    getByTestId(...args: any[]): Locator;
    click(...args: any[]): Promise<any>;
    fill(...args: any[]): Promise<any>;
    evaluate(...args: any[]): Promise<any>;
    waitForFunction(...args: any[]): Promise<any>;
    waitForResponse(predicate: (response: any) => any, options?: any): Promise<any>;
    waitForResponse(...args: any[]): Promise<any>;
    waitForSelector(...args: any[]): Promise<any>;
    waitForTimeout(...args: any[]): Promise<any>;
    waitForEvent(...args: any[]): Promise<any>;
    url(...args: any[]): any;
    screenshot(...args: any[]): Promise<any>;
    on(event: string, handler: (...args: any[]) => any): any;
    once(event: string, handler: (...args: any[]) => any): any;
    [key: string]: any;
  };

  export type Browser = any;
  export type BrowserContext = any;
  export type APIRequestContext = any;
  export type TestInfo = any;

  type PlaywrightFixtures = {
    page: Page;
    context: BrowserContext;
    browser: Browser;
    request: APIRequestContext;
  };

  type PlaywrightTestFn = (fixtures: PlaywrightFixtures, testInfo?: TestInfo) => any;

  type PlaywrightExpect = {
    (actual: any, message?: string): any;
    soft(actual: any, message?: string): any;
    poll(fn: (...args: any[]) => any, options?: any): any;
    arrayContaining(expected: any[]): any;
    objectContaining(expected: Record<string, any>): any;
  };

  interface PlaywrightTest {
    (name: string, fn: PlaywrightTestFn): void;
    (name: string, details: any, fn: PlaywrightTestFn): void;
    describe: {
      (name: string, fn: () => void): void;
      skip(name: string, fn: () => void): void;
      only(name: string, fn: () => void): void;
      [key: string]: any;
    };
    beforeEach(fn: PlaywrightTestFn | (() => any)): void;
    afterEach(fn: PlaywrightTestFn | (() => any)): void;
    beforeAll(fn: PlaywrightTestFn | (() => any)): void;
    afterAll(fn: PlaywrightTestFn | (() => any)): void;
    setTimeout(timeout: number): void;
    skip(...args: any[]): any;
    only: PlaywrightTest;
    fixme: PlaywrightTest;
    [key: string]: any;
  }

  export const test: PlaywrightTest;
  export const expect: PlaywrightExpect;
}
