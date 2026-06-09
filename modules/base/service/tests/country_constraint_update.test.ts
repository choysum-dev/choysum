// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import Country from '@/base/service/models/country';
import { GrpcCode, ChoysumError } from '@/core/service/error';
import { Constraint, ValidationPipelineError } from '@/core/service/api/constraint';
import { getReadonlyCtx, withContext as withModelContext } from '@/core/service/api/context';

import { countryCode8, uid } from './_helpers';

Object.defineProperty(Country, 'validateConstraintCurrentName', {
  value: function validateConstraintCurrentName(self: Country, ctx: any) {
    if (ctx.mode !== 'update') return;
    if (ctx.current?.Name === 'country_before_update' && self.Name === 'country_after_blocked') {
      throw new Error(`blocked rename from ${ctx.current?.Name} to ${self.Name}`);
    }
  },
  configurable: true,
  writable: true,
});

Constraint<Country>('Name', { priority: 500 })(Country, 'validateConstraintCurrentName', undefined as any);

test('base.country: repository update passes current row into constraint context', async () => {
  const created = await Country.Create(
    {
      Name: 'country_before_update',
      Code: countryCode8(),
      ZipRequired: false,
      StateRequired: false,
      IsActive: true,
    } as any,
    ['Id'] as any
  );

  let error: unknown;
  try {
    await Country.UpdateById(
      String((created as any).Id),
      {
        Name: 'country_after_blocked',
      } as any,
      ['Id', 'Name'] as any
    );
  } catch (err) {
    error = err;
  }

  expect(error instanceof ChoysumError).toBe(true);
  const repositoryError = error as ChoysumError;
  expect(repositoryError.domain).toBe('core.repository');
  expect(repositoryError.code).toBe('validation_failed');
  expect(repositoryError.grpcCode).toBe(GrpcCode.InvalidArgument);
  expect(repositoryError.metadata.mode).toBe('update');
  expect(repositoryError.metadata.method).toBe('validateConstraintCurrentName');

  expect((repositoryError as any).cause instanceof ValidationPipelineError).toBe(true);
  const pipelineError = (repositoryError as any).cause as ValidationPipelineError;
  expect(pipelineError.issues.length).toBe(1);
  expect(pipelineError.issues[0]?.method).toBe('validateConstraintCurrentName');
  expect(pipelineError.issues[0]?.message).toBe('blocked rename from country_before_update to country_after_blocked');

  const updated = await Country.UpdateById(
    String((created as any).Id),
    {
      Name: uid('country_after_ok'),
    } as any,
    ['Id', 'Name'] as any
  );

  expect(Boolean((updated as any).Id)).toBe(true);
  expect(String((updated as any).Name).startsWith('country_after_ok_')).toBe(true);
});

test('base.country: repository update rejects writes to select-only fields', async () => {
  const created = await Country.Create(
    {
      Name: uid('country_select_field'),
      Code: countryCode8(),
      ZipRequired: false,
      StateRequired: false,
      IsActive: true,
    } as any,
    ['Id'] as any
  );

  let error: unknown;
  try {
    await Country.UpdateById(
      String((created as any).Id),
      {
        DisplayName: 'forbidden_display_name',
      } as any,
      ['Id', 'DisplayName'] as any
    );
  } catch (err) {
    error = err;
  }

  expect(error instanceof ChoysumError).toBe(true);
  const repositoryError = error as ChoysumError;
  expect(repositoryError.domain).toBe('core.repository');
  expect(repositoryError.code).toBe('validation_failed');
  expect(repositoryError.grpcCode).toBe(GrpcCode.InvalidArgument);
  expect(repositoryError.metadata.mode).toBe('update');
  expect(repositoryError.metadata.field).toBe('DisplayName');
  expect(repositoryError.metadata.issueCode).toBe('platform_write_to_select_field');
  expect(repositoryError.metadata.fields).toBe('DisplayName');
  expect(repositoryError.metadata.fieldCount).toBe('1');

  const fieldIssues = JSON.parse(repositoryError.metadata.fieldIssues || '{}') as Record<string, any[]>;
  expect(Array.isArray(fieldIssues.DisplayName)).toBe(true);
  expect(fieldIssues.DisplayName?.[0]?.scope).toBe('platform');
  expect(fieldIssues.DisplayName?.[0]?.code).toBe('platform_write_to_select_field');

  expect((repositoryError as any).cause instanceof ValidationPipelineError).toBe(true);
  const pipelineError = (repositoryError as any).cause as ValidationPipelineError;
  expect(pipelineError.issues.length).toBe(1);
  expect(pipelineError.issues[0]?.scope).toBe('platform');
  expect(pipelineError.issues[0]?.field).toBe('DisplayName');
  expect(pipelineError.issues[0]?.code).toBe('platform_write_to_select_field');
});

test('base.country: repository create rejects writes to select-only fields by default', async () => {
  let error: unknown;
  try {
    await Country.Create(
      {
        Name: uid('country_create_select_deny'),
        Code: countryCode8(),
        DisplayName: 'forbidden_create_display_name',
        ZipRequired: false,
        StateRequired: false,
        IsActive: true,
      } as any,
      ['Id'] as any
    );
  } catch (err) {
    error = err;
  }

  expect(error instanceof ChoysumError).toBe(true);
  const repositoryError = error as ChoysumError;
  expect(repositoryError.domain).toBe('core.repository');
  expect(repositoryError.code).toBe('validation_failed');
  expect(repositoryError.grpcCode).toBe(GrpcCode.InvalidArgument);
  expect(repositoryError.metadata.mode).toBe('create');
  expect(repositoryError.metadata.field).toBe('DisplayName');
  expect(repositoryError.metadata.issueCode).toBe('platform_write_to_select_field');
});

test('base.country: repository create accepts select-only field via context whitelist', async () => {
  const created = await withModelContext(
    {
      validation: {
        platformCreateWriteWhitelistByModel: {
          'base.Country': ['DisplayName'],
        },
      },
    } as any,
    async () =>
      await Country.Create(
        {
          Name: uid('country_create_select_allow'),
          Code: countryCode8(),
          DisplayName: 'allowed_create_display_name',
          ZipRequired: false,
          StateRequired: false,
          IsActive: true,
        } as any,
        ['Id'] as any
      )
  );

  expect(Boolean((created as any).Id)).toBe(true);
});

test('base.country: model-specific create whitelist overrides global whitelist', async () => {
  let error: unknown;

  await withModelContext(
    {
      validation: {
        platformCreateWriteWhitelist: ['DisplayName'],
        platformCreateWriteWhitelistByModel: {
          'base.Country': [],
        },
      },
    } as any,
    async () => {
      try {
        await Country.Create(
          {
            Name: uid('country_create_select_model_override'),
            Code: countryCode8(),
            DisplayName: 'should_be_denied_by_model_override',
            ZipRequired: false,
            StateRequired: false,
            IsActive: true,
          } as any,
          ['Id'] as any
        );
      } catch (err) {
        error = err;
      }
    }
  );

  expect(error instanceof ChoysumError).toBe(true);
  const repositoryError = error as ChoysumError;
  expect(repositoryError.domain).toBe('core.repository');
  expect(repositoryError.code).toBe('validation_failed');
  expect(repositoryError.metadata.mode).toBe('create');
  expect(repositoryError.metadata.issueCode).toBe('platform_write_to_select_field');
});

test('base.country: repository create rejects unknown fields when platform unknown guard is enabled', async () => {
  let error: unknown;

  await withModelContext(
    {
      validation: {
        platformRejectUnknownFields: true,
      },
    } as any,
    async () => {
      try {
        await Country.Create(
          {
            Name: uid('country_create_unknown_field'),
            Code: countryCode8(),
            UnknownWriteField: 'forbidden',
            ZipRequired: false,
            StateRequired: false,
            IsActive: true,
          } as any,
          ['Id'] as any
        );
      } catch (err) {
        error = err;
      }
    }
  );

  expect(error instanceof ChoysumError).toBe(true);
  const repositoryError = error as ChoysumError;
  expect(repositoryError.domain).toBe('core.repository');
  expect(repositoryError.code).toBe('validation_failed');
  expect(repositoryError.metadata.mode).toBe('create');
  expect(repositoryError.metadata.field).toBe('UnknownWriteField');
  expect(repositoryError.metadata.issueCode).toBe('platform_unknown_field');
});

test('base.country: repository create writes whitelist audit into request context', async () => {
  (globalThis as any).__choysumValidationAudit = { version: 1, platformCreateWhitelistHits: [] };

  const auditHits = await withModelContext(
    {
      validation: {
        platformCreateWriteWhitelistByModel: {
          'base.Country': ['DisplayName'],
        },
      },
    } as any,
    async () => {
      await Country.Create(
        {
          Name: uid('country_create_audit_whitelist'),
          Code: countryCode8(),
          DisplayName: 'audit_whitelisted',
          ZipRequired: false,
          StateRequired: false,
          IsActive: true,
        } as any,
        ['Id'] as any
      );

      const ctx = getReadonlyCtx() as any;
      const localHits = (ctx?.__validationAudit?.platformCreateWhitelistHits || []) as Array<Record<string, unknown>>;
      if (localHits.length > 0) {
        return localHits;
      }

      const globalHits = ((globalThis as any).__choysumValidationAudit?.platformCreateWhitelistHits || []) as Array<Record<string, unknown>>;
      return globalHits;
    }
  );

  expect(Array.isArray(auditHits)).toBe(true);
  expect(auditHits.length).toBe(1);
  expect(auditHits[0]?.version).toBe(1);
  expect(auditHits[0]?.source === 'request_context' || auditHits[0]?.source === 'global_fallback').toBe(true);
  expect(auditHits[0]?.model).toBe('base.Country');
  expect(auditHits[0]?.mode).toBe('create');
  expect(auditHits[0]?.fields).toEqual(['DisplayName']);
});

test('base.country: repository create accepts select-only field via global whitelist when no model-specific override', async () => {
  const created = await withModelContext(
    {
      validation: {
        platformCreateWriteWhitelist: ['DisplayName'],
      },
    } as any,
    async () =>
      await Country.Create(
        {
          Name: uid('country_create_select_global_allow'),
          Code: countryCode8(),
          DisplayName: 'allowed_by_global_whitelist',
          ZipRequired: false,
          StateRequired: false,
          IsActive: true,
        } as any,
        ['Id'] as any
      )
  );

  expect(Boolean((created as any).Id)).toBe(true);
});

test('base.country: repository update still rejects select-only field even with create whitelist configured', async () => {
  const created = await Country.Create(
    {
      Name: uid('country_update_select_create_whitelist_only'),
      Code: countryCode8(),
      ZipRequired: false,
      StateRequired: false,
      IsActive: true,
    } as any,
    ['Id'] as any
  );

  let error: unknown;
  await withModelContext(
    {
      validation: {
        platformCreateWriteWhitelistByModel: {
          'base.Country': ['DisplayName'],
        },
      },
    } as any,
    async () => {
      try {
        await Country.UpdateById(
          String((created as any).Id),
          {
            DisplayName: 'still_forbidden_on_update',
          } as any,
          ['Id', 'DisplayName'] as any
        );
      } catch (err) {
        error = err;
      }
    }
  );

  expect(error instanceof ChoysumError).toBe(true);
  const repositoryError = error as ChoysumError;
  expect(repositoryError.domain).toBe('core.repository');
  expect(repositoryError.code).toBe('validation_failed');
  expect(repositoryError.metadata.mode).toBe('update');
  expect(repositoryError.metadata.issueCode).toBe('platform_write_to_select_field');
});

test('base.country: repository create ignores unknown fields when platform unknown guard is disabled', async () => {
  const created = await Country.Create(
    {
      Name: uid('country_create_unknown_field_guard_off'),
      Code: countryCode8(),
      UnknownWriteField: 'ignored_when_guard_off',
      ZipRequired: false,
      StateRequired: false,
      IsActive: true,
    } as any,
    ['Id'] as any
  );

  expect(Boolean((created as any).Id)).toBe(true);
});

test('base.country: repository update rejects unknown fields when platform unknown guard is enabled', async () => {
  const created = await Country.Create(
    {
      Name: uid('country_update_unknown_field_guard_on'),
      Code: countryCode8(),
      ZipRequired: false,
      StateRequired: false,
      IsActive: true,
    } as any,
    ['Id'] as any
  );

  let error: unknown;
  await withModelContext(
    {
      validation: {
        platformRejectUnknownFields: true,
      },
    } as any,
    async () => {
      try {
        await Country.UpdateById(
          String((created as any).Id),
          {
            UnknownWriteField: 'forbidden_on_update_when_guard_on',
          } as any,
          ['Id'] as any
        );
      } catch (err) {
        error = err;
      }
    }
  );

  expect(error instanceof ChoysumError).toBe(true);
  const repositoryError = error as ChoysumError;
  expect(repositoryError.domain).toBe('core.repository');
  expect(repositoryError.code).toBe('validation_failed');
  expect(repositoryError.metadata.mode).toBe('update');
  expect(repositoryError.metadata.field).toBe('UnknownWriteField');
  expect(repositoryError.metadata.issueCode).toBe('platform_unknown_field');
});
