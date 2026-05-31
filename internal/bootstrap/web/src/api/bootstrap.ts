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

async function hashPasswordForBootstrap(password: string, username: string): Promise<string> {
  const prefixMarker = '$CH$';
  if (password.startsWith(prefixMarker)) {
    return password;
  }

  if (!globalThis.isSecureContext || !globalThis.crypto?.subtle) {
    throw new Error('Client password hashing is unavailable in this browser context.');
  }

  const encoder = new TextEncoder();
  const data = encoder.encode(`${password}:${username}`);
  const hashBuffer = await crypto.subtle.digest('SHA-256', data);
  const hashArray = new Uint8Array(hashBuffer);
  const hashHex = Array.prototype.map.call(hashArray, x => x.toString(16).padStart(2, '0')).join('');
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
