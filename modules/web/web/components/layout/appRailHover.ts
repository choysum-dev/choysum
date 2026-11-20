// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/** Delay before opening a collapsed-rail L2 flyout on pointer enter. */
export const APP_RAIL_HOVER_OPEN_MS = 100;

/** Grace period before closing the flyout after pointer leave. */
export const APP_RAIL_HOVER_CLOSE_MS = 200;

/**
 * After a leaf click, ignore flyout re-open (click-through / synthetic enter)
 * for this long, then unlock so a later hover on the app icon works again.
 */
export const APP_RAIL_LEAF_SUPPRESS_MS = 400;
