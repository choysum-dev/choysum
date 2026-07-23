// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import type { CSRFProvider, RequestLifecycleProvider, TokenProvider } from '../../rpc/types';

let tokenProvider: TokenProvider | undefined;
let csrfProvider: CSRFProvider | undefined;
let lifecycleProvider: RequestLifecycleProvider | undefined;

export function setTokenProvider(provider: TokenProvider | (() => TokenProvider) | null | undefined): void {
  if (provider == null) {
    tokenProvider = undefined;
    return;
  }
  tokenProvider = typeof provider === 'function' ? provider() : provider;
}

export function setCSRFProvider(provider: CSRFProvider | (() => CSRFProvider) | null | undefined): void {
  if (provider == null) {
    csrfProvider = undefined;
    return;
  }
  csrfProvider = typeof provider === 'function' ? provider() : provider;
}

export function setLifecycleProvider(provider: RequestLifecycleProvider | (() => RequestLifecycleProvider) | null | undefined): void {
  if (provider == null) {
    lifecycleProvider = undefined;
    return;
  }
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
