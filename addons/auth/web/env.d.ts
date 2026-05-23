// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * Auth web environment variables injected at build time.
 */
interface ImportMetaEnv {
  /**
   * Whether client-side password hashing is enabled.
   */
  readonly CHOYSUM_CLIENT_HASHING_ENABLED: boolean;

  /**
   * Whether CSRF protection is enabled.
   */
  readonly CHOYSUM_CSRF_ENABLED: boolean;

  /**
   * Whether self-service registration is enabled.
   */
  readonly CHOYSUM_ENABLE_REGISTRATION: boolean;
}
