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

test('runDefaultGetPipeline treats whitespace application as non-core missing ctor', async () => {
  const meta = MetadataStorage.instance.getModelMetadata(PipelinePartnerDemo as any);
  const prevApp = meta.application;
  (meta as any).application = '   ';

  const originalWarn = console.warn;
  const warnings: string[] = [];
  console.warn = (...args: any[]) => {
    warnings.push(args.map(x => String(x)).join(' '));
  };

  try {
    const out = await runDefaultGetPipeline(PipelinePartnerDemo as any, {} as any);
    expect(warnings.some(msg => msg.includes('FIELD_DEFAULT_MODEL_MISSING'))).toBe(true);
    expect((out as any).Name).toBe('from-column');
  } finally {
    (meta as any).application = prevApp;
    console.warn = originalWarn;
  }
});

test('runDefaultGetPipeline handles undefined application and empty modelName', async () => {
  const meta = MetadataStorage.instance.getModelMetadata(PipelinePartnerDemo as any);
  const prevApp = meta.application;
  const prevModel = meta.modelName;
  (meta as any).application = undefined;
  (meta as any).modelName = '';

  __setLookupFieldDefaultModelForTest('partner', undefined);

  const originalWarn = console.warn;
  const warnings: string[] = [];
  console.warn = (...args: any[]) => {
    warnings.push(args.map(x => String(x)).join(' '));
  };

  try {
    // lookup uses meta.application (undefined) → miss; warn uses trimmed empty application fallback.
    const out = await runDefaultGetPipeline(PipelinePartnerDemo as any, {} as any);
    expect(warnings.some(msg => msg.includes('FIELD_DEFAULT_MODEL_MISSING'))).toBe(true);
    expect((out as any).Name).toBe('from-column');
  } finally {
    (meta as any).application = prevApp;
    (meta as any).modelName = prevModel;
    console.warn = originalWarn;
  }
});

test('runDefaultGetPipeline ignores non-object GetEffective results', async () => {
  const meta = MetadataStorage.instance.getModelMetadata(PipelinePartnerDemo as any);
  const prevModel = meta.modelName;
  (meta as any).modelName = undefined;

  const calls: any[] = [];
  __setLookupFieldDefaultModelForTest('partner', {
    async GetEffective(modelName, fieldNames) {
      calls.push({ modelName, fieldNames });
      // Truthy non-object so `typeof effRaw === 'object'` is false.
      return 42 as any;
    },
  });

  try {
    const out = await runDefaultGetPipeline(PipelinePartnerDemo as any, {} as any);
    expect(calls[0]?.modelName).toBe('');
    expect((out as any).Name).toBe('from-column');
    expect((out as any).Code).toBeUndefined();
  } finally {
    (meta as any).modelName = prevModel;
    restoreLookup('partner');
  }
});

test('runDefaultGetPipeline stringifies non-Error GetEffective failures', async () => {
  __setLookupFieldDefaultModelForTest('partner', {
    async GetEffective() {
      throw 'bare-string-failure';
    },
  });

  const originalWarn = console.warn;
  const warnings: string[] = [];
  console.warn = (...args: any[]) => {
    warnings.push(args.map(x => String(x)).join(' '));
  };

  try {
    const out = await runDefaultGetPipeline(PipelinePartnerDemo as any, {} as any);
    expect(warnings.some(msg => msg.includes('FIELD_DEFAULT_GET_EFFECTIVE_FAILED') && msg.includes('bare-string-failure'))).toBe(
      true
    );
    expect((out as any).Name).toBe('from-column');
  } finally {
    console.warn = originalWarn;
    restoreLookup('partner');
  }
});

@Model('PipelinePropsPartner', { application: 'partner' })
class PipelinePropsPartner extends BaseModel {
  @Field({ type: 'varchar', size: 64, default: 'from-column' })
  Name!: string;

  @Field({ type: 'properties' })
  PartnerProperties!: Record<string, unknown>;
}

test('runDefaultGetPipeline fills properties item defaults (PP3)', async () => {
  const { __setLookupPropertyDefinitionModelForTest, __clearLookupPropertyDefinitionModelForTest } = await import(
    './properties_lookup'
  );
  __setLookupFieldDefaultModelForTest('partner', {
    async GetEffective() {
      return {};
    },
  });
  __setLookupPropertyDefinitionModelForTest('partner', {
    Search: async () => [
      {
        TargetModel: 'PipelinePropsPartner',
        PropertiesField: 'PartnerProperties',
        ContainerId: null,
        Definition: [
          { name: 'tax_id', type: 'char', default: 'T-DEFAULT' },
          { name: 'vip', type: 'boolean', default: false },
          { name: 'note', type: 'text' },
        ],
      },
    ],
  });
  try {
    const out = await runDefaultGetPipeline(PipelinePropsPartner as any, {} as any);
    expect((out as any).PartnerProperties).toEqual({ tax_id: 'T-DEFAULT', vip: false });

    const withExisting = await runDefaultGetPipeline(PipelinePropsPartner as any, {
      PartnerProperties: { tax_id: 'KEEP', note: 'n' },
    } as any);
    expect((withExisting as any).PartnerProperties).toEqual({ tax_id: 'KEEP', note: 'n', vip: false });

    const emptySchema = await (async () => {
      __setLookupPropertyDefinitionModelForTest('partner', { Search: async () => [] });
      return runDefaultGetPipeline(PipelinePropsPartner as any, {} as any);
    })();
    expect((emptySchema as any).PartnerProperties).toBeUndefined();
  } finally {
    __clearLookupPropertyDefinitionModelForTest();
    restoreLookup('partner');
  }
});

test('runDefaultGetPipeline warns when properties defaults resolution fails', async () => {
  const { __setLookupPropertyDefinitionModelForTest, __clearLookupPropertyDefinitionModelForTest } = await import(
    './properties_lookup'
  );
  __setLookupFieldDefaultModelForTest('partner', {
    async GetEffective() {
      return {};
    },
  });
  __setLookupPropertyDefinitionModelForTest('partner', {
    Search: async () => {
      throw new Error('props-schema-down');
    },
  });
  const originalWarn = console.warn;
  const warnings: string[] = [];
  console.warn = (...args: any[]) => {
    warnings.push(args.map(x => String(x)).join(' '));
  };
  try {
    const out = await runDefaultGetPipeline(PipelinePropsPartner as any, {} as any);
    expect(warnings.some(msg => msg.includes('PROPERTIES_DEFAULT_GET_FAILED') && msg.includes('props-schema-down'))).toBe(
      true
    );
    expect((out as any).Name).toBe('from-column');

    warnings.length = 0;
    __setLookupPropertyDefinitionModelForTest('partner', {
      Search: async () => {
        throw 'bare-props-fail';
      },
    });
    await runDefaultGetPipeline(PipelinePropsPartner as any, {} as any);
    expect(warnings.some(msg => msg.includes('PROPERTIES_DEFAULT_GET_FAILED') && msg.includes('bare-props-fail'))).toBe(true);

    // Schema items without defaults still materialize an empty map when field is undefined.
    __setLookupPropertyDefinitionModelForTest('partner', {
      Search: async () => [
        {
          TargetModel: 'PipelinePropsPartner',
          PropertiesField: 'PartnerProperties',
          ContainerId: null,
          Definition: [{ name: 'only', type: 'char' }],
        },
      ],
    });
    const emptyDefaults = await runDefaultGetPipeline(PipelinePropsPartner as any, {} as any);
    expect((emptyDefaults as any).PartnerProperties).toEqual({});
  } finally {
    console.warn = originalWarn;
    __clearLookupPropertyDefinitionModelForTest();
    restoreLookup('partner');
  }
});
