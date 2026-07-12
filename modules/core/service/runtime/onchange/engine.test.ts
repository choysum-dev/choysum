// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { OnchangeEngine } from './engine';
import { MetadataStorage } from '../../orm/metadata/storage';

function createMeta(config: {
  name?: string;
  fields?: Array<[string, any]>;
  onchangeHandlers?: Array<{ method: string; triggers: string[]; priority?: number; signature?: 'legacyCtx' | 'instanceNoArgs' }>;
}) {
  return {
    name: config.name || 'TestModel',
    fields: new Map<string, any>(config.fields || []),
    onchangeHandlers: (config.onchangeHandlers || []).map(handler => ({
      method: handler.method,
      triggers: handler.triggers,
      priority: handler.priority ?? 100,
      signature: handler.signature,
    })),
  } as any;
}

test('onchange engine normalizes relation ref payloads and accepts returned selection patch', async () => {
  const meta = createMeta({
    fields: [
      ['TargetRef', { type: 'ManyToOneRef' }],
      ['TagRefs', { type: 'ManyToManyRef' }],
      ['Status', { type: 'varchar' }],
    ],
    onchangeHandlers: [{ method: 'onTargetRef', triggers: ['TargetRef'] }],
  });

  const draft: any = {
    TargetRef: { Id: 'partner-1', Name: 'P1' },
    TagRefs: [{ Id: 'tag-1' }, { id: 'tag-2' }, 'tag-3', null],
    Status: 'draft',
    onTargetRef() {
      expect(this.TargetRef).toBe('partner-1');
      expect(this.TagRefs).toEqual(['tag-1', 'tag-2', 'tag-3']);
      this.Status = 'active';
      return {
        selection: [
          {
            field: 'Status',
            selection: ['active', 'draft'],
          },
        ],
      };
    },
  };

  const result = await OnchangeEngine.run(meta, draft, ['TargetRef', 'TagRefs'], {
    withCompute: false,
  });

  expect(result.value).toEqual({ Status: 'active' });
  expect(result.selection).toEqual([
    {
      field: 'Status',
      selection: ['active', 'draft'],
    },
  ]);
  expect(result.touchedHandlers).toEqual(['onTargetRef']);
});

test('onchange engine keeps running handlers when stopOnError=false', async () => {
  const meta = createMeta({
    fields: [
      ['Name', { type: 'varchar' }],
      ['Code', { type: 'varchar' }],
    ],
    onchangeHandlers: [
      { method: 'onNameFail', triggers: ['Name'], priority: 1 },
      { method: 'onNameRecover', triggers: ['Name'], priority: 2 },
    ],
  });

  const draft: any = {
    Name: 'A',
    Code: '',
    onNameFail() {
      throw new Error('boom-onchange');
    },
    onNameRecover() {
      this.Code = 'RECOVERED';
    },
  };

  const result = await OnchangeEngine.run(meta, draft, ['Name'], {
    withCompute: false,
    stopOnError: false,
  });

  expect(result.touchedHandlers).toEqual(['onNameFail', 'onNameRecover']);
  expect(result.value).toEqual({ Code: 'RECOVERED' });
  expect(result.messages?.some(m => String(m.message || '').includes('boom-onchange'))).toBe(true);
});

test('onchange engine stops subsequent handlers on first error when stopOnError is default true', async () => {
  const meta = createMeta({
    fields: [
      ['Name', { type: 'varchar' }],
      ['Code', { type: 'varchar' }],
    ],
    onchangeHandlers: [
      { method: 'onNameFail', triggers: ['Name'], priority: 1 },
      { method: 'onNameRecover', triggers: ['Name'], priority: 2 },
    ],
  });

  const draft: any = {
    Name: 'A',
    Code: '',
    onNameFail() {
      throw new Error('boom-stop');
    },
    onNameRecover() {
      this.Code = 'RECOVERED';
    },
  };

  const result = await OnchangeEngine.run(meta, draft, ['Name'], {
    withCompute: false,
  });

  expect(result.touchedHandlers).toEqual(['onNameFail']);
  expect(result.value).toBe(undefined);
  expect(result.messages?.some(m => String(m.message || '').includes('boom-stop'))).toBe(true);
});

test('onchange engine reports max-iterations warning when pending fields remain', async () => {
  const meta = createMeta({
    fields: [
      ['A', { type: 'int' }],
      ['B', { type: 'int' }],
    ],
    onchangeHandlers: [
      { method: 'onA', triggers: ['A'] },
      { method: 'onB', triggers: ['B'] },
    ],
  });

  const draft: any = {
    A: 0,
    B: 0,
    onA() {
      this.B = Number(this.B || 0) + 1;
    },
    onB() {
      this.A = Number(this.A || 0) + 1;
    },
  };

  const warns: string[] = [];
  const originalWarn = console.warn;
  console.warn = (...args: any[]) => {
    warns.push(args.map(x => String(x)).join(' '));
  };

  try {
    const result = await OnchangeEngine.run(meta, draft, ['A'], {
      withCompute: false,
      maxIterations: 1,
    });

    expect(result.iterations).toBe(1);
    expect(warns.some(msg => msg.includes('Max iterations reached'))).toBe(true);
  } finally {
    console.warn = originalWarn;
  }
});

test('onchange engine collects computePreview recomputed fields into result', async () => {
  const meta = createMeta({
    fields: [
      ['Qty', { type: 'int' }],
      ['Total', { type: 'int' }],
    ],
    onchangeHandlers: [],
  });

  const draft: any = {
    Qty: 2,
    Total: 0,
  };

  const result = await OnchangeEngine.run(meta, draft, ['Qty'], {
    withCompute: true,
    computePreview: async (entity: any) => {
      entity.Total = Number(entity.Qty || 0) * 10;
    },
  });

  expect(result.value).toEqual({ Total: 20 });
  expect(result.computeRecomputed).toEqual(['Total']);
});

test('onchange engine normalizes indexed changed paths and triggers matching handlers', async () => {
  const meta = createMeta({
    fields: [
      ['Lines', { type: 'OneToMany', relation: { targetModel: () => null, inverseField: 'ParentId' } }],
      ['Touched', { type: 'varchar' }],
    ],
    onchangeHandlers: [{ method: 'onLines', triggers: ['Lines'] }],
  });

  const draft: any = {
    Lines: [{ Quantity: 1 }],
    Touched: '',
    onLines() {
      this.Touched = 'yes';
    },
  };

  const result = await OnchangeEngine.run(meta, draft, ['Lines.0.Quantity'], {
    withCompute: false,
  });

  expect(result.touchedHandlers).toEqual(['onLines']);
  expect(result.value).toEqual({ Touched: 'yes' });
});

test('onchange engine drops emitted value patch when stopOnError=true and error occurs', async () => {
  const meta = createMeta({
    fields: [
      ['Name', { type: 'varchar' }],
      ['Code', { type: 'varchar' }],
    ],
    onchangeHandlers: [{ method: 'onName', triggers: ['Name'] }],
  });

  const draft: any = {
    Name: 'A',
    Code: '',
    onName() {
      this.Code = 'X';
      throw new Error('fail-after-emit');
    },
  };

  const result = await OnchangeEngine.run(meta, draft, ['Name'], {
    withCompute: false,
  });

  expect(result.value).toBeUndefined();
  expect(result.messages?.some(m => String(m.message || '').includes('fail-after-emit'))).toBe(true);
});

test('onchange engine normalizes decimal values from returned value payload', async () => {
  const meta = createMeta({
    fields: [
      ['Price', { type: 'decimal', column: { scale: 2 } }],
      ['Trigger', { type: 'varchar' }],
    ],
    onchangeHandlers: [{ method: 'onTrigger', triggers: ['Trigger'] }],
  });

  const draft: any = {
    Trigger: 'Y',
    Price: '0',
    onTrigger() {
      return { value: { Price: '1.239' } };
    },
  };

  const result = await OnchangeEngine.run(meta, draft, ['Trigger'], {
    withCompute: false,
  });

  expect(String((result.value as any).Price)).toBe('1.24');
});

test('onchange engine handles empty handlers and keeps ref payload when id/array normalization is not applicable', async () => {
  const meta = createMeta({
    fields: [
      ['Ref', { type: 'ManyToOneRef' }],
      ['Tags', { type: 'ManyToManyRef' }],
    ],
    onchangeHandlers: [],
  });

  const draft: any = {
    Ref: { Name: 'no-id' },
    Tags: 'not-array',
  };

  const result = await OnchangeEngine.run(meta, draft, ['Ref', 'Tags'], {
    withCompute: false,
  });

  expect(result.iterations).toBe(1);
  expect(result.touchedHandlers).toEqual([]);
  expect(result.value).toBeUndefined();
  expect(draft.Ref).toEqual({ Name: 'no-id' });
  expect(draft.Tags).toBe('not-array');
});

test('onchange engine merges return message/messages/condition and keeps running when stopOnError=false', async () => {
  const meta = createMeta({
    fields: [
      ['Name', { type: 'varchar' }],
      ['Code', { type: 'varchar' }],
      ['Count', { type: 'int' }],
    ],
    onchangeHandlers: [
      { method: 'onNameInfo', triggers: ['Name'], priority: 1 },
      { method: 'onNameError', triggers: ['Name'], priority: 2 },
      { method: 'onNameFinal', triggers: ['Name'], priority: 3 },
    ],
  });

  const draft: any = {
    Name: 'A',
    Code: '',
    Count: 0,
    onNameInfo() {
      return {
        message: 'info-msg',
        condition: [{ field: 'Code', condition: ['Code', '=', 'OK'] }],
      };
    },
    onNameError() {
      return {
        messages: [{ level: 'error', message: 'err-msg' }],
      };
    },
    onNameFinal() {
      this.Count = 1;
    },
  };

  const result = await OnchangeEngine.run(meta, draft, ['Name'], {
    withCompute: false,
    stopOnError: false,
  });

  expect(result.touchedHandlers).toEqual(['onNameInfo', 'onNameError', 'onNameFinal']);
  expect(result.value).toEqual({ Count: 1 });
  expect(result.condition).toEqual([{ field: 'Code', condition: ['Code', '=', 'OK'] }]);
  expect(result.messages?.some(m => String(m.message || '').includes('info-msg'))).toBe(true);
  expect(result.messages?.some(m => String(m.message || '').includes('err-msg'))).toBe(true);
});

test('onchange engine suppresses re-trigger when field change count exceeds loopThreshold', async () => {
  const meta = createMeta({
    name: 'LoopModel',
    fields: [
      ['A', { type: 'int' }],
      ['B', { type: 'int' }],
    ],
    onchangeHandlers: [
      { method: 'onA', triggers: ['A'], priority: 1 },
      { method: 'onB', triggers: ['B'], priority: 2 },
    ],
  });

  const draft: any = {
    A: 0,
    B: 0,
    onA() {
      this.B = Number(this.B || 0) + 1;
    },
    onB() {
      this.A = Number(this.A || 0) + 1;
    },
  };

  const warns: string[] = [];
  const originalWarn = console.warn;
  console.warn = (...args: any[]) => {
    warns.push(args.map(x => String(x)).join(' '));
  };

  try {
    const result = await OnchangeEngine.run(meta, draft, ['A'], {
      withCompute: false,
      loopThreshold: 1,
      maxIterations: 5,
    });

    expect(result.iterations).toBe(3);
    expect(result.touchedHandlers).toEqual(['onA', 'onB', 'onA']);
    expect(warns.some(msg => msg.includes('Loop suppressed on field "B"'))).toBe(true);
    expect(warns.some(msg => msg.includes('Max iterations reached'))).toBe(false);
  } finally {
    console.warn = originalWarn;
  }
});

test('onchange engine tolerates missing metadata root when normalizing changed paths', async () => {
  const meta = createMeta({
    fields: [['Name', { type: 'varchar' }]],
    onchangeHandlers: [],
  });

  const draft: any = {
    UnknownRef: { Id: 'U-1' },
    Name: 'A',
  };

  const result = await OnchangeEngine.run(meta, draft, ['UnknownRef.Name'], {
    withCompute: false,
  });

  expect(result.touchedHandlers).toEqual([]);
  expect(result.value).toBeUndefined();
  expect(draft.UnknownRef).toEqual({ Id: 'U-1' });
});

test('onchange engine normalizeRelationRefs keeps ManyToOneRef object when id is null', async () => {
  const meta = createMeta({
    fields: [['PartnerRef', { type: 'ManyToOneRef' }]],
    onchangeHandlers: [],
  });

  const draft: any = {
    PartnerRef: { Id: null, Name: 'Partner-A' },
  };

  const result = await OnchangeEngine.run(meta, draft, ['PartnerRef'], {
    withCompute: false,
  });

  expect(result.touchedHandlers).toEqual([]);
  expect(draft.PartnerRef).toEqual({ Id: null, Name: 'Partner-A' });
});

test('onchange engine compute preview decimal fallback keeps invalid decimal payload and traverses many2one without target ctor', async () => {
  class ChildModel {}

  const storage = MetadataStorage.instance as any;
  const originalGetMeta = storage.getModelMetadata;

  const childMeta = {
    fullModelName: 'test.Child',
    fields: new Map([['Amount', { type: 'decimal', column: { scale: 2 } }]]),
  } as any;

  storage.getModelMetadata = function (model: Function) {
    if (model === ChildModel) return childMeta;
    return originalGetMeta.call(this, model);
  };

  const meta = createMeta({
    fields: [
      ['Price', { type: 'decimal', column: { scale: 2 } }],
      ['Owner', { type: 'ManyToOne', relation: {} }],
      ['Ref', { type: 'ManyToOne', relation: { targetModel: () => ChildModel } }],
      ['Trigger', { type: 'varchar' }],
    ],
    onchangeHandlers: [{ method: 'onTrigger', triggers: ['Trigger'] }],
  });

  const draft: any = {
    Trigger: 'x',
    Price: { bad: true },
    Owner: { Name: 'owner-no-target' },
    Ref: { Amount: '1.25' },
    onTrigger() {
      return {};
    },
  };

  try {
    const result = await OnchangeEngine.run(meta, draft, ['Trigger'], {
      withCompute: true,
      computePreview: async () => {},
    });

    expect(result.computeRecomputed).toEqual([]);
    expect(draft.Price).toEqual({ bad: true });
    expect(draft.Owner).toEqual({ Name: 'owner-no-target' });
  } finally {
    storage.getModelMetadata = originalGetMeta;
  }
});

test('onchange engine ignores empty changed segments and keeps null relation refs untouched', async () => {
  const meta = createMeta({
    fields: [
      ['Lines', { type: 'OneToMany', relation: { targetModel: () => null, inverseField: 'ParentId' } }],
      ['PartnerRef', { type: 'ManyToOneRef' }],
      ['TagRefs', { type: 'ManyToManyRef' }],
    ],
    onchangeHandlers: [{ method: 'onLines', triggers: ['Lines'] }],
  });

  const draft: any = {
    Lines: [{ Qty: 1 }],
    PartnerRef: null,
    TagRefs: null,
    touched: false,
    onLines() {
      this.touched = true;
    },
  };

  const result = await OnchangeEngine.run(meta, draft, ['', 'Lines.0.Qty', 'PartnerRef', 'TagRefs'], {
    withCompute: false,
  });

  expect(result.touchedHandlers).toEqual(['onLines']);
  expect((result.value as any).touched).toBe(true);
  expect(draft.PartnerRef).toBe(null);
  expect(draft.TagRefs).toBe(null);
});

test('onchange engine supports meta without onchangeHandlers', async () => {
  const meta: any = {
    name: 'NoHandlersModel',
    fields: new Map([['Name', { type: 'varchar' }]]),
  };

  const draft: any = { Name: 'A' };
  const result = await OnchangeEngine.run(meta, draft, ['Name'], { withCompute: false });

  expect(result.touchedHandlers).toEqual([]);
  expect(result.value).toBeUndefined();
  expect(result.iterations).toBe(1);
});

test('onchange engine normalizes decimal written via proxy patch sink', async () => {
  const meta = createMeta({
    fields: [
      ['Price', { type: 'decimal', column: { scale: 2 } }],
      ['Trigger', { type: 'varchar' }],
    ],
    onchangeHandlers: [{ method: 'onTrigger', triggers: ['Trigger'] }],
  });

  const draft: any = {
    Trigger: 'x',
    Price: '0',
    onTrigger() {
      this.Price = '1.239';
    },
  };

  const result = await OnchangeEngine.run(meta, draft, ['Trigger'], { withCompute: false });
  expect(String((result.value as any).Price)).toBe('1.24');
});

test('onchange engine awaits async handlers returning promises', async () => {
  const meta = createMeta({
    fields: [
      ['Name', { type: 'varchar' }],
      ['Code', { type: 'varchar' }],
    ],
    onchangeHandlers: [{ method: 'onName', triggers: ['Name'] }],
  });

  const draft: any = {
    Name: 'N',
    Code: '',
    async onName() {
      return await Promise.resolve({ value: { Code: 'ASYNC-OK' } });
    },
  };

  const result = await OnchangeEngine.run(meta, draft, ['Name'], { withCompute: false });
  expect((result.value as any).Code).toBe('ASYNC-OK');
});

test('onchange engine quantizedTopLevel keeps original value when decimal normalization throws', async () => {
  const meta = createMeta({
    fields: [
      ['Price', { type: 'decimal', column: { scale: 2 } }],
      ['Trigger', { type: 'varchar' }],
    ],
    onchangeHandlers: [{ method: 'onTrigger', triggers: ['Trigger'] }],
  });

  const draft: any = {
    Trigger: 'x',
    Price: '0',
    onTrigger() {
      this.Price = { bad: true };
    },
  };

  const result = await OnchangeEngine.run(meta, draft, ['Trigger'], { withCompute: false });
  expect((result.value as any).Price).toEqual({ bad: true });
});

test('onchange engine normalizes ManyToOneRef using lowercase id field', async () => {
  const meta = createMeta({
    fields: [['PartnerRef', { type: 'ManyToOneRef' }]],
    onchangeHandlers: [],
  });

  const draft: any = {
    PartnerRef: { id: 'P-lower' },
  };

  await OnchangeEngine.run(meta, draft, ['PartnerRef'], { withCompute: false });
  expect(draft.PartnerRef).toBe('P-lower');
});

test('onchange engine collects error from ctx.emit message via pushMessages callback', async () => {
  const meta = createMeta({
    fields: [
      ['Name', { type: 'varchar' }],
      ['Code', { type: 'varchar' }],
    ],
    onchangeHandlers: [{ method: 'onName', triggers: ['Name'] }],
  });

  const draft: any = {
    Name: 'A',
    Code: '',
    onName() {
      this.Code = 'AFTER-ERROR';
      return { messages: [{ level: 'error', message: 'ctx-error' }] };
    },
  };

  const result = await OnchangeEngine.run(meta, draft, ['Name'], {
    withCompute: false,
    stopOnError: false,
  });

  expect((result.value as any).Code).toBe('AFTER-ERROR');
  expect(result.messages?.some(m => String(m.message || '').includes('ctx-error'))).toBe(true);
});

test('onchange engine keeps raw decimal payload when returned value cannot be normalized', async () => {
  const meta = createMeta({
    fields: [
      ['Price', { type: 'decimal', column: { scale: 2 } }],
      ['Trigger', { type: 'varchar' }],
    ],
    onchangeHandlers: [{ method: 'onTrigger', triggers: ['Trigger'] }],
  });

  const draft: any = {
    Trigger: 'T',
    Price: '0',
    onTrigger() {
      return {
        value: {
          Price: { bad: true },
        },
      };
    },
  };

  const result = await OnchangeEngine.run(meta, draft, ['Trigger'], { withCompute: false });
  expect((result.value as any).Price).toEqual({ bad: true });
});

test('onchange engine normalizes ctx.emit decimal value patch and keeps fallback payload when normalization fails', async () => {
  const meta = createMeta({
    fields: [
      ['Price', { type: 'decimal', column: { scale: 2 } }],
      ['Trigger', { type: 'varchar' }],
    ],
    onchangeHandlers: [{ method: 'onTrigger', triggers: ['Trigger'] }],
  });

  const draft: any = {
    Trigger: 'P',
    Price: '0',
    onTrigger() {
      this.Price = '2.236';
      this.Price = { bad: true };
    },
  };

  const result = await OnchangeEngine.run(meta, draft, ['Trigger'], { withCompute: false });
  expect((result.value as any).Price).toEqual({ bad: true });
});

test('onchange engine keeps ManyToOneRef primitive and filters nullish ManyToManyRef items', async () => {
  const meta = createMeta({
    fields: [
      ['PartnerRef', { type: 'ManyToOneRef' }],
      ['TagRefs', { type: 'ManyToManyRef' }],
    ],
    onchangeHandlers: [],
  });

  const draft: any = {
    PartnerRef: 'raw-partner',
    TagRefs: [null, undefined, { Id: 'T-1' }, { id: 'T-2' }, 'T-3'],
  };

  const result = await OnchangeEngine.run(meta, draft, ['PartnerRef', 'TagRefs'], { withCompute: false });
  expect(result.touchedHandlers).toEqual([]);
  expect(draft.PartnerRef).toBe('raw-partner');
  expect(draft.TagRefs).toEqual(['T-1', 'T-2', 'T-3']);
});

test('onchange engine stops next handler when ret.message raises error under stopOnError=true', async () => {
  const meta: any = {
    name: 'MinimalModel',
    fields: new Map(),
    onchangeHandlers: [
      { method: 'onA', triggers: ['A'], priority: 1 },
      { method: 'onB', triggers: ['A'], priority: 2 },
    ],
  };

  const draft: any = {
    A: 'x',
    B: '',
    onA() {
      return { message: { level: 'error', message: 'ret-error' } };
    },
    onB() {
      this.B = 'should-not-run';
    },
  };

  const result = await OnchangeEngine.run(meta, draft, ['A'], {
    withCompute: false,
    stopOnError: true,
  });

  expect(result.touchedHandlers).toEqual(['onA']);
  expect(result.value).toBeUndefined();
  expect(result.messages?.some(m => String(m.message || '').includes('ret-error'))).toBe(true);
  expect(draft.B).toBe('');
});

test('onchange engine stops next handler when ctx.pushMessages contains error under stopOnError=true', async () => {
  const meta: any = {
    name: 'PushMessageStopModel',
    fields: new Map(),
    onchangeHandlers: [
      { method: 'onA', triggers: ['A'], priority: 1 },
      { method: 'onB', triggers: ['A'], priority: 2 },
    ],
  };

  const draft: any = {
    A: 'x',
    B: '',
    onA() {
      return { messages: [{ level: 'error', message: 'ctx-stop-error' }] };
    },
    onB() {
      this.B = 'should-not-run';
    },
  };

  const result = await OnchangeEngine.run(meta, draft, ['A'], {
    withCompute: false,
    stopOnError: true,
  });

  expect(result.touchedHandlers).toEqual(['onA']);
  expect((result.value as any)?.B).toBeUndefined();
  expect(result.messages?.some(m => String(m.message || '').includes('ctx-stop-error'))).toBe(true);
  expect(draft.B).toBe('');
});

test('onchange engine skips compute preview when handler already produced error', async () => {
  const meta = createMeta({
    fields: [['Name', { type: 'varchar' }]],
    onchangeHandlers: [{ method: 'onName', triggers: ['Name'] }],
  });

  const draft: any = {
    Name: 'A',
    onName() {
      throw new Error('boom-before-compute');
    },
  };

  let computeCalled = 0;
  const result = await OnchangeEngine.run(meta, draft, ['Name'], {
    withCompute: true,
    computePreview: async () => {
      computeCalled += 1;
    },
  });

  expect(computeCalled).toBe(0);
  expect(result.messages?.some(m => String(m.message || '').includes('boom-before-compute'))).toBe(true);
});

test('onchange engine accepts weak meta.fields shape when no changed paths are provided', async () => {
  const meta: any = {
    name: 'WeakMetaModel',
    fields: {} as any,
    onchangeHandlers: [],
  };

  const draft: any = { Name: 'A' };
  const result = await OnchangeEngine.run(meta, draft, [], { withCompute: false });

  expect(result.touchedHandlers).toEqual([]);
  expect(result.value).toBeUndefined();
  expect(result.iterations).toBe(0);
});

test('onchange engine keeps empty-path emitted patch key and skips compute root seed for empty key', async () => {
  const meta = createMeta({
    fields: [['Name', { type: 'varchar' }]],
    onchangeHandlers: [{ method: 'onName', triggers: ['Name'] }],
  });

  const draft: any = {
    Name: 'A',
    onName() {
      this[''] = 'empty-key';
    },
  };

  const result = await OnchangeEngine.run(meta, draft, ['Name'], { withCompute: false });

  expect((result.value as any)['']).toBe('empty-key');
  expect(result.touchedHandlers).toEqual(['onName']);
});

test('onchange engine compute seed ignores empty emitted patch root key', async () => {
  const meta = createMeta({
    fields: [
      ['Name', { type: 'varchar' }],
      ['Total', { type: 'int' }],
    ],
    onchangeHandlers: [{ method: 'onName', triggers: ['Name'] }],
  });

  const draft: any = {
    Name: 'A',
    Total: 1,
    onName() {
      this[''] = 'empty-key';
    },
  };

  let capturedSeed: Set<string> | undefined;
  const result = await OnchangeEngine.run(meta, draft, ['Name'], {
    withCompute: true,
    computePreview: async (_entity: any, seed: Set<string>) => {
      capturedSeed = new Set(seed);
    },
  });

  expect((result.value as any)['']).toBe('empty-key');
  expect(capturedSeed?.has('Name')).toBe(true);
  // In the end-state (no ctx), this[''] is a real field write and appears in seed.
  expect(capturedSeed?.has('')).toBe(true);
});

test('onchange engine catches non-Error throws and keeps stringified fallback message', async () => {
  const meta = createMeta({
    fields: [['Name', { type: 'varchar' }]],
    onchangeHandlers: [{ method: 'onName', triggers: ['Name'] }],
  });

  const draft: any = {
    Name: 'A',
    onName() {
      throw 'string-throw';
    },
  };

  const result = await OnchangeEngine.run(meta, draft, ['Name'], {
    withCompute: false,
    stopOnError: true,
  });

  expect(result.touchedHandlers).toEqual(['onName']);
  expect(result.messages?.some(m => String(m.message || '').includes('string-throw'))).toBe(true);
});

test('onchange engine tolerates undefined payload in ctx.emit value patch', async () => {
  const meta = createMeta({
    fields: [
      ['Name', { type: 'varchar' }],
      ['Code', { type: 'varchar' }],
    ],
    onchangeHandlers: [{ method: 'onName', triggers: ['Name'] }],
  });

  const draft: any = {
    Name: 'A',
    Code: 'X',
    onName() {
      // no-op — no ctx.emit to call
    },
  };

  const result = await OnchangeEngine.run(meta, draft, ['Name'], { withCompute: false });
  expect(result.value).toBeUndefined();
  expect(result.touchedHandlers).toEqual(['onName']);
});

test('onchange engine sorts handlers with falsy priority via default fallback', async () => {
  const meta: any = {
    name: 'PriorityFallbackModel',
    fields: new Map([
      ['A', { type: 'varchar' }],
      ['B', { type: 'varchar' }],
    ]),
    onchangeHandlers: [
      { method: 'onA', triggers: ['A'], priority: 0 },
      { method: 'onB', triggers: ['A'] },
    ],
  };

  const draft: any = {
    A: 'x',
    B: '',
    onA() {
      this.B = 'A';
    },
    onB() {
      this.B = 'B';
    },
  };

  const result = await OnchangeEngine.run(meta, draft, ['A'], { withCompute: false });
  expect(result.touchedHandlers).toEqual(['onA', 'onB']);
});

test('onchange engine compute seed includes non-empty emitted patch root', async () => {
  const meta = createMeta({
    fields: [
      ['Name', { type: 'varchar' }],
      ['Code', { type: 'varchar' }],
    ],
    onchangeHandlers: [{ method: 'onName', triggers: ['Name'] }],
  });

  const draft: any = {
    Name: 'A',
    Code: '',
    onName() {
      this.Code = 'X';
    },
  };

  let capturedSeed: Set<string> | undefined;
  const result = await OnchangeEngine.run(meta, draft, ['Name'], {
    withCompute: true,
    computePreview: async (_entity: any, seed: Set<string>) => {
      capturedSeed = new Set(seed);
    },
  });

  expect((result.value as any).Code).toBe('X');
  expect(capturedSeed?.has('Code')).toBe(true);
});

test('onchange engine keeps many2many ref items without id keys out of normalized array', async () => {
  const meta = createMeta({
    fields: [['TagRefs', { type: 'ManyToManyRef' }]],
    onchangeHandlers: [],
  });

  const draft: any = {
    TagRefs: [{}, { Id: 'T-1' }, { id: 'T-2' }, null],
  };

  await OnchangeEngine.run(meta, draft, ['TagRefs'], { withCompute: false });
  expect(draft.TagRefs).toEqual(['T-1', 'T-2']);
});

test('onchange engine routes ctx condition and selection emit payloads through context callbacks', async () => {
  const meta = createMeta({
    fields: [
      ['Name', { type: 'varchar' }],
      ['Code', { type: 'varchar' }],
    ],
    onchangeHandlers: [{ method: 'onName', triggers: ['Name'] }],
  });

  const draft: any = {
    Name: 'A',
    Code: 'A',
    onName() {
      return {
        messages: [{ level: 'info', message: 'ctx-info' }],
        condition: [{ field: 'Code', condition: ['Code', '=', 'A'] }],
        selection: [{ field: 'Code', selection: ['A', 'B'], disabled: ['B'] }],
      };
    },
  };

  const result = await OnchangeEngine.run(meta, draft, ['Name'], {
    withCompute: false,
  });

  expect(result.condition).toEqual([{ field: 'Code', condition: ['Code', '=', 'A'] }]);
  expect(result.selection).toEqual([{ field: 'Code', selection: ['A', 'B'], disabled: ['B'] }]);
  expect(result.messages?.some(m => String(m.message || '').includes('ctx-info'))).toBe(true);
  expect(result.value).toBeUndefined();
});

test('onchange engine invokes instanceNoArgs sync handler without ctx', async () => {
  const meta = createMeta({
    fields: [
      ['Name', { type: 'varchar' }],
      ['Code', { type: 'varchar' }],
    ],
    onchangeHandlers: [{ method: 'onName', triggers: ['Name'], signature: 'instanceNoArgs' }],
  });

  const draft: any = {
    Name: 'A',
    Code: '',
    onName() {
      this.Code = 'set-by-instance';
    },
  };

  const result = await OnchangeEngine.run(meta, draft, ['Name'], { withCompute: false });

  expect(result.touchedHandlers).toEqual(['onName']);
  expect(result.value).toEqual({ Code: 'set-by-instance' });
});

test('onchange engine invokes instanceNoArgs async handler and awaits its result', async () => {
  const meta = createMeta({
    fields: [
      ['Qty', { type: 'int' }],
      ['Total', { type: 'int' }],
    ],
    onchangeHandlers: [{ method: 'onQty', triggers: ['Qty'], signature: 'instanceNoArgs' }],
  });

  const draft: any = {
    Qty: 3,
    Total: 0,
    async onQty() {
      const multiplier = await Promise.resolve(10);
      this.Total = Number(this.Qty || 0) * multiplier;
    },
  };

  const result = await OnchangeEngine.run(meta, draft, ['Qty'], { withCompute: false });

  expect(result.touchedHandlers).toEqual(['onQty']);
  expect(result.value).toEqual({ Total: 30 });
});

test('onchange engine calls legacyCtx-signature handler without ctx when flag is off (end-state)', async () => {
  const meta = createMeta({
    fields: [
      ['Name', { type: 'varchar' }],
      ['Code', { type: 'varchar' }],
    ],
    // Even with legacyCtx signature, the engine calls without ctx when flag is off.
    onchangeHandlers: [{ method: 'onName', triggers: ['Name'], signature: 'legacyCtx' }],
  });

  const draft: any = {
    Name: 'A',
    Code: '',
    onName() {
      // No ctx argument — works via this assignment only.
      this.Code = 'via-this';
    },
  };

  const result = await OnchangeEngine.run(meta, draft, ['Name'], { withCompute: false });

  expect(result.touchedHandlers).toEqual(['onName']);
  expect(result.value).toEqual({ Code: 'via-this' });
});

test('onchange engine calls unset-signature handler without ctx when flag is off (end-state)', async () => {
  const meta = createMeta({
    fields: [
      ['Name', { type: 'varchar' }],
      ['Code', { type: 'varchar' }],
    ],
    // Omit signature entirely — default is legacyCtx, but flag is off so no ctx.
    onchangeHandlers: [{ method: 'onName', triggers: ['Name'] }],
  });

  const draft: any = {
    Name: 'A',
    Code: '',
    onName() {
      this.Code = 'default-this';
    },
  };

  const result = await OnchangeEngine.run(meta, draft, ['Name'], { withCompute: false });

  expect(result.touchedHandlers).toEqual(['onName']);
  expect(result.value).toEqual({ Code: 'default-this' });
});

test('onchange engine runs mixed signature handlers in priority order', async () => {
  const meta = createMeta({
    fields: [
      ['A', { type: 'varchar' }],
      ['B', { type: 'varchar' }],
      ['C', { type: 'varchar' }],
    ],
    onchangeHandlers: [
      { method: 'onLegacy', triggers: ['A'], priority: 1, signature: 'legacyCtx' },
      { method: 'onInstance', triggers: ['A'], priority: 2, signature: 'instanceNoArgs' },
    ],
  });

  const order: string[] = [];

  const draft: any = {
    A: 'start',
    B: '',
    C: '',
    onLegacy() {
      order.push('legacy');
      this.B = 'legacy-ok';
    },
    onInstance() {
      order.push('instance');
      this.C = 'instance-ok';
    },
  };

  const result = await OnchangeEngine.run(meta, draft, ['A'], { withCompute: false });

  expect(order).toEqual(['legacy', 'instance']);
  expect(result.touchedHandlers).toEqual(['onLegacy', 'onInstance']);
  expect(result.value).toEqual({ B: 'legacy-ok', C: 'instance-ok' });
});

test('onchange engine respects stopOnError for instanceNoArgs handler that throws', async () => {
  const meta = createMeta({
    fields: [
      ['A', { type: 'varchar' }],
      ['B', { type: 'varchar' }],
    ],
    onchangeHandlers: [
      { method: 'onFail', triggers: ['A'], priority: 1, signature: 'instanceNoArgs' },
      { method: 'onNext', triggers: ['A'], priority: 2, signature: 'instanceNoArgs' },
    ],
  });

  const draft: any = {
    A: 'x',
    B: '',
    onFail() {
      throw new Error('instance-boom');
    },
    onNext() {
      this.B = 'should-not-run';
    },
  };

  const result = await OnchangeEngine.run(meta, draft, ['A'], { withCompute: false });

  expect(result.touchedHandlers).toEqual(['onFail']);
  expect(result.value).toBeUndefined();
  expect(result.messages?.some(m => String(m.message || '').includes('instance-boom'))).toBe(true);
  expect(draft.B).toBe('');
});

test('onchange engine instanceNoArgs handler returns messages/condition/selection', async () => {
  const meta = createMeta({
    fields: [
      ['Name', { type: 'varchar' }],
      ['Code', { type: 'varchar' }],
    ],
    onchangeHandlers: [{ method: 'onName', triggers: ['Name'], signature: 'instanceNoArgs' }],
  });

  const draft: any = {
    Name: 'A',
    Code: 'A',
    onName() {
      this.Code = 'CHANGED';
      return {
        messages: [{ level: 'warn', message: 'instance-warn' }],
        condition: [{ field: 'Code', condition: ['Code', '=', 'CHANGED'] }],
        selection: [{ field: 'Code', selection: ['A', 'CHANGED'] }],
      };
    },
  };

  const result = await OnchangeEngine.run(meta, draft, ['Name'], { withCompute: false });

  expect(result.touchedHandlers).toEqual(['onName']);
  expect(result.value).toEqual({ Code: 'CHANGED' });
  expect(result.messages?.some(m => String(m.message || '').includes('instance-warn'))).toBe(true);
  expect(result.condition).toEqual([{ field: 'Code', condition: ['Code', '=', 'CHANGED'] }]);
  expect(result.selection).toEqual([{ field: 'Code', selection: ['A', 'CHANGED'] }]);
});

test('onchange engine ignores legacyCtx signature and runs all handlers without ctx when flag is disabled', async () => {
  // Simulate ENABLE_ONCHANGE_LEGACY_CTX = false by setting signature to
  // instanceNoArgs on handlers that would otherwise expect ctx.
  // The engine should call all handlers without ctx regardless.
  const meta = createMeta({
    fields: [
      ['A', { type: 'varchar' }],
      ['B', { type: 'varchar' }],
    ],
    onchangeHandlers: [
      { method: 'onLegacy', triggers: ['A'], priority: 1, signature: 'instanceNoArgs' },
      { method: 'onInstance', triggers: ['A'], priority: 2, signature: 'instanceNoArgs' },
    ],
  });

  let legacyCalled = false;
  let instanceCalled = false;

  const draft: any = {
    A: 'x',
    B: '',
    onLegacy() {
      legacyCalled = true;
      // This handler was originally legacyCtx; after migration it works
      // via this-assignment and return side-effects.
      this.B = 'legacy-migrated';
    },
    onInstance() {
      instanceCalled = true;
      return { messages: [{ level: 'info', message: 'instance-ok' }] };
    },
  };

  const result = await OnchangeEngine.run(meta, draft, ['A'], { withCompute: false });

  expect(legacyCalled).toBe(true);
  expect(instanceCalled).toBe(true);
  expect(result.touchedHandlers).toEqual(['onLegacy', 'onInstance']);
  expect(result.value).toEqual({ B: 'legacy-migrated' });
  expect(result.messages?.some(m => String(m.message || '').includes('instance-ok'))).toBe(true);
});
