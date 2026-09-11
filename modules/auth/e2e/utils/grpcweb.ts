// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { expect, type Page, type E2EResponse } from '@choysum/e2e';

/**
 * Options for waiting on a successful gRPC-Web unary response.
 */
export type GrpcWebOkOptions = {
  timeoutMs?: number;
};

/**
 * Parsed unary gRPC-Web result.
 */
export type GrpcWebUnaryResult = {
  response: E2EResponse;
  grpcStatus: string;
  grpcMessage: string;
};

type GrpcWebStatus = {
  status: string;
  message: string;
};

type AnyPage = Page & {
  __choysum_e2e_page__?: boolean;
  waitForResponse: (...args: any[]) => Promise<any>;
};

async function waitForGrpcWebUnaryResponse(page: AnyPage, fullMethod: string, timeoutMs: number): Promise<E2EResponse> {
  const res: any = await page.waitForResponse(
    { urlIncludes: fullMethod, method: 'POST', contentTypePrefix: 'application/grpc-web' },
    { timeout: timeoutMs }
  );

  const status = typeof res.status === 'function' ? res.status() : Number(res.status);
  expect(status, `HTTP status for ${fullMethod}`).toBe(200);
  return res as E2EResponse;
}

async function readGrpcWebStatus(res: E2EResponse, fullMethod: string): Promise<GrpcWebStatus> {
  // Strict mode: enforce gRPC status=0.
  // Note: grpc-status is typically a *trailer* in gRPC-Web and may not be visible via Response.headers().
  // We therefore parse the gRPC-Web body trailer frame (flag 0x80).
  const headers = res.headers();
  const headerGrpcStatus = headers['grpc-status'];
  const headerGrpcMessage = headers['grpc-message'];
  if (typeof headerGrpcStatus === 'string' && headerGrpcStatus !== '') {
    const msg = decodeGrpcMessage(typeof headerGrpcMessage === 'string' ? headerGrpcMessage : '');
    return { status: headerGrpcStatus, message: msg };
  }

  const body = await res.body();
  const bytes = body instanceof Uint8Array ? body : new Uint8Array(body as ArrayBuffer);
  const trailerText = extractGrpcWebTrailerText(bytes);
  expect(trailerText, `missing grpc-web trailer frame for ${fullMethod}`).toBeTruthy();

  const trailerHeaders = parseTrailerHeaders(String(trailerText));
  const grpcStatus = trailerHeaders['grpc-status'];
  const grpcMessage = trailerHeaders['grpc-message'];
  const msg = decodeGrpcMessage(grpcMessage);

  expect(grpcStatus, `missing grpc-status trailer for ${fullMethod}${msg ? ` (grpc-message=${msg})` : ''}`).toBeTruthy();
  return { status: String(grpcStatus), message: msg };
}

/**
 * Wait for one unary gRPC-Web call and return its parsed grpc-status/message.
 */
export async function waitForGrpcWebUnary(page: Page, fullMethod: string, opts: GrpcWebOkOptions = {}): Promise<GrpcWebUnaryResult> {
  const timeoutMs = typeof opts.timeoutMs === 'number' ? opts.timeoutMs : 30_000;
  const response = await waitForGrpcWebUnaryResponse(page as AnyPage, fullMethod, timeoutMs);
  const grpc = await readGrpcWebStatus(response, fullMethod);
  return {
    response,
    grpcStatus: grpc.status,
    grpcMessage: grpc.message,
  };
}

/**
 * Wait for a unary gRPC-Web call to complete with grpc-status=0.
 */
export async function waitForGrpcWebUnaryOk(page: Page, fullMethod: string, opts: GrpcWebOkOptions = {}): Promise<E2EResponse> {
  const unary = await waitForGrpcWebUnary(page, fullMethod, opts);
  expect(unary.grpcStatus, formatGrpcAssertMessage(fullMethod, unary.grpcMessage)).toBe('0');
  return unary.response;
}

/**
 * Format the assertion message used for grpc-status checks.
 */
function formatGrpcAssertMessage(fullMethod: string, decodedGrpcMessage: string): string {
  return `grpc-status for ${fullMethod}${decodedGrpcMessage ? ` (grpc-message=${decodedGrpcMessage})` : ''}`;
}

/**
 * Extract the trailing header block from a gRPC-Web response body.
 */
function extractGrpcWebTrailerText(body: Uint8Array): string {
  // gRPC-Web framing: 1 byte flags + 4 bytes length (big-endian) + payload.
  // Trailer frame is indicated by MSB flag 0x80; payload is ASCII header block.
  if (!body || body.length < 5) return '';

  let offset = 0;
  while (offset + 5 <= body.length) {
    const flags = body[offset];
    const len = ((body[offset + 1] << 24) | (body[offset + 2] << 16) | (body[offset + 3] << 8) | body[offset + 4]) >>> 0;
    offset += 5;
    if (offset + len > body.length) break;
    const payload = body.subarray(offset, offset + len);
    offset += len;

    if ((flags & 0x80) !== 0) {
      if (typeof TextDecoder !== 'undefined') {
        return new TextDecoder('utf-8').decode(payload);
      }
      let s = '';
      for (let i = 0; i < payload.length; i++) s += String.fromCharCode(payload[i]);
      return s;
    }
  }
  return '';
}

/**
 * Parse a gRPC-Web trailer block into lowercase header keys.
 */
function parseTrailerHeaders(trailerBlock: string): Record<string, string> {
  const out: Record<string, string> = {};
  const lines = String(trailerBlock || '')
    .split(/\r?\n/)
    .map(s => s.trim())
    .filter(Boolean);
  for (const line of lines) {
    const idx = line.indexOf(':');
    if (idx <= 0) continue;
    const k = line.slice(0, idx).trim().toLowerCase();
    const v = line.slice(idx + 1).trim();
    if (!k) continue;
    out[k] = v;
  }
  return out;
}

/**
 * Decode a percent-encoded grpc-message value.
 */
function decodeGrpcMessage(v: any): string {
  const raw = String(v ?? '').trim();
  if (!raw) return '';
  try {
    // grpc-message uses percent-encoding.
    return decodeURIComponent(raw);
  } catch {
    return raw;
  }
}
