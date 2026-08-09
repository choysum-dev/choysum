// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { GrpcCode, ChoysumError } from '@/core/service/error';
import { ValidationPipelineError } from '../../../metadata';
import { selectPrimaryValidationIssue, wrapRepositoryValidationError } from '..';

test('repository validation error helper builds rich metadata for sql/kernel/global issues', () => {
  const error = new ValidationPipelineError('pipeline failed', [
    {
      scope: 'sql',
      field: 'Name',
      method: 'checkUnique',
      code: 'sql_unique',
      message: 'name duplicate',
      severity: 'error',
      meta: {
        sqlConstraint: ' uq_name ',
        sqlField: ' Name ',
        sqlColumns: ['Name', '', null],
      },
    },
    {
      scope: 'kernel',
      field: 'Age',
      code: 'kernel_int_invalid',
      message: 'age invalid',
      severity: 'warning',
    },
    {
      scope: 'platform',
      code: 'platform_reject_unknown',
      message: 'unknown field',
      severity: 'error',
    },
  ] as any);

  const wrapped = wrapRepositoryValidationError(
    {
      fullModelName: '',
      modelName: 'demo.Model',
      name: '',
    } as any,
    error,
    'create'
  );

  expect(wrapped instanceof ChoysumError).toBe(true);
  expect(wrapped.code).toBe('validation_failed');
  expect(wrapped.grpcCode).toBe(GrpcCode.InvalidArgument);
  expect(wrapped.message).toBe('name duplicate');

  expect(wrapped.metadata.mode).toBe('create');
  expect(wrapped.metadata.issueCount).toBe('3');
  expect(wrapped.metadata.model).toBe('demo.Model');
  expect(wrapped.metadata.scope).toBe('sql');
  expect(wrapped.metadata.field).toBe('Name');
  expect(wrapped.metadata.method).toBe('checkUnique');
  expect(wrapped.metadata.issueCode).toBe('sql_unique');
  expect(wrapped.metadata.sqlCode).toBe('sql_unique');
  expect(wrapped.metadata.sqlConstraint).toBe('uq_name');
  expect(wrapped.metadata.sqlField).toBe('Name');
  expect(wrapped.metadata.sqlColumns).toBe('Name');
  expect(wrapped.metadata.kernelCode).toBe('kernel_int_invalid');
  expect(wrapped.metadata.fields).toBe('Name,Age');
  expect(wrapped.metadata.fieldCount).toBe('2');
  expect(wrapped.metadata.globalIssueCount).toBe('1');

  const fieldIssues = JSON.parse(wrapped.metadata.fieldIssues || '{}');
  expect(Object.keys(fieldIssues).sort()).toEqual(['Age', 'Name']);

  const fieldSummary = JSON.parse(wrapped.metadata.fieldIssueSummary || '{}');
  expect(fieldSummary.Name.firstCode).toBe('sql_unique');
  expect(fieldSummary.Age.firstCode).toBe('kernel_int_invalid');
  expect(fieldSummary.Age.firstKernelCode).toBe('kernel_int_invalid');
  expect(fieldSummary.Age.kernelCode).toBe('kernel_int_invalid');

  const globalIssues = JSON.parse(wrapped.metadata.globalIssues || '[]');
  expect(globalIssues.length).toBe(1);
  expect(globalIssues[0].code).toBe('platform_reject_unknown');

  const issues = JSON.parse(wrapped.metadata.issues || '[]');
  expect(issues.length).toBe(3);
  expect(issues[0].field).toBe('Name');
  expect(issues[1].field).toBe('Age');
  expect(issues[2].field).toBe(undefined);
});

test('repository validation error helper prefers status-bearing constraint issue over earlier kernel errors', () => {
  const wrapped = wrapRepositoryValidationError(
    {
      fullModelName: 'web.SavedFilter',
      modelName: 'SavedFilter',
      name: '',
    } as any,
    new ValidationPipelineError('pipeline failed', [
      {
        scope: 'kernel',
        field: 'Name',
        code: 'required',
        message: 'Name is required',
        severity: 'error',
      },
      {
        scope: 'constraint',
        method: 'validateSavedFilterConstraint',
        code: 'constraint_execution_failed',
        message: 'Authentication required',
        severity: 'error',
        meta: {
          causeCode: 'PermissionDenied',
          causeDomain: 'web',
          grpcCode: GrpcCode.Unauthenticated,
        },
      },
    ] as any),
    'create'
  );

  expect(wrapped.grpcCode).toBe(GrpcCode.Unauthenticated);
  expect(wrapped.message).toBe('Authentication required');
  expect(wrapped.metadata.causeCode).toBe('PermissionDenied');
  expect(wrapped.metadata.causeDomain).toBe('web');
  expect(wrapped.metadata.issueCode).toBe('constraint_execution_failed');
});

test('selectPrimaryValidationIssue falls back to first error when no grpc meta is present', () => {
  const primary = selectPrimaryValidationIssue([
    { scope: 'kernel', code: 'required', message: 'a', severity: 'error' },
    { scope: 'platform', code: 'platform_x', message: 'b', severity: 'error' },
  ] as any);
  expect(primary?.code).toBe('required');
});

test('selectPrimaryValidationIssue ignores OK, non-integer, and out-of-range grpcCode as status-bearing', () => {
  const issues = [
    {
      scope: 'constraint',
      code: 'bad_ok',
      message: 'ok code',
      severity: 'error',
      meta: { grpcCode: 0 },
    },
    {
      scope: 'constraint',
      code: 'bad_frac',
      message: 'fraction',
      severity: 'error',
      meta: { grpcCode: 7.5 },
    },
    {
      scope: 'constraint',
      code: 'bad_range',
      message: 'range',
      severity: 'error',
      meta: { grpcCode: 99 },
    },
    {
      scope: 'kernel',
      code: 'required',
      message: 'fallback',
      severity: 'error',
    },
  ] as any;
  // Invalid grpcCode values are not status-bearing, so selection falls back to the first error.
  expect(selectPrimaryValidationIssue(issues)?.code).toBe('bad_ok');

  const wrapped = wrapRepositoryValidationError(
    { fullModelName: 'demo.Model', modelName: 'Model', name: '' } as any,
    new ValidationPipelineError('pipeline failed', issues),
    'create'
  );
  expect(wrapped.grpcCode).toBe(GrpcCode.InvalidArgument);
  expect(wrapped.message).toBe('ok code');
});

test('repository validation error helper falls back message and keeps minimal metadata when issues are empty', () => {
  const wrapped = wrapRepositoryValidationError(
    {
      fullModelName: '',
      modelName: '',
      name: '',
    } as any,
    new ValidationPipelineError('', []),
    'update'
  );

  expect(wrapped.code).toBe('validation_failed');
  expect(wrapped.message).toBe('validation failed');
  expect(wrapped.metadata.mode).toBe('update');
  expect(wrapped.metadata.issueCount).toBe('0');
  expect(wrapped.metadata.model).toBeUndefined();
  expect(wrapped.metadata.scope).toBeUndefined();
  expect(wrapped.metadata.field).toBeUndefined();
  expect(wrapped.metadata.method).toBeUndefined();
  expect(wrapped.metadata.issueCode).toBeUndefined();
  expect(wrapped.metadata.sqlCode).toBeUndefined();
  expect(wrapped.metadata.kernelCode).toBeUndefined();
  expect(wrapped.metadata.fieldIssues).toBeUndefined();
  expect(wrapped.metadata.fieldIssueSummary).toBeUndefined();
  expect(wrapped.metadata.globalIssues).toBeUndefined();
  expect(wrapped.metadata.issues).toBe('[]');
});

test('repository validation error helper omits sql metadata when sql issue metadata is blank', () => {
  const wrapped = wrapRepositoryValidationError(
    {
      fullModelName: 'demo.Model',
      modelName: '',
      name: '',
    } as any,
    new ValidationPipelineError('sql fallback', [
      {
        scope: 'sql',
        field: 'Code',
        code: 'sql_unknown',
        message: '',
        severity: 'warning',
        meta: {
          sqlConstraint: '   ',
          sqlField: '',
          sqlColumns: 'not-array',
        },
      },
    ] as any),
    'preview'
  );

  expect(wrapped.message).toBe('sql fallback');
  expect(wrapped.metadata.model).toBe('demo.Model');
  expect(wrapped.metadata.sqlCode).toBe('sql_unknown');
  expect(wrapped.metadata.sqlConstraint).toBeUndefined();
  expect(wrapped.metadata.sqlField).toBeUndefined();
  expect(wrapped.metadata.sqlColumns).toBeUndefined();
});

test('repository validation error helper normalizes missing issue scope/code/severity and sql meta fallbacks', () => {
  const wrapped = wrapRepositoryValidationError(
    {
      fullModelName: '',
      modelName: '',
      name: 'demo.Model',
    } as any,
    new ValidationPipelineError('fallback issue payload', [
      {
        scope: 'sql',
        field: 'Code',
        code: 'sql_fallback',
        message: 'sql fallback',
        severity: 'error',
        meta: {},
      },
      {
        field: 'Code',
        message: 'missing scope/code/severity',
      },
    ] as any),
    'update'
  );

  expect(wrapped.message).toBe('sql fallback');
  expect(wrapped.metadata.model).toBe('demo.Model');
  expect(wrapped.metadata.sqlCode).toBe('sql_fallback');
  expect(wrapped.metadata.sqlConstraint).toBeUndefined();
  expect(wrapped.metadata.sqlField).toBeUndefined();
  expect(wrapped.metadata.sqlColumns).toBeUndefined();

  const fieldIssues = JSON.parse(wrapped.metadata.fieldIssues || '{}');
  expect(fieldIssues.Code.length).toBe(2);
  expect(fieldIssues.Code[1]).toEqual({
    scope: '',
    code: '',
    message: 'missing scope/code/severity',
    severity: '',
  });

  const fieldSummary = JSON.parse(wrapped.metadata.fieldIssueSummary || '{}');
  expect(fieldSummary.Code.issueCount).toBe('2');
});

test('repository validation error helper handles sql issue without meta payload', () => {
  const wrapped = wrapRepositoryValidationError(
    {
      fullModelName: 'demo.Model',
      modelName: '',
      name: '',
    } as any,
    new ValidationPipelineError('sql no meta', [
      {
        scope: 'sql',
        field: 'Code',
        code: 'sql_no_meta',
        message: 'sql no meta',
        severity: 'error',
      },
    ] as any),
    'create'
  );

  expect(wrapped.metadata.model).toBe('demo.Model');
  expect(wrapped.metadata.sqlCode).toBe('sql_no_meta');
  expect(wrapped.metadata.sqlConstraint).toBeUndefined();
  expect(wrapped.metadata.sqlField).toBeUndefined();
  expect(wrapped.metadata.sqlColumns).toBeUndefined();
});
