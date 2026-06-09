// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * Auth web environment variables injected at build time.
 */
interface ImportMetaEnv {
  /**
   * Whether CSRF protection is enabled.
   */
  readonly CHOYSUM_CSRF_ENABLED: boolean;

  /**
   * Whether self-service registration is enabled.
   */
  readonly CHOYSUM_ENABLE_REGISTRATION: boolean;
}
