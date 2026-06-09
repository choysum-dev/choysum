// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import type { CSRFProvider, RequestLifecycleProvider, TokenProvider } from '../../rpc/types';

let tokenProvider: TokenProvider | undefined;
let csrfProvider: CSRFProvider | undefined;
let lifecycleProvider: RequestLifecycleProvider | undefined;

export function setTokenProvider(provider: TokenProvider | (() => TokenProvider)): void {
  tokenProvider = typeof provider === 'function' ? provider() : provider;
}

export function setCSRFProvider(provider: CSRFProvider | (() => CSRFProvider)): void {
  csrfProvider = typeof provider === 'function' ? provider() : provider;
}

export function setLifecycleProvider(provider: RequestLifecycleProvider | (() => RequestLifecycleProvider)): void {
  lifecycleProvider = typeof provider === 'function' ? provider() : provider;
}

export function getTokenProvider(): TokenProvider | undefined {
  return tokenProvider;
}

export function getCSRFProvider(): CSRFProvider | undefined {
  return csrfProvider;
}

export function getLifecycleProvider(): RequestLifecycleProvider | undefined {
  return lifecycleProvider;
}
