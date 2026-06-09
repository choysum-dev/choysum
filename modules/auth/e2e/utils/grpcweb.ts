// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { expect, type Page, type Response } from '@playwright/test';

/**
 * Options for waiting on a successful gRPC-Web unary response.
 */
export type GrpcWebOkOptions = {
  timeoutMs?: number;
};

/**
 * Wait for a unary gRPC-Web call to complete with grpc-status=0.
 */
export async function waitForGrpcWebUnaryOk(page: Page, fullMethod: string, opts: GrpcWebOkOptions = {}): Promise<Response> {
  const timeoutMs = typeof opts.timeoutMs === 'number' ? opts.timeoutMs : 30_000;

  const res = await page.waitForResponse(
    r => {
      const url = r.url();
      if (!url.includes(fullMethod)) return false;
      const req = r.request();
      if (req.method() !== 'POST') return false;
      const ct = String(req.headers()['content-type'] || '').toLowerCase();
      return ct.startsWith('application/grpc-web');
    },
    { timeout: timeoutMs }
  );

  expect(res.status(), `HTTP status for ${fullMethod}`).toBe(200);

  // Strict mode: enforce gRPC status=0.
  // Note: grpc-status is typically a *trailer* in gRPC-Web and may not be visible via Response.headers().
  // We therefore parse the gRPC-Web body trailer frame (flag 0x80).
  const headers = res.headers();
  const headerGrpcStatus = headers['grpc-status'];
  const headerGrpcMessage = headers['grpc-message'];
  if (typeof headerGrpcStatus === 'string' && headerGrpcStatus !== '') {
    const msg = decodeGrpcMessage(typeof headerGrpcMessage === 'string' ? headerGrpcMessage : '');
    expect(headerGrpcStatus, formatGrpcAssertMessage(fullMethod, msg)).toBe('0');
    return res;
  }

  const body = await res.body();
  const trailerText = extractGrpcWebTrailerText(body);
  expect(trailerText, `missing grpc-web trailer frame for ${fullMethod}`).toBeTruthy();

  const trailerHeaders = parseTrailerHeaders(String(trailerText));
  const grpcStatus = trailerHeaders['grpc-status'];
  const grpcMessage = trailerHeaders['grpc-message'];
  const msg = decodeGrpcMessage(grpcMessage);

  expect(grpcStatus, `missing grpc-status trailer for ${fullMethod}${msg ? ` (grpc-message=${msg})` : ''}`).toBeTruthy();
  expect(String(grpcStatus), formatGrpcAssertMessage(fullMethod, msg)).toBe('0');

  return res;
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
function extractGrpcWebTrailerText(body: Buffer): string {
  // gRPC-Web framing: 1 byte flags + 4 bytes length (big-endian) + payload.
  // Trailer frame is indicated by MSB flag 0x80; payload is ASCII header block.
  if (!body || body.length < 5) return '';

  let offset = 0;
  while (offset + 5 <= body.length) {
    const flags = body.readUInt8(offset);
    const len = body.readUInt32BE(offset + 1);
    offset += 5;
    if (offset + len > body.length) break;
    const payload = body.subarray(offset, offset + len);
    offset += len;

    if ((flags & 0x80) !== 0) {
      return payload.toString('utf8');
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
