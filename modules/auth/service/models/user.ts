// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { BaseModel, Model, Field } from '@/core/service';
import { getCurrentReq, getOrInitReqServiceState } from '@/core/service/api/context';
import type { Insertable } from '@/core/service/api/input';
import type { ConditionEnvelope, RecordRuleOp } from '@/core/service/api/authz';
import { ChoysumError } from '@/core/service/error';
import { newAuthError, wrapAuthError, GrpcCode, AuthErrCode } from '../error';
import Session from './session';
import Role from './role';
import RoleMethodAccess from './role_method_access';
import Token from './token';
import UserRole from './user_role';
import { parseModelFullName, parseServiceFullName } from '@/core/service/utils/model_parsing';
import { uniqStrings } from '@/core/service/utils/normalization';
import { buildAuthzContextCacheKey, buildMethodAccessCacheKey } from './_request_cache_invalidation';
import { withPermissionGraphBypass, sortStrings, getCompanyScopeFromRequestContext } from './_user_authz_shared';
import { evaluateRoleMethodAccess, evaluateUiDerivedMethodDecision, resolveMethodAccessMeta } from './_user_method_access';
import {
  buildScopePreferences,
  computeTokenCompanyScope,
  createSwitchCompanyScopeAuditEmitter,
  normalizeRequestedEnabledCompanyIds,
  normalizeScopeId,
  validateSwitchCompanyScopeInput,
} from './_user_lifecycle_scope';
import {
  ensureCreatedUserIdOrThrow,
  ensureRegistrationIdentityUnique,
  issueLoginTokensAndSession,
  provisionRegisteredUserBaseline,
  refreshTokensWithLatestMetadata,
  revokeLogoutArtifacts,
  validateAndHashRegistrationInput,
  validateLoginCandidateOrThrow,
} from './_user_lifecycle_auth';
import { buildAclAggregation } from './_user_permission_state_acl';
import { buildUiPermissionProjection } from './_user_permission_state_ui';
import { evaluateFieldRules } from './_user_field_rule_eval';
import { evaluateRecordRuleCondition } from './_user_record_rule_eval';
import { buildAuthzContext, computeEffectiveRoleScopes, computePermStateVersion, expandRoleClosure, maxUpdatedAt } from './_user_authz_context';

/**
 * Auth user model with identity, token, and company-scope operations.
 */
@Model('User')
export default class User extends BaseModel {
  /**
   * Unique username used for sign-in.
   */
  @Field({ type: 'varchar', column: { size: 100, unique: true, notNull: true } })
  Username: string;

  /**
   * Primary email address for the user.
   */
  @Field({ type: 'varchar', column: { size: 100, unique: true } })
  readonly Email: string;

  /**
   * Optional phone number for the user.
   */
  @Field({ type: 'varchar', column: { size: 20, unique: true } })
  Phone: string;

  /**
   * Stored password hash for local authentication.
   */
  @Field({ type: 'varchar', column: { size: 255, notNull: true } })
  PasswordHash: string;

  /**
   * Given name used in profile and display contexts.
   */
  @Field({ type: 'varchar', column: { size: 100, index: true } })
  FirstName: string;

  /**
   * Family name used in profile and display contexts.
   */
  @Field({ type: 'varchar', column: { size: 100, index: true } })
  LastName: string;

  /**
   * Computed full name derived from first and last name.
   */
  @Field({
    type: 'varchar',
    column: {
      size: 200,
      compute: {
        expr: (self: User) => self.FirstName + ' ' + self.LastName,
        deps: ['FirstName', 'LastName'],
      },
    },
  })
  FullName: string;

  /**
   * Optional avatar image for the user profile.
   */
  @Field({ type: 'image', column: { index: true } })
  Avatar?: string;

  /**
   * Preferred language reserved for future localization support.
   */
  @Field({ type: 'varchar', column: { size: 20 } })
  Language: string;

  /**
   * Preferred timezone reserved for future localization support.
   */
  @Field({ type: 'varchar', column: { size: 40 } })
  Timezone: string;

  /**
   * User-specific UI and company-scope preferences.
   */
  @Field({ type: 'jsonobject', column: { default: () => {} } })
  Preferences: {
    // Company scope preferences (P3).
    activeCompanyId?: string;
    enabledCompanyIds?: string[];

    // UI appearance preferences.
    avatar?: string;
    theme?: 'light' | 'dark' | 'auto';
    density?: 'comfortable' | 'compact' | 'standard';

    // Notification preferences.
    notifications?: {
      email?: boolean;
      push?: boolean;
      inApp?: boolean;
    };

    // Formatting and display preferences.
    display?: {
      dateFormat?: string;
      timeFormat?: string;
      currency?: string;
    };

    // Additional infrequently used preferences.
  };

  /**
   * Primary company assigned by the base company module.
   */
  @Field({ type: 'ManyToOneRef', targetModel: 'base.Company' })
  CompanyId: string;

  /**
   * Additional company ids available to the user in multi-company mode.
   */
  @Field({ type: 'ManyToManyRef', targetModel: 'base.Company' })
  CompanyIds: string[];

  /**
   * Whether the account is currently active.
   */
  @Field({ type: 'boolean', column: { default: () => true, index: true } })
  IsActive: boolean;

  /**
   * Timestamp of the most recent successful login.
   */
  @Field({ type: 'datetime', column: { index: true } })
  LastLogin: Date;

  /**
   * Sessions currently associated with the user.
   */
  @Field({
    type: 'OneToMany',
    relation: { targetModel: () => Session, inverseField: 'UserId' },
  })
  Sessions: Session[];

  /**
   * Tokens currently associated with the user.
   */
  @Field({
    type: 'OneToMany',
    relation: { targetModel: () => Token, inverseField: 'UserId' },
  })
  Tokens: Token[];

  /**
   * Roles granted to the user through the auth graph.
   */
  @Field({
    type: 'ManyToMany',
    relation: {
      joinModel: () => UserRole,
      targetModel: () => Role,
      joinField: 'UserId',
      inverseJoinField: 'RoleId',
    },
  })
  Roles: Role[];

  /**
   * Register a new local user and provision the default auth baseline.
   */
  static async Register(userData: Partial<Insertable<User>>, password: string): Promise<string> {
    const passwordHash = validateAndHashRegistrationInput(userData as any, password);
    await ensureRegistrationIdentityUnique(userData as any, {
      searchByUsername: async (username: string) => await this.Search(['Username', '=', username] as any),
      searchByEmail: async (email: string) => await this.Search(['Email', '=', email] as any),
    });

    try {
      // Create the user record first so follow-up provisioning has a stable id.
      const created = await this.Create({
        ...userData,
        PasswordHash: passwordHash,
      });

      const userId = ensureCreatedUserIdOrThrow((created as any)?.Id);
      await provisionRegisteredUserBaseline(userId, (userData as any)?.Preferences, {
        updateUserCompanyContext: async values => {
          await this.UpdateById(userId, values as any, ['Id'] as any);
        },
      });

      return userId;
    } catch (error) {
      throw wrapAuthError(error, {
        code: AuthErrCode.USER_CREATION_FAILED,
        message: 'User registration failed',
      }).withMetadata({ username: String(userData.Username || '') });
    }
  }

  /**
   * Authenticate a local user and create a token pair.
   */
  static async Login(usernameOrEmail: string, password: string, ipAddress?: string, deviceInfo?: string, rememberMe?: boolean): Promise<TokenPair> {
    const users = await this.Search({
      Or: [
        ['Username', '=', usernameOrEmail],
        ['Email', '=', usernameOrEmail],
      ],
    });
    const user = validateLoginCandidateOrThrow((users || [])[0] as any, usernameOrEmail, password) as any;

    try {
      return await issueLoginTokensAndSession(
        user as any,
        {
          extractUserMetadata: async u => await this.extractUserMetadata(u as any),
          updateLastLogin: async (uid: string, timestamp: Date) => {
            await this.UpdateById(uid, { LastLogin: timestamp } as any);
          },
        },
        { ipAddress, deviceInfo, rememberMe }
      );
    } catch (error) {
      throw wrapAuthError(error, {
        code: AuthErrCode.TOKEN_CREATION_FAILED,
        message: 'Login failed: unable to create token pair',
      }).withMetadata({ userId: user.Id, username: user.Username });
    }
  }

  /**
   * Build token metadata from the current user record and company scope.
   */
  static async extractUserMetadata(user: User): Promise<TokenMetadata> {
    const userId = String((user as any)?.Id || '').trim();
    const userVersion = Number(new Date((user as any)?.UpdatedAt || Date.now()));
    const permStateVersion = userId ? await this._computePermStateVersion(userId) : 0;
    const companyScope = computeTokenCompanyScope(user as any);

    return {
      language: user.Language,
      timezone: user.Timezone,
      allowedCompanyIds: companyScope.allowedCompanyIds,
      activeCompanyId: companyScope.activeCompanyId,
      enabledCompanyIds: companyScope.enabledCompanyIds,
      userVersion,
      permStateVersion,
    };
  }

  /**
   * Refresh a token pair using the latest user metadata snapshot.
   */
  static async RefreshTokens(refreshToken: string): Promise<TokenPair> {
    try {
      return await refreshTokensWithLatestMetadata(refreshToken, {
        browseUser: async (userId: string) => await this.Browse(userId),
        extractUserMetadata: async (user: any) => await this.extractUserMetadata(user),
      });
    } catch (error) {
      throw wrapAuthError(error, {
        code: AuthErrCode.TOKEN_REFRESH_FAILED,
        message: 'Token refresh failed',
      });
    }
  }

  /**
   * Switch the current company scope for the authenticated user.
   * - activeCompanyId: company used for current write operations.
   * - enabledCompanyIds: readable company scope; defaults to Preferences or [activeCompanyId].
   *
   * Strict fail-closed validation:
   * - enabled ⊆ allowed
   * - active ∈ enabled
   */
  static async SwitchCompanyScope(activeCompanyId: string, enabledCompanyIds?: any): Promise<TokenPair> {
    const userId = String(this.userId || '').trim();
    if (!userId) {
      throw newAuthError({
        code: AuthErrCode.VALIDATION_FAILED,
        message: 'User is not logged in',
      }).withGrpcCode(GrpcCode.Unauthenticated);
    }

    const audit = createSwitchCompanyScopeAuditEmitter('auth.user.switch_company_scope');
    const normalizedRequestedEnabled = normalizeRequestedEnabledCompanyIds(enabledCompanyIds);
    const active = normalizeScopeId(activeCompanyId);
    if (!active) {
      audit.emitOnce({
        ok: false,
        userId,
        targetActive: active,
        targetEnabled: null,
        reason: 'activeCompanyId cannot be empty',
      });
      throw newAuthError({
        code: AuthErrCode.VALIDATION_FAILED,
        message: 'activeCompanyId cannot be empty',
      }).withGrpcCode(GrpcCode.InvalidArgument);
    }

    try {
      const user = await this.Browse(userId);
      await user.load(['CompanyIds']);
      const validated = validateSwitchCompanyScopeInput(user as any, active, enabledCompanyIds);
      if (!validated.ok) {
        if (validated.code === 'enabled_type') {
          audit.emitOnce({
            ok: false,
            userId,
            targetActive: active,
            targetEnabled: null,
            reason: 'enabledCompanyIds must be a string[] or omitted',
          });
          throw newAuthError({
            code: AuthErrCode.VALIDATION_FAILED,
            message: 'enabledCompanyIds must be a string[] or omitted',
          }).withGrpcCode(GrpcCode.InvalidArgument);
        }

        if (validated.code === 'enabled_unauthorized') {
          const targetEnabled = validated.enabled || [];
          audit.emitOnce({
            ok: false,
            userId,
            targetActive: active,
            targetEnabled,
            reason: 'enabledCompanyIds contains an unauthorized company',
            companyId: validated.companyId,
          });
          throw newAuthError({
            code: AuthErrCode.VALIDATION_FAILED,
            message: 'enabledCompanyIds contains an unauthorized company',
          })
            .withGrpcCode(GrpcCode.InvalidArgument)
            .withMetadata({ companyId: validated.companyId || '' });
        }

        if (validated.code === 'active_outside_allowed') {
          audit.emitOnce({
            ok: false,
            userId,
            targetActive: active,
            targetEnabled: validated.enabled || [],
            reason: 'activeCompanyId is outside the allowed company scope',
          });
          throw newAuthError({
            code: AuthErrCode.VALIDATION_FAILED,
            message: 'activeCompanyId is outside the allowed company scope',
          })
            .withGrpcCode(GrpcCode.InvalidArgument)
            .withMetadata({ activeCompanyId: active });
        }

        audit.emitOnce({
          ok: false,
          userId,
          targetActive: active,
          targetEnabled: validated.enabled || [],
          reason: 'activeCompanyId must be included in enabledCompanyIds',
        });
        throw newAuthError({
          code: AuthErrCode.VALIDATION_FAILED,
          message: 'activeCompanyId must be included in enabledCompanyIds',
        }).withGrpcCode(GrpcCode.InvalidArgument);
      }

      const enabled = validated.enabled;
      const nextPrefs: any = buildScopePreferences(validated.prefs, active, enabled);

      // NOTE: Switching company scope is a self-service auth operation.
      // Users may legitimately have zero roles at this stage; record rules are deny-by-default for write.
      // We bypass RecordRule/FieldRule for this internal preference update (still guarded by strict validation above).
      await withPermissionGraphBypass(async () => {
        await this.UpdateById(userId, { Preferences: nextPrefs } as any);
      });

      // Reload the record so UpdatedAt and metadata reflect the persisted preferences.
      const updated = await this.Browse(userId);
      await updated.load(['CompanyIds']);
      const metadata = await this.extractUserMetadata(updated);

      audit.emitOnce({
        ok: true,
        userId,
        targetActive: active,
        targetEnabled: enabled,
        reason: 'ok',
      });
      return await Token.CreateTokenPair(userId, metadata);
    } catch (error) {
      // Emit an audit failure record without changing error semantics.
      if (!audit.wasEmitted()) {
        try {
          const msg = typeof (error as any)?.message === 'string' ? (error as any).message : String(error);
          const code = (error as any)?.code ? String((error as any).code) : undefined;
          audit.emitOnce({
            ok: false,
            userId,
            targetActive: active,
            targetEnabled: normalizedRequestedEnabled,
            reason: msg,
            errorCode: code,
          });
        } catch {}
      }

      // Preserve auth-domain errors, especially VALIDATION_FAILED, without wrapping them.
      if (error instanceof ChoysumError && error.domain === 'auth') {
        throw error;
      }
      throw wrapAuthError(error, {
        code: AuthErrCode.UNKNOWN,
        message: 'Switch company scope failed',
      });
    }
  }

  /**
   * Revoke the current token or every token owned by the current user.
   *
   * @param token - Access token to revoke.
   * @param allDevices - Whether to revoke every token owned by the user.
   * @param deviceInfo - Device information used for audit metadata.
   * @returns True when logout succeeds.
   */
  static async Logout(token: string, allDevices: boolean = false, deviceInfo?: string): Promise<boolean> {
    if (!token) {
      throw newAuthError({
        code: AuthErrCode.VALIDATION_FAILED,
        message: 'Token is required',
      }).withGrpcCode(GrpcCode.InvalidArgument);
    }

    try {
      await revokeLogoutArtifacts(token, allDevices);

      return true;
    } catch (error) {
      throw wrapAuthError(error, {
        code: AuthErrCode.TOKEN_REVOCATION_FAILED,
        message: 'Logout failed',
      }).withMetadata({
        allDevices: String(allDevices),
        deviceInfo: deviceInfo || '',
      });
    }
  }

  /**
   * Get the frontend permission projection used for UX-only pruning.
   *
   * The return shape follows docs/frontend_permission_design.md.
   */
  static async GetPermissionState(): Promise<{
    permStateVersion: number;
    byCompany: Record<
      string,
      {
        ui?: {
          routes?: string[];
          menus?: string[];
          actions?: string[];
        };
      }
    >;
  }> {
    try {
      const userId = String(this.userId || '').trim();
      const companyIds = Array.isArray(this.companyIds) ? this.companyIds.map(c => String(c || '').trim()).filter(Boolean) : [];
      const hasCompany = companyIds.length > 0;

      if (!userId) {
        return { permStateVersion: 0, byCompany: {} };
      }

      return await withPermissionGraphBypass(async () => {
        const permStateVersion = await this._computePermStateVersion(userId);
        const authz = await this._getAuthzContext();
        const roleIds = authz.roleIds;

        const acl = await buildAclAggregation(roleIds, authz.roleScopesById);
        const byCompany = await buildUiPermissionProjection(roleIds, authz.roleScopesById, authz.enabledCompanyIds || [], acl);
        return { permStateVersion, byCompany };
      });
    } catch {
      // fail-closed
      return { permStateVersion: 0, byCompany: {} };
    }
  }

  /**
   * Build or reuse the request-scoped authorization context for the current user.
   */
  private static async _getAuthzContext(): Promise<{
    userId: string;
    activeCompanyId: string;
    enabledCompanyIds: string[];
    roleScopesById: Record<string, { global: boolean; companies: string[] }>;
    roleIds: string[];
    rolesByCompany: Record<string, string[]>;
    roles: string[];
  }> {
    const req = getCurrentReq();
    const state = getOrInitReqServiceState(req);

    const userId = String((this.userId as any) || '').trim();
    const companyScope = getCompanyScopeFromRequestContext();
    const enabledCompanyIdsKey = sortStrings(uniqStrings(companyScope.enabledCompanyIds));
    const companyScopeKey = `${companyScope.activeCompanyId}::${enabledCompanyIdsKey.join(',')}`;

    const build = async (args: { userId: string; activeCompanyId: string; enabledCompanyIds: string[] }) => buildAuthzContext(args);

    if (!state) {
      return await build({ userId, activeCompanyId: companyScope.activeCompanyId, enabledCompanyIds: enabledCompanyIdsKey });
    }

    const KEY = buildAuthzContextCacheKey(userId, companyScopeKey);
    const existing = state[KEY];
    if (existing) {
      if (typeof existing?.then === 'function') {
        const v = await existing;
        state[KEY] = v;
        return v;
      }
      return existing;
    }

    const p = build({ userId, activeCompanyId: companyScope.activeCompanyId, enabledCompanyIds: enabledCompanyIdsKey })
      .then((v: any) => {
        try {
          state[KEY] = v;
        } catch {
          // ignore
        }
        return v;
      })
      .catch((e: any) => {
        try {
          delete state[KEY];
        } catch {
          // ignore
        }
        throw e;
      });
    state[KEY] = p;
    return await p;
  }

  /**
   * Return the latest UpdatedAt timestamp matching a condition.
   */
  private static async _maxUpdatedAt(model: any, cond: any): Promise<number> {
    return await maxUpdatedAt(model, cond);
  }

  private static async _computePermStateVersion(userId: string): Promise<number> {
    return await computePermStateVersion(userId);
  }

  private static async _expandRoleClosure(directRoleIds: string[]): Promise<string[]> {
    return await expandRoleClosure(directRoleIds);
  }

  private static async _computeEffectiveRoleScopes(userRoles: any[]): Promise<Map<string, { global: boolean; companies: Set<string> }>> {
    return await computeEffectiveRoleScopes(userRoles);
  }

  /**
   * Check whether the user can call a gRPC service method under P1 ACL rules.
   *
   * Keep ACL evaluation centralized in auth so other modules do not query auth tables directly.
   * This path is deny-by-default, deny-wins, and admin-aware.
   */
  static async CheckMethodAccess(companyId: string, serviceFullName: string): Promise<boolean> {
    try {
      const userId = this.userId;
      if (!userId || !serviceFullName) return false;

      const parsed = parseServiceFullName(serviceFullName);
      if (!parsed) return false;
      const { appName, modelName, methodName } = parsed;
      const normalizedFullMethod = `/${appName}.${modelName}/${methodName}`;

      const normalizedCompanyId = String(companyId || '').trim();
      const hasCompany = normalizedCompanyId.length > 0;
      if (!hasCompany) return false;

      // Company view must be within enabledCompanyIds (fail-closed).
      const { enabledCompanyIds } = getCompanyScopeFromRequestContext();
      if (enabledCompanyIds.length > 0 && !enabledCompanyIds.includes(normalizedCompanyId)) return false;

      // Permission graph reads must bypass RecordRule/FieldRule.
      // Otherwise a request with activeCompanyId=c1 can never evaluate roles scoped to c2,
      // because UserRole/RoleMethodAccess reads may be fail-closed by RecordRule injection.
      return await withPermissionGraphBypass(async () => {
        const authz = await this._getAuthzContext();
        const roleIds = authz.rolesByCompany?.[normalizedCompanyId] || [];
        if (roleIds.length === 0) return false;

        const req = getCurrentReq();
        const state = getOrInitReqServiceState(req);
        const cacheKey = buildMethodAccessCacheKey(String(authz.userId || '').trim(), normalizedCompanyId, normalizedFullMethod);
        const cached = state?.[cacheKey];
        if (typeof cached === 'boolean') return cached;

        const accessMeta = await resolveMethodAccessMeta(appName, modelName, methodName);
        if (!accessMeta) return false;

        const accessResult = await evaluateRoleMethodAccess(roleIds, accessMeta.scopeOr);

        if (accessResult.denied) {
          if (state) state[cacheKey] = false;
          return false;
        }

        if (accessResult.allowed) {
          if (state) state[cacheKey] = true;
          return true;
        }

        const uiDecision = await evaluateUiDerivedMethodDecision(roleIds, accessMeta.modelKey, accessMeta.methodLower);
        if (uiDecision.denied) {
          if (state) state[cacheKey] = false;
          return false;
        }

        const result = Boolean(uiDecision.allowed);
        if (state) state[cacheKey] = result;

        return result;
      });
    } catch {
      // fail-closed
      return false;
    }
  }

  /**
   * Get the effective RecordRule condition for the current user, model, and operation.
   *
   * Notes:
   * - userId comes from the current request identity.
   * - model should be "<app>.<Model>" such as "base.Company".
   * - op should be "read" | "write" | "create" | "delete".
   * - the return value is a ConditionEnvelope that maps to google.protobuf.Value on the proto side.
   */
  static async GetRecordRuleCondition(model: string, op: string): Promise<ConditionEnvelope> {
    try {
      const userId = this.userId;
      if (!userId) {
        return { kind: 'false', reason: 'missing_identity_user_id' };
      }

      const rawModel = String(model || '').trim();
      const rawOp = String(op || '')
        .trim()
        .toLowerCase();
      if (!rawModel) return { kind: 'false', reason: 'missing_model' };

      const opValue = rawOp as RecordRuleOp;
      if (!['read', 'write', 'create', 'delete'].includes(opValue)) {
        return { kind: 'false', reason: 'invalid_op' };
      }

      const parsedModel = parseModelFullName(rawModel);
      if (!parsedModel) return { kind: 'false', reason: 'invalid_model_full_name' };

      const { enabledCompanyIds } = getCompanyScopeFromRequestContext();
      const authz = await this._getAuthzContext();

      return await evaluateRecordRuleCondition({
        appName: parsedModel.appName,
        modelName: parsedModel.modelName,
        hasCompany: enabledCompanyIds.length > 0,
        opValue,
        roleIds: authz.roles || [],
        roleScopesById: authz.roleScopesById as Record<string, { global: boolean; companies: string[] }>,
      });
    } catch {
      // fail-closed
      return { kind: 'false', reason: 'internal_error' };
    }
  }

  /**
   * Get the effective FieldRule spec for the current user and model.
   *
   * Notes:
   * - userId comes from the current request identity.
   * - model should be "<app>.<Model>" such as "auth.User".
   * - the return value is a structured object that maps to google.protobuf.Value on the proto side.
   */
  static async GetFieldRuleSpec(model: string): Promise<{ denyReadFields: string[]; denyWriteFields: string[]; reason?: string }> {
    try {
      const userId = this.userId;
      if (!userId) {
        throw newAuthError({
          code: AuthErrCode.VALIDATION_FAILED,
          message: 'Missing identity information (userId)',
        }).withGrpcCode(GrpcCode.Unauthenticated);
      }

      const rawModel = String(model || '').trim();
      if (!rawModel) {
        throw newAuthError({
          code: AuthErrCode.VALIDATION_FAILED,
          message: 'Missing model name (model)',
        }).withGrpcCode(GrpcCode.InvalidArgument);
      }

      const parsedModel = parseModelFullName(rawModel);
      if (!parsedModel) {
        throw newAuthError({
          code: AuthErrCode.VALIDATION_FAILED,
          message: 'Invalid model full name; expected "<app>.<Model>"',
        })
          .withGrpcCode(GrpcCode.InvalidArgument)
          .withMetadata({ model: rawModel });
      }

      return await withPermissionGraphBypass(async () => {
        const authz = await this._getAuthzContext();
        const roleIds = Array.isArray(authz?.roles) ? authz.roles : [];
        return await evaluateFieldRules({
          appName: parsedModel.appName,
          modelName: parsedModel.modelName,
          rawModel,
          roleIds,
        });
      });
    } catch (error) {
      throw wrapAuthError(error, {
        code: AuthErrCode.UNKNOWN,
        message: 'Failed to compute field rule spec',
      }).withMetadata({ model: String(model || '').trim() });
    }
  }
}
