// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * Default global upload byte cap aligned with `document.attachment.maxUploadBytes`
 * (Viper default 20971520 / 20 MiB). Field-level `maxUploadBytes` must not exceed this
 * at decorator time; runtime effective cap is min(field, current config).
 */
export const DEFAULT_GLOBAL_MAX_UPLOAD_BYTES = 20 * 1024 * 1024;
