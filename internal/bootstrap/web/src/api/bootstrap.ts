// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

import { ConnectError, createClient } from '@connectrpc/connect';
import { createGrpcWebTransport } from '@connectrpc/connect-web';

import { InitializationState, Workspace } from '../gen/bootstrap/internal/bootstrap/proto/bootstrap_pb';
import type { Workspace_GetInitializationStatus_Resp, Workspace_Initialize_Req } from '../gen/bootstrap/internal/bootstrap/proto/bootstrap_pb';

export type InitializeWorkspaceInput = Pick<Workspace_Initialize_Req, 'adminUsername' | 'password' | 'idempotencyKey'>;

export interface InitializeWorkspaceResult {
  accepted: boolean;
  operationId: string;
  nextPollAfterMs: bigint;
  state: InitializationState;
}

export interface BootstrapAPIError extends Error {
  bootstrapCode?: string;
  bootstrapDetails?: string;
}

export type InitializationStatus = Pick<
  Workspace_GetInitializationStatus_Resp,
  'operationId' | 'state' | 'stage' | 'progressPercent' | 'readyForLogin' | 'redirectUrl' | 'errorCode' | 'errorMessage' | 'nextPollAfterMs'
>;

const workspaceClient = createClient(
  Workspace,
  createGrpcWebTransport({
    baseUrl: '/',
  })
);

function parseBootstrapErrorDetail(detail: string): { code: string; details: string } | null {
  const trimmed = detail.trim();
  if (trimmed === '') {
    return null;
  }
  const matched = /^(BOOTSTRAP_[A-Z0-9_]+):\s*(.+)$/.exec(trimmed);
  if (!matched) {
    return null;
  }
  return {
    code: matched[1].trim(),
    details: matched[2].trim(),
  };
}

function normalizeBootstrapAPIError(error: unknown): Error {
  if (error instanceof ConnectError) {
    const detail = error.rawMessage?.trim() || error.message || 'bootstrap rpc failed';
    const normalized = new Error(`RPC ${error.code}: ${detail}`) as BootstrapAPIError;
    normalized.name = 'BootstrapAPIError';
    const parsed = parseBootstrapErrorDetail(detail);
    if (parsed) {
      normalized.bootstrapCode = parsed.code;
      normalized.bootstrapDetails = parsed.details;
    }
    return normalized;
  }
  if (error instanceof Error) {
    return error;
  }
  return new Error('unknown bootstrap rpc error');
}

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

async function hashPasswordForBootstrap(password: string, username: string): Promise<string> {
  const prefixMarker = '$CH$';
  if (password.startsWith(prefixMarker)) {
    return password;
  }

  const encoder = new TextEncoder();
  const data = encoder.encode(`${password}:${username}`);

  if (!globalThis.isSecureContext || !globalThis.crypto?.subtle) {
    return `${prefixMarker}${sha256HexFallback(`${password}:${username}`)}`;
  }

  const hashBuffer = await crypto.subtle.digest('SHA-256', data);
  const hashArray = new Uint8Array(hashBuffer);
  const hashHex = Array.from(hashArray, x => x.toString(16).padStart(2, '0')).join('');
  return `${prefixMarker}${hashHex}`;
}

export async function initializeWorkspace(input: InitializeWorkspaceInput): Promise<InitializeWorkspaceResult> {
  try {
    const adminUsername = input.adminUsername.trim();
    const password = await hashPasswordForBootstrap(input.password, adminUsername);

    const resp = await workspaceClient.initialize({
      adminUsername,
      password,
      idempotencyKey: input.idempotencyKey,
    });

    return {
      accepted: resp.accepted,
      operationId: resp.operationId,
      nextPollAfterMs: resp.nextPollAfterMs,
      state: resp.state,
    };
  } catch (error) {
    throw normalizeBootstrapAPIError(error);
  }
}

export async function getInitializationStatus(operationId: string): Promise<InitializationStatus> {
  try {
    const resp = await workspaceClient.getInitializationStatus({
      operationId: operationId.trim(),
    });

    return {
      operationId: resp.operationId,
      state: resp.state,
      stage: resp.stage,
      progressPercent: resp.progressPercent,
      readyForLogin: resp.readyForLogin,
      redirectUrl: resp.redirectUrl,
      errorCode: resp.errorCode,
      errorMessage: resp.errorMessage,
      nextPollAfterMs: resp.nextPollAfterMs,
    };
  } catch (error) {
    throw normalizeBootstrapAPIError(error);
  }
}
