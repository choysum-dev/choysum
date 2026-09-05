// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

// Minimal vite/client ambient for Choysum Go-native typecheck.
// Self-contained (no relative /// <reference>) so node_modules/vite is not required.

interface ImportMetaEnv {
  readonly BASE_URL: string;
  readonly MODE: string;
  readonly DEV: boolean;
  readonly PROD: boolean;
  readonly SSR: boolean;
  readonly [key: string]: any;
}

interface ImportMeta {
  readonly url: string;
  readonly env: ImportMetaEnv;
  readonly hot?: {
    readonly data: any;
    accept: (...args: any[]) => void;
    dispose: (cb: (...args: any[]) => void) => void;
    invalidate: (...args: any[]) => void;
  };
  glob: (pattern: string | string[], options?: Record<string, any>) => Record<string, any>;
}

declare module "*.module.css" {
  const classes: { readonly [key: string]: string };
  export default classes;
}
declare module "*.module.scss" {
  const classes: { readonly [key: string]: string };
  export default classes;
}
declare module "*.module.sass" {
  const classes: { readonly [key: string]: string };
  export default classes;
}

declare module "*.css" {
  const src: string;
  export default src;
}
declare module "*.scss" {
  const src: string;
  export default src;
}
declare module "*.sass" {
  const src: string;
  export default src;
}
declare module "*.less" {
  const src: string;
  export default src;
}

declare module "*.svg" {
  const src: string;
  export default src;
}
declare module "*.png" {
  const src: string;
  export default src;
}
declare module "*.jpg" {
  const src: string;
  export default src;
}
declare module "*.jpeg" {
  const src: string;
  export default src;
}
declare module "*.gif" {
  const src: string;
  export default src;
}
declare module "*.webp" {
  const src: string;
  export default src;
}
declare module "*.ico" {
  const src: string;
  export default src;
}
declare module "*.woff" {
  const src: string;
  export default src;
}
declare module "*.woff2" {
  const src: string;
  export default src;
}
declare module "*.ttf" {
  const src: string;
  export default src;
}
declare module "*.eot" {
  const src: string;
  export default src;
}
declare module "*.mp4" {
  const src: string;
  export default src;
}
declare module "*.webm" {
  const src: string;
  export default src;
}
declare module "*.json" {
  const value: any;
  export default value;
}

declare module "*?raw" {
  const content: string;
  export default content;
}

declare module "*?url" {
  const src: string;
  export default src;
}

declare module "vite/client" {}
