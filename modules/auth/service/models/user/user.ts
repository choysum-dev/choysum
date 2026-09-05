// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { Model, Field, Compute } from '@/core/service';
import AttachmentOwnerMixin from '@/core/service/mixins/attachment_owner_model';
import { getCurrentReq, getOrInitReqServiceState, memoizeInReqState } from '@/core/service/api/context';
import type { Insertable } from '@/core/service/api/input';
import type { ConditionEnvelope, FieldRuleSpec, RecordRuleOp } from '@/core/service/api/authz';
import { ChoysumError } from '@/core/service/error';
import { newAuthError, wrapAuthError, GrpcCode, AuthErrCode } from '../../error';
import { _t, _lt } from '../../i18n';
import Session from '../session';
import Role from '../role';
import Token from '../token';
import UserRole from '../user_role';
import { parseModelFullName, parseServiceFullName } from '@/core/service/utils/model_parsing';
import { uniqStrings } from '@/core/service/utils/normalization';
import { isIanaTimezone, listIanaTimezoneSelection } from '@/core/service/utils/datetime';
import { Constraint } from '@/core/service/api/constraint';
import type LanguageModel from '@/base/service/models/language';
import type Company from '@/base/service/models/company';
import { createServiceByModel } from '@/core/service/rpc';
import { buildAuthzContextCacheKey, buildMethodAccessCacheKey } from '../_request_cache_invalidation';
import { withPermissionGraphBypass, sortStrings, getCompanyScopeFromRequestContext } from './_authz_shared';
import { evaluateRoleMethodAccess, evaluateUiDerivedMethodDecision, resolveMethodAccessMeta } from './_method_access';
import type { MethodAccessDecision } from './_method_access';

const Language = createServiceByModel<typeof LanguageModel>('base.Language');
import {
  buildScopePreferences,
  computeTokenCompanyScope,
  createSwitchCompanyScopeAuditEmitter,
  normalizeRequestedEnabledCompanyIds,
  normalizeScopeId,
  validateSwitchCompanyScopeInput,
} from './_lifecycle_scope';
import {
  ensureCreatedUserIdOrThrow,
  ensureRegistrationIdentityUnique,
  issueLoginTokensAndSession,
  persistBrowserTimezoneIfEmpty,
  provisionRegisteredUserBaseline,
  refreshTokensWithLatestMetadata,
  revokeLogoutArtifacts,
  validateAndHashRegistrationInput,
  validateLoginCandidateOrThrow,
} from './_lifecycle_auth';

import { buildAclAggregation } from './_permission_state_acl';
import { buildUiPermissionProjection } from './_permission_state_ui';
import { evaluateFieldRules } from './_field_rule_eval';
import { evaluateRecordRuleCondition } from './_record_rule_eval';
import { buildAuthzContext, computePermStateVersion } from './_authz_context';

/**
 * Auth user model with identity, token, and company-scope operations.
 *
 * Layout: this file is the only non-`_` entry under `models/user/`. Register /
 * Login / Refresh / Logout / SwitchCompanyScope orchestrate here and call
 * adjacent `user/_lifecycle_*` / `user/_authz_*` / eval helpers. Other modules
 * extend via `@Model('User') export default class User extends UserBase` (import UserBase
 * from `@/auth/service/models` or `@/auth/service/models/user/user`).
 *
 * Extends {@link AttachmentOwnerMixin} for Avatar bind/unbind entry points (dial
 * `document.AttachmentBinding`; not on BaseModel).
 */
@Model('User')
export default class User extends AttachmentOwnerMixin {
  /**
   * Unique username used for sign-in.
   */
  @Field({
    type: 'varchar',
    size: 100,
    unique: true,
    notNull: true,
    string: _lt('Username', { scope: 'auth.model.User.fields' }),
  })
  Username: string;

  /**
   * Primary email address for the user.
   */
  @Field({
    type: 'varchar',
    size: 100,
    unique: true,
    string: _lt('Email', { scope: 'auth.model.User.fields' }),
  })
  readonly Email: string;

  /**
   * Optional phone number for the user.
   */
  @Field({
    type: 'varchar',
    size: 20,
    unique: true,
    string: _lt('Phone', { scope: 'auth.model.User.fields' }),
  })
  Phone: string;

  /**
   * Stored password hash for local authentication.
   */
  @Field({
    type: 'varchar',
    size: 255,
    notNull: true,
    copy: false,
    string: _lt('Password Hash', { scope: 'auth.model.User.fields' }),
  })
  PasswordHash: string;

  /**
   * Given name used in profile and display contexts.
   */
  @Field({
    type: 'varchar',
    size: 100,
    index: true,
    string: _lt('First Name', { scope: 'auth.model.User.fields' }),
  })
  FirstName: string;

  /**
   * Family name used in profile and display contexts.
   */
  @Field({
    type: 'varchar',
    size: 100,
    index: true,
    string: _lt('Last Name', { scope: 'auth.model.User.fields' }),
  })
  LastName: string;

  /**
   * Computed full name derived from first and last name.
   */
  @Field({
    type: 'varchar',
    size: 200,
    string: _lt('Full Name', { scope: 'auth.model.User.fields' }),
  })
  FullName: string;

  @Compute<User>('FullName', {
    deps: ['FirstName', 'LastName'],
  })
  computeFullName() {
    return this.FirstName + ' ' + this.LastName;
  }

  /**
   * Optional avatar image for the user profile.
   */
  @Field({
    type: 'image',
    index: true,
    string: _lt('Avatar', { scope: 'auth.model.User.fields' }),
  })
  Avatar?: string;

  /**
   * Preferred terminology language (e.g. zh_CN). Written by FE language switch when logged in.
   */
  @Field({
    type: 'varchar',
    size: 20,
    string: _lt('Language', { scope: 'auth.model.User.fields' }),
    help: _lt('When set, must match an active base.Language POSIX terminology code.', {
      scope: 'auth.model.User.fields',
    }),
  })
  Language: string;

  /**
   * Preferred IANA timezone for localization and display.
   */
  @Field({
    type: 'selection',
    selection: () => listIanaTimezoneSelection(),
    size: 64,
    string: _lt('Time Zone', { scope: 'auth.model.User.fields' }),
    help: _lt('Display timezone; empty uses browser timezone on first login.', {
      scope: 'auth.model.User.fields',
    }),
  })
  Timezone: string | null;

  /**
   * User-specific UI and company-scope preferences.
   */
  @Field({
    type: 'jsonobject',
    default: () => {},
    string: _lt('Preferences', { scope: 'auth.model.User.fields' }),
  })
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
  @Field<Company>({
    type: 'ManyToOneRef',
    relation: { targetModel: 'base.Company' },
    condition: ['IsActive', '=', true],
    string: _lt('Company', { scope: 'auth.model.User.fields' }),
  })
  CompanyId: string;

  /**
   * Additional company ids available to the user in multi-company mode.
   */
  @Field<Company>({
    type: 'ManyToManyRef',
    relation: { targetModel: 'base.Company' },
    condition: ['IsActive', '=', true],
    string: _lt('Accessible Companies', { scope: 'auth.model.User.fields' }),
    help: _lt('Companies the user may switch to; must stay within role assignments.', {
      scope: 'auth.model.User.fields',
    }),
  })
  CompanyIds: string[];

  /**
   * Whether the account is currently active.
   */
  @Field({
    type: 'boolean',
    default: () => true,
    index: true,
    string: _lt('Active', { scope: 'auth.model.User.fields' }),
  })
  IsActive: boolean;

  /**
   * Timestamp of the most recent successful login.
   */
  @Field({
    type: 'datetime',
    index: true,
    copy: false,
    string: _lt('Last Login', { scope: 'auth.model.User.fields' }),
  })
  LastLogin: Date;

  /**
   * Sessions currently associated with the user.
   */
  @Field({
    type: 'OneToMany',
    relation: { targetModel: () => Session, inverseField: 'UserId' },
    copy: false,
    string: _lt('Sessions', { scope: 'auth.model.User.fields' }),
  })
  Sessions: Session[];

  /**
   * Tokens currently associated with the user.
   */
  @Field({
    type: 'OneToMany',
    relation: { targetModel: () => Token, inverseField: 'UserId' },
    copy: false,
    string: _lt('Tokens', { scope: 'auth.model.User.fields' }),
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
    condition: ['IsActive', '=', true],
    string: _lt('Roles', { scope: 'auth.model.User.fields' }),
  })
  Roles: Role[];

  /**
   * Cleared / blank timezone is stored as null; non-empty values must be valid IANA ids.
   */
  @Constraint<User>(['Timezone'])
  validateTimezoneConstraint(): void {
    const raw = this.Timezone;
    if (raw == null || !String(raw).trim()) {
      this.Timezone = null;
      return;
    }
    const timezone = String(raw).trim();
    if (!isIanaTimezone(timezone)) {
      throw newAuthError({
        code: AuthErrCode.VALIDATION_FAILED,
        message: _t('Invalid IANA timezone: %s', { scope: 'service/models/user' }, timezone),
      }).withGrpcCode(GrpcCode.InvalidArgument);
    }
    this.Timezone = timezone;
  }

  /**
   * User.Language is a POSIX terminology code that must reference an active base.Language row.
   */
  @Constraint<User>(['Language'])
  async validateLanguageConstraint(): Promise<void> {
    const raw = this.Language as any;
    if (raw == null || !String(raw).trim()) {
      (this as any).Language = null;
      return;
    }
    const code = String(raw).trim();
    const active = await Language.Search(
      {
        And: [
          ['Code', '=', code],
          ['IsActive', '=', true],
        ],
      } as any,
      { fields: ['Code'], limit: 1 } as any
    );
    if (!active?.length) {
      throw newAuthError({
        code: AuthErrCode.VALIDATION_FAILED,
        message: _t('Invalid or inactive language: %s', { scope: 'service/models/user' }, code),
      }).withGrpcCode(GrpcCode.InvalidArgument);
    }
    (this as any).Language = code;
  }

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

      // D20: persist browser IANA when registration left Timezone empty.
      await persistBrowserTimezoneIfEmpty(
        { Id: userId, Timezone: (created as any)?.Timezone ?? (userData as any)?.Timezone } as any,
        {
          updateTimezone: async (uid, timezone) => {
            await this.UpdateById(uid, { Timezone: timezone } as any, ['Id'] as any);
          },
        }
      );

      return userId;
    } catch (error) {
      throw wrapAuthError(error, {
        code: AuthErrCode.USER_CREATION_FAILED,
        message: _t('User registration failed', { scope: 'service/models/user@Register' }),
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
      // D20: first login with empty User.Timezone + baggage clientTz → persist and refresh metadata.
      const loginUser = await persistBrowserTimezoneIfEmpty(user as any, {
        updateTimezone: async (uid, timezone) => {
          await this.UpdateById(uid, { Timezone: timezone } as any);
        },
        reloadUser: async uid => (await this.Browse(uid)) as any,
      });

      return await issueLoginTokensAndSession(
        loginUser as any,
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
        message: _t('Login failed: unable to create token pair', { scope: 'service/models/user@Login' }),
      }).withMetadata({ userId: user.Id, username: user.Username });
    }
  }

  /**
   * Build token metadata from the current user record and company scope.
   */
  static async extractUserMetadata(user: User): Promise<TokenMetadata> {
    const userId = String((user as any)?.Id || '').trim();
    const userVersion = Number(new Date((user as any)?.UpdatedAt || Date.now()));
    const permStateVersion = userId ? await computePermStateVersion(userId) : 0;
    const companyScope = computeTokenCompanyScope(user as any);

    let companyTimezone: string | undefined;
    const activeCompanyId = String(companyScope.activeCompanyId || '').trim();
    if (activeCompanyId) {
      try {
        const CompanyService = createServiceByModel<typeof Company>('base.Company');
        const company = await (CompanyService as any).Browse(activeCompanyId, ['Timezone']);
        const tz = String(company?.Timezone || '').trim();
        if (tz && isIanaTimezone(tz)) companyTimezone = tz;
      } catch {
        // Company may be unavailable during early bootstrap; leave companyTimezone unset.
      }
    }

    return {
      language: user.Language,
      timezone: user.Timezone || undefined,
      companyTimezone,
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
        message: _t('Token refresh failed', { scope: 'service/models/user' }),
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
        message: _t('User is not logged in', { scope: 'service/models/user' }),
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
        message: _t('activeCompanyId cannot be empty', { scope: 'service/models/user' }),
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
            message: _t('enabledCompanyIds must be a string[] or omitted', { scope: 'service/models/user' }),
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
            message: _t('enabledCompanyIds contains an unauthorized company', { scope: 'service/models/user' }),
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
            message: _t('activeCompanyId is outside the allowed company scope', { scope: 'service/models/user' }),
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
          message: _t('activeCompanyId must be included in enabledCompanyIds', { scope: 'service/models/user' }),
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
        message: _t('Switch company scope failed', { scope: 'service/models/user' }),
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
        message: _t('Token is required', { scope: 'service/models/user' }),
      }).withGrpcCode(GrpcCode.InvalidArgument);
    }

    try {
      await revokeLogoutArtifacts(token, allDevices);

      return true;
    } catch (error) {
      throw wrapAuthError(error, {
        code: AuthErrCode.TOKEN_REVOCATION_FAILED,
        message: _t('Logout failed', { scope: 'service/models/user' }),
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
        const permStateVersion = await computePermStateVersion(userId);
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

    const KEY = buildAuthzContextCacheKey(userId, companyScopeKey);

    return await memoizeInReqState(state, KEY, async () =>
      buildAuthzContext({ userId, activeCompanyId: companyScope.activeCompanyId, enabledCompanyIds: enabledCompanyIdsKey })
    );
  }

  /**
   * Check whether the user can call a gRPC service method under P1 ACL rules.
   *
   * Keep ACL evaluation centralized in auth so other modules do not query auth tables directly.
   * This path is deny-by-default, deny-wins, and admin-aware.
   *
   * PR-E-5: returns `{ allowed, reason, hitRuleIds }` so the Go guard / audit log can record
   * the same diagnostics vocabulary as RR/FR. Callers that only need a boolean should read `.allowed`.
   */
  static async CheckMethodAccess(companyId: string, serviceFullName: string): Promise<MethodAccessDecision> {
    const deny = (reason: string, hitRuleIds: string[]): MethodAccessDecision => ({
      allowed: false,
      reason,
      hitRuleIds,
    });
    const allow = (reason: string, hitRuleIds: string[]): MethodAccessDecision => ({
      allowed: true,
      reason,
      hitRuleIds,
    });

    try {
      const userId = this.userId;
      if (!userId || !serviceFullName) return deny('missing_identity_or_method', []);

      const parsed = parseServiceFullName(serviceFullName);
      if (!parsed) return deny('invalid_service_full_name', []);
      const { appName, modelName, methodName } = parsed;
      const normalizedFullMethod = `/${appName}.${modelName}/${methodName}`;

      const normalizedCompanyId = String(companyId || '').trim();
      const hasCompany = normalizedCompanyId.length > 0;
      if (!hasCompany) return deny('missing_company_id', []);

      // Company view must be within enabledCompanyIds (fail-closed).
      const { enabledCompanyIds } = getCompanyScopeFromRequestContext();
      if (enabledCompanyIds.length > 0 && !enabledCompanyIds.includes(normalizedCompanyId)) {
        return deny('company_not_in_enabled_scope', []);
      }

      // Permission graph reads must bypass RecordRule/FieldRule.
      // Otherwise a request with activeCompanyId=c1 can never evaluate roles scoped to c2,
      // because UserRole/RoleMethodAccess reads may be fail-closed by RecordRule injection.
      return await withPermissionGraphBypass(async () => {
        const authz = await this._getAuthzContext();
        const roleIds = authz.rolesByCompany?.[normalizedCompanyId] || [];
        if (roleIds.length === 0) return deny('no_roles_for_company', []);

        const req = getCurrentReq();
        const state = getOrInitReqServiceState(req);
        const cacheKey = buildMethodAccessCacheKey(String(authz.userId || '').trim(), normalizedCompanyId, normalizedFullMethod);
        const cached = state?.[cacheKey];
        if (cached && typeof cached === 'object' && typeof (cached as MethodAccessDecision).allowed === 'boolean') {
          return cached as MethodAccessDecision;
        }
        // Legacy boolean cache entries (pre-E-5) — recompute with diagnostics.
        if (typeof cached === 'boolean') {
          delete state[cacheKey];
        }

        const accessMeta = await resolveMethodAccessMeta(appName, modelName, methodName);
        if (!accessMeta) return deny('method_meta_not_found', []);

        const accessResult = await evaluateRoleMethodAccess(roleIds, accessMeta.scopeOr, accessMeta.methodLower);

        if (accessResult.denied) {
          const decision = deny(accessResult.reason, accessResult.hitRuleIds);
          if (state) state[cacheKey] = decision;
          return decision;
        }

        if (accessResult.allowed) {
          const decision = allow(accessResult.reason, accessResult.hitRuleIds);
          if (state) state[cacheKey] = decision;
          return decision;
        }

        const uiDecision = await evaluateUiDerivedMethodDecision(roleIds, accessMeta.modelKey, accessMeta.methodLower);
        if (uiDecision.denied) {
          const decision = deny(uiDecision.reason, uiDecision.hitRuleIds);
          if (state) state[cacheKey] = decision;
          return decision;
        }

        const decision = uiDecision.allowed
          ? allow(uiDecision.reason, uiDecision.hitRuleIds)
          : deny(uiDecision.reason, uiDecision.hitRuleIds);
        if (state) state[cacheKey] = decision;
        return decision;
      });
    } catch {
      // fail-closed
      return deny('internal_error', []);
    }
  }

  /**
   * Get the effective RecordRule condition for the current user, model, and operation.
   *
   * Notes:
   * - userId comes from the current request identity.
   * - model should be "<app>.<Model>" such as "base.Company".
   * - op should be "read" | "write" | "create" | "delete".
   * - returns a ConditionEnvelope (proto message; expr may still be Value-shaped).
   */
  static async GetRecordRuleCondition(model: string, op: string): Promise<ConditionEnvelope> {
    try {
      const userId = this.userId;
      if (!userId) {
        return { kind: 'false', reason: 'missing_identity_user_id' };
      }

      const modelFullName = String(model || '').trim();
      const rawOp = String(op || '')
        .trim()
        .toLowerCase();
      if (!modelFullName) return { kind: 'false', reason: 'missing_model' };

      const opValue = rawOp as RecordRuleOp;
      if (!['read', 'write', 'create', 'delete'].includes(opValue)) {
        return { kind: 'false', reason: 'invalid_op' };
      }

      const parsedModel = parseModelFullName(modelFullName);
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
   * - returns a FieldRuleSpec protobuf message.
   */
  static async GetFieldRuleSpec(model: string): Promise<FieldRuleSpec> {
    try {
      const userId = this.userId;
      if (!userId) {
        throw newAuthError({
          code: AuthErrCode.VALIDATION_FAILED,
          message: _t('Missing identity information (userId)', { scope: 'service/models/user' }),
        }).withGrpcCode(GrpcCode.Unauthenticated);
      }

      const modelFullName = String(model || '').trim();
      if (!modelFullName) {
        throw newAuthError({
          code: AuthErrCode.VALIDATION_FAILED,
          message: _t('Missing model name (model)', { scope: 'service/models/user' }),
        }).withGrpcCode(GrpcCode.InvalidArgument);
      }

      const parsedModel = parseModelFullName(modelFullName);
      if (!parsedModel) {
        throw newAuthError({
          code: AuthErrCode.VALIDATION_FAILED,
          message: _t('Invalid model full name; expected "<app>.<Model>"', { scope: 'service/models/user' }),
        })
          .withGrpcCode(GrpcCode.InvalidArgument)
          .withMetadata({ model: modelFullName });
      }

      return await withPermissionGraphBypass(async () => {
        const authz = await this._getAuthzContext();
        const roleIds = Array.isArray(authz?.roles) ? authz.roles : [];
        return await evaluateFieldRules({
          appName: parsedModel.appName,
          modelName: parsedModel.modelName,
          modelFullName,
          roleIds,
        });
      });
    } catch (error) {
      throw wrapAuthError(error, {
        code: AuthErrCode.UNKNOWN,
        message: _t('Failed to compute field rule spec', { scope: 'service/models/user' }),
      }).withMetadata({ model: String(model || '').trim() });
    }
  }
}
