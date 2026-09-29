// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * Ambient modules for dayjs subpath imports.
 *
 * type-fetch intentionally skips dayjs/locale/* and dayjs/plugin/* (see
 * shouldSkipTypeFetchSubpathSpecifier). CLI `choysum test typecheck` injects
 * the same stubs via internal/typecheck SubpathStubOverlay; the IDE reads this
 * committed file under modules/ instead.
 */
declare module 'dayjs/locale/*';
declare module 'dayjs/plugin/*';
