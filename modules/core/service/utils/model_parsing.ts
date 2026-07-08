// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * Parse an application-qualified model name (e.g. "auth.User").
 */
export function parseModelFullName(v: string): { appName: string; modelName: string } | null {
  const s = String(v || '').trim();
  if (!s) return null;
  const lastDot = s.lastIndexOf('.');
  if (lastDot <= 0 || lastDot >= s.length - 1) return null;
  const appName = s.slice(0, lastDot).trim();
  const modelName = s.slice(lastDot + 1).trim();
  if (!appName || !modelName) return null;
  return { appName, modelName };
}

/**
 * Parse /app.Model/Method into application, model, and method parts.
 */
export function parseServiceFullName(v: string): { appName: string; modelName: string; methodName: string } | null {
  const s = String(v || '').trim();
  if (!s) return null;

  const noLead = s.startsWith('/') ? s.slice(1) : s;
  const parts = noLead.split('/').filter(Boolean);
  if (parts.length !== 2) return null;
  const service = parts[0].trim();
  const methodName = parts[1].trim();
  if (!service || !methodName) return null;

  const parsed = parseModelFullName(service);
  if (!parsed) return null;
  return { ...parsed, methodName };
}
