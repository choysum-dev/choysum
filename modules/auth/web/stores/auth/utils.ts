// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { isClient } from '@vueuse/core';
import { newAuthError, wrapAuthError, AuthErrCode, ChoysumError } from '../../error';

/**
 * Safely parse a JWT payload.
 */
export function parseJwt(token: string): TokenPayload {
  try {
    if (!token || typeof token !== 'string' || token.split('.').length !== 3) {
      throw newAuthError({
        code: AuthErrCode.TOKEN_INVALID,
        message: 'Invalid JWT token format',
      });
    }

    const base64Url = token.split('.')[1];
    if (!base64Url) {
      throw newAuthError({
        code: AuthErrCode.TOKEN_INVALID,
        message: 'JWT token is missing the payload segment',
      });
    }

    const base64 = base64Url.replace(/-/g, '+').replace(/_/g, '/');
    const jsonPayload = decodeURIComponent(
      atob(base64)
        .split('')
        .map(c => '%' + ('00' + c.charCodeAt(0).toString(16)).slice(-2))
        .join('')
    );

    return JSON.parse(jsonPayload);
  } catch (error) {
    throw wrapAuthError(error, {
      code: AuthErrCode.TOKEN_PARSING_FAILED,
      message: 'Failed to parse JWT token',
    });
  }
}

/**
 * Extract the auth identity payload from a JWT.
 */
export function extractIdentity(token: string): TokenIdentity {
  try {
    let payload: TokenPayload;

    try {
      payload = parseJwt(token);
    } catch (error) {
      throw wrapAuthError(error, {
        code: AuthErrCode.IDENTITY_EXTRACTION_FAILED,
        message: 'Failed to extract identity: token parsing failed',
      });
    }

    if (!payload.sub) {
      throw newAuthError({
        code: AuthErrCode.IDENTITY_EXTRACTION_FAILED,
        message: 'Token is missing the required subject identity (sub/user_id)',
      });
    }

    const metadata = payload?.meta;
    const safeMetadata = metadata && typeof metadata === 'object' ? (metadata as Record<string, any>) : {};

    return {
      userId: payload.sub || '',
      tokenId: payload.jti || '',
      metadata: safeMetadata,
    };
  } catch (error) {
    throw wrapAuthError(error, {
      code: AuthErrCode.IDENTITY_EXTRACTION_FAILED,
      message: 'Failed to extract user identity',
    });
  }
}

/**
 * Collect a JSON-encoded client device fingerprint.
 */
export function getDeviceInfo(): string {
  if (!isClient) {
    throw newAuthError({
      code: AuthErrCode.DEVICE_INFO_FAILED,
      message: 'Device information is only available in a client environment',
    });
  }

  try {
    return JSON.stringify({
      userAgent: navigator.userAgent,
      language: navigator.language,
      platform: navigator.platform,
      vendor: navigator.vendor,
      screen: {
        width: window.screen.width,
        height: window.screen.height,
      },
      timezone: Intl.DateTimeFormat().resolvedOptions().timeZone,
    });
  } catch (error) {
    throw wrapAuthError(error, {
      code: AuthErrCode.DEVICE_INFO_FAILED,
      message: 'Failed to collect device information',
    });
  }
}

/**
 * Compute a SHA-256 hex digest without relying on WebCrypto.
 */
function sha256HexFallback(input: string): string {
  const K = [
    0x428a2f98, 0x71374491, 0xb5c0fbcf, 0xe9b5dba5, 0x3956c25b, 0x59f111f1, 0x923f82a4, 0xab1c5ed5, 0xd807aa98, 0x12835b01, 0x243185be, 0x550c7dc3, 0x72be5d74,
    0x80deb1fe, 0x9bdc06a7, 0xc19bf174, 0xe49b69c1, 0xefbe4786, 0x0fc19dc6, 0x240ca1cc, 0x2de92c6f, 0x4a7484aa, 0x5cb0a9dc, 0x76f988da, 0x983e5152, 0xa831c66d,
    0xb00327c8, 0xbf597fc7, 0xc6e00bf3, 0xd5a79147, 0x06ca6351, 0x14292967, 0x27b70a85, 0x2e1b2138, 0x4d2c6dfc, 0x53380d13, 0x650a7354, 0x766a0abb, 0x81c2c92e,
    0x92722c85, 0xa2bfe8a1, 0xa81a664b, 0xc24b8b70, 0xc76c51a3, 0xd192e819, 0xd6990624, 0xf40e3585, 0x106aa070, 0x19a4c116, 0x1e376c08, 0x2748774c, 0x34b0bcb5,
    0x391c0cb3, 0x4ed8aa4a, 0x5b9cca4f, 0x682e6ff3, 0x748f82ee, 0x78a5636f, 0x84c87814, 0x8cc70208, 0x90befffa, 0xa4506ceb, 0xbef9a3f7, 0xc67178f2,
  ];
  const H = [0x6a09e667, 0xbb67ae85, 0x3c6ef372, 0xa54ff53a, 0x510e527f, 0x9b05688c, 0x1f83d9ab, 0x5be0cd19];

  const rotr = (x: number, n: number): number => (x >>> n) | (x << (32 - n));
  const ch = (x: number, y: number, z: number): number => (x & y) ^ (~x & z);
  const maj = (x: number, y: number, z: number): number => (x & y) ^ (x & z) ^ (y & z);
  const bigSigma0 = (x: number): number => rotr(x, 2) ^ rotr(x, 13) ^ rotr(x, 22);
  const bigSigma1 = (x: number): number => rotr(x, 6) ^ rotr(x, 11) ^ rotr(x, 25);
  const smallSigma0 = (x: number): number => rotr(x, 7) ^ rotr(x, 18) ^ (x >>> 3);
  const smallSigma1 = (x: number): number => rotr(x, 17) ^ rotr(x, 19) ^ (x >>> 10);

  const bytes = new TextEncoder().encode(input);
  const bitLen = bytes.length * 8;
  const padLen = (64 - ((bytes.length + 1 + 8) % 64)) % 64;
  const totalLen = bytes.length + 1 + padLen + 8;
  const data = new Uint8Array(totalLen);
  data.set(bytes, 0);
  data[bytes.length] = 0x80;

  const bitLenHi = Math.floor(bitLen / 0x100000000);
  const bitLenLo = bitLen >>> 0;
  const tail = totalLen - 8;
  data[tail] = (bitLenHi >>> 24) & 0xff;
  data[tail + 1] = (bitLenHi >>> 16) & 0xff;
  data[tail + 2] = (bitLenHi >>> 8) & 0xff;
  data[tail + 3] = bitLenHi & 0xff;
  data[tail + 4] = (bitLenLo >>> 24) & 0xff;
  data[tail + 5] = (bitLenLo >>> 16) & 0xff;
  data[tail + 6] = (bitLenLo >>> 8) & 0xff;
  data[tail + 7] = bitLenLo & 0xff;

  const w = new Uint32Array(64);

  for (let offset = 0; offset < data.length; offset += 64) {
    for (let i = 0; i < 16; i++) {
      const j = offset + i * 4;
      w[i] = ((data[j] << 24) | (data[j + 1] << 16) | (data[j + 2] << 8) | data[j + 3]) >>> 0;
    }
    for (let i = 16; i < 64; i++) {
      w[i] = (smallSigma1(w[i - 2]) + w[i - 7] + smallSigma0(w[i - 15]) + w[i - 16]) >>> 0;
    }

    let a = H[0];
    let b = H[1];
    let c = H[2];
    let d = H[3];
    let e = H[4];
    let f = H[5];
    let g = H[6];
    let h = H[7];

    for (let i = 0; i < 64; i++) {
      const t1 = (h + bigSigma1(e) + ch(e, f, g) + K[i] + w[i]) >>> 0;
      const t2 = (bigSigma0(a) + maj(a, b, c)) >>> 0;
      h = g;
      g = f;
      f = e;
      e = (d + t1) >>> 0;
      d = c;
      c = b;
      b = a;
      a = (t1 + t2) >>> 0;
    }

    H[0] = (H[0] + a) >>> 0;
    H[1] = (H[1] + b) >>> 0;
    H[2] = (H[2] + c) >>> 0;
    H[3] = (H[3] + d) >>> 0;
    H[4] = (H[4] + e) >>> 0;
    H[5] = (H[5] + f) >>> 0;
    H[6] = (H[6] + g) >>> 0;
    H[7] = (H[7] + h) >>> 0;
  }

  return H.map(v => v.toString(16).padStart(8, '0')).join('');
}

/**
 * Hash a password on the client before it is sent to the backend.
 */
export async function hashPasswordClient(password: string, username: string): Promise<string> {
  const prefixMarker = '$CH$';

  if (!isClient) return password;

  const dataToHash = `${password}:${username}`;

  // Browsers may disable WebCrypto in non-secure contexts, so keep a JS fallback.
  if (!globalThis.isSecureContext || !globalThis.crypto?.subtle) {
    console.warn('[auth] WebCrypto unavailable in current context, fallback to JS SHA-256.');
    return `${prefixMarker}${sha256HexFallback(dataToHash)}`;
  }

  try {
    // Prefer the native Web Crypto implementation when it is available.
    const encoder = new TextEncoder();
    const data = encoder.encode(dataToHash);
    const hashBuffer = await crypto.subtle.digest('SHA-256', data);

    // Convert the digest bytes into a hex string.
    const hashArray = new Uint8Array(hashBuffer);
    const hashHex = Array.prototype.map.call(hashArray, x => x.toString(16).padStart(2, '0')).join('');

    // Prefix the hash so the backend can detect client-side hashing.
    return `${prefixMarker}${hashHex}`;
  } catch (error) {
    // Some browsers expose crypto but still reject digest calls, so keep the fallback consistent.
    console.warn('[auth] Client password hashing failed, fallback to JS SHA-256.', error);
    return `${prefixMarker}${sha256HexFallback(dataToHash)}`;
  }
}

/**
 * Read the CSRF token from the browser cookie jar.
 */
export function getCsrfTokenFromCookie(): string | null {
  if (!isClient) {
    return null;
  }

  try {
    const match = document.cookie.match(new RegExp('(^|;\\s*)(XSRF-TOKEN)=([^;]*)'));
    if (!match) {
      return null;
    }
    return decodeURIComponent(match[3]);
  } catch (error) {
    throw wrapAuthError(error, {
      code: AuthErrCode.CSRF_TOKEN_FAILED,
      message: 'Failed to read CSRF token',
    });
  }
}

/**
 * Wrap an async function with loading-state bookkeeping.
 */
export function withLoading<T, Args extends unknown[]>(fn: (...args: Args) => Promise<T>, loadingRef: { value: boolean }): (...args: Args) => Promise<T> {
  return async (...args: Args): Promise<T> => {
    loadingRef.value = true;
    try {
      return await fn(...args);
    } finally {
      loadingRef.value = false;
    }
  };
}
