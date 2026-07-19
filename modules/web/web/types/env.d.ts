// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * Environment variable interface that other modules can extend.
 */
declare module '*.vue' {
  const component: any;
  export default component;

  // NOTE: tsc (without Volar) cannot infer named exports from SFCs.
  // These declarations allow type-only imports used across the modules workspace.
  export type ViewMode = any;
  export type ViewContainer = any;
  export type SelectionExpose<T = any> = any;
  export type RowEventPayload<T = any> = any;
  export type ValueClickPayload<T = any> = any;
  export type TagClickPayload<T = any> = any;
}

interface ImportMetaEnv {
  /**
   * Base application URL, for example "http://localhost:8000".
   */
  readonly BASE_URL: string;

  /**
   * Current build mode, for example "production" or "development".
   */
  readonly MODE: string;

  /**
   * Whether this is a production build.
   */
  readonly PROD: boolean;

  /**
   * Whether this is a development build.
   */
  readonly DEV: boolean;

  /**
   * Whether this is a server-side rendering build.
   */
  readonly SSR: boolean;

  /**
   * Choysum application name.
   */
  readonly CHOYSUM_APP_NAME: string;

  /**
   * Choysum application version.
   */
  readonly CHOYSUM_APP_VERSION: string;

  /**
   * Whether maintenance mode is enabled.
   */
  readonly CHOYSUM_MAINTENANCE_MODE: string;
}

/**
 * Environment variable type declarations.
 * These values are injected during the build and accessed through import.meta.env.
 */
interface ImportMeta {
  readonly env: ImportMetaEnv;
}

/**
 * Static asset import type declarations.
 */
declare module '*.svg' {
  const content: string;
  export default content;
}

declare module '*.png' {
  const content: string;
  export default content;
}

declare module '*.jpg' {
  const content: string;
  export default content;
}

declare module '*.scss' {
  const content: { [className: string]: string };
  export default content;
}

declare module '*.css' {
  const content: { [className: string]: string };
  export default content;
}
