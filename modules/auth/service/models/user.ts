// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { BaseModel, Decimal, Model, Field } from '@/core/service';
import { Onchange } from '@/core/service/api/onchange';
import { getIdentity, getReadonlyCtx, withContext, getCurrentReq, getOrInitReqServiceState, getJsCtxAndReq } from '@/core/service/api/context';
import type { Insertable } from '@/core/service/api/input';
import type { ConditionEnvelope, RecordRuleOp } from '@/core/service/api/authz';
import { ChoysumError } from '@/core/service/error';
import { createServiceByModel } from '@/core/service/rpc';
import { newAuthError, wrapAuthError, GrpcCode, AuthErrCode } from '../error';
import Session from './session';
import Role from './role';
import RoleMethodAccess from './role_method_access';
import RoleUiResource from './role_ui_resource';
import RoleRecordRule from './role_record_rule';
import RoleInheritance from './role_inheritance';
import RoleFieldRule from './role_field_rule';
import Token from './token';
import UserRole from './user_role';
import IrUiResource from '@/meta/service/models/ir_ui_resource';
import IrUiResourceMenuRoute from '@/meta/service/models/ir_ui_resource_menu_route';
import IrUiResourceRouteAction from '@/meta/service/models/ir_ui_resource_route_action';
import type IrApplicationModel from '@/meta/service/models/ir_application';
import type IrFieldModel from '@/meta/service/models/ir_field';
import type IrModelModel from '@/meta/service/models/ir_model';
import type IrServiceModel from '@/meta/service/models/ir_service';
import type Company from '@/base/service/models/company';
import { parseModelFullName, parseServiceFullName } from '@/core/service/utils/model_parsing';
import { uniqStrings, normalizeRpcRequireKey, rpcServiceWildcard } from '@/core/service/utils/normalization';
import { buildAuthzContextCacheKey, buildMethodAccessCacheKey, buildUiGrantCacheKey } from './_request_cache_invalidation';

const IrApplication = createServiceByModel<typeof IrApplicationModel>('meta.IrApplication');
const IrField = createServiceByModel<typeof IrFieldModel>('meta.IrField');
const IrModel = createServiceByModel<typeof IrModelModel>('meta.IrModel');
const IrService = createServiceByModel<typeof IrServiceModel>('meta.IrService');
const CompanyService = createServiceByModel<typeof Company>('base.Company');

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
    // Validate the minimum required registration fields.
    if (!userData.Username || !password) {
      throw newAuthError({
        code: AuthErrCode.VALIDATION_FAILED,
        message: 'Username and password are required',
      }).withGrpcCode(GrpcCode.InvalidArgument);
    }

    // Enforce username uniqueness.
    const existing = await this.Search(['Username', '=', userData.Username]);
    if (existing.length > 0) {
      throw newAuthError({
        code: AuthErrCode.USERNAME_TAKEN,
        message: 'Username is already in use',
      })
        .withGrpcCode(GrpcCode.AlreadyExists)
        .withMetadata({ username: userData.Username });
    }

    // Enforce email uniqueness when an email was provided.
    if (userData.Email) {
      const emailExists = await this.Search(['Email', '=', userData.Email]);
      if (emailExists.length > 0) {
        throw newAuthError({
          code: AuthErrCode.EMAIL_TAKEN,
          message: 'Email is already registered',
        })
          .withGrpcCode(GrpcCode.AlreadyExists)
          .withMetadata({ email: userData.Email });
      }
    }

    try {
      // Create the user record first so follow-up provisioning has a stable id.
      const created = await this.Create({
        ...userData,
        PasswordHash: this._hashPassword(password),
      });

      const userId = String((created as any)?.Id || '').trim();
      if (!userId) {
        throw newAuthError({
          code: AuthErrCode.USER_CREATION_FAILED,
          message: 'User registration failed: missing user id',
        }).withGrpcCode(GrpcCode.Internal);
      }

      // Provision the default company and base role after anonymous registration.
      // This flow reads permission graph models before a stable request identity exists,
      // so it bypasses RecordRule and FieldRule checks and attaches explicit company scope
      // before writing company-scoped data.
      await this._withPermissionGraphBypass(async () => {
        const main = await CompanyService.Search(['Code', '=', 'MAIN'] as any, { fields: ['Id'], limit: 1 } as any);
        const mainCompanyId = String((main as any)?.[0]?.Id || '').trim();
        if (!mainCompanyId) {
          throw newAuthError({
            code: AuthErrCode.VALIDATION_FAILED,
            message: 'Registration failed: Main Company (Code=MAIN) was not found',
          }).withGrpcCode(GrpcCode.FailedPrecondition);
        }

        await withContext({ activeCompanyId: mainCompanyId, enabledCompanyIds: [mainCompanyId] }, async () => {
          const mergedPrefs: any = (userData as any)?.Preferences && typeof (userData as any).Preferences === 'object' ? (userData as any).Preferences : {};
          mergedPrefs.activeCompanyId = mainCompanyId;
          mergedPrefs.enabledCompanyIds = [mainCompanyId];

          await this.UpdateById(
            userId,
            {
              CompanyId: mainCompanyId,
              CompanyIds: [mainCompanyId],
              Preferences: mergedPrefs,
            } as any,
            ['Id'] as any
          );

          const baseRole = await Role.Search(
            ['Code', '=', 'base.user'] as any,
            {
              fields: ['Id'],
              limit: 1,
            } as any
          );
          const baseUserRoleId = String((baseRole as any)?.[0]?.Id || '').trim();
          if (!baseUserRoleId) {
            throw newAuthError({
              code: AuthErrCode.ROLE_NOT_FOUND,
              message: 'Registration failed: base.user role is not initialized; load auth bootstrap data first',
            })
              .withGrpcCode(GrpcCode.FailedPrecondition)
              .withMetadata({ roleCode: 'base.user', externalId: 'auth.role_base_user' });
          }

          const existingUserRole = await UserRole.Search(
            {
              And: [
                ['UserId', '=', userId],
                ['RoleId', '=', baseUserRoleId],
                ['CompanyId', '=', mainCompanyId],
              ],
            } as any,
            { fields: ['Id'], limit: 1 } as any
          );
          if (!existingUserRole || existingUserRole.length === 0) {
            await UserRole.Create(
              {
                UserId: { Id: userId } as any,
                RoleId: { Id: baseUserRoleId } as any,
                CompanyId: mainCompanyId,
              } as any,
              ['Id'] as any
            );
          }

          // Ensure base.user has the minimum RPC allowlist required by the web bootstrap path.
          await this._ensureBaseUserRpcAllow(baseUserRoleId);
        });
      });

      return userId;
    } catch (error) {
      throw wrapAuthError(error, {
        code: AuthErrCode.USER_CREATION_FAILED,
        message: 'User registration failed',
      }).withMetadata({ username: userData.Username });
    }
  }

  /**
   * Ensure the base.user role keeps the minimum RPC allowlist for web auth flows.
   */
  private static async _ensureBaseUserRpcAllow(roleId: string): Promise<void> {
    const rid = String(roleId || '').trim();
    if (!rid) return;

    const targets: Array<{ app: string; model: string; method: string }> = [
      { app: 'auth', model: 'User', method: 'Browse' },
      { app: 'auth', model: 'User', method: 'GetPermissionState' },
      { app: 'auth', model: 'User', method: 'SwitchCompanyScope' },
      { app: 'base', model: 'Company', method: 'Search' },
    ];

    const resolveModelId = async (app: string, name: string): Promise<string> => {
      const hit = (
        await IrModel.Search(
          {
            And: [
              ['Application', '=', app],
              ['Name', '=', name],
            ],
          } as any,
          { fields: ['Id'], limit: 1 } as any
        )
      )?.[0] as any;
      return String(hit?.Id || '').trim();
    };

    const resolveService = async (modelId: string, method: string): Promise<{ id: string; name: string }> => {
      const rows = await IrService.Search({ And: [['ModelId', '=', modelId]] } as any, { fields: ['Id', 'Name'], limit: 5000 } as any);
      const target = String(method || '')
        .trim()
        .toLowerCase();
      const hit = (rows || []).find(
        (r: any) =>
          String((r as any).Name || '')
            .trim()
            .toLowerCase() === target
      ) as any;
      const id = String(hit?.Id || '').trim();
      const name = String(hit?.Name || '').trim();
      return { id, name };
    };

    for (const t of targets) {
      const modelId = await resolveModelId(t.app, t.model);
      if (!modelId) continue;

      const svc = await resolveService(modelId, t.method);
      if (!svc.id) continue;

      const existing = await RoleMethodAccess.Search(
        {
          And: [
            ['RoleId', '=', rid],
            ['IrServiceId', '=', svc.id],
            ['IrModelId', 'is', null],
            ['IrApplicationId', 'is', null],
          ],
        } as any,
        { fields: ['Id', 'Mode'], limit: 1 } as any
      );

      if (!existing || existing.length === 0) {
        await RoleMethodAccess.Create(
          {
            RoleId: { Id: rid } as any,
            IrServiceId: svc.id,
            IrModelId: null,
            IrApplicationId: null,
            Mode: 'allow',
          } as any,
          ['Id'] as any
        );
      } else {
        const row: any = existing[0];
        const id = String(row?.Id || '').trim();
        const mode = String(row?.Mode || '').toLowerCase();
        if (id && mode !== 'allow') {
          await RoleMethodAccess.UpdateById(
            id,
            {
              IrServiceId: svc.id,
              IrModelId: null,
              IrApplicationId: null,
              Mode: 'allow',
            } as any,
            ['Id'] as any
          );
        }
      }
    }
  }

  /**
   * Authenticate a local user and create a token pair.
   */
  static async Login(usernameOrEmail: string, password: string, ipAddress?: string, deviceInfo?: string, rememberMe?: boolean): Promise<TokenPair> {
    // Resolve the user by username or email.

    const users = await this.Search({
      Or: [
        ['Username', '=', usernameOrEmail],
        ['Email', '=', usernameOrEmail],
      ],
    });

    if (users.length === 0) {
      throw newAuthError({
        code: AuthErrCode.USER_NOT_FOUND,
        message: 'User not found or password is incorrect',
      }).withGrpcCode(GrpcCode.NotFound);
    }

    const user = users[0];

    // Verify the supplied password against the stored hash.
    if (!this._verifyPassword(password, user.PasswordHash)) {
      throw newAuthError({
        code: AuthErrCode.INVALID_PASSWORD,
        message: 'User not found or password is incorrect',
      })
        .withGrpcCode(GrpcCode.Unauthenticated)
        .withMetadata({ username: usernameOrEmail });
    }

    // Reject inactive accounts before token issuance.
    if (!user.IsActive) {
      throw newAuthError({
        code: AuthErrCode.ACCOUNT_DISABLED,
        message: 'Account is disabled',
      })
        .withGrpcCode(GrpcCode.PermissionDenied)
        .withMetadata({ userId: user.Id, username: user.Username });
    }

    try {
      // Load CompanyIds explicitly before building token metadata.
      await user.load(['CompanyIds']);
      const metadata = await this.extractUserMetadata(user);

      // Issue the token pair with the latest metadata snapshot.
      const tokens = await Token.CreateTokenPair(user.Id, metadata);

      // Update the login timestamp after successful issuance.
      await this.UpdateById(user.Id, { LastLogin: new Date() });

      // Persist a session record when request context details are available.
      if (ipAddress || deviceInfo) {
        // P0.2 stores only the access token id (jti), never the raw access JWT.
        const accessIdentity = $choysum.auth.validateToken(tokens.accessToken, 'access', false);
        await Session.Create({
          UserId: { Id: user.Id },
          AccessTokenId: String(accessIdentity?.tokenId || '').trim(),
          DeviceInfo: deviceInfo || '',
          IpAddress: ipAddress || '',
          ExpiresAt: new Date(tokens.expiresAt),
          LastActivityAt: new Date(),
          Status: 'active',
          Metadata: {
            rememberMe: !!rememberMe,
            createdAt: Date.now(),
          },
        });
      }

      return tokens;
    } catch (error) {
      throw wrapAuthError(error, {
        code: AuthErrCode.TOKEN_CREATION_FAILED,
        message: 'Login failed: unable to create token pair',
      }).withMetadata({ userId: user.Id, username: user.Username });
    }
  }

  /**
   * Normalize a model reference or scalar value into a string id.
   */
  private static _maybeId(value: any): string | undefined {
    if (!value) return undefined;
    if (typeof value === 'string') return value;
    if (typeof value === 'object') return String((value as any).Id ?? (value as any).id ?? '').trim() || undefined;
    return undefined;
  }

  /**
   * Build token metadata from the current user record and company scope.
   */
  static async extractUserMetadata(user: User): Promise<TokenMetadata> {
    const userId = String((user as any)?.Id || '').trim();
    const userVersion = Number(new Date((user as any)?.UpdatedAt || Date.now()));
    const permStateVersion = userId ? await this._computePermStateVersion(userId) : 0;

    const normalizeId = (v: any): string => {
      const id = this._maybeId(v);
      if (id) return String(id).trim();
      return String(v ?? '').trim();
    };
    const uniq = (xs: string[]): string[] => Array.from(new Set(xs));

    const allowedCompanyIds = uniq(
      [normalizeId((user as any).CompanyId), ...(Array.isArray((user as any).CompanyIds) ? (user as any).CompanyIds : [])].map(normalizeId).filter(Boolean)
    );

    const prefs: any = (user as any)?.Preferences || {};
    const prefActive = typeof prefs?.activeCompanyId === 'string' ? normalizeId(prefs.activeCompanyId) : '';
    const prefEnabled = Array.isArray(prefs?.enabledCompanyIds) ? prefs.enabledCompanyIds.map(normalizeId).filter(Boolean) : [];

    let activeCompanyId = '';
    if (prefActive && allowedCompanyIds.includes(prefActive)) activeCompanyId = prefActive;
    else {
      const fallback = normalizeId((user as any).CompanyId);
      if (fallback && allowedCompanyIds.includes(fallback)) activeCompanyId = fallback;
      else if (allowedCompanyIds.length) activeCompanyId = allowedCompanyIds[0];
    }

    let enabledCompanyIds = prefEnabled.filter((cid: string) => allowedCompanyIds.includes(cid));
    if (!enabledCompanyIds.length && activeCompanyId) enabledCompanyIds = [activeCompanyId];
    if (activeCompanyId && !enabledCompanyIds.includes(activeCompanyId)) enabledCompanyIds = [activeCompanyId, ...enabledCompanyIds];
    if (!enabledCompanyIds.length && allowedCompanyIds.length) enabledCompanyIds = [allowedCompanyIds[0]];

    return {
      language: user.Language,
      timezone: user.Timezone,
      allowedCompanyIds,
      activeCompanyId: activeCompanyId || undefined,
      enabledCompanyIds,
      userVersion,
      permStateVersion,
    };
  }

  /**
   * Refresh a token pair using the latest user metadata snapshot.
   */
  static async RefreshTokens(refreshToken: string): Promise<TokenPair> {
    try {
      // Resolve the user id from the validated refresh token.
      const id = await Token.ValidateToken(refreshToken, 'refresh');
      const userId = id.userId;

      // Reload the user record and rebuild metadata before issuing fresh tokens.
      const user = await this.Browse(userId);
      await user.load(['CompanyIds']);
      const metadata = await this.extractUserMetadata(user);

      // Pass the refreshed metadata through to token issuance.
      return await Token.RefreshTokens(refreshToken, metadata);
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

    let auditEmitted = false;
    const emitAudit = (payload: Record<string, any>): void => {
      try {
        const { req } = getJsCtxAndReq();
        const traceId = typeof req?.traceId === 'string' ? req.traceId : '';
        const out = {
          event: 'auth.user.switch_company_scope',
          traceId,
          ...payload,
        };
        if (payload?.ok === false) {
          console.warn(`[AUDIT] ${JSON.stringify(out)}`);
        } else {
          console.info(`[AUDIT] ${JSON.stringify(out)}`);
        }
      } catch {
        try {
          if (payload?.ok === false) {
            console.warn('[AUDIT] auth.user.switch_company_scope');
          } else {
            console.info('[AUDIT] auth.user.switch_company_scope');
          }
        } catch {}
      }
    };

    const emitAuditOnce = (payload: Record<string, any>): void => {
      if (auditEmitted) return;
      auditEmitted = true;
      emitAudit(payload);
    };

    const normalizeId = (v: any): string => String(v ?? '').trim();
    const uniq = (xs: string[]): string[] => Array.from(new Set(xs));
    const active = normalizeId(activeCompanyId);
    if (!active) {
      emitAuditOnce({
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

      const allowed = uniq(
        [normalizeId((user as any).CompanyId), ...(Array.isArray((user as any).CompanyIds) ? (user as any).CompanyIds : [])].map(normalizeId).filter(Boolean)
      );

      const normalizeJsonObject = (v: any): any => {
        if (!v) return {};
        if (typeof v === 'string') {
          const s = v.trim();
          if (!s) return {};
          try {
            const parsed = JSON.parse(s);
            return parsed && typeof parsed === 'object' && !Array.isArray(parsed) ? parsed : {};
          } catch {
            return {};
          }
        }
        if (typeof v === 'object') {
          // Prefer a serializable snapshot to handle proxies and special runtime objects.
          try {
            const snap = JSON.parse(JSON.stringify(v));
            if (snap && typeof snap === 'object' && !Array.isArray(snap)) return snap;
          } catch {
            // Ignore snapshot failures and fall back to the original object.
          }
          return v;
        }
        return {};
      };

      const prefs: any = normalizeJsonObject((user as any)?.Preferences);
      const prefEnabled = Array.isArray(prefs?.enabledCompanyIds) ? prefs.enabledCompanyIds.map(normalizeId).filter(Boolean) : [];

      let enabled: string[];
      if (enabledCompanyIds === undefined || enabledCompanyIds === null) {
        enabled = prefEnabled.length ? prefEnabled : [active];
      } else if (Array.isArray(enabledCompanyIds)) {
        enabled = enabledCompanyIds.map(normalizeId).filter(Boolean);
      } else {
        emitAuditOnce({
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

      enabled = uniq(enabled);

      // Ensure enabled companies stay within the allowed set.
      for (const cid of enabled) {
        if (!allowed.includes(cid)) {
          emitAuditOnce({
            ok: false,
            userId,
            targetActive: active,
            targetEnabled: enabled,
            reason: 'enabledCompanyIds contains an unauthorized company',
            companyId: cid,
          });
          throw newAuthError({
            code: AuthErrCode.VALIDATION_FAILED,
            message: 'enabledCompanyIds contains an unauthorized company',
          })
            .withGrpcCode(GrpcCode.InvalidArgument)
            .withMetadata({ companyId: cid });
        }
      }
      if (!allowed.includes(active)) {
        emitAuditOnce({
          ok: false,
          userId,
          targetActive: active,
          targetEnabled: enabled,
          reason: 'activeCompanyId is outside the allowed company scope',
        });
        throw newAuthError({
          code: AuthErrCode.VALIDATION_FAILED,
          message: 'activeCompanyId is outside the allowed company scope',
        })
          .withGrpcCode(GrpcCode.InvalidArgument)
          .withMetadata({ activeCompanyId: active });
      }

      // Ensure the active company is present in the enabled set.
      if (!enabled.includes(active)) {
        emitAuditOnce({
          ok: false,
          userId,
          targetActive: active,
          targetEnabled: enabled,
          reason: 'activeCompanyId must be included in enabledCompanyIds',
        });
        throw newAuthError({
          code: AuthErrCode.VALIDATION_FAILED,
          message: 'activeCompanyId must be included in enabledCompanyIds',
        }).withGrpcCode(GrpcCode.InvalidArgument);
      }

      // Write Preferences back while preserving other UI settings.
      // jsonobject fields may be backed by proxies or special runtime objects, so the
      // serializable snapshot above acts as the merge baseline.
      const basePrefs: any = prefs && typeof prefs === 'object' && !Array.isArray(prefs) ? prefs : {};
      const nextPrefs: any = {
        ...basePrefs,
        activeCompanyId: active,
        enabledCompanyIds: enabled,
      };

      // NOTE: Switching company scope is a self-service auth operation.
      // Users may legitimately have zero roles at this stage; record rules are deny-by-default for write.
      // We bypass RecordRule/FieldRule for this internal preference update (still guarded by strict validation above).
      await this._withPermissionGraphBypass(async () => {
        await this.UpdateById(userId, { Preferences: nextPrefs } as any);
      });

      // Reload the record so UpdatedAt and metadata reflect the persisted preferences.
      const updated = await this.Browse(userId);
      await updated.load(['CompanyIds']);
      const metadata = await this.extractUserMetadata(updated);

      emitAuditOnce({
        ok: true,
        userId,
        targetActive: active,
        targetEnabled: enabled,
        reason: 'ok',
      });
      return await Token.CreateTokenPair(userId, metadata);
    } catch (error) {
      // Emit an audit failure record without changing error semantics.
      if (!auditEmitted) {
        try {
          const msg = typeof (error as any)?.message === 'string' ? (error as any).message : String(error);
          const code = (error as any)?.code ? String((error as any).code) : undefined;
          emitAuditOnce({
            ok: false,
            userId,
            targetActive: active,
            targetEnabled: Array.isArray(enabledCompanyIds) ? enabledCompanyIds.map(normalizeId).filter(Boolean) : null,
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
      // Validate and decode the current access token.
      const identity = await Token.ValidateToken(token, 'access');
      const userId = identity.userId;

      if (allDevices) {
        // Revoke every token owned by the current user.
        await Token.RevokeAllUserTokens(userId, undefined, 'User initiated logout on all devices');

        // Revoke every session owned by the current user.
        await Session.RevokeAllForUser(userId);
      } else {
        // Revoke only the current access token.
        await Token.RevokeToken(token, 'User initiated logout');

        // Revoke the linked session when one exists for this access token id.
        try {
          const tokenId = String((identity as any)?.tokenId || '').trim();
          const sessions = tokenId ? await Session.Search(['AccessTokenId', '=', tokenId] as any) : [];
          if (sessions.length > 0) {
            await Session.RevokeSession(sessions[0].Id);
          }
        } catch (error) {
          // Ignore lookup failures because a session record may not exist.
        }
      }

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

      return await this._withPermissionGraphBypass(async () => {
        const permStateVersion = await this._computePermStateVersion(userId);

        const authz = await this._getAuthzContext();

        // 1) Compute effective roles, including inherited role closure and company scope.
        const roleScopesById = authz.roleScopesById;
        const roleIds = authz.roleIds;
        const byCompany: Record<string, { ui?: { routes?: string[]; menus?: string[]; actions?: string[] } }> = {};
        const ensureBucket = (k: string) => {
          if (!byCompany[k]) byCompany[k] = { ui: { routes: [], menus: [], actions: [] } };
          if (!byCompany[k].ui) byCompany[k].ui = { routes: [], menus: [], actions: [] };
          if (!byCompany[k].ui!.routes) byCompany[k].ui!.routes = [];
          if (!byCompany[k].ui!.menus) byCompany[k].ui!.menus = [];
          if (!byCompany[k].ui!.actions) byCompany[k].ui!.actions = [];
          return byCompany[k];
        };

        ensureBucket('*');
        for (const cid of authz.enabledCompanyIds || []) ensureBucket(cid);

        if (roleIds.length === 0) {
          return { permStateVersion, byCompany };
        }

        // 2) Compute effective RPC allow and deny sets per company for UI require checks.
        const accesses = await RoleMethodAccess.Search(['RoleId', 'in', roleIds] as any, {
          fields: ['RoleId', 'IrServiceId', 'IrModelId', 'IrApplicationId', 'Mode'],
          limit: 50000,
        });

        const applyToScope = (roleId: string, fn: (companyKey: string) => void) => {
          const scope = (roleScopesById as any)?.[roleId] as { global: boolean; companies?: string[] } | undefined;
          if (!scope) return;
          if (scope.global) fn('*');
          else for (const cid of scope.companies || []) fn(cid);
        };

        type ServiceAgg = { allow: Set<string>; deny: Set<string>; allowAll: boolean; denyAll: boolean };
        const aggByCompany = new Map<string, Map<string, ServiceAgg>>(); // companyKey -> serviceFullName -> agg
        const companyGlobalAllow = new Set<string>();
        const companyGlobalDeny = new Set<string>();

        const ensureAgg = (companyKey: string, serviceFullName: string): ServiceAgg => {
          let m = aggByCompany.get(companyKey);
          if (!m) {
            m = new Map();
            aggByCompany.set(companyKey, m);
          }
          let a = m.get(serviceFullName);
          if (!a) {
            a = { allow: new Set(), deny: new Set(), allowAll: false, denyAll: false };
            m.set(serviceFullName, a);
          }
          return a;
        };

        const irServiceIds = Array.from(new Set((accesses || []).map(a => String((a as any).IrServiceId || '').trim()).filter(Boolean)));
        const irModelIds = Array.from(new Set((accesses || []).map(a => String((a as any).IrModelId || '').trim()).filter(Boolean)));
        const irApplicationIds = Array.from(new Set((accesses || []).map(a => String((a as any).IrApplicationId || '').trim()).filter(Boolean)));

        const serviceById = new Map<string, { modelId: string; name: string }>();
        const modelById = new Map<string, { app: string; name: string }>();

        // 4.1) Resolve service -> model + method
        const modelIdsFromServices = new Set<string>();
        if (irServiceIds.length > 0) {
          const services = await IrService.Search(['Id', 'in', irServiceIds] as any, { fields: ['Id', 'ModelId', 'Name'], limit: 50000 });
          for (const s of services || []) {
            const sid = String((s as any).Id || '').trim();
            const mid = this._maybeId((s as any).ModelId);
            const name = String((s as any).Name || '').trim();
            if (!sid || !mid || !name) continue;
            serviceById.set(sid, { modelId: mid, name });
            modelIdsFromServices.add(mid);
          }
        }

        // 4.2) Resolve model -> app + name
        const needModelIds = Array.from(new Set([...irModelIds, ...Array.from(modelIdsFromServices)]));
        if (needModelIds.length > 0) {
          const models = await IrModel.Search(['Id', 'in', needModelIds] as any, { fields: ['Id', 'Name', 'Application'], limit: 50000 });
          for (const m of models || []) {
            const mid = String((m as any).Id || '').trim();
            const app = String((m as any).Application || '').trim();
            const name = String((m as any).Name || '').trim();
            if (!mid || !app || !name) continue;
            modelById.set(mid, { app, name });
          }
        }

        // 4.3) Resolve applicationId -> applicationName, and applicationName -> models
        const appNameById = new Map<string, string>();
        if (irApplicationIds.length > 0) {
          const apps = await IrApplication.Search(['Id', 'in', irApplicationIds] as any, { fields: ['Id', 'Name'], limit: 50000 } as any);
          for (const a of apps || []) {
            const id = String((a as any).Id || '').trim();
            const name = String((a as any).Name || '').trim();
            if (!id || !name) continue;
            appNameById.set(id, name);
          }
        }

        const modelsByApp = new Map<string, Array<{ app: string; name: string }>>();
        const getModelsForApp = async (appName: string): Promise<Array<{ app: string; name: string }>> => {
          const k = String(appName || '').trim();
          if (!k) return [];
          const cached = modelsByApp.get(k);
          if (cached) return cached;
          const rows = await IrModel.Search(['Application', '=', k] as any, { fields: ['Application', 'Name'], limit: 50000 } as any);
          const out = (rows || [])
            .map((r: any) => ({ app: String((r as any).Application || '').trim(), name: String((r as any).Name || '').trim() }))
            .filter((r: { app: string; name: string }) => r.app && r.name);
          modelsByApp.set(k, out);
          return out;
        };

        let allModels: Array<{ app: string; name: string }> | undefined;
        const getAllModels = async (): Promise<Array<{ app: string; name: string }>> => {
          if (allModels) return allModels;
          const rows = await IrModel.Search([] as any, { fields: ['Application', 'Name'], limit: 50000 } as any);
          const out = (rows || [])
            .map((r: any) => ({ app: String((r as any).Application || '').trim(), name: String((r as any).Name || '').trim() }))
            .filter((r: { app: string; name: string }) => r.app && r.name);
          allModels = out;
          return out;
        };

        // 4.4) Apply rules into per-company aggregates
        for (const a of accesses || []) {
          const roleId = this._maybeId((a as any).RoleId);
          const sid = String((a as any).IrServiceId || '').trim();
          const mid = String((a as any).IrModelId || '').trim();
          const aid = String((a as any).IrApplicationId || '').trim();
          const mode = String((a as any).Mode || '').toLowerCase();
          if (!roleId || (mode !== 'allow' && mode !== 'deny')) continue;

          // global scope
          if (!sid && !mid && !aid) {
            applyToScope(roleId, companyKey => {
              if (mode === 'allow') companyGlobalAllow.add(companyKey);
              else companyGlobalDeny.add(companyKey);
            });
            const models = await getAllModels();
            applyToScope(roleId, companyKey => {
              for (const m of models) {
                const serviceFullName = `${m.app}.${m.name}`;
                const agg = ensureAgg(companyKey, serviceFullName);
                if (mode === 'allow') agg.allowAll = true;
                else agg.denyAll = true;
              }
            });
            continue;
          }

          // application scope
          if (!sid && !mid && aid) {
            const appName = appNameById.get(aid);
            if (!appName) continue;
            const models = await getModelsForApp(appName);
            applyToScope(roleId, companyKey => {
              for (const m of models) {
                const serviceFullName = `${m.app}.${m.name}`;
                const agg = ensureAgg(companyKey, serviceFullName);
                if (mode === 'allow') agg.allowAll = true;
                else agg.denyAll = true;
              }
            });
            continue;
          }

          // model scope
          if (!sid && mid && !aid) {
            const mdl = modelById.get(mid);
            if (!mdl) continue;
            const serviceFullName = `${mdl.app}.${mdl.name}`;
            applyToScope(roleId, companyKey => {
              const agg = ensureAgg(companyKey, serviceFullName);
              if (mode === 'allow') agg.allowAll = true;
              else agg.denyAll = true;
            });
            continue;
          }

          // service(method) scope
          if (sid && !mid && !aid) {
            const svc = serviceById.get(sid);
            if (!svc) continue;
            const mdl = modelById.get(svc.modelId);
            if (!mdl) continue;
            const serviceFullName = `${mdl.app}.${mdl.name}`;
            const methodName = svc.name;
            applyToScope(roleId, companyKey => {
              const agg = ensureAgg(companyKey, serviceFullName);
              if (mode === 'allow') agg.allow.add(methodName);
              else agg.deny.add(methodName);
            });
            continue;
          }
        }

        // 2.5) Synthesize requires keys per company for internal evaluation only.
        const requiresAllowKeysByCompany = new Map<string, Set<string>>();
        const requiresDenyKeysByCompany = new Map<string, Set<string>>();
        for (const [companyKey, svcMap] of aggByCompany.entries()) {
          if (!requiresAllowKeysByCompany.has(companyKey)) requiresAllowKeysByCompany.set(companyKey, new Set<string>());
          if (!requiresDenyKeysByCompany.has(companyKey)) requiresDenyKeysByCompany.set(companyKey, new Set<string>());
          const requiresAllowSet = requiresAllowKeysByCompany.get(companyKey)!;
          const requiresDenySet = requiresDenyKeysByCompany.get(companyKey)!;

          for (const [serviceFullName, agg] of svcMap.entries()) {
            const serviceWildcard = `rpc:/${serviceFullName}/*`;

            // Keep deny-wins semantics by emitting only wildcard deny for denyAll scopes.
            if (agg.denyAll) {
              requiresDenySet.add(serviceWildcard);
              continue;
            }

            const hasAnyAllow = agg.allowAll || agg.allow.size > 0;
            if (!hasAnyAllow) {
              // Stay fail-closed when no allow entry exists.
              continue;
            }

            // Emit the service-level allow key for coarse route and menu checks.
            requiresAllowSet.add(serviceWildcard);

            if (agg.deny.size > 0) {
              // Emit method-level keys only when allow and deny entries coexist.
              for (const m of agg.allow) requiresAllowSet.add(`rpc:/${serviceFullName}/${m}`);
              for (const m of agg.deny) requiresDenySet.add(`rpc:/${serviceFullName}/${m}`);
            }
          }
        }

        // 3) Generate ui.routes, ui.menus, and ui.actions from UI grants and RPC requires.
        const uiGrants = await RoleUiResource.Search(['RoleId', 'in', roleIds] as any, {
          fields: ['RoleId', 'IrApplicationId', 'IrUiResourceId', 'Mode'],
          limit: 50000,
        });

        const explicitGlobalUiAllowByCompany = new Set<string>();
        const explicitGlobalUiDenyByCompany = new Set<string>();
        const explicitAppUiAllowByCompany = new Map<string, Set<string>>();
        const explicitAppUiDenyByCompany = new Map<string, Set<string>>();
        const explicitResourceUiAllowByCompany = new Map<string, Set<string>>();
        const explicitResourceUiDenyByCompany = new Map<string, Set<string>>();

        const ensureExplicitAppSet = (companyKey: string): Set<string> => {
          let set = explicitAppUiAllowByCompany.get(companyKey);
          if (!set) {
            set = new Set<string>();
            explicitAppUiAllowByCompany.set(companyKey, set);
          }
          return set;
        };

        const ensureExplicitAppDenySet = (companyKey: string): Set<string> => {
          let set = explicitAppUiDenyByCompany.get(companyKey);
          if (!set) {
            set = new Set<string>();
            explicitAppUiDenyByCompany.set(companyKey, set);
          }
          return set;
        };

        const ensureExplicitResourceSet = (companyKey: string): Set<string> => {
          let set = explicitResourceUiAllowByCompany.get(companyKey);
          if (!set) {
            set = new Set<string>();
            explicitResourceUiAllowByCompany.set(companyKey, set);
          }
          return set;
        };

        const ensureExplicitResourceDenySet = (companyKey: string): Set<string> => {
          let set = explicitResourceUiDenyByCompany.get(companyKey);
          if (!set) {
            set = new Set<string>();
            explicitResourceUiDenyByCompany.set(companyKey, set);
          }
          return set;
        };

        for (const g of uiGrants || []) {
          const roleId = this._maybeId((g as any).RoleId);
          const appId = this._normalizeScopeRefId((g as any).IrApplicationId);
          const resourceId = this._normalizeUiResourceId((g as any).IrUiResourceId);
          const mode = String((g as any).Mode ?? 'allow')
            .trim()
            .toLowerCase();
          if (!roleId) continue;
          if (mode !== 'allow' && mode !== 'deny') continue;

          // global scope
          if (!appId && !resourceId) {
            applyToScope(roleId, companyKey => {
              if (mode === 'allow') explicitGlobalUiAllowByCompany.add(companyKey);
              else explicitGlobalUiDenyByCompany.add(companyKey);
            });
            continue;
          }

          // application scope
          if (appId && !resourceId) {
            applyToScope(roleId, companyKey => {
              if (mode === 'allow') ensureExplicitAppSet(companyKey).add(appId);
              else ensureExplicitAppDenySet(companyKey).add(appId);
            });
            continue;
          }

          // resource scope
          if (!appId && resourceId) {
            applyToScope(roleId, companyKey => {
              if (mode === 'allow') ensureExplicitResourceSet(companyKey).add(resourceId);
              else ensureExplicitResourceDenySet(companyKey).add(resourceId);
            });
            continue;
          }
        }

        const resources = await IrUiResource.Search(
          [] as any,
          {
            fields: ['Id', 'Name', 'Type', 'ParentId', 'IrApplicationId', 'Requires'],
            limit: 100000,
          } as any
        );

        const resourceNameById = new Map<string, string>();
        for (const row of resources || []) {
          const id = String((row as any)?.Id ?? (row as any)?.id ?? '').trim();
          const name = String((row as any)?.Name ?? (row as any)?.name ?? '').trim();
          if (!id || !name) continue;
          resourceNameById.set(id, name);
        }

        const allResources = (resources || [])
          .map((row: any) => ({
            dbId: String(row?.Id ?? row?.id ?? '').trim(),
            resourceId: String(row?.Name ?? row?.name ?? '').trim(),
            type: String(row?.Type || '')
              .trim()
              .toUpperCase(),
            parentId: (() => {
              const pid = this._normalizeUiResourceId(row?.ParentId ?? row?.parentId);
              return pid ? String(resourceNameById.get(pid) || '').trim() : '';
            })(),
            appId: this._normalizeScopeRefId(row?.IrApplicationId),
            requires: this._parseJsonStringArray((row as any)?.Requires ?? (row as any)?.requires),
          }))
          .filter(r => !!r.resourceId && (r.type === 'ROUTE' || r.type === 'MENU' || r.type === 'ACTION'));

        const menuRouteRows = await IrUiResourceMenuRoute.Search(
          [] as any,
          {
            fields: ['MenuUiResourceId', 'RouteUiResourceId'],
            limit: 100000,
          } as any
        );

        const routeActionRows = await IrUiResourceRouteAction.Search(
          [] as any,
          {
            fields: ['RouteUiResourceId', 'ActionUiResourceId'],
            limit: 100000,
          } as any
        );

        const resourceMetaById = new Map<string, (typeof allResources)[number]>();
        for (const resource of allResources) {
          resourceMetaById.set(resource.resourceId, resource);
        }

        const menuParentById = new Map<string, string>();
        for (const r of allResources) {
          if (r.type === 'MENU' && r.parentId) menuParentById.set(r.resourceId, r.parentId);
        }

        const menuIdsByRouteId = new Map<string, Set<string>>();
        for (const row of menuRouteRows || []) {
          const menuDbId = this._normalizeUiResourceId((row as any)?.MenuUiResourceId);
          const routeDbId = this._normalizeUiResourceId((row as any)?.RouteUiResourceId);
          const menuId = menuDbId ? String(resourceNameById.get(menuDbId) || '').trim() : '';
          const routeId = routeDbId ? String(resourceNameById.get(routeDbId) || '').trim() : '';
          if (!menuId || !routeId) continue;
          let menuIds = menuIdsByRouteId.get(routeId);
          if (!menuIds) {
            menuIds = new Set<string>();
            menuIdsByRouteId.set(routeId, menuIds);
          }
          menuIds.add(menuId);
        }

        const routeIdsByActionId = new Map<string, Set<string>>();
        for (const row of routeActionRows || []) {
          const routeDbId = this._normalizeUiResourceId((row as any)?.RouteUiResourceId);
          const actionDbId = this._normalizeUiResourceId((row as any)?.ActionUiResourceId);
          const routeId = routeDbId ? String(resourceNameById.get(routeDbId) || '').trim() : '';
          const actionId = actionDbId ? String(resourceNameById.get(actionDbId) || '').trim() : '';
          if (!routeId || !actionId) continue;
          let routeIds = routeIdsByActionId.get(actionId);
          if (!routeIds) {
            routeIds = new Set<string>();
            routeIdsByActionId.set(actionId, routeIds);
          }
          routeIds.add(routeId);
        }

        const companyKeys = Array.from(new Set(['*', ...(authz.enabledCompanyIds || [])]));
        for (const companyKey of companyKeys) {
          const bucket = ensureBucket(companyKey);
          const ui = bucket.ui!;

          const hasGlobalAllow = companyGlobalAllow.has(companyKey);
          const hasGlobalDeny = companyGlobalDeny.has(companyKey);
          const hasExplicitGlobalUiAllow = explicitGlobalUiAllowByCompany.has(companyKey);
          const hasExplicitGlobalUiDeny = explicitGlobalUiDenyByCompany.has(companyKey);
          const explicitAppAllow = explicitAppUiAllowByCompany.get(companyKey) ?? new Set<string>();
          const explicitAppDeny = explicitAppUiDenyByCompany.get(companyKey) ?? new Set<string>();
          const explicitResourceAllow = explicitResourceUiAllowByCompany.get(companyKey) ?? new Set<string>();
          const explicitResourceDeny = explicitResourceUiDenyByCompany.get(companyKey) ?? new Set<string>();
          const hasAnyExplicitUiDeny = hasExplicitGlobalUiDeny || explicitAppDeny.size > 0 || explicitResourceDeny.size > 0;

          const isExplicitUiDenied = (resource: { dbId: string; resourceId: string; appId: string | null }): boolean =>
            hasExplicitGlobalUiDeny ||
            (!!resource.appId && explicitAppDeny.has(resource.appId)) ||
            explicitResourceDeny.has(resource.dbId) ||
            explicitResourceDeny.has(resource.resourceId);

          const isExplicitUiAllowed = (resource: { dbId: string; resourceId: string; appId: string | null }): boolean =>
            !isExplicitUiDenied(resource) &&
            (hasExplicitGlobalUiAllow ||
              (!!resource.appId && explicitAppAllow.has(resource.appId)) ||
              explicitResourceAllow.has(resource.dbId) ||
              explicitResourceAllow.has(resource.resourceId));

          // explicit UI global allow has highest priority for UI whitelist materialization.
          if (!hasAnyExplicitUiDeny && ((hasGlobalAllow && !hasGlobalDeny) || hasExplicitGlobalUiAllow)) {
            ui.routes = ['*'];
            ui.menus = ['*'];
            ui.actions = ['*'];
            continue;
          }

          const requiresAllowSet = requiresAllowKeysByCompany.get(companyKey) ?? new Set<string>();
          const requiresDenySet = requiresDenyKeysByCompany.get(companyKey) ?? new Set<string>();

          const routeSet = new Set<string>();
          const menuSet = new Set<string>();
          const actionSet = new Set<string>();
          const explicitRouteSet = new Set<string>();
          const explicitActionSet = new Set<string>();

          for (const r of allResources) {
            const explicitlyDenied = isExplicitUiDenied(r);
            if (explicitlyDenied) continue;
            const allowedByExplicit = isExplicitUiAllowed(r);
            const allowedByRequires = this._isUiResourceAllowed(r.requires, requiresAllowSet, requiresDenySet);
            if (!allowedByExplicit && !allowedByRequires) continue;
            if (r.type === 'ROUTE') {
              routeSet.add(r.resourceId);
              if (allowedByExplicit) explicitRouteSet.add(r.resourceId);
            } else if (r.type === 'MENU') menuSet.add(r.resourceId);
            else if (r.type === 'ACTION') {
              actionSet.add(r.resourceId);
              if (allowedByExplicit) explicitActionSet.add(r.resourceId);
            }
          }

          // Keep explicit action grants navigable by projecting their owning routes.
          for (const actionId of explicitActionSet) {
            for (const routeId of routeIdsByActionId.get(actionId) ?? []) {
              const route = resourceMetaById.get(routeId);
              if (!route || route.type !== 'ROUTE' || isExplicitUiDenied(route)) continue;
              routeSet.add(routeId);
              explicitRouteSet.add(routeId);
            }
          }

          for (const routeId of explicitRouteSet) {
            for (const menuId of menuIdsByRouteId.get(routeId) ?? []) {
              const menu = resourceMetaById.get(menuId);
              if (!menu || isExplicitUiDenied(menu)) continue;
              menuSet.add(menuId);
            }
          }

          for (const menuId of Array.from(menuSet)) {
            let parentId = menuParentById.get(menuId);
            let deniedByAncestor = false;
            while (parentId) {
              const parent = resourceMetaById.get(parentId);
              if (parent && isExplicitUiDenied(parent)) {
                deniedByAncestor = true;
                break;
              }
              parentId = menuParentById.get(parentId);
            }
            if (deniedByAncestor) menuSet.delete(menuId);
          }

          // Backfill parent menus so visible children never appear without parents.
          const pending = Array.from(menuSet);
          while (pending.length > 0) {
            const cur = pending.pop()!;
            const parentId = menuParentById.get(cur);
            if (!parentId) continue;
            if (menuSet.has(parentId)) continue;
            const parent = resourceMetaById.get(parentId);
            if (parent && isExplicitUiDenied(parent)) {
              menuSet.delete(cur);
              continue;
            }
            menuSet.add(parentId);
            pending.push(parentId);
          }

          ui.routes = this._sortStrings(Array.from(routeSet));
          ui.menus = this._sortStrings(Array.from(menuSet));
          ui.actions = this._sortStrings(Array.from(actionSet));
        }

        return { permStateVersion, byCompany };
      });
    } catch {
      // fail-closed
      return { permStateVersion: 0, byCompany: {} };
    }
  }

  private static async _withPermissionGraphBypass<T>(fn: () => Promise<T>): Promise<T> {
    const { req } = getJsCtxAndReq();
    if (!req) return await fn();

    if (!req.__choysumServiceState) req.__choysumServiceState = {};
    const state: any = req.__choysumServiceState;

    const prevRR = typeof state.recordRuleBypassDepth === 'number' ? state.recordRuleBypassDepth : 0;
    const prevFR = typeof state.fieldRuleBypassDepth === 'number' ? state.fieldRuleBypassDepth : 0;
    const hadCompanyMode = Object.prototype.hasOwnProperty.call(req, 'companyMode');
    const prevCompanyMode = req.companyMode;

    state.recordRuleBypassDepth = prevRR + 1;
    state.fieldRuleBypassDepth = prevFR + 1;
    req.companyMode = 'skip';
    try {
      return await fn();
    } finally {
      const nextRR = prevRR;
      const nextFR = prevFR;
      if (nextRR > 0) state.recordRuleBypassDepth = nextRR;
      else delete state.recordRuleBypassDepth;
      if (nextFR > 0) state.fieldRuleBypassDepth = nextFR;
      else delete state.fieldRuleBypassDepth;

      if (hadCompanyMode) req.companyMode = prevCompanyMode;
      else delete req.companyMode;
    }
  }

  /**
   * Return a stable sorted copy of a string list.
   */
  private static _sortStrings(xs: string[]): string[] {
    return (xs || []).slice().sort((a, b) => (a < b ? -1 : a > b ? 1 : 0));
  }

  /**
   * Read active and enabled company scope from request overrides or identity metadata.
   */
  private static _getCompanyScopeFromRequestContext(): { activeCompanyId: string; enabledCompanyIds: string[] } {
    // IMPORTANT: Use core runtime/context helpers to respect request-level overrides set by withContext().
    // Some tests (and potentially other flows) rely on Symbol.for('choysum.ctx.override') rather than
    // mutating jsCtx.ctx directly.
    const ctx: any = (getReadonlyCtx() ?? {}) as any;
    const identity: any = (getIdentity() ?? {}) as any;
    const meta: any = (identity?.metadata ?? identity?.Metadata ?? {}) as any;

    const activeCompanyId = String(ctx?.activeCompanyId ?? meta?.activeCompanyId ?? '').trim();
    const enabledCompanyIds = uniqStrings(ctx?.enabledCompanyIds ?? meta?.enabledCompanyIds ?? []);

    if (activeCompanyId && !enabledCompanyIds.includes(activeCompanyId)) {
      return { activeCompanyId, enabledCompanyIds: [activeCompanyId, ...enabledCompanyIds] };
    }
    return { activeCompanyId, enabledCompanyIds };
  }

  /**
   * Parse a string or structured value into a normalized string array.
   */
  private static _parseJsonStringArray(raw: any): string[] {
    const normalize = (xs: any[]): string[] => uniqStrings((xs || []).map(v => String(v ?? '').trim()).filter(Boolean));
    const tryObjectSnapshot = (value: any): string[] | null => {
      if (!value || typeof value !== 'object') return null;
      try {
        const snap = JSON.parse(JSON.stringify(value));
        if (Array.isArray(snap)) return normalize(snap);
        if (!snap || typeof snap !== 'object') return null;

        for (const key of ['value', 'values', 'items']) {
          if (Array.isArray((snap as any)[key])) return normalize((snap as any)[key]);
        }

        const numericKeys = Object.keys(snap)
          .filter(key => /^\d+$/.test(key))
          .sort((a, b) => Number(a) - Number(b));
        if (numericKeys.length > 0) return normalize(numericKeys.map(key => (snap as any)[key]));
      } catch {
        // fallthrough
      }
      return null;
    };

    if (Array.isArray(raw)) return normalize(raw);
    if (raw == null) return [];

    if (typeof raw === 'string') {
      const s = raw.trim();
      if (!s) return [];
      try {
        const parsed = JSON.parse(s);
        if (Array.isArray(parsed)) return normalize(parsed);
      } catch {
        // fallthrough
      }
      return normalize([s]);
    }

    const snapResult = tryObjectSnapshot(raw);
    if (snapResult) return snapResult;

    try {
      if (typeof (raw as any)?.toString === 'function') {
        const s = String((raw as any).toString() || '').trim();
        if (!s) return [];
        const parsed = JSON.parse(s);
        if (Array.isArray(parsed)) return normalize(parsed);
      }
    } catch {
      // fallthrough
    }
    return [];
  }

  /**
   * Evaluate a single RPC require key against allow and deny sets.
   */
  private static _hasRpcPermission(req: string, allowSet: Set<string>, denySet: Set<string>): boolean {
    const k = normalizeRpcRequireKey(req);
    if (!k) return false;
    const wildcard = rpcServiceWildcard(k);

    if (denySet.has(k)) return false;
    if (wildcard && denySet.has(wildcard)) return false;

    if (allowSet.has(k)) return true;
    if (wildcard && allowSet.has(wildcard)) return true;
    return false;
  }

  /**
   * Check whether all UI resource requires are satisfied by RPC permissions.
   */
  private static _isUiResourceAllowed(requires: string[], allowSet: Set<string>, denySet: Set<string>): boolean {
    const reqs = uniqStrings((requires || []).map(v => String(v || '').trim()).filter(Boolean));
    if (reqs.length === 0) return true;

    for (const req of reqs) {
      if (!this._hasRpcPermission(req, allowSet, denySet)) return false;
    }
    return true;
  }

  /**
   * Check whether a require key targets the specified model and method.
   */
  private static _requireMatchesMethod(req: string, modelKey: string, methodLower: string): boolean {
    const k = normalizeRpcRequireKey(req);
    if (!k || !k.startsWith('rpc:/')) return false;
    const body = k.slice('rpc:/'.length);
    const parts = body.split('/');
    if (parts.length !== 2) return false;
    const mk = String(parts[0] || '').trim();
    const mm = String(parts[1] || '')
      .trim()
      .toLowerCase();
    if (!mk || !mm) return false;
    if (
      mk.toLowerCase() !==
      String(modelKey || '')
        .trim()
        .toLowerCase()
    )
      return false;
    return mm === '*' || mm === methodLower;
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
    const companyScope = this._getCompanyScopeFromRequestContext();
    const enabledCompanyIdsKey = this._sortStrings(uniqStrings(companyScope.enabledCompanyIds));
    const companyScopeKey = `${companyScope.activeCompanyId}::${enabledCompanyIdsKey.join(',')}`;

    const build = async (args: { userId: string; activeCompanyId: string; enabledCompanyIds: string[] }) => {
      const { activeCompanyId, enabledCompanyIds } = args;

      if (!userId) {
        return {
          userId: '',
          activeCompanyId,
          enabledCompanyIds,
          roleScopesById: {},
          roleIds: [],
          rolesByCompany: {},
          roles: [],
        };
      }

      return await this._withPermissionGraphBypass(async () => {
        const hasCompany = enabledCompanyIds.length > 0;
        const userRoleCond: any = hasCompany
          ? {
              And: [
                ['UserId', '=', userId],
                {
                  Or: [
                    ['CompanyId', '=', null],
                    ['CompanyId', 'in', enabledCompanyIds],
                  ],
                },
              ],
            }
          : {
              And: [
                ['UserId', '=', userId],
                ['CompanyId', '=', null],
              ],
            };

        const userRoles = await UserRole.Search(userRoleCond, { fields: ['RoleId', 'CompanyId'], limit: 5000 });
        const roleScopes = await this._computeEffectiveRoleScopes(userRoles || []);
        const roleIds = Array.from(roleScopes.keys());

        const roleScopesById: Record<string, { global: boolean; companies: string[] }> = {};
        for (const [rid, scope] of roleScopes.entries()) {
          roleScopesById[rid] = {
            global: !!scope.global,
            companies: scope.global ? [] : this._sortStrings(Array.from(scope.companies)),
          };
        }

        const globalRoleIds: string[] = [];
        const scopedByCompany = new Map<string, Set<string>>();
        for (const [rid, scope] of roleScopes.entries()) {
          if (scope.global) {
            globalRoleIds.push(rid);
          } else {
            for (const cid of scope.companies) {
              if (!cid) continue;
              let s = scopedByCompany.get(cid);
              if (!s) {
                s = new Set<string>();
                scopedByCompany.set(cid, s);
              }
              s.add(rid);
            }
          }
        }

        const rolesByCompany: Record<string, string[]> = {};
        for (const cid of enabledCompanyIds) {
          const set = new Set<string>();
          for (const rid of globalRoleIds) set.add(rid);
          const extra = scopedByCompany.get(cid);
          if (extra) for (const rid of extra) set.add(rid);
          rolesByCompany[cid] = this._sortStrings(Array.from(set));
        }

        const activeRoles = activeCompanyId && rolesByCompany[activeCompanyId] ? rolesByCompany[activeCompanyId] : this._sortStrings(globalRoleIds);

        return {
          userId,
          activeCompanyId,
          enabledCompanyIds,
          roleScopesById,
          roleIds: this._sortStrings(roleIds),
          rolesByCompany,
          roles: activeRoles,
        };
      });
    };

    if (!state) {
      return await build({ userId, activeCompanyId: companyScope.activeCompanyId, enabledCompanyIds: enabledCompanyIdsKey });
    }

    const KEY = buildAuthzContextCacheKey(userId, companyScopeKey);
    const existing = state[KEY];
    if (existing) {
      // Avoid leaking a Promise into req.__choysumServiceState: Go-side unmarshalling
      // treats Promises as unsupported JavaScript types.
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
    try {
      const rows = await model.Search(cond, { fields: ['UpdatedAt'], orderBy: { field: 'UpdatedAt', order: 'desc' }, limit: 1 });
      const v = (rows as any)?.[0]?.UpdatedAt;
      const n = Number(new Date(v || 0));
      return Number.isFinite(n) ? n : 0;
    } catch {
      return 0;
    }
  }

  /**
   * Compute a permission-state version from user-role and role graph timestamps.
   */
  private static async _computePermStateVersion(userId: string): Promise<number> {
    const uid = String(userId || '').trim();
    if (!uid) return 0;
    try {
      return await this._withPermissionGraphBypass(async () => {
        const urMax = await this._maxUpdatedAt(UserRole, ['UserId', '=', uid] as any);

        const userRoles = await UserRole.Search(['UserId', '=', uid] as any, { fields: ['RoleId'], limit: 5000 });
        const directRoleIds = Array.from(new Set((userRoles || []).map(ur => this._maybeId((ur as any).RoleId)).filter(Boolean) as string[]));

        const effectiveRoleIds = await this._expandRoleClosure(directRoleIds);
        if (effectiveRoleIds.length === 0) return urMax;

        const roleMax = await this._maxUpdatedAt(Role, ['Id', 'in', effectiveRoleIds] as any);
        const inhMax = await this._maxUpdatedAt(RoleInheritance, {
          Or: [
            ['ParentRoleId', 'in', effectiveRoleIds],
            ['ChildRoleId', 'in', effectiveRoleIds],
          ],
        } as any);
        const maMax = await this._maxUpdatedAt(RoleMethodAccess, ['RoleId', 'in', effectiveRoleIds] as any);

        return Math.max(urMax, roleMax, inhMax, maMax);
      });
    } catch {
      return 0;
    }
  }

  /**
   * Expand direct roles through the full inheritance closure.
   */
  private static async _expandRoleClosure(directRoleIds: string[]): Promise<string[]> {
    const seed = Array.from(new Set((directRoleIds || []).map(s => String(s || '').trim()).filter(Boolean)));
    if (seed.length === 0) return [];

    const all = new Set<string>(seed);
    const pending = seed.slice();
    const totalRoleCount = Number(await Role.Count([] as any)) || 0;
    const maxSteps = Math.max(50, totalRoleCount * 5);
    let steps = 0;

    while (pending.length > 0 && steps < maxSteps) {
      steps++;
      const batch = pending.splice(0, 200);
      const edges = await RoleInheritance.Search(['ParentRoleId', 'in', batch] as any, { fields: ['ParentRoleId', 'ChildRoleId'], limit: 5000 });
      for (const e of edges || []) {
        const childId = this._maybeId((e as any).ChildRoleId);
        if (!childId) continue;
        if (!all.has(childId)) {
          all.add(childId);
          pending.push(childId);
        }
      }
    }
    return Array.from(all);
  }

  /**
   * Compute effective global and company-scoped role coverage from user-role assignments.
   */
  private static async _computeEffectiveRoleScopes(userRoles: any[]): Promise<Map<string, { global: boolean; companies: Set<string> }>> {
    type RoleScope = { global: boolean; companies: Set<string> };
    const roleScopes = new Map<string, RoleScope>();

    const ensureScope = (roleId: string): RoleScope => {
      let s = roleScopes.get(roleId);
      if (!s) {
        s = { global: false, companies: new Set<string>() };
        roleScopes.set(roleId, s);
      }
      return s;
    };

    const mergeScope = (roleId: string, companyId: string | null | undefined): boolean => {
      const s = ensureScope(roleId);
      if (companyId === null || companyId === undefined || String(companyId ?? '').trim() === '') {
        if (s.global) return false;
        s.global = true;
        return true;
      }
      if (s.global) return false;
      const cid = String(companyId).trim();
      if (!cid || s.companies.has(cid)) return false;
      s.companies.add(cid);
      return true;
    };

    // direct roles
    for (const ur of userRoles || []) {
      const roleId = this._maybeId((ur as any).RoleId);
      if (!roleId) continue;
      mergeScope(roleId, (ur as any).CompanyId as any);
    }

    const directRoleIds = Array.from(new Set((userRoles || []).map(ur => this._maybeId((ur as any).RoleId)).filter(Boolean) as string[]));
    if (directRoleIds.length === 0) return roleScopes;

    // inheritance closure with scope propagation
    const totalRoleCount = Number(await Role.Count([] as any)) || 0;
    const maxSteps = Math.max(50, totalRoleCount * 5);
    let steps = 0;
    const pending: string[] = directRoleIds.slice();
    for (const rid of pending) ensureScope(rid);

    while (pending.length > 0 && steps < maxSteps) {
      steps++;
      const batch = pending.splice(0, 200);
      const edges = await RoleInheritance.Search(['ParentRoleId', 'in', batch] as any, { fields: ['ParentRoleId', 'ChildRoleId'], limit: 5000 });
      for (const e of edges || []) {
        const parentId = this._maybeId((e as any).ParentRoleId) || '';
        const childId = this._maybeId((e as any).ChildRoleId);
        if (!childId) continue;

        const parentScope = roleScopes.get(parentId);
        if (!parentScope) {
          ensureScope(childId);
          continue;
        }

        let changed = false;
        if (parentScope.global) {
          changed = mergeScope(childId, null) || changed;
        } else {
          for (const cid of parentScope.companies) {
            changed = mergeScope(childId, cid) || changed;
          }
        }
        if (changed) pending.push(childId);
      }
    }

    return roleScopes;
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
      const { enabledCompanyIds } = this._getCompanyScopeFromRequestContext();
      if (enabledCompanyIds.length > 0 && !enabledCompanyIds.includes(normalizedCompanyId)) return false;

      // Permission graph reads must bypass RecordRule/FieldRule.
      // Otherwise a request with activeCompanyId=c1 can never evaluate roles scoped to c2,
      // because UserRole/RoleMethodAccess reads may be fail-closed by RecordRule injection.
      return await this._withPermissionGraphBypass(async () => {
        const authz = await this._getAuthzContext();
        const roleIds = authz.rolesByCompany?.[normalizedCompanyId] || [];
        if (roleIds.length === 0) return false;

        const req = getCurrentReq();
        const state = getOrInitReqServiceState(req);
        const cacheKey = buildMethodAccessCacheKey(String(authz.userId || '').trim(), normalizedCompanyId, normalizedFullMethod);
        const cached = state?.[cacheKey];
        if (typeof cached === 'boolean') return cached;

        // 2) Resolve meta model/service ids
        const models = await IrModel.Search(
          {
            And: [
              ['Name', '=', modelName],
              ['Application', '=', appName],
            ],
          } as any,
          { fields: ['Id'], limit: 1 }
        );
        const modelId = String(models?.[0]?.Id || '').trim();
        if (!modelId) return false;

        // Meta IR service names are not guaranteed to be case-stable (and can differ across build paths).
        // Follow the established pattern from tests: fetch by ModelId then match by name.
        const serviceRows = await IrService.Search({ And: [['ModelId', '=', modelId]] } as any, { fields: ['Id', 'Name'], limit: 5000 } as any);
        const targetMethod = String(methodName || '')
          .trim()
          .toLowerCase();
        const matched = (serviceRows || []).find(
          (r: any) =>
            String((r as any).Name || '')
              .trim()
              .toLowerCase() === targetMethod
        ) as any;
        const irServiceId = String(matched?.Id || '').trim();
        if (!irServiceId) return false;

        // IrApplicationId must come from meta.IrApplication.Id (not Name).
        // If the application cannot be resolved, we can still evaluate service/model/global scopes.
        const apps = await IrApplication.Search({ And: [['Name', '=', appName]] } as any, { fields: ['Id'], limit: 1 } as any);
        const irApplicationId = String((apps?.[0] as any)?.Id || '').trim();

        // 3) Evaluate allow/deny (deny wins) across 4 scopes.
        // Match order does not override deny-wins: any deny in any matched scope denies.
        const scopeOr: any[] = [
          {
            And: [
              ['IrServiceId', '=', irServiceId],
              ['IrModelId', 'is', null],
              ['IrApplicationId', 'is', null],
            ],
          },
          {
            And: [
              ['IrServiceId', 'is', null],
              ['IrModelId', '=', modelId],
              ['IrApplicationId', 'is', null],
            ],
          },
          {
            And: [
              ['IrServiceId', 'is', null],
              ['IrModelId', 'is', null],
              ['IrApplicationId', 'is', null],
            ],
          },
        ];
        if (irApplicationId) {
          scopeOr.splice(2, 0, {
            And: [
              ['IrServiceId', 'is', null],
              ['IrModelId', 'is', null],
              ['IrApplicationId', '=', irApplicationId],
            ],
          });
        }

        const accesses = await RoleMethodAccess.Search(
          {
            And: [['RoleId', 'in', roleIds], { Or: scopeOr } as any],
          } as any,
          { fields: ['Mode'], limit: 5000 }
        );

        let denied = false;
        let seenAllow = false;
        for (const a of accesses || []) {
          const mode = String((a as any).Mode || '').toLowerCase();
          if (mode === 'deny') {
            denied = true;
            break;
          }
          if (mode === 'allow') seenAllow = true;
        }

        if (denied) {
          if (state) state[cacheKey] = false;
          return false;
        }

        if (seenAllow) {
          if (state) state[cacheKey] = true;
          return true;
        }

        const uiDecision = await this._evaluateUiDerivedMethodDecision(
          roleIds,
          `${appName}.${modelName}`,
          String(methodName || '')
            .trim()
            .toLowerCase()
        );
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
   * Normalize a UI resource reference into its string id.
   */
  private static _normalizeUiResourceId(raw: any): string {
    if (raw == null) return '';
    if (typeof raw === 'object' && raw !== null) {
      return String((raw as any).Id ?? (raw as any).id ?? '').trim();
    }
    return String(raw || '').trim();
  }

  /**
   * Normalize an application or scope reference into its string id.
   */
  private static _normalizeScopeRefId(raw: any): string {
    if (raw == null) return '';
    if (typeof raw === 'object' && raw !== null) {
      return String((raw as any).Id ?? (raw as any).id ?? '').trim();
    }
    return String(raw || '').trim();
  }

  /**
   * Expand role UI grants into materialized resource rows and lookup tables.
   */
  private static async _loadUiGrantExpansionForRoles(roleIds: string[]): Promise<{
    resources: any[];
    hasGlobalAllow: boolean;
    hasGlobalDeny: boolean;
    appModesById: Record<string, ('allow' | 'deny')[]>;
    resourceModesByKey: Record<string, ('allow' | 'deny')[]>;
  }> {
    const ids = this._sortStrings(uniqStrings(roleIds || []));
    if (ids.length === 0) {
      return { resources: [], hasGlobalAllow: false, hasGlobalDeny: false, appModesById: {}, resourceModesByKey: {} };
    }

    const req = getCurrentReq();
    const state = getOrInitReqServiceState(req);
    const roleSig = ids.join(',');
    const cacheKey = buildUiGrantCacheKey(roleSig);
    const cached = state?.[cacheKey];
    if (cached && Array.isArray((cached as any).resources)) {
      return cached as any;
    }

    const grants = await RoleUiResource.Search(
      {
        And: [['RoleId', 'in', ids]],
      } as any,
      { fields: ['IrApplicationId', 'IrUiResourceId', 'Mode'], limit: 100000 } as any
    );

    let hasGlobalAllow = false;
    let hasGlobalDeny = false;
    const appIDs = new Set<string>();
    const resourceIDs = new Set<string>();
    const appModesById = new Map<string, Set<'allow' | 'deny'>>();
    const resourceModesByKey = new Map<string, Set<'allow' | 'deny'>>();
    const sortGrantModes = (modes: Iterable<'allow' | 'deny'>): Array<'allow' | 'deny'> =>
      Array.from(new Set(modes)).sort((a, b) => a.localeCompare(b)) as Array<'allow' | 'deny'>;
    for (const g of grants || []) {
      const mode =
        String((g as any).Mode ?? 'allow')
          .trim()
          .toLowerCase() === 'deny'
          ? 'deny'
          : 'allow';
      const appID = this._normalizeScopeRefId((g as any).IrApplicationId);
      const uiID = this._normalizeUiResourceId((g as any).IrUiResourceId);
      if (!appID && !uiID) {
        if (mode === 'deny') hasGlobalDeny = true;
        else hasGlobalAllow = true;
        continue;
      }

      if (appID) {
        appIDs.add(appID);
        if (!appModesById.has(appID)) appModesById.set(appID, new Set());
        appModesById.get(appID)!.add(mode);
      }
      if (uiID) {
        resourceIDs.add(uiID);
        if (!resourceModesByKey.has(uiID)) resourceModesByKey.set(uiID, new Set());
        resourceModesByKey.get(uiID)!.add(mode);
      }
    }

    if (!hasGlobalAllow && !hasGlobalDeny && appIDs.size === 0 && resourceIDs.size === 0) {
      const empty = { resources: [], hasGlobalAllow: false, hasGlobalDeny: false, appModesById: {}, resourceModesByKey: {} };
      if (state) state[cacheKey] = empty;
      return empty;
    }

    const byId = new Map<string, any>();
    const mergeRows = (rows: any[]) => {
      for (const row of rows || []) {
        const id = this._normalizeUiResourceId((row as any)?.Id ?? (row as any)?.id);
        if (!id) continue;
        if (!byId.has(id)) byId.set(id, row);
      }
    };

    if (hasGlobalAllow || hasGlobalDeny) {
      const allRows = await IrUiResource.Search([] as any, { fields: ['Id', 'Name', 'IrApplicationId', 'Requires'], limit: 100000 } as any);
      mergeRows(allRows as any[]);
    } else {
      const appIDList = uniqStrings(Array.from(appIDs));
      if (appIDList.length > 0) {
        const rows = await IrUiResource.Search(
          {
            And: [['IrApplicationId', 'in', appIDList]],
          } as any,
          { fields: ['Id', 'Name', 'IrApplicationId', 'Requires'], limit: 100000 } as any
        );
        mergeRows(rows as any[]);
      }

      const resourceIDList = uniqStrings(Array.from(resourceIDs));
      if (resourceIDList.length > 0) {
        const byIdRows = await IrUiResource.Search(
          {
            And: [['Id', 'in', resourceIDList]],
          } as any,
          { fields: ['Id', 'Name', 'IrApplicationId', 'Requires'], limit: 100000 } as any
        );
        mergeRows(byIdRows as any[]);

        const byResourceIdRows = await IrUiResource.Search(
          {
            And: [['Name', 'in', resourceIDList]],
          } as any,
          { fields: ['Id', 'Name', 'IrApplicationId', 'Requires'], limit: 100000 } as any
        );
        mergeRows(byResourceIdRows as any[]);
      }
    }

    const resources = Array.from(byId.values());
    const appModesRecord: Record<string, Array<'allow' | 'deny'>> = {};
    for (const [key, value] of appModesById.entries()) {
      appModesRecord[key] = sortGrantModes(value);
    }

    const resourceModesRecord: Record<string, Array<'allow' | 'deny'>> = {};
    for (const [key, value] of resourceModesByKey.entries()) {
      resourceModesRecord[key] = sortGrantModes(value);
    }

    const out = {
      resources,
      hasGlobalAllow,
      hasGlobalDeny,
      appModesById: appModesRecord,
      resourceModesByKey: resourceModesRecord,
    };
    if (state) state[cacheKey] = out;
    return out;
  }

  /**
   * Evaluate allow and deny decisions derived from UI resource requires.
   */
  private static async _evaluateUiDerivedMethodDecision(
    roleIds: string[],
    modelKey: string,
    methodLower: string
  ): Promise<{ allowed: boolean; denied: boolean }> {
    const mkey = String(modelKey || '').trim();
    const m = String(methodLower || '')
      .trim()
      .toLowerCase();
    if (!mkey || !m) return { allowed: false, denied: false };

    const expansion = await this._loadUiGrantExpansionForRoles(roleIds);
    const resources = expansion.resources || [];
    if (resources.length === 0) return { allowed: false, denied: false };

    let allowed = false;
    let denied = false;

    for (const row of resources) {
      const requires = this._parseJsonStringArray((row as any)?.Requires ?? (row as any)?.requires);
      if (requires.length === 0) continue;

      let matchesMethod = false;
      for (const req of requires) {
        if (this._requireMatchesMethod(req, mkey, m)) {
          matchesMethod = true;
          break;
        }
      }
      if (!matchesMethod) continue;

      const matchedModes = new Set<'allow' | 'deny'>();
      if (expansion.hasGlobalAllow) matchedModes.add('allow');
      if (expansion.hasGlobalDeny) matchedModes.add('deny');

      const appId = this._normalizeScopeRefId((row as any)?.IrApplicationId);
      for (const mode of expansion.appModesById?.[appId || ''] || []) matchedModes.add(mode);

      const resourceKeys = uniqStrings([
        this._normalizeUiResourceId((row as any)?.Id ?? (row as any)?.id),
        String((row as any)?.Name ?? (row as any)?.name ?? '').trim(),
      ]);
      for (const key of resourceKeys) {
        for (const mode of expansion.resourceModesByKey?.[key] || []) matchedModes.add(mode);
      }

      if (matchedModes.has('deny')) {
        denied = true;
        break;
      }
      if (matchedModes.has('allow')) allowed = true;
    }

    return { allowed, denied };
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

      const { activeCompanyId, enabledCompanyIds } = this._getCompanyScopeFromRequestContext();
      const hasCompany = enabledCompanyIds.length > 0;

      // 1) Resolve meta application/model ids
      // IrApplicationId must come from meta.IrApplication.Id (not Name).
      // If the application cannot be resolved, we can still evaluate model/global scopes.
      const apps = await IrApplication.Search({ And: [['Name', '=', parsedModel.appName]] } as any, { fields: ['Id'], limit: 1 } as any);
      const irApplicationId = String((apps as any)?.[0]?.Id || '').trim();

      const models = await IrModel.Search(
        {
          And: [
            ['Name', '=', parsedModel.modelName],
            ['Application', '=', parsedModel.appName],
          ],
        } as any,
        { fields: ['Id', 'CompanyScoped'], limit: 1 }
      );
      const modelHit: any = (models as any)?.[0] as any;
      const modelId = String(modelHit?.Id || '').trim();
      if (!modelId) return { kind: 'false', reason: 'model_not_found' };

      /**
       * Decide whether company gating should be applied for the target model.
       */
      const computeCompanyGateMode = async (): Promise<{ enabled: boolean; reason?: string }> => {
        if (!hasCompany) return { enabled: false, reason: 'no_company_context' };

        // request-scope memoization
        const req = getCurrentReq();
        const state = req ? getOrInitReqServiceState(req) : undefined;
        const key = `companyGateMode::${modelId}`;
        const existing = state ? state[key] : undefined;
        if (existing) {
          // Avoid leaking a Promise into jsCtx.req: Go-side unmarshalling treats Promises
          // as unsupported JavaScript types.
          if (typeof existing?.then === 'function') {
            const v = await existing;
            try {
              state[key] = v;
            } catch {
              // ignore
            }
            return v;
          }
          return existing;
        }

        const p = (async (): Promise<{ enabled: boolean; reason?: string }> => {
          const companyScoped = Boolean(modelHit?.CompanyScoped);
          if (!companyScoped) return { enabled: false, reason: 'model_not_company_scoped' };

          const hasCompanyIdField =
            Number(
              await IrField.Count({
                And: [
                  ['ModelId', '=', modelId],
                  ['Name', '=', 'CompanyId'],
                ],
              } as any)
            ) > 0;
          if (!hasCompanyIdField) return { enabled: false, reason: 'company_scoped_missing_company_id_field' };

          return { enabled: true };
        })()
          .then((v: any) => {
            if (state) {
              try {
                state[key] = v;
              } catch {
                // ignore
              }
            }
            return v;
          })
          .catch(() => {
            if (state) {
              try {
                delete state[key];
              } catch {
                // ignore
              }
            }
            return { enabled: false, reason: 'meta_company_gate_error' };
          });

        if (state) state[key] = p;
        const v = await p;
        if (state) {
          try {
            state[key] = v;
          } catch {
            // ignore
          }
        }
        return v;
      };

      const companyGate = await computeCompanyGateMode();

      type RoleScope = { global: boolean; companies: string[] };

      // 2) Resolve active-view roles from request-scoped AuthzContext
      const authz = await this._getAuthzContext();
      const roleIds = authz.roles || [];
      const roleScopesById = authz.roleScopesById as Record<string, RoleScope>;

      if (roleIds.length === 0) {
        // RecordRule is an additional restriction layer only.
        // No roles => no extra restriction (ACL / company filter / business logic still apply).
        return { kind: 'true', reason: `no_roles_${opValue}_allow` };
      }

      // 4) Query record rules in scope (pick-one: model > application > global)
      const permFieldByOp: Record<RecordRuleOp, keyof RoleRecordRule> = {
        read: 'PermRead',
        write: 'PermWrite',
        create: 'PermCreate',
        delete: 'PermDelete',
      };
      const permField = permFieldByOp[opValue];

      const baseAnd: any[] = [
        ['RoleId', 'in', roleIds],
        [permField as any, '=', true],
      ];

      /**
       * Check whether any record rule exists in the requested scope.
       */
      const hasAnyRule = async (scopeAnd: any[]): Promise<boolean> => {
        const n = Number(
          await RoleRecordRule.Count({
            And: [...scopeAnd, ...baseAnd],
          } as any)
        );
        if (!Number.isFinite(n)) throw new Error('invalid_role_record_rule_count');
        return n > 0;
      };

      type PickedScope = 'model' | 'application' | 'global';

      const modelScopeAnd: any[] = [
        ['IrModelId', '=', modelId],
        ['IrApplicationId', 'is', null],
      ];
      const applicationScopeAnd: any[] = [
        ['IrModelId', 'is', null],
        ['IrApplicationId', '=', irApplicationId],
      ];
      const globalScopeAnd: any[] = [
        ['IrModelId', 'is', null],
        ['IrApplicationId', 'is', null],
      ];

      let picked: PickedScope | null = null;
      let pickedScopeAnd: any[] = [];

      if (await hasAnyRule(modelScopeAnd)) {
        picked = 'model';
        pickedScopeAnd = modelScopeAnd;
      } else if (irApplicationId && (await hasAnyRule(applicationScopeAnd))) {
        picked = 'application';
        pickedScopeAnd = applicationScopeAnd;
      } else if (await hasAnyRule(globalScopeAnd)) {
        picked = 'global';
        pickedScopeAnd = globalScopeAnd;
      }

      const rules =
        picked == null
          ? []
          : await RoleRecordRule.Search(
              {
                And: [...pickedScopeAnd, ...baseAnd],
              } as any,
              { fields: ['RoleId', 'Condition'], limit: 5000 }
            );

      if (!rules || rules.length === 0) {
        return { kind: 'true', reason: `no_rules_${opValue}_allow` };
      }

      // 5) OR-merge all conditions (empty/null condition means TRUE).
      // If the target model is company-scoped, AND-gate each rule by the role's company scope
      // to prevent cross-company leakage when ctx.enabledCompanyIds spans multiple companies.

      /**
       * Build the company gate expression for a role scope when company gating is active.
       */
      const buildCompanyGate = (scope: RoleScope): any => {
        if (!companyGate.enabled) return null;
        if (scope.global) return null;
        const ids = scope.companies || [];
        // No companies means the rule can only apply to shared rows.
        const companyIn: any = ['CompanyId', 'in', ids] as any;
        const shared: any = ['CompanyId', 'is', null] as any;
        return { Or: [companyIn, shared] } as any;
      };

      const exprs = (rules || [])
        .map(r => {
          const roleId = this._maybeId((r as any).RoleId) || '';
          const scope = roleScopesById?.[roleId] || { global: true, companies: [] };
          const gate = buildCompanyGate(scope);
          const cond = (r as any).Condition;

          const isTrueCond = cond === undefined || cond === null || (Array.isArray(cond) && cond.length === 0);

          if (isTrueCond) {
            // Empty condition = TRUE; still apply company gate if enabled
            return gate;
          }
          if (!gate) return cond;
          return { And: [gate, cond] } as any;
        })
        .filter(v => v !== undefined && v !== null);

      if (exprs.length === 0) {
        return { kind: 'true', reason: companyGate.enabled ? 'rules_with_empty_condition_or_company_gate' : 'rules_with_empty_condition' };
      }
      if (exprs.length === 1) {
        return { kind: 'expr', expr: exprs[0], reason: 'single_rule' };
      }
      return { kind: 'expr', expr: { Or: exprs } as any, reason: 'or_merged' };
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

      return await this._withPermissionGraphBypass(async () => {
        // 0) Resolve application ids (meta.IrApplication.Name == appName)
        // NOTE: application name is unique in most setups, but tests/bootstrap may re-materialize rows.
        // We intentionally accept multiple ids and match rules against any of them.
        const apps = await IrApplication.Search(
          ['Name', '=', parsedModel.appName] as any,
          { fields: ['Id', 'UpdatedAt'], orderBy: { field: 'UpdatedAt', order: 'desc' }, limit: 5000 } as any
        );
        const applicationIds: string[] = [];
        const applicationIdSet = new Set<string>();
        for (const a of apps || []) {
          const id = String((a as any)?.Id || '').trim();
          if (!id) continue;
          if (applicationIdSet.has(id)) continue;
          applicationIdSet.add(id);
          applicationIds.push(id);
        }
        if (applicationIds.length === 0) {
          throw newAuthError({
            code: AuthErrCode.VALIDATION_FAILED,
            message: 'Application does not exist',
          })
            .withGrpcCode(GrpcCode.InvalidArgument)
            .withMetadata({ application: parsedModel.appName, model: rawModel });
        }

        // 1) Resolve meta model ids.
        // IMPORTANT: meta_ir_model can be re-materialized during tests/bootstrapping.
        // We have observed cases where a RoleFieldRule references an IrModelId whose meta_ir_model row
        // either has a different Application value or is missing it. To make model-scope overrides work
        // reliably, we intentionally match ALL meta models by Name (ignoring Application) and then
        // normalize fields by logical name downstream.
        const models = await IrModel.Search(
          ['Name', '=', parsedModel.modelName] as any,
          { fields: ['Id', 'UpdatedAt'], orderBy: { field: 'UpdatedAt', order: 'desc' }, limit: 5000 } as any
        );
        const modelIds: string[] = [];
        const modelIdSet = new Set<string>();
        for (const m of models || []) {
          const id = String((m as any)?.Id || '').trim();
          if (!id) continue;
          if (modelIdSet.has(id)) continue;
          modelIdSet.add(id);
          modelIds.push(id);
        }
        if (modelIds.length === 0) {
          throw newAuthError({
            code: AuthErrCode.VALIDATION_FAILED,
            message: 'Model does not exist',
          })
            .withGrpcCode(GrpcCode.InvalidArgument)
            .withMetadata({ model: rawModel });
        }

        // 2) Reuse request-scoped authz context (company-view aware)
        const authz = await this._getAuthzContext();
        const roleIds = Array.isArray(authz?.roles) ? authz.roles : [];

        if (roleIds.length === 0) {
          // allow-by-default
          return { denyReadFields: [], denyWriteFields: [], reason: 'no_roles_allow_by_default' };
        }

        // 3) Load meta fields for the model ids (Id + Name)
        const fields = await IrField.Search(['ModelId', 'in', modelIds] as any, { fields: ['Id', 'Name'], limit: 5000 } as any);
        const fieldNameById = new Map<string, string>();
        const fieldIdsByName = new Map<string, string[]>();
        const fieldIdSet = new Set<string>();
        const SYSTEM_FIELDS = new Set(['Id', 'CreatedAt', 'UpdatedAt', 'DeletedAt', 'DisplayName']);
        for (const f of fields || []) {
          const id = String((f as any)?.Id || '').trim();
          const name = String((f as any)?.Name || '').trim();
          if (!id || !name) continue;
          if (SYSTEM_FIELDS.has(name)) continue;
          fieldNameById.set(id, name);
          if (!fieldIdSet.has(id)) fieldIdSet.add(id);
          const xs = fieldIdsByName.get(name) || [];
          xs.push(id);
          fieldIdsByName.set(name, xs);
        }

        // If the model has no fields, treat as allow-by-default.
        if (fieldIdsByName.size === 0) {
          return { denyReadFields: [], denyWriteFields: [], reason: 'no_fields_allow_by_default' };
        }

        /**
         * Normalize a model, field, or application reference into a string id.
         */
        const normalizeId = (v: any): string | null => {
          if (v == null) return null;
          if (typeof v === 'string') {
            const s = v.trim();
            return s ? s : null;
          }

          // ManyToOneRef values may roundtrip as objects in some runtimes (e.g. { Id }, { id }, { value }).
          if (typeof v === 'object') {
            const raw = (v as any)?.Id ?? (v as any)?.id ?? (v as any)?.value ?? (v as any)?.Value;
            const s = String(raw ?? '').trim();
            return s ? s : null;
          }

          const s = String(v ?? '').trim();
          return s ? s : null;
        };

        /**
         * Normalize a FieldRule selection value into allow or deny.
         */
        const normalizePerm = (v: any): 'allow' | 'deny' | null => {
          if (v == null) return null;

          // Selection fields may roundtrip as objects in some runtimes (e.g. { value, label }).
          if (typeof v === 'object') {
            const raw = (v as any)?.value ?? (v as any)?.Value ?? (v as any)?.id ?? (v as any)?.Id;
            if (raw != null && raw !== v) return normalizePerm(raw);
          }

          const s = String(v ?? '')
            .trim()
            .toLowerCase();
          if (!s) return null;
          if (s === 'allow' || s === 'deny') return s;
          return null;
        };

        // 4) Load all candidate rules (field/model/application/global) for current roles.
        // We match against ANY meta ids of the same logical model/application (see steps 0/1).
        // NOTE: '= null' will be normalized to IS NULL by repository/query normalizer.
        const rules = await RoleFieldRule.Search(
          {
            And: [
              ['RoleId', 'in', roleIds],
              {
                Or: [
                  // field exact
                  {
                    And: [
                      ['IrModelId', 'in', modelIds],
                      ['IrFieldId', 'in', Array.from(fieldIdSet)],
                      ['IrApplicationId', '=', null],
                    ],
                  },
                  // model wildcard
                  {
                    And: [
                      ['IrModelId', 'in', modelIds],
                      ['IrFieldId', '=', null],
                      ['IrApplicationId', '=', null],
                    ],
                  },
                  // application wildcard
                  {
                    And: [
                      ['IrApplicationId', 'in', applicationIds],
                      ['IrModelId', '=', null],
                      ['IrFieldId', '=', null],
                    ],
                  },
                  // global wildcard
                  {
                    And: [
                      ['IrApplicationId', '=', null],
                      ['IrModelId', '=', null],
                      ['IrFieldId', '=', null],
                    ],
                  },
                ],
              },
            ],
          } as any,
          {
            fields: ['Id', 'IrApplicationId', 'IrModelId', 'IrFieldId', 'PermRead', 'PermWrite'],
            limit: 5000,
          } as any
        );

        if (!rules || rules.length === 0) {
          return { denyReadFields: [], denyWriteFields: [], reason: 'no_field_rules_allow_by_default' };
        }

        // 5) Partition rules by scope
        // IMPORTANT: field/model ids can differ across meta re-materializations.
        // We normalize everything down to the logical field *name*.
        const fieldRulesByFieldName = new Map<string, any[]>();
        const modelRules: any[] = [];
        const appRules: any[] = [];
        const globalRules: any[] = [];

        /**
         * Read a field from search rows across multiple runtime key shapes.
         */
        const pick = (obj: any, keys: string[]): any => {
          if (!obj || (typeof obj !== 'object' && typeof obj !== 'function')) return undefined;

          // 1) Fast path: direct key hits (own or prototype)
          for (const k of keys) {
            // Search() results may be class instances with prototype getters in some runtimes.
            // Use `in` so we can read both own props and prototype-backed fields (including null).
            if (k in obj) return (obj as any)[k];
          }

          // 2) Fallback: case/underscore-insensitive match across own + prototype keys.
          // Some driver/plugin combos may produce keys like `irFieldId`/`ir_field_id`/`IrFieldId`.
          const norm = (s: string): string =>
            String(s ?? '')
              .replace(/[^a-zA-Z0-9]/g, '')
              .toLowerCase();

          const normalizedToActual = new Map<string, string>();
          let cur: any = obj;
          while (cur && cur !== Object.prototype) {
            for (const k of Reflect.ownKeys(cur)) {
              if (typeof k !== 'string') continue;
              const nk = norm(k);
              if (!nk) continue;
              if (!normalizedToActual.has(nk)) normalizedToActual.set(nk, k);
            }
            cur = Object.getPrototypeOf(cur);
          }

          for (const want of keys) {
            const actual = normalizedToActual.get(norm(want));
            if (!actual) continue;
            try {
              return (obj as any)[actual];
            } catch {
              // ignore
            }
          }

          return undefined;
        };

        for (const r of rules || []) {
          const rid = String((r as any)?.Id ?? '').trim();
          const irApp = normalizeId(pick(r, ['IrApplicationId', 'ir_application_id', 'irApplicationId']));
          const irModel = normalizeId(pick(r, ['IrModelId', 'ir_model_id', 'irModelId']));
          const irField = normalizeId(pick(r, ['IrFieldId', 'ir_field_id', 'irFieldId']));
          const permRead = normalizePerm(pick(r, ['PermRead', 'perm_read', 'permRead']));
          const permWrite = normalizePerm(pick(r, ['PermWrite', 'perm_write', 'permWrite']));

          // Materialize a minimal, normalized view of the rule.
          // IMPORTANT: do not spread `r` (Search results may expose fields via prototype getters).
          const rule = {
            __rid: rid,
            irApp,
            irModel,
            irField,
            permRead,
            permWrite,
          };

          // Prefer not to trust invalid shapes (fail-closed by ignoring them).
          // DB CHECK + model Create/Update validation should prevent these.
          const isField = irField != null && irModel != null && irApp == null;
          const isModel = irField == null && irModel != null && irApp == null;
          const isApp = irField == null && irModel == null && irApp != null;
          const isGlobal = irField == null && irModel == null && irApp == null;

          if (isField) {
            if (!fieldIdSet.has(irField)) continue;
            if (!modelIdSet.has(irModel)) continue;
            const fieldName = fieldNameById.get(irField);
            if (!fieldName) continue;
            const xs = fieldRulesByFieldName.get(fieldName) || [];
            xs.push(rule);
            fieldRulesByFieldName.set(fieldName, xs);
          } else if (isModel) {
            if (!modelIdSet.has(irModel)) continue;
            modelRules.push(rule);
          } else if (isApp) {
            if (!applicationIdSet.has(irApp)) continue;
            appRules.push(rule);
          } else if (isGlobal) {
            globalRules.push(rule);
          }
        }

        /**
         * Resolve the effective decision inside one rule bucket.
         */
        const decideInScope = (xs: any[], dim: 'read' | 'write'): 'allow' | 'deny' | undefined => {
          let hasAllow = false;
          for (const r of xs || []) {
            const v = dim === 'read' ? (r as any)?.permRead : (r as any)?.permWrite;
            if (v === 'deny') return 'deny';
            if (v === 'allow') hasAllow = true;
          }
          return hasAllow ? 'allow' : undefined;
        };

        /**
         * Resolve the effective read or write decision for one logical field.
         */
        const decideEffective = (fieldName: string, dim: 'read' | 'write'): 'allow' | 'deny' => {
          const buckets: any[][] = [fieldRulesByFieldName.get(fieldName) || [], modelRules, appRules, globalRules];
          for (const b of buckets) {
            const d = decideInScope(b, dim);
            if (d) return d;
          }
          return 'allow';
        };

        const denyReadFields: string[] = [];
        const denyWriteFields: string[] = [];
        const fieldNames = Array.from(fieldIdsByName.keys()).sort();
        for (const name of fieldNames) {
          const readDecision = decideEffective(name, 'read');
          let writeDecision = decideEffective(name, 'write');

          // Consistency fail-closed: write implies read
          if (readDecision === 'deny') writeDecision = 'deny';

          if (readDecision === 'deny') denyReadFields.push(name);
          if (writeDecision === 'deny') denyWriteFields.push(name);
        }

        denyReadFields.sort();
        denyWriteFields.sort();

        return { denyReadFields, denyWriteFields, reason: 'ok' };
      });
    } catch (error) {
      throw wrapAuthError(error, {
        code: AuthErrCode.UNKNOWN,
        message: 'Failed to compute field rule spec',
      }).withMetadata({ model: String(model || '').trim() });
    }
  }

  /**
   * Hash a password before persistence.
   */
  private static _hashPassword(password: string): string {
    // Detect client-side hashed inputs by prefix.
    const prefixMarker = '$CH$';
    if (password.startsWith(prefixMarker)) {
      // Strip the prefix and hash again on the server to complete double hashing.
      const clientHashed = password.substring(prefixMarker.length);
      return $choysum.crypto.hashPassword(clientHashed);
    }

    // Fall back to the traditional single server-side hash.
    return $choysum.crypto.hashPassword(password);
  }

  /**
   * Verify a plaintext or client-hashed password against the stored hash.
   */
  private static _verifyPassword(password: string, hashedPassword: string): boolean {
    // Detect client-side hashed inputs by prefix.
    const prefixMarker = '$CH$';
    if (password.startsWith(prefixMarker)) {
      // Strip the client hash prefix before verification.
      password = password.substring(prefixMarker.length);
    }

    return $choysum.crypto.verifyPassword(password, hashedPassword);
  }
}
