// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { BaseModel, Compute, Field } from '@/core/service';
import { Constraint, ValidationPipelineError } from '@/core/service/api/constraint';
import { MetadataStorage } from '@/core/service/api/metadata';
import { ValidationEngine } from '@/core/service/api/validation';
import { RepositoryFactory } from '../../orm/repository/repository_factory';

const engineCallLog: Array<Record<string, unknown>> = [];
type ObjectRecord = Record<string, unknown>;

class ConstraintEngineModel extends BaseModel {
  Name?: string;
  Status?: string;

  static resetLog() {
    engineCallLog.length = 0;
  }

  static checkName(self: ConstraintEngineModel, ctx: any) {
    engineCallLog.push({
      method: 'checkName',
      currentName: ctx.current?.Name,
      mergedName: self.Name,
      mergedStatus: self.Status,
    });
  }

  static checkStatus(self: ConstraintEngineModel) {
    engineCallLog.push({
      method: 'checkStatus',
      mergedStatus: self.Status,
    });

    if (self.Status === 'blocked') {
      throw new Error('status blocked');
    }
  }

  static checkPreview(self: ConstraintEngineModel) {
    engineCallLog.push({
      method: 'checkPreview',
      previewFlag: Boolean((self as any).__preview),
      mergedName: self.Name,
    });
  }
}

class ConstraintEngineEdgeModel extends BaseModel {
  Name?: string;

  static checkRaisesPipeline(_self: ConstraintEngineEdgeModel) {
    throw new ValidationPipelineError('nested validation failed', [
      {
        scope: 'constraint',
        method: 'checkRaisesPipeline',
        code: 'nested_constraint_error',
        message: 'nested pipeline issue',
        severity: 'error',
      },
    ] as any);
  }
}

class PlatformValidationModel extends BaseModel {
  @Field({ type: 'varchar', size: 64 })
  Name?: string;

  @Field({ type: 'varchar', size: 64 })
  VirtualName?: string;

  @Compute<PlatformValidationModel>('VirtualName', {
    deps: ['Name' as any],
    store: false,
  })
  computeVirtualName() {
    return this.Name || '';
  }

  @Field({ type: 'varchar', size: 64 })
  ComputedName?: string;

  @Compute<PlatformValidationModel>('ComputedName', {
    deps: ['Name' as any],
  })
  computeComputedName() {
    return this.Name || '';
  }
}

class PlatformReadonlyModel extends BaseModel {
  @Field({ type: 'varchar', size: 64 })
  Name?: string;
}

class PlatformCompanyTargetModel extends BaseModel {
  @Field({ type: 'varchar', size: 20 })
  CompanyId?: string;
}

class PlatformCompanySourceModel extends BaseModel {
  @Field({
    type: 'ManyToOne',
    relation: { targetModel: () => PlatformCompanyTargetModel },
  })
  TargetId?: PlatformCompanyTargetModel;
}

class PlatformCompanyRefSourceModel extends BaseModel {
  @Field({
    type: 'ManyToOneRef',
    relation: { targetModel: () => PlatformCompanyTargetModel },
  })
  TargetRefId?: string;
}

class PlatformCompanyRefStringSourceModel extends BaseModel {
  @Field({
    type: 'ManyToOneRef',
    relation: { targetModel: 'test.PlatformCompanyTargetModel' },
  })
  TargetRefId?: string;
}

class PlatformBaseCompanyRefSourceModel extends BaseModel {
  @Field({
    type: 'ManyToOneRef',
    relation: { targetModel: 'base.Company' },
  })
  CompanyId?: string;
}

class KernelValidationTargetModel extends BaseModel {
  @Field({ type: 'varchar', size: 20 })
  Name?: string;
}

class KernelValidationModel extends BaseModel {
  @Field({ type: 'varchar', size: 64, notNull: true })
  RequiredName?: string;

  @Field({ type: 'decimal', precision: 5, scale: 2 })
  Amount?: any;

  @Field({
    type: 'ManyToOne',
    relation: { targetModel: () => KernelValidationTargetModel },
  })
  TargetId?: KernelValidationTargetModel;

  @Field({
    type: 'ManyToOneRef',
    relation: { targetModel: 'base.Company' },
  })
  CompanyId?: string;
}

Constraint<ConstraintEngineModel>('Name', { priority: 1, alwaysOnCreate: true })(ConstraintEngineModel, 'checkName', undefined as any);
Constraint<ConstraintEngineModel>('Status', { priority: 2 })(ConstraintEngineModel, 'checkStatus', undefined as any);
Constraint<ConstraintEngineModel>('Name', { priority: 3, preview: true })(ConstraintEngineModel, 'checkPreview', undefined as any);
Constraint<ConstraintEngineEdgeModel>('Name', { priority: 1 })(ConstraintEngineEdgeModel, 'checkRaisesPipeline', undefined as any);
Constraint<ConstraintEngineEdgeModel>('Name', { priority: 2 })(ConstraintEngineEdgeModel, 'checkMissingMethod', undefined as any);

test('validation engine merges current and incoming values for constraint self', async () => {
  ConstraintEngineModel.resetLog();
  const metadata = MetadataStorage.instance.getModelMetadata(ConstraintEngineModel as any);

  await ValidationEngine.validateOrThrow({
    mode: 'update',
    model: ConstraintEngineModel as any,
    metadata,
    current: { Id: '1', Name: 'Old Name', Status: 'draft' },
    values: { Name: 'New Name' },
    changedFields: new Set(['Name']),
    repository: {} as any,
    requestContext: {},
  });

  expect(engineCallLog.length).toBe(2);
  expect(engineCallLog[0]).toEqual({
    method: 'checkName',
    currentName: 'Old Name',
    mergedName: 'New Name',
    mergedStatus: 'draft',
  });
  expect(engineCallLog[1]).toEqual({
    method: 'checkPreview',
    previewFlag: false,
    mergedName: 'New Name',
  });
});

test('validation engine reports constraint failures as pipeline error', async () => {
  ConstraintEngineModel.resetLog();
  const metadata = MetadataStorage.instance.getModelMetadata(ConstraintEngineModel as any);

  let error: unknown;
  try {
    await ValidationEngine.validateOrThrow({
      mode: 'update',
      model: ConstraintEngineModel as any,
      metadata,
      current: { Id: '1', Name: 'Old Name', Status: 'draft' },
      values: { Status: 'blocked' },
      changedFields: new Set(['Status']),
      repository: {} as any,
      requestContext: {},
    });
  } catch (err) {
    error = err;
  }

  expect(error instanceof ValidationPipelineError).toBe(true);
  const pipelineError = error as ValidationPipelineError;
  expect(pipelineError.issues.length).toBe(1);
  expect(pipelineError.issues[0]?.scope).toBe('constraint');
  expect(pipelineError.issues[0]?.method).toBe('checkStatus');
  expect(pipelineError.issues[0]?.code).toBe('constraint_execution_failed');
});

test('validation engine unwraps nested ValidationPipelineError from constraint method', async () => {
  const metadata = MetadataStorage.instance.getModelMetadata(ConstraintEngineEdgeModel as any);

  const issues = await ValidationEngine.validate(
    {
      mode: 'update',
      model: ConstraintEngineEdgeModel as any,
      metadata,
      current: { Id: '1', Name: 'old' },
      values: { Name: 'next' },
      changedFields: new Set(['Name']),
      repository: {} as any,
      requestContext: {},
    },
    {
      includeKernel: false,
      includePlatform: false,
      includeConstraints: true,
    }
  );

  expect(issues.some(issue => issue.code === 'nested_constraint_error')).toBe(true);
  expect(issues.some(issue => issue.code === 'constraint_method_missing')).toBe(true);
});

test('validation engine only runs preview constraints in preview mode', async () => {
  ConstraintEngineModel.resetLog();
  const metadata = MetadataStorage.instance.getModelMetadata(ConstraintEngineModel as any);
  const previewSelf = Object.assign(Object.create(ConstraintEngineModel.prototype), {
    __preview: true,
    Name: 'Preview Name',
    Status: 'draft',
  }) as ConstraintEngineModel;

  await ValidationEngine.validateOrThrow({
    mode: 'preview',
    model: ConstraintEngineModel as any,
    metadata,
    self: previewSelf,
    current: { Id: '1', Name: 'Old Name', Status: 'draft' },
    values: { Name: 'Preview Name' },
    changedFields: new Set(['Name']),
    repository: {} as any,
    requestContext: {},
  });

  expect(engineCallLog).toEqual([
    {
      method: 'checkPreview',
      previewFlag: true,
      mergedName: 'Preview Name',
    },
  ]);
});

test('validation engine reports platform issues for writes to select-only and computed fields', async () => {
  const metadata = MetadataStorage.instance.getModelMetadata(PlatformValidationModel as any);

  const issues = await ValidationEngine.validate(
    {
      mode: 'update',
      model: PlatformValidationModel as any,
      metadata,
      current: { Id: '1', Name: 'Old Name' },
      values: { VirtualName: 'Virtual', ComputedName: 'Computed' },
      changedFields: new Set(['VirtualName', 'ComputedName']),
      repository: {} as any,
      requestContext: {},
    },
    {
      includeKernel: false,
      includeConstraints: false,
    }
  );

  expect(issues).toEqual([
    {
      scope: 'platform',
      field: 'VirtualName',
      code: 'platform_write_to_select_field',
      message: 'field "VirtualName" is select-only and cannot be written',
      severity: 'error',
    },
    {
      scope: 'platform',
      field: 'ComputedName',
      code: 'platform_write_to_computed_field',
      message: 'field "ComputedName" is computed and cannot be written directly',
      severity: 'error',
    },
  ]);
});

test('validation engine reports platform issues for create writes to select-only and computed fields', async () => {
  const metadata = MetadataStorage.instance.getModelMetadata(PlatformValidationModel as any);

  const issues = await ValidationEngine.validate(
    {
      mode: 'create',
      model: PlatformValidationModel as any,
      metadata,
      values: { VirtualName: 'Virtual', ComputedName: 'Computed' },
      changedFields: new Set(['VirtualName', 'ComputedName']),
      repository: {} as any,
      requestContext: {},
    },
    {
      includeKernel: false,
      includeConstraints: false,
    }
  );

  expect(issues).toEqual([
    {
      scope: 'platform',
      field: 'VirtualName',
      code: 'platform_write_to_select_field',
      message: 'field "VirtualName" is select-only and cannot be written',
      severity: 'error',
    },
    {
      scope: 'platform',
      field: 'ComputedName',
      code: 'platform_write_to_computed_field',
      message: 'field "ComputedName" is computed and cannot be written directly',
      severity: 'error',
    },
  ]);
});

test('validation engine allows create writes to select-only/computed fields via whitelist', async () => {
  const metadata = MetadataStorage.instance.getModelMetadata(PlatformValidationModel as any);

  const issues = await ValidationEngine.validate(
    {
      mode: 'create',
      model: PlatformValidationModel as any,
      metadata,
      values: { VirtualName: 'Virtual', ComputedName: 'Computed' },
      changedFields: new Set(['VirtualName', 'ComputedName']),
      repository: {} as any,
      requestContext: {},
    },
    {
      includeKernel: false,
      includeConstraints: false,
      platformCreateWriteWhitelist: ['VirtualName', 'ComputedName'],
    }
  );

  expect(issues).toEqual([]);
});

test('validation engine reports platform issue for readonly model writes', async () => {
  const metadata = MetadataStorage.instance.getModelMetadata(PlatformReadonlyModel as any);
  MetadataStorage.instance.setModelMetadata(
    PlatformReadonlyModel as any,
    {
      ...metadata,
      readonly: true,
    } as any
  );

  const readonlyMeta = MetadataStorage.instance.getModelMetadata(PlatformReadonlyModel as any);

  const createIssues = await ValidationEngine.validate(
    {
      mode: 'create',
      model: PlatformReadonlyModel as any,
      metadata: readonlyMeta,
      values: { Name: 'readonly-create' },
      changedFields: new Set(['Name']),
      repository: {} as any,
      requestContext: {},
    },
    {
      includeKernel: false,
      includeConstraints: false,
    }
  );

  expect(createIssues.length).toBe(1);
  expect(createIssues[0]?.scope).toBe('platform');
  expect(createIssues[0]?.code).toBe('platform_model_readonly');

  const updateIssues = await ValidationEngine.validate(
    {
      mode: 'update',
      model: PlatformReadonlyModel as any,
      metadata: readonlyMeta,
      values: { Name: 'readonly-update' },
      changedFields: new Set(['Name']),
      repository: {} as any,
      requestContext: {},
    },
    {
      includeKernel: false,
      includeConstraints: false,
    }
  );

  expect(updateIssues.length).toBe(1);
  expect(updateIssues[0]?.scope).toBe('platform');
  expect(updateIssues[0]?.code).toBe('platform_model_readonly');
});

test('validation engine does not bypass readonly rule even with create whitelist', async () => {
  const metadata = MetadataStorage.instance.getModelMetadata(PlatformReadonlyModel as any);
  MetadataStorage.instance.setModelMetadata(
    PlatformReadonlyModel as any,
    {
      ...metadata,
      readonly: true,
    } as any
  );

  const readonlyMeta = MetadataStorage.instance.getModelMetadata(PlatformReadonlyModel as any);
  const issues = await ValidationEngine.validate(
    {
      mode: 'create',
      model: PlatformReadonlyModel as any,
      metadata: readonlyMeta,
      values: { Name: 'readonly-create' },
      changedFields: new Set(['Name']),
      repository: {} as any,
      requestContext: {},
    },
    {
      includeKernel: false,
      includeConstraints: false,
      platformCreateWriteWhitelist: ['Name'],
    }
  );

  expect(issues.length).toBe(1);
  expect(issues[0]?.code).toBe('platform_model_readonly');
});

test('validation engine rejects unknown write fields when platform unknown-field guard is enabled', async () => {
  const metadata = MetadataStorage.instance.getModelMetadata(PlatformValidationModel as any);

  const issues = await ValidationEngine.validate(
    {
      mode: 'create',
      model: PlatformValidationModel as any,
      metadata,
      values: { UnknownField: 'x' },
      changedFields: new Set(['UnknownField']),
      repository: {} as any,
      requestContext: {},
    },
    {
      includeKernel: false,
      includeConstraints: false,
      platformRejectUnknownFields: true,
    }
  );

  expect(issues.length).toBe(1);
  expect(issues[0]?.scope).toBe('platform');
  expect(issues[0]?.code).toBe('platform_unknown_field');
  expect(issues[0]?.field).toBe('UnknownField');
});

test('validation engine reports whitelist hit callback for create write allowlist fields', async () => {
  const metadata = MetadataStorage.instance.getModelMetadata(PlatformValidationModel as any);
  const hits: string[][] = [];

  const issues = await ValidationEngine.validate(
    {
      mode: 'create',
      model: PlatformValidationModel as any,
      metadata,
      values: { VirtualName: 'Virtual', ComputedName: 'Computed' },
      changedFields: new Set(['VirtualName', 'ComputedName']),
      repository: {} as any,
      requestContext: {},
    },
    {
      includeKernel: false,
      includeConstraints: false,
      platformCreateWriteWhitelist: ['VirtualName', 'ComputedName'],
      onPlatformCreateWhitelistHit: fields => hits.push(fields),
    }
  );

  expect(issues).toEqual([]);
  expect(hits.length).toBe(1);
  expect(hits[0]).toEqual(['ComputedName', 'VirtualName']);
});

test('validation engine reports platform issue for cross-company many2one reference', async () => {
  const targetMetadata = MetadataStorage.instance.getModelMetadata(PlatformCompanyTargetModel as any);
  MetadataStorage.instance.setModelMetadata(
    PlatformCompanyTargetModel as any,
    {
      ...targetMetadata,
      companyScoped: true,
    } as any
  );

  RepositoryFactory.setRepository(
    PlatformCompanyTargetModel as any,
    {
      withDeleted() {
        return this;
      },
      async search() {
        return [{ Id: 'target_ref_1', CompanyId: 'company_b' }];
      },
    } as any
  );

  const metadata = MetadataStorage.instance.getModelMetadata(PlatformCompanySourceModel as any);
  const issues = await ValidationEngine.validate(
    {
      mode: 'update',
      model: PlatformCompanySourceModel as any,
      metadata,
      values: { TargetId: 'target_ref_1' },
      changedFields: new Set(['TargetId']),
      repository: {} as any,
      requestContext: {
        enabledCompanyIds: ['company_a'],
      },
    } as any,
    {
      includeKernel: false,
      includeConstraints: false,
    }
  );

  expect(issues).toEqual([
    {
      scope: 'platform',
      field: 'TargetId',
      code: 'platform_cross_company_reference_violation',
      message: 'reference "TargetId" points to company "company_b", which is outside ctx.enabledCompanyIds',
      severity: 'error',
    },
  ]);
});

test('validation engine reports platform issue for cross-company many2one-ref reference', async () => {
  RepositoryFactory.setRepository(
    PlatformCompanyTargetModel as any,
    {
      withDeleted() {
        return this;
      },
      async search() {
        return [{ Id: 'target_ref_2', CompanyId: 'company_b' }];
      },
    } as any
  );

  const metadata = MetadataStorage.instance.getModelMetadata(PlatformCompanyRefSourceModel as any);
  const issues = await ValidationEngine.validate(
    {
      mode: 'update',
      model: PlatformCompanyRefSourceModel as any,
      metadata,
      values: { TargetRefId: 'target_ref_2' },
      changedFields: new Set(['TargetRefId']),
      repository: {} as any,
      requestContext: {
        enabledCompanyIds: ['company_a'],
      },
    } as any,
    {
      includeKernel: false,
      includeConstraints: false,
    }
  );

  expect(issues).toEqual([
    {
      scope: 'platform',
      field: 'TargetRefId',
      code: 'platform_cross_company_reference_violation',
      message: 'reference "TargetRefId" points to company "company_b", which is outside ctx.enabledCompanyIds',
      severity: 'error',
    },
  ]);
});

test('validation engine reports not-visible issue when company-scoped target cannot be loaded', async () => {
  RepositoryFactory.setRepository(
    PlatformCompanyTargetModel as any,
    {
      withDeleted() {
        return this;
      },
      async search() {
        return [];
      },
    } as any
  );

  const metadata = MetadataStorage.instance.getModelMetadata(PlatformCompanyRefSourceModel as any);
  const issues = await ValidationEngine.validate(
    {
      mode: 'update',
      model: PlatformCompanyRefSourceModel as any,
      metadata,
      values: { TargetRefId: 'target_ref_missing' },
      changedFields: new Set(['TargetRefId']),
      repository: {} as any,
      requestContext: {
        enabledCompanyIds: ['company_a'],
      },
    } as any,
    {
      includeKernel: false,
      includeConstraints: false,
    }
  );

  expect(issues).toEqual([
    {
      scope: 'platform',
      field: 'TargetRefId',
      code: 'platform_cross_company_reference_not_visible',
      message: 'reference "TargetRefId" with id "target_ref_missing" is not visible in current company scope',
      severity: 'error',
    },
  ]);
});

test('validation engine preview mode bypasses cross-company reference checks', async () => {
  let repoSearchCalls = 0;
  RepositoryFactory.setRepository(
    PlatformCompanyTargetModel as any,
    {
      withDeleted() {
        return this;
      },
      async search() {
        repoSearchCalls += 1;
        return [{ Id: 'target_ref_preview', CompanyId: 'company_b' }];
      },
    } as any
  );

  const metadata = MetadataStorage.instance.getModelMetadata(PlatformCompanyRefSourceModel as any);
  const issues = await ValidationEngine.validate(
    {
      mode: 'preview',
      model: PlatformCompanyRefSourceModel as any,
      metadata,
      values: { TargetRefId: 'target_ref_preview' },
      changedFields: new Set(['TargetRefId']),
      repository: {} as any,
      requestContext: {
        enabledCompanyIds: ['company_a'],
      },
    } as any,
    {
      includeKernel: false,
      includeConstraints: false,
    }
  );

  expect(issues).toEqual([]);
  expect(repoSearchCalls).toBe(0);
});

test('validation engine resolves string targetModel for many2one-ref cross-company checks', async () => {
  const previousPool = (globalThis as any).pool;
  (globalThis as any).pool = {
    get(name: string) {
      if (name === 'test.PlatformCompanyTargetModel') {
        return PlatformCompanyTargetModel;
      }
      return undefined;
    },
  };

  try {
    RepositoryFactory.setRepository(
      PlatformCompanyTargetModel as any,
      {
        withDeleted() {
          return this;
        },
        async search() {
          return [{ Id: 'target_ref_3', CompanyId: 'company_b' }];
        },
      } as any
    );

    const metadata = MetadataStorage.instance.getModelMetadata(PlatformCompanyRefStringSourceModel as any);
    const issues = await ValidationEngine.validate(
      {
        mode: 'update',
        model: PlatformCompanyRefStringSourceModel as any,
        metadata,
        values: { TargetRefId: 'target_ref_3' },
        changedFields: new Set(['TargetRefId']),
        repository: {} as any,
        requestContext: {
          enabledCompanyIds: ['company_a'],
        },
      } as any,
      {
        includeKernel: false,
        includeConstraints: false,
      }
    );

    expect(issues).toEqual([
      {
        scope: 'platform',
        field: 'TargetRefId',
        code: 'platform_cross_company_reference_violation',
        message: 'reference "TargetRefId" points to company "company_b", which is outside ctx.enabledCompanyIds',
        severity: 'error',
      },
    ]);
  } finally {
    (globalThis as any).pool = previousPool;
  }
});

test('validation engine enforces enabledCompanyIds for base.Company many2one-ref targets', async () => {
  const metadata = MetadataStorage.instance.getModelMetadata(PlatformBaseCompanyRefSourceModel as any);

  const issues = await ValidationEngine.validate(
    {
      mode: 'update',
      model: PlatformBaseCompanyRefSourceModel as any,
      metadata,
      values: { CompanyId: 'company_b' },
      changedFields: new Set(['CompanyId']),
      repository: {} as any,
      requestContext: {
        enabledCompanyIds: ['company_a'],
      },
    } as any,
    {
      includeKernel: false,
      includeConstraints: false,
    }
  );

  expect(issues).toEqual([
    {
      scope: 'platform',
      field: 'CompanyId',
      code: 'platform_cross_company_reference_violation',
      message: 'reference "CompanyId" points to company "company_b", which is outside ctx.enabledCompanyIds',
      severity: 'error',
    },
  ]);
});

test('validation engine allows base.Company many2one-ref when enabledCompanyIds is empty', async () => {
  const metadata = MetadataStorage.instance.getModelMetadata(PlatformBaseCompanyRefSourceModel as any);

  const issues = await ValidationEngine.validate(
    {
      mode: 'update',
      model: PlatformBaseCompanyRefSourceModel as any,
      metadata,
      values: { CompanyId: 'company_b' },
      changedFields: new Set(['CompanyId']),
      repository: {} as any,
      requestContext: {
        enabledCompanyIds: [],
      },
    } as any,
    {
      includeKernel: false,
      includeConstraints: false,
    }
  );

  expect(issues).toEqual([]);
});

test('validation engine kernel: create requires notNull fields when no default', async () => {
  const metadata = MetadataStorage.instance.getModelMetadata(KernelValidationModel as any);

  const issues = await ValidationEngine.validate(
    {
      mode: 'create',
      model: KernelValidationModel as any,
      metadata,
      values: {},
      changedFields: new Set(),
      repository: {} as any,
      requestContext: {},
    },
    {
      includePlatform: false,
      includeConstraints: false,
    }
  );

  expect(issues.length).toBe(1);
  expect(issues[0]?.scope).toBe('kernel');
  expect(issues[0]?.code).toBe('kernel_required_missing');
  expect(String(issues[0]?.message || '').includes('RequiredName')).toBe(true);
});

test('validation engine kernel: update rejects null for notNull fields', async () => {
  const metadata = MetadataStorage.instance.getModelMetadata(KernelValidationModel as any);

  const issues = await ValidationEngine.validate(
    {
      mode: 'update',
      model: KernelValidationModel as any,
      metadata,
      values: { RequiredName: null },
      changedFields: new Set(['RequiredName']),
      repository: {} as any,
      requestContext: {},
    },
    {
      includePlatform: false,
      includeConstraints: false,
    }
  );

  expect(issues.length).toBe(1);
  expect(issues[0]?.scope).toBe('kernel');
  expect(issues[0]?.code).toBe('kernel_required_null');
  expect(String(issues[0]?.message || '').includes('RequiredName')).toBe(true);
});

test('validation engine kernel: decimal precision/format validation', async () => {
  const metadata = MetadataStorage.instance.getModelMetadata(KernelValidationModel as any);

  const invalidFormatIssues = await ValidationEngine.validate(
    {
      mode: 'update',
      model: KernelValidationModel as any,
      metadata,
      values: { Amount: 'abc' },
      changedFields: new Set(['Amount']),
      repository: {} as any,
      requestContext: {},
    },
    {
      includePlatform: false,
      includeConstraints: false,
    }
  );

  expect(invalidFormatIssues.length).toBe(1);
  expect(invalidFormatIssues[0]?.code).toBe('kernel_decimal_invalid');
  expect(String(invalidFormatIssues[0]?.message || '').includes('Amount')).toBe(true);

  const overflowPrecisionIssues = await ValidationEngine.validate(
    {
      mode: 'update',
      model: KernelValidationModel as any,
      metadata,
      values: { Amount: '123456' },
      changedFields: new Set(['Amount']),
      repository: {} as any,
      requestContext: {},
    },
    {
      includePlatform: false,
      includeConstraints: false,
    }
  );

  expect(overflowPrecisionIssues.length).toBe(1);
  expect(overflowPrecisionIssues[0]?.code).toBe('kernel_decimal_invalid');
  expect(String(overflowPrecisionIssues[0]?.message || '').includes('Amount')).toBe(true);
});

// ---------------------------------------------------------------------------
// Instance constraint (non-static, this-based) tests
// ---------------------------------------------------------------------------

const instanceCallLog: Array<Record<string, unknown>> = [];

class InstanceConstraintModel extends BaseModel {
  @Field({ type: 'varchar', size: 64 })
  Name?: string;

  @Field({ type: 'varchar', size: 32 })
  Status?: string;

  @Field({ type: 'int', notNull: true, default: () => 0 })
  Rank?: number;

  static resetLog() {
    instanceCallLog.length = 0;
  }

  // Instance constraint (NOT static): reads and writes via `this`.
  validateName(): void {
    instanceCallLog.push({
      method: 'validateName',
      thisName: this.Name,
      thisStatus: this.Status,
    });
    if (this.Name) {
      this.Name = (this.Name as string).trim().toUpperCase();
    }
  }

  // Instance constraint: throws on blocked status.
  validateStatus(): void {
    instanceCallLog.push({
      method: 'validateStatus',
      thisStatus: this.Status,
    });
    if (this.Status === 'blocked') {
      throw new Error('status blocked via instance');
    }
  }

  // Instance constraint: async, mutates multiple fields.
  async validateRank(): Promise<void> {
    instanceCallLog.push({
      method: 'validateRank',
      thisRank: this.Rank,
    });
    if (this.Rank != null && this.Rank < 0) {
      this.Rank = 0;
    }
  }
}

// Manually register instance constraints via the decorator — isStatic is detected
// automatically from the property descriptor.
Constraint<InstanceConstraintModel>('Name', { priority: 1, alwaysOnCreate: true })(InstanceConstraintModel.prototype, 'validateName', undefined as any);
Constraint<InstanceConstraintModel>('Status', { priority: 2 })(InstanceConstraintModel.prototype, 'validateStatus', undefined as any);
Constraint<InstanceConstraintModel>('Rank', { priority: 3, alwaysOnCreate: true })(InstanceConstraintModel.prototype, 'validateRank', undefined as any);

test('instance constraint method executes with this readable', async () => {
  InstanceConstraintModel.resetLog();
  const metadata = MetadataStorage.instance.getModelMetadata(InstanceConstraintModel as any);

  await ValidationEngine.validateOrThrow({
    mode: 'update',
    model: InstanceConstraintModel as any,
    metadata,
    current: { Id: '1', Name: 'old', Status: 'draft' },
    values: { Name: '  new name  ' },
    changedFields: new Set(['Name']),
    repository: {} as any,
    requestContext: {},
  });

  expect(instanceCallLog.length).toBeGreaterThanOrEqual(1);
  const nameEntry = instanceCallLog.find(e => e.method === 'validateName');
  expect(nameEntry).toBeDefined();
  expect(nameEntry?.thisName).toBe('  new name  ');
});

test('instance constraint writeback updates ctx.values', async () => {
  InstanceConstraintModel.resetLog();
  const metadata = MetadataStorage.instance.getModelMetadata(InstanceConstraintModel as any);

  const values: ObjectRecord = { Name: '  trimmed up  ' };
  await ValidationEngine.validateOrThrow({
    mode: 'update',
    model: InstanceConstraintModel as any,
    metadata,
    current: { Id: '1', Name: 'old' },
    values,
    changedFields: new Set(['Name']),
    repository: {} as any,
    requestContext: {},
  });

  // The instance constraint should have normalized Name to uppercase trimmed form.
  expect(values.Name).toBe('TRIMMED UP');
});

test('instance constraint error is reported as constraint_execution_failed', async () => {
  InstanceConstraintModel.resetLog();
  const metadata = MetadataStorage.instance.getModelMetadata(InstanceConstraintModel as any);

  let error: unknown;
  try {
    await ValidationEngine.validateOrThrow({
      mode: 'update',
      model: InstanceConstraintModel as any,
      metadata,
      current: { Id: '1', Name: 'old', Status: 'draft' },
      values: { Status: 'blocked' },
      changedFields: new Set(['Status']),
      repository: {} as any,
      requestContext: {},
    });
  } catch (err) {
    error = err;
  }

  expect(error instanceof ValidationPipelineError).toBe(true);
  const pipelineError = error as ValidationPipelineError;
  expect(pipelineError.issues.some(i => i.code === 'constraint_execution_failed')).toBe(true);
});

test('instance constraint async method executes and mutates', async () => {
  InstanceConstraintModel.resetLog();
  const metadata = MetadataStorage.instance.getModelMetadata(InstanceConstraintModel as any);

  const values: ObjectRecord = { Rank: -5 };
  await ValidationEngine.validateOrThrow({
    mode: 'create',
    model: InstanceConstraintModel as any,
    metadata,
    values,
    changedFields: new Set(['Rank']),
    repository: {} as any,
    requestContext: {},
  });

  // validateRank should have clamped negative rank to 0.
  expect(values.Rank).toBe(0);
  const rankEntry = instanceCallLog.find(e => e.method === 'validateRank');
  expect(rankEntry).toBeDefined();
});

test('static and instance constraints run together in priority order', async () => {
  // Mixed model: one static + one instance constraint.
  const mixedCallOrder: string[] = [];

  class MixedConstraintModel extends BaseModel {
    @Field({ type: 'varchar', size: 64 })
    Name?: string;

    static staticCheck(_self: MixedConstraintModel) {
      mixedCallOrder.push('static');
    }

    instanceCheck(): void {
      mixedCallOrder.push('instance');
    }
  }

  Constraint<MixedConstraintModel>('Name', { priority: 1 })(MixedConstraintModel, 'staticCheck', undefined as any);
  Constraint<MixedConstraintModel>('Name', { priority: 2 })(MixedConstraintModel.prototype, 'instanceCheck', undefined as any);

  const metadata = MetadataStorage.instance.getModelMetadata(MixedConstraintModel as any);

  await ValidationEngine.validate({
    mode: 'update',
    model: MixedConstraintModel as any,
    metadata,
    current: { Id: '1', Name: 'old' },
    values: { Name: 'next' },
    changedFields: new Set(['Name']),
    repository: {} as any,
    requestContext: {},
  });

  expect(mixedCallOrder).toEqual(['static', 'instance']);
});

test('instance constraint write to unknown field is silently skipped', async () => {
  class UnknownFieldModel extends BaseModel {
    @Field({ type: 'varchar', size: 64 })
    Name?: string;

    writeUnknown(): void {
      // Attempt to write a field not declared on the model metadata.
      (this as any).NonExistentField = 'should-not-leak';
      this.Name = 'valid';
    }
  }

  Constraint<UnknownFieldModel>('Name', { priority: 1 })(UnknownFieldModel.prototype, 'writeUnknown', undefined as any);

  const metadata = MetadataStorage.instance.getModelMetadata(UnknownFieldModel as any);
  const values: ObjectRecord = { Name: 'original' };

  await ValidationEngine.validate({
    mode: 'update',
    model: UnknownFieldModel as any,
    metadata,
    current: { Id: '1', Name: 'orig' },
    values,
    changedFields: new Set(['Name']),
    repository: {} as any,
    requestContext: {},
  });

  expect(values.Name).toBe('valid');
  expect('NonExistentField' in values).toBe(false);
});

test('post-constraint mutation triggers kernel re-validation for new fields', async () => {
  class MutatingConstraintModel extends BaseModel {
    @Field({ type: 'varchar', size: 64, notNull: true })
    RequiredField?: string;

    @Field({ type: 'varchar', size: 64 })
    OtherField?: string;

    normalizeRequired(): void {
      // This constraint mutates a field that was NOT in the original changedFields.
      this.RequiredField = (this.RequiredField || '').trim();
    }
  }

  Constraint<MutatingConstraintModel>('OtherField', { priority: 1, alwaysOnCreate: true })(
    MutatingConstraintModel.prototype,
    'normalizeRequired',
    undefined as any
  );

  const metadata = MetadataStorage.instance.getModelMetadata(MutatingConstraintModel as any);

  // Create with only OtherField; RequiredField is notNull but missing.
  // The instance constraint reads RequiredField (gets undefined initially)
  // and writes back an empty string — which kernel should reject as null.
  const issues = await ValidationEngine.validate({
    mode: 'create',
    model: MutatingConstraintModel as any,
    metadata,
    values: { OtherField: 'trigger' },
    changedFields: new Set(['OtherField']),
    repository: {} as any,
    requestContext: {},
  });

  // The post-constraint kernel re-validation should catch the empty RequiredField.
  const kernelIssues = issues.filter(i => i.scope === 'kernel');
  expect(kernelIssues.length).toBeGreaterThanOrEqual(1);
  expect(kernelIssues.some(i => i.code === 'kernel_required_missing' || i.code === 'kernel_required_null')).toBe(true);
});

test('validation engine kernel: relation shape validation for many2one/many2one-ref', async () => {
  const metadata = MetadataStorage.instance.getModelMetadata(KernelValidationModel as any);

  const invalidMany2OneIssues = await ValidationEngine.validate(
    {
      mode: 'update',
      model: KernelValidationModel as any,
      metadata,
      values: { TargetId: { foo: 'bar' } as any },
      changedFields: new Set(['TargetId']),
      repository: {} as any,
      requestContext: {},
    },
    {
      includePlatform: false,
      includeConstraints: false,
    }
  );

  expect(invalidMany2OneIssues.length).toBe(1);
  expect(invalidMany2OneIssues[0]?.code).toBe('kernel_relation_shape_invalid');
  expect(String(invalidMany2OneIssues[0]?.message || '').includes('TargetId')).toBe(true);

  const invalidMany2OneRefIssues = await ValidationEngine.validate(
    {
      mode: 'update',
      model: KernelValidationModel as any,
      metadata,
      values: { CompanyId: { foo: 'bar' } as any },
      changedFields: new Set(['CompanyId']),
      repository: {} as any,
      requestContext: {},
    },
    {
      includePlatform: false,
      includeConstraints: false,
    }
  );

  expect(invalidMany2OneRefIssues.length).toBe(1);
  expect(invalidMany2OneRefIssues[0]?.code).toBe('kernel_relation_shape_invalid');
  expect(String(invalidMany2OneRefIssues[0]?.message || '').includes('CompanyId')).toBe(true);
});

test('validation engine private helpers normalize reference ids and enabled company ids', () => {
  const resolveReferenceId = (ValidationEngine as any).resolveReferenceId as (raw: unknown) => string | undefined;
  const extractEnabledCompanyIds = (ValidationEngine as any).extractEnabledCompanyIds as (ctx: unknown) => string[];

  expect(resolveReferenceId(null)).toBe(undefined);
  expect(resolveReferenceId('  A  ')).toBe('A');
  expect(resolveReferenceId(12)).toBe('12');
  expect(resolveReferenceId(BigInt(99))).toBe('99');
  expect(resolveReferenceId({ id: '  ' })).toBe(undefined);
  expect(resolveReferenceId({ Id: 'ID-1' })).toBe('ID-1');

  expect(extractEnabledCompanyIds({ enabledCompanyIds: [' c1 ', 'c2', 'c1'] })).toEqual(['c1', 'c2']);
  expect(extractEnabledCompanyIds({ EnabledCompanyIds: ['X'] })).toEqual(['X']);
  expect(extractEnabledCompanyIds({ activeCompanyId: 'A1' })).toEqual(['A1']);
  expect(extractEnabledCompanyIds({ ActiveCompanyId: 'A2' })).toEqual(['A2']);
  expect(extractEnabledCompanyIds(undefined)).toEqual([]);
});

test('validation engine private helpers resolve target ctor and base-company target markers', () => {
  const resolveReferenceTargetCtor = (ValidationEngine as any).resolveReferenceTargetCtor as (meta: any) => any;
  const isBaseCompanyTarget = (ValidationEngine as any).isBaseCompanyTarget as (meta: any, targetMeta?: any) => boolean;

  class DummyTarget extends BaseModel {}

  const previousPool = (globalThis as any).pool;
  try {
    (globalThis as any).pool = {
      get(name: string) {
        if (name === 'test.DummyTarget') return DummyTarget;
        return undefined;
      },
    };

    expect(resolveReferenceTargetCtor({ relation: { targetModel: () => DummyTarget } })).toBe(DummyTarget);
    expect(resolveReferenceTargetCtor({ relation: { targetModel: 'test.DummyTarget' } })).toBe(DummyTarget);

    const throwingResolver = (() => {
      throw new Error('resolver failed');
    }) as any;
    throwingResolver.prototype = DummyTarget.prototype;
    expect(resolveReferenceTargetCtor({ relation: { targetModel: throwingResolver } })).toBe(throwingResolver);

    expect(resolveReferenceTargetCtor({ relation: { targetModel: 'test.UnknownTarget' } })).toBe(undefined);

    expect(isBaseCompanyTarget({ relation: { targetModel: 'base.Company' } })).toBe(true);
    expect(isBaseCompanyTarget({}, { fullModelName: 'base.Company' })).toBe(true);
    expect(isBaseCompanyTarget({}, { application: 'base', modelName: 'Company' })).toBe(true);
    expect(isBaseCompanyTarget({ relation: { targetModel: 'test.Other' } }, { application: 'test', modelName: 'Other' })).toBe(false);
  } finally {
    (globalThis as any).pool = previousPool;
  }
});

test('validation engine platform uses values keys when changedFields is empty and skips internal fields', async () => {
  const metadata = MetadataStorage.instance.getModelMetadata(PlatformValidationModel as any);
  const issues = await ValidationEngine.validate(
    {
      mode: 'create',
      model: PlatformValidationModel as any,
      metadata,
      values: {
        __internal: 'x',
        Name: 'ok',
      },
      changedFields: new Set(),
      repository: {} as any,
      requestContext: {},
    },
    {
      includeKernel: false,
      includeConstraints: false,
      platformRejectUnknownFields: true,
      platformCreateWriteWhitelist: [null as any, ' Name ', ''],
    }
  );

  expect(issues).toEqual([]);
});

test('validation engine platform short-circuits when reference id or target ctor is missing', async () => {
  class MissingTargetRefModel extends BaseModel {
    @Field({ type: 'ManyToOneRef', relation: { targetModel: 'test.UnknownTarget' } })
    TargetRefId?: string;
  }

  const metadata = MetadataStorage.instance.getModelMetadata(MissingTargetRefModel as any);
  const issues = await ValidationEngine.validate(
    {
      mode: 'update',
      model: MissingTargetRefModel as any,
      metadata,
      values: {
        TargetRefId: { Id: null },
      },
      changedFields: new Set(['TargetRefId']),
      repository: {} as any,
      requestContext: {
        enabledCompanyIds: ['company_a'],
      },
    } as any,
    {
      includeKernel: false,
      includeConstraints: false,
    }
  );

  expect(issues).toEqual([]);
});

test('validation engine platform skips company check for non-company-scoped target and enforces base.Company via targetMeta', async () => {
  class NonCompanyScopedTarget extends BaseModel {
    @Field({ type: 'varchar', size: 20 })
    Name?: string;
  }

  class NonCompanyScopedSource extends BaseModel {
    @Field({ type: 'ManyToOne', relation: { targetModel: () => NonCompanyScopedTarget } })
    TargetId?: NonCompanyScopedTarget;
  }

  class BaseCompanyViaTargetMeta extends BaseModel {
    @Field({ type: 'ManyToOne', relation: { targetModel: () => PlatformCompanyTargetModel } })
    CompanyId?: PlatformCompanyTargetModel;
  }

  const targetMeta = MetadataStorage.instance.getModelMetadata(PlatformCompanyTargetModel as any);
  MetadataStorage.instance.setModelMetadata(
    PlatformCompanyTargetModel as any,
    {
      ...targetMeta,
      fullModelName: 'base.Company',
      companyScoped: false,
    } as any
  );

  const noCompanyMeta = MetadataStorage.instance.getModelMetadata(NonCompanyScopedSource as any);
  const noCompanyIssues = await ValidationEngine.validate(
    {
      mode: 'update',
      model: NonCompanyScopedSource as any,
      metadata: noCompanyMeta,
      values: { TargetId: 'target_no_scope' },
      changedFields: new Set(['TargetId']),
      repository: {} as any,
      requestContext: { enabledCompanyIds: ['company_a'] },
    } as any,
    {
      includeKernel: false,
      includeConstraints: false,
    }
  );

  const baseCompanyMeta = MetadataStorage.instance.getModelMetadata(BaseCompanyViaTargetMeta as any);
  const baseCompanyIssues = await ValidationEngine.validate(
    {
      mode: 'update',
      model: BaseCompanyViaTargetMeta as any,
      metadata: baseCompanyMeta,
      values: { CompanyId: 'company_b' },
      changedFields: new Set(['CompanyId']),
      repository: {} as any,
      requestContext: { enabledCompanyIds: ['company_a'] },
    } as any,
    {
      includeKernel: false,
      includeConstraints: false,
    }
  );

  expect(noCompanyIssues).toEqual([]);
  expect(baseCompanyIssues.length).toBe(1);
  expect(baseCompanyIssues[0]?.code).toBe('platform_cross_company_reference_violation');
});

test('validation engine private helpers cover empty resolver and prototype constraint method resolution', async () => {
  const resolveReferenceTargetCtor = (ValidationEngine as any).resolveReferenceTargetCtor as (meta: any) => any;
  const shouldRunConstraint = (ValidationEngine as any).shouldRunConstraint as (handler: any, ctx: any) => boolean;
  const buildConstraintSelf = (ValidationEngine as any).buildConstraintSelf as (ctx: any) => any;
  const resolveConstraintMethod = (ValidationEngine as any).resolveConstraintMethod as (model: any, handler: any) => any;

  class InstanceConstraintModel extends BaseModel {
    Name?: string;
    checkInstance(this: InstanceConstraintModel) {
      return this.Name;
    }
  }

  expect(resolveReferenceTargetCtor({ relation: { targetModel: undefined } })).toBe(undefined);

  expect(shouldRunConstraint({ fields: [], alwaysOnCreate: false, preview: false }, { mode: 'create', changedFields: new Set<string>() })).toBe(true);
  expect(shouldRunConstraint({ fields: ['Name'], alwaysOnCreate: true, preview: false }, { mode: 'create', changedFields: new Set<string>() })).toBe(true);

  const self = buildConstraintSelf({
    model: InstanceConstraintModel,
    values: { Name: 'next' },
  });
  expect(self.Name).toBe('next');

  const instanceMethod = resolveConstraintMethod(InstanceConstraintModel as any, {
    method: 'checkInstance',
    isStatic: false,
  });
  expect(typeof instanceMethod).toBe('function');
  const result = await instanceMethod(Object.assign(Object.create(InstanceConstraintModel.prototype), { Name: 'ok' }), {} as any);
  expect(result).toBe(undefined);
});

test('validation engine wraps non-error thrown values from constraint methods', async () => {
  class NonErrorConstraintModel extends BaseModel {
    Name?: string;
    static raiseNonError() {
      throw 'non-error-throw';
    }
  }

  Constraint<NonErrorConstraintModel>('Name', { priority: 1 })(NonErrorConstraintModel, 'raiseNonError', undefined as any);

  const metadata = MetadataStorage.instance.getModelMetadata(NonErrorConstraintModel as any);
  const issues = await ValidationEngine.validate(
    {
      mode: 'update',
      model: NonErrorConstraintModel as any,
      metadata,
      values: { Name: 'x' },
      changedFields: new Set(['Name']),
      repository: {} as any,
      requestContext: {},
    },
    {
      includeKernel: false,
      includePlatform: false,
      includeConstraints: true,
    }
  );

  expect(issues.length).toBe(1);
  expect(issues[0]?.code).toBe('constraint_execution_failed');
  expect(issues[0]?.message).toBe('non-error-throw');
});

test('validation engine kernel fallback maps non-kernel errors and non-error throws', async () => {
  const issuesFromError = await ValidationEngine.validateKernelRules({
    mode: 'update',
    metadata: {
      fields: {
        forEach() {
          throw new Error('kernel-fallback-error');
        },
      },
    } as any,
    values: {},
  } as any);

  expect(issuesFromError.length).toBe(1);
  expect(issuesFromError[0]?.code).toBe('kernel_validation_failed');
  expect(String(issuesFromError[0]?.message || '').includes('kernel-fallback-error')).toBe(true);

  const issuesFromString = await ValidationEngine.validateKernelRules({
    mode: 'update',
    metadata: {
      fields: {
        forEach() {
          throw 'kernel-fallback-string';
        },
      },
    } as any,
    values: {},
  } as any);

  expect(issuesFromString.length).toBe(1);
  expect(issuesFromString[0]?.code).toBe('kernel_validation_failed');
  expect(issuesFromString[0]?.message).toBe('kernel-fallback-string');
});

test('validation engine platform skips unresolved target ctor and empty target company id rows', async () => {
  class MissingCtorRefModel extends BaseModel {
    @Field({ type: 'ManyToOneRef', relation: { targetModel: 'test.MissingCtorModel' } })
    TargetRefId?: string;
  }

  const missingCtorMeta = MetadataStorage.instance.getModelMetadata(MissingCtorRefModel as any);
  const missingCtorIssues = await ValidationEngine.validate(
    {
      mode: 'update',
      model: MissingCtorRefModel as any,
      metadata: missingCtorMeta,
      values: { TargetRefId: 'missing-ref-1' },
      changedFields: new Set(['TargetRefId']),
      repository: {} as any,
      requestContext: {
        enabledCompanyIds: ['company_a'],
      },
    } as any,
    {
      includeKernel: false,
      includeConstraints: false,
    }
  );

  expect(missingCtorIssues).toEqual([]);

  const originalTargetMeta = MetadataStorage.instance.getModelMetadata(PlatformCompanyTargetModel as any);
  MetadataStorage.instance.setModelMetadata(
    PlatformCompanyTargetModel as any,
    {
      ...originalTargetMeta,
      fullModelName: 'test.PlatformCompanyTargetModel',
      application: 'test',
      modelName: 'PlatformCompanyTargetModel',
      companyScoped: true,
    } as any
  );

  try {
    RepositoryFactory.setRepository(
      PlatformCompanyTargetModel as any,
      {
        withDeleted() {
          return this;
        },
        async search() {
          return [{ Id: 'target_ref_empty_company' }];
        },
      } as any
    );

    const refMeta = MetadataStorage.instance.getModelMetadata(PlatformCompanyRefSourceModel as any);
    const emptyCompanyIssues = await ValidationEngine.validate(
      {
        mode: 'update',
        model: PlatformCompanyRefSourceModel as any,
        metadata: refMeta,
        values: { TargetRefId: 'target_ref_empty_company' },
        changedFields: new Set(['TargetRefId']),
        repository: {} as any,
        requestContext: {
          enabledCompanyIds: ['company_a'],
        },
      } as any,
      {
        includeKernel: false,
        includeConstraints: false,
      }
    );

    expect(emptyCompanyIssues).toEqual([]);
  } finally {
    MetadataStorage.instance.setModelMetadata(PlatformCompanyTargetModel as any, originalTargetMeta as any);
  }
});

test('validation engine helper branches cover primitive fallback ids and resolver without prototype', () => {
  const resolveReferenceId = (ValidationEngine as any).resolveReferenceId as (raw: unknown) => string | undefined;
  const resolveReferenceTargetCtor = (ValidationEngine as any).resolveReferenceTargetCtor as (meta: any) => any;
  const extractEnabledCompanyIds = (ValidationEngine as any).extractEnabledCompanyIds as (ctx: unknown) => string[];

  const resolverWithoutPrototype = (() => undefined) as any;

  expect(resolveReferenceId(true as any)).toBe(undefined);
  expect(resolveReferenceTargetCtor({ relation: { targetModel: resolverWithoutPrototype } })).toBe(undefined);
  expect(extractEnabledCompanyIds({ enabledCompanyIds: [null, ' company_a ', undefined] } as any)).toEqual(['company_a']);
});

test('validation engine constraint execution sorts same-priority handlers by method name', async () => {
  class ConstraintTieModel extends BaseModel {
    Name?: string;

    static order: string[] = [];

    static resetOrder() {
      this.order = [];
    }

    static zeta() {
      this.order.push('zeta');
    }

    static alpha() {
      this.order.push('alpha');
    }
  }

  Constraint<ConstraintTieModel>('Name', { priority: 9 })(ConstraintTieModel, 'zeta', undefined as any);
  Constraint<ConstraintTieModel>('Name', { priority: 9 })(ConstraintTieModel, 'alpha', undefined as any);

  const metadata = MetadataStorage.instance.getModelMetadata(ConstraintTieModel as any);
  ConstraintTieModel.resetOrder();

  await ValidationEngine.validate(
    {
      mode: 'update',
      model: ConstraintTieModel as any,
      metadata,
      values: { Name: 'next' },
      changedFields: new Set(['Name']),
      repository: {} as any,
      requestContext: {},
    },
    {
      includeKernel: false,
      includePlatform: false,
      includeConstraints: true,
    }
  );

  expect(ConstraintTieModel.order).toEqual(['alpha', 'zeta']);
});

// ---------------------------------------------------------------------------
// Inheritance override: engine-level execution source verification
// ---------------------------------------------------------------------------

const inheritLog: string[] = [];

class InheritEngineParent extends BaseModel {
  @Field({ type: 'varchar', size: 64 })
  Name?: string;

  // Parent instance constraint.
  validateName(): void {
    inheritLog.push('parent-validateName');
  }
}
Constraint<InheritEngineParent>('Name', { priority: 10 })(InheritEngineParent.prototype, 'validateName', undefined as any);

class InheritEngineChild extends InheritEngineParent {
  // Child re-decorates same method name with different priority.
  validateName(): void {
    inheritLog.push('child-validateName');
  }
}
Constraint<InheritEngineChild>('Name', { priority: 5 })(InheritEngineChild.prototype, 'validateName', undefined as any);

class InheritEngineGrandchild extends InheritEngineChild {
  // Grandchild inherits without re-decoration → child's handler runs.
}

test('constraint override: child implementation is actually executed (not parent)', async () => {
  inheritLog.length = 0;
  const metadata = MetadataStorage.instance.getModelMetadata(InheritEngineChild as any);

  await ValidationEngine.validate({
    mode: 'update',
    model: InheritEngineChild as any,
    metadata,
    current: { Id: '1', Name: 'old' },
    values: { Name: 'next' },
    changedFields: new Set(['Name']),
    repository: {} as any,
    requestContext: {},
  });

  expect(inheritLog).toEqual(['child-validateName']);
});

test('constraint reuse: grandchild without re-decoration inherits child handler', async () => {
  inheritLog.length = 0;
  const metadata = MetadataStorage.instance.getModelMetadata(InheritEngineGrandchild as any);

  await ValidationEngine.validate({
    mode: 'update',
    model: InheritEngineGrandchild as any,
    metadata,
    current: { Id: '1', Name: 'old' },
    values: { Name: 'next' },
    changedFields: new Set(['Name']),
    repository: {} as any,
    requestContext: {},
  });

  // Grandchild reuses child's handler (closest ancestor with same method name).
  expect(inheritLog).toEqual(['child-validateName']);
});
