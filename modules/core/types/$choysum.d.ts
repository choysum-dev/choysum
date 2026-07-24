// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * Token pair.
 */
interface TokenPair {
  accessToken: string;
  refreshToken: string;
  expiresAt: number;
  refreshExpiresAt: number;
}

/**
 * Token metadata.
 *
 * Follows JWT conventions and extends them with custom fields.
 */
interface TokenMetadata {
  language?: string; // Language preference, mapped to ctx.lang by default.
  timezone?: string; // Timezone preference, mapped to ctx.tz by default.

  // Active company IANA timezone, mapped to ctx.companyTz (business day authority).
  companyTimezone?: string;

  // Authorization scope used to derive identity.allowedCompanyIds.
  allowedCompanyIds?: string[];

  // Active operating company, mapped to ctx.activeCompanyId by default.
  activeCompanyId?: string;

  // Company set for the current read boundary, mapped to ctx.enabledCompanyIds.
  enabledCompanyIds?: string[];

  // Hint that frontend user/profile data may need a refresh.
  userVersion?: number;

  // Hint that the frontend PermissionState cache is stale.
  permStateVersion?: number;
}

interface TokenIdentity {
  userId: string;

  /**
   * Token ID.
   */
  tokenId: string;

  /**
   * Metadata associated with the token.
   */
  metadata?: TokenMetadata;
}

/**
 * JS Context (Spec) — see docs/js_context_definition.md
 */
type JsBusinessContext = Readonly<{
  lang?: string;
  /** Display timezone (user → empty browser baggage → company → UTC). */
  tz?: string;
  /** Active company IANA timezone; business-day authority only. */
  companyTz?: string;
  activeCompanyId?: string;
  enabledCompanyIds?: string[];
  [key: string]: unknown;
}>;

type JsIdentitySnapshot = Readonly<{
  userId?: string;
  tokenId?: string;
  roles?: Array<{ id: string; displayName: string }>;
  allowedCompanyIds?: string[];
}>;

type JsRequestMeta = Readonly<{
  requestId?: string;
  traceId?: string;
  spanId?: string;
  kind?: 'grpc' | 'grpc-web' | 'http' | 'local' | 'background';
  depth?: number;
}>;

type JsCtx = Readonly<{
  ctx?: JsBusinessContext;
  identity?: JsIdentitySnapshot;
  req?: JsRequestMeta;
}>;

/**
 * TokenPayload interface.
 * Describes the payload structure stored in JWT tokens.
 */
interface TokenPayload {
  // Standard JWT claims.
  iss?: string; // Issuer.
  sub?: string; // Subject, usually the user ID.
  aud?: string | string[]; // Audience.
  exp?: number; // Expiration time as a Unix timestamp.
  nbf?: number; // Not-before time.
  iat?: number; // Issued-at time.
  jti?: string; // JWT ID
  meta?: Record<string, unknown>; // Custom metadata fields.
}

// Request context injected at runtime and hard-cut to JsCtx.
type ChoysumRequestContext = JsCtx;

declare var $choysum: {
  db: {
    dialectName: 'postgres' | 'mysql' | 'sqlite' | 'mssql';
    query: (sql: string, parameters: string) => Promise<string>;
    execute: (sql: string, parameters: string) => Promise<string>;
    savepoint: (name: string) => Promise<string>;
    rollbackToSavepoint: (name: string) => Promise<void>;
    releaseSavepoint: (name: string) => Promise<void>;
  };
  crypto: {
    hashPassword: (password: string) => string;
    verifyPassword: (password: string, hashedPassword: string) => boolean;
    generateToken: () => string;
  };
  xid: {
    New: () => string;
  };
  request: {
    id: string;
    service: string;
    args: unknown[];
    context: ChoysumRequestContext;
  };
  moduleManagement: {
    install(params: { moduleName: string; withDemo?: boolean; operatorUserId?: string; jobId?: string; action?: string }): Promise<{
      ok: boolean;
      errorDomain?: string;
      errorCode?: string;
      errorMessage?: string;
    }>;
    uninstall(params: { moduleName: string; operatorUserId?: string; jobId?: string; action?: string }): Promise<{
      ok: boolean;
      errorDomain?: string;
      errorCode?: string;
      errorMessage?: string;
    }>;
    upgrade(params: { moduleName: string; operatorUserId?: string; jobId?: string; action?: string }): Promise<{
      ok: boolean;
      errorDomain?: string;
      errorCode?: string;
      errorMessage?: string;
    }>;
    reload(): Promise<{ triggered: boolean; failed: boolean; error?: string }>;
    syncIndex(params: { originType?: 'local' | 'registry'; force?: boolean }): Promise<{
      ok: boolean;
      originType?: string;
      total?: number;
      success?: number;
      failed?: number;
      durationMs?: number;
      error?: string;
    }>;
  };
  /**
   * Authentication service capabilities.
   */
  auth: {
    /**
     * Whether the authentication service is enabled.
     */
    enabled: boolean;

    /**
     * Creates a pair of access and refresh tokens.
     * @param userId User ID.
     * @param metadata Optional metadata to associate with the token pair.
     * @returns Token pair containing the access and refresh tokens.
     * @throws When the authentication service is disabled or token creation fails.
     */
    createTokens(userId: string, metadata?: TokenMetadata | Record<string, unknown>): TokenPair;

    /**
     * Creates a new token pair from a refresh token.
     * @param refreshToken Refresh token.
     * @returns The newly issued token pair.
     * @throws When the authentication service is disabled or refresh fails.
     */
    refreshTokens(refreshToken: string, metadata?: TokenMetadata | Record<string, unknown>): TokenPair;

    /**
     * Revokes a specific token.
     * @param token Token to revoke.
     * @param reason Optional revoke reason that will be recorded in the database.
     * @returns Whether the revoke operation succeeded.
     * @throws When the authentication service is disabled or revoke fails.
     */
    revokeToken(token: string, reason?: string): boolean;

    /**
     * Revokes all tokens for a user.
     * @param userId User ID.
     * @param exceptTokenId Optional token ID to keep active.
     * @param reason Optional revoke reason that will be recorded in the database.
     * @returns The number of revoked tokens.
     * @throws When the authentication service is disabled or revoke fails.
     */
    revokeAllUserTokens(userId: string, exceptTokenId?: string, reason?: string): number;

    /**
     * Validates token validity.
     * @param token Token to validate.
     * @param tokenType Token type, either 'access' or 'refresh'.
     * @param checkRevoked Optional flag to check revoke state in the database and avoid duplicate checks.
     * @returns Identity information associated with the token.
     * @throws When the authentication service is disabled or the token is invalid.
     */
    validateToken(token: string, tokenType: string, checkRevoked?: boolean): TokenIdentity;
  };
  utils?: {
    isDecimal: (v: unknown) => boolean;
    isDecimalLike: (v: unknown) => boolean;
    isDecimalLeak: (v: unknown) => boolean;
    serialize: <T = unknown>(v: T) => T;
    deserialize: <T = unknown>(v: T) => T;
    decimalEqual: (a: unknown, b: unknown) => boolean;

    // NEW
    asBigdecimal: <T = unknown>(v: T) => T | { $bigdecimal: string };
    isBigdecimalEnvelope: (v: unknown) => boolean;
    toDecimalString: (v: unknown) => string | undefined;
  };

  /**
   * gRPC Bridge (Go Runtime Injected)
   */
  grpc: {
    unary: <TResponse = unknown, TRequest = unknown>(service: string, method: string, data: TRequest) => Promise<TResponse>;
    stream: <TResponse = unknown, TRequest = unknown>(service: string, method: string, data: TRequest) => TResponse;
    registerProto: (path: string, content: string) => void;
  };
};
