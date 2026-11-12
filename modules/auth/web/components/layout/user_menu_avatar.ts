// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * Resolve sidebar user-menu avatar src from User.Avatar attachment values.
 * Internal document binding content URLs need a Bearer fetch → data URL for <img>.
 *
 * `auth.User` Browse returns image fields as bare binding-id strings (no query
 * executor enrichment). Prefer attachment preview URLs when present; otherwise
 * synthesize `/_document/bindings/{id}/content`.
 */

import {
  resolveBindingId,
  resolvePreviewUrl,
} from '@/web/web/components/field/attachmentFieldHelpers';
import {
  fetchAuthenticatedPreviewAsDataUrl,
  isInternalDocumentBindingContentUrl,
} from '@/web/web/components/field/attachmentPreviewAuth';

function bindingContentUrl(bindingId: string): string {
  return `/_document/bindings/${bindingId}/content`;
}

/** True for values that are already usable as an <img> src (or relative path). */
function isUrlLike(value: string): boolean {
  return (
    value.startsWith('data:') ||
    value.startsWith('blob:') ||
    value.startsWith('https://') ||
    value.startsWith('http://') ||
    value.startsWith('/')
  );
}

/** Sync preview/download URL from Avatar bag (no auth fetch). */
export function resolveUserMenuAvatarPreviewUrl(avatar: unknown): string | undefined {
  const fromObject = resolvePreviewUrl(avatar);
  if (fromObject) return fromObject;

  if (typeof avatar === 'string') {
    const trimmed = avatar.trim();
    if (!trimmed) return undefined;
    if (isUrlLike(trimmed)) return trimmed;
    return bindingContentUrl(trimmed);
  }

  const bindingId = resolveBindingId(avatar);
  if (bindingId) return bindingContentUrl(bindingId);
  return undefined;
}

export type FetchAuthenticatedPreview = (url: string) => Promise<string | undefined>;

/**
 * Display src for ChoyAvatarImage: passthrough public/data URLs; auth-fetch
 * internal `/_document/bindings/.../content` URLs to a data URL.
 */
export async function resolveUserMenuAvatarDisplaySrc(
  avatar: unknown,
  fetchAuthed: FetchAuthenticatedPreview = fetchAuthenticatedPreviewAsDataUrl,
): Promise<string | undefined> {
  const preview = resolveUserMenuAvatarPreviewUrl(avatar);
  if (!preview) return undefined;
  if (preview.startsWith('data:') || preview.startsWith('blob:')) return preview;
  if (!isInternalDocumentBindingContentUrl(preview)) return preview;
  return (await fetchAuthed(preview)) || undefined;
}
