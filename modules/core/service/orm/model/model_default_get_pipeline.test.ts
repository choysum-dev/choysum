// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import BaseModel from './model';
import { Field, Model } from '../decorator';
import { MetadataStorage } from '../metadata/storage';
import { withContext } from '../../runtime/context/scope';
import { runDefaultGetPipeline } from './model_default_get_pipeline';
import { __setLookupFieldDefaultModelForTest } from './field_default_lookup';

@Model('PipelineCoreDemo', { application: 'core' })
class PipelineCoreDemo extends BaseModel {
  @Field({ type: 'varchar', size: 64, default: 'from-column' })
  Name!: string;

  @Field({ type: 'varchar', size: 64 })
  Code!: string;
}

@Model('PipelinePartnerDemo', { application: 'partner' })
class PipelinePartnerDemo extends BaseModel {
  @Field({ type: 'varchar', size: 64, default: 'from-column' })
  Name!: string;

  @Field({ type: 'varchar', size: 64 })
  Code!: string;
}

function restoreLookup(application: string) {
  __setLookupFieldDefaultModelForTest(application, undefined);
}

test('runDefaultGetPipeline keeps payload keys and explicit null over lower layers', async () => {
  const lookupCalls: any[] = [];
  __setLookupFieldDefaultModelForTest('partner', {
    async GetEffective(modelName, fieldNames) {
      lookupCalls.push({ modelName, fieldNames });
      return { Name: 'from-field-default', Code: 'from-field-default' };
    },
  });

  try {
    const out = await withContext({ default_Name: 'from-context', default_Code: 'from-context' }, async () =>
      runDefaultGetPipeline(PipelinePartnerDemo as any, { Name: 'payload', Code: null } as any)
    );

    expect(out).toEqual({ Name: 'payload', Code: null });
    expect(lookupCalls.length).toBe(1);
  } finally {
    restoreLookup('partner');
  }
});

test('runDefaultGetPipeline fills from context default_<Field> via withContext', async () => {
  __setLookupFieldDefaultModelForTest('partner', {
    async GetEffective() {
      return { Name: 'from-field-default' };
    },
  });

  try {
    const out = await withContext({ default_Name: 'from-context' }, async () =>
      runDefaultGetPipeline(PipelinePartnerDemo as any, {} as any)
    );

    expect((out as any).Name).toBe('from-context');
  } finally {
    restoreLookup('partner');
  }
});

test('runDefaultGetPipeline skips FieldDefault lookup for core models', async () => {
  let lookedUp = false;
  __setLookupFieldDefaultModelForTest('core', {
    async GetEffective() {
      lookedUp = true;
      return { Name: 'from-field-default' };
    },
  });

  const originalWarn = console.warn;
  const warnings: string[] = [];
  console.warn = (...args: any[]) => {
    warnings.push(args.map(x => String(x)).join(' '));
  };

  try {
    const out = await withContext({ default_Code: 'ctx-code' }, async () => runDefaultGetPipeline(PipelineCoreDemo as any, {} as any));

    expect(lookedUp).toBe(false);
    expect(warnings.some(msg => msg.includes('FIELD_DEFAULT_MODEL_MISSING'))).toBe(false);
    expect((out as any).Code).toBe('ctx-code');
    expect((out as any).Name).toBe('from-column');
  } finally {
    console.warn = originalWarn;
    restoreLookup('core');
  }
});

test('runDefaultGetPipeline merges stub GetEffective for non-core models', async () => {
  const lookupCalls: any[] = [];
  __setLookupFieldDefaultModelForTest('partner', {
    async GetEffective(modelName, fieldNames) {
      lookupCalls.push({ modelName, fieldNames: [...fieldNames].sort() });
      return { Code: 'from-effective' };
    },
  });

  try {
    const out = await runDefaultGetPipeline(PipelinePartnerDemo as any, {} as any);

    expect(lookupCalls.length).toBe(1);
    expect(lookupCalls[0]?.modelName).toBe('PipelinePartnerDemo');
    expect(lookupCalls[0]?.fieldNames).toContain('Name');
    expect(lookupCalls[0]?.fieldNames).toContain('Code');
    expect((out as any).Code).toBe('from-effective');
    // FieldDefault did not set Name; column default fills it.
    expect((out as any).Name).toBe('from-column');
  } finally {
    restoreLookup('partner');
  }
});

test('runDefaultGetPipeline warns when non-core FieldDefault ctor is missing', async () => {
  restoreLookup('partner');

  const originalWarn = console.warn;
  const warnings: string[] = [];
  console.warn = (...args: any[]) => {
    warnings.push(args.map(x => String(x)).join(' '));
  };

  try {
    const out = await runDefaultGetPipeline(PipelinePartnerDemo as any, {} as any);

    expect(warnings.some(msg => msg.includes('FIELD_DEFAULT_MODEL_MISSING') && msg.includes('partner'))).toBe(true);
    expect((out as any).Name).toBe('from-column');
  } finally {
    console.warn = originalWarn;
  }
});

test('runDefaultGetPipeline warns and continues when GetEffective throws', async () => {
  __setLookupFieldDefaultModelForTest('partner', {
    async GetEffective() {
      throw new Error('effective-query-failed');
    },
  });

  const originalWarn = console.warn;
  const warnings: string[] = [];
  console.warn = (...args: any[]) => {
    warnings.push(args.map(x => String(x)).join(' '));
  };

  try {
    const out = await runDefaultGetPipeline(PipelinePartnerDemo as any, {} as any);

    expect(
      warnings.some(msg => msg.includes('FIELD_DEFAULT_GET_EFFECTIVE_FAILED') && msg.includes('effective-query-failed'))
    ).toBe(true);
    expect((out as any).Name).toBe('from-column');
    expect((out as any).Code).toBeUndefined();
  } finally {
    console.warn = originalWarn;
    restoreLookup('partner');
  }
});

test('BaseModel.DefaultGet delegates to runDefaultGetPipeline', async () => {
  __setLookupFieldDefaultModelForTest('partner', {
    async GetEffective() {
      return {};
    },
  });

  try {
    const out = await withContext({ default_Code: 'via-hook' }, async () => PipelinePartnerDemo.DefaultGet({} as any));
    expect((out as any).Code).toBe('via-hook');
    expect((out as any).Name).toBe('from-column');
  } finally {
    restoreLookup('partner');
  }
});

test('lookup override clear restores pool miss behavior', async () => {
  __setLookupFieldDefaultModelForTest('partner', {
    async GetEffective() {
      return { Code: 'injected' };
    },
  });
  restoreLookup('partner');

  const originalWarn = console.warn;
  const warnings: string[] = [];
  console.warn = (...args: any[]) => {
    warnings.push(args.map(x => String(x)).join(' '));
  };

  try {
    const out = await runDefaultGetPipeline(PipelinePartnerDemo as any, {} as any);
    expect((out as any).Code).toBeUndefined();
    expect(warnings.some(msg => msg.includes('FIELD_DEFAULT_MODEL_MISSING'))).toBe(true);
  } finally {
    console.warn = originalWarn;
  }
});

// Ensure MetadataStorage still has application after decorator registration (sanity for harness reuse).
test('pipeline demo models expose application metadata', () => {
  expect(MetadataStorage.instance.getModelMetadata(PipelineCoreDemo as any).application).toBe('core');
  expect(MetadataStorage.instance.getModelMetadata(PipelinePartnerDemo as any).application).toBe('partner');
});
