// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import BaseModel from '../../orm/model/model';
import { MetadataStorage } from '../../orm/metadata/storage';
import { resolveInstanceHandler, createEntityBackedModelInstance } from './handler_runtime';

function fakeMeta(type: any, overrides?: Record<string, unknown>) {
  const base = MetadataStorage.instance.getModelMetadata(type as any) || {
    type,
    fields: new Map(),
    modelName: type?.name || 'Unknown',
  } as any;
  return { ...base, ...overrides };
}

test('resolveInstanceHandler resolves a method from prototype', () => {
  class Target {
    computeX() {
      return 42;
    }
  }
  const meta = fakeMeta(Target, {
    type: Target,
  });

  const fn = resolveInstanceHandler(meta as any, 'X', 'computeX', '@Compute');
  expect(typeof fn).toBe('function');
  expect(fn.call(new Target())).toBe(42);
});

test('resolveInstanceHandler throws when method name is empty', () => {
  class Target {}
  const meta = fakeMeta(Target, { type: Target });

  expect(() => resolveInstanceHandler(meta as any, 'X', '', '@Compute')).toThrow(
    '@Compute handler is missing method name'
  );
});

test('resolveInstanceHandler throws when method not found on prototype', () => {
  class Target {}
  const meta = fakeMeta(Target, { type: Target });

  expect(() => resolveInstanceHandler(meta as any, 'X', 'missingMethod', '@Compute')).toThrow(
    '@Compute handler not found'
  );
});

test('createEntityBackedModelInstance get trap prefers entityRecord over prototype', () => {
  class Target {
    Name = 'proto-name';
    greet() {
      return `Hello ${this.Name}`;
    }
  }
  const meta = fakeMeta(Target, { type: Target });
  const instance = createEntityBackedModelInstance(meta as any, { Name: 'entity-name' });

  expect((instance as any).Name).toBe('entity-name');
  expect(typeof (instance as any).greet).toBe('function');
});

test('createEntityBackedModelInstance set trap writes to entityRecord for plain keys', () => {
  class Target {
    Name = '';
  }
  const meta = fakeMeta(Target, { type: Target });
  const instance = createEntityBackedModelInstance(meta as any, {});

  (instance as any).Name = 'updated';
  expect((instance as any).Name).toBe('updated');
});

test('createEntityBackedModelInstance set trap respects prototype setters', () => {
  let setterCalled = false;
  let setterValue: unknown;

  class Target {
    private _name = '';

    get Name() {
      return this._name;
    }
    set Name(v: string) {
      setterCalled = true;
      setterValue = v;
      this._name = v;
    }
  }
  const meta = fakeMeta(Target, { type: Target });
  const instance = createEntityBackedModelInstance(meta as any, {});

  (instance as any).Name = 'via-setter';
  expect(setterCalled).toBe(true);
  expect(setterValue).toBe('via-setter');
  expect((instance as any).Name).toBe('via-setter');
});

test('createEntityBackedModelInstance deleteProperty removes from entityRecord', () => {
  class Target {
    Name = 'default';
  }
  const meta = fakeMeta(Target, { type: Target });
  const instance = createEntityBackedModelInstance(meta as any, { Name: 'test', Age: 25 });

  delete (instance as any).Name;
  expect((instance as any).Name).toBeUndefined();
  expect('Name' in (instance as any)).toBe(false);
  expect((instance as any).Age).toBe(25);
});

test('createEntityBackedModelInstance has trap checks entityRecord first', () => {
  class Target {
    protoMethod() { return 1; }
  }
  const meta = fakeMeta(Target, { type: Target });
  const instance = createEntityBackedModelInstance(meta as any, { Name: 'test' });

  expect('Name' in (instance as any)).toBe(true);
  expect('protoMethod' in (instance as any)).toBe(true);
  expect('missing' in (instance as any)).toBe(false);
});

test('createEntityBackedModelInstance ownKeys merges entity and target keys', () => {
  class Target {
    protoMethod() { return 1; }
  }
  const meta = fakeMeta(Target, { type: Target });
  const instance = createEntityBackedModelInstance(meta as any, { Name: 'test', Age: 25 });

  const keys = Reflect.ownKeys(instance as any);
  expect(keys).toContain('Name');
  expect(keys).toContain('Age');
  expect(keys).toContain('protoMethod');
});

test('createEntityBackedModelInstance getOwnPropertyDescriptor returns entity descriptor', () => {
  class Target {
    Name = 'proto-name';
  }
  const meta = fakeMeta(Target, { type: Target });
  const instance = createEntityBackedModelInstance(meta as any, { Name: 'entity-name' });

  const desc = Object.getOwnPropertyDescriptor(instance as any, 'Name');
  expect(desc).toBeTruthy();
  expect(desc!.value).toBe('entity-name');
  expect(desc!.enumerable).toBe(true);
  expect(desc!.writable).toBe(true);
});

test('createEntityBackedModelInstance defineProperty sets entityRecord value', () => {
  class Target {}
  const meta = fakeMeta(Target, { type: Target });
  const instance = createEntityBackedModelInstance(meta as any, {});

  Object.defineProperty(instance as any, 'Name', {
    value: 'defined',
    configurable: true,
    enumerable: true,
    writable: false,
  });

  // defineProperty writes to entityRecord even for non-writable descriptors
  // because the proxy's defineProperty trap has value-in-descriptor branch.
  expect((instance as any).Name).toBe('defined');
});

test('createEntityBackedModelInstance handles non-string keys gracefully', () => {
  class Target {}
  const meta = fakeMeta(Target, { type: Target });
  const instance = createEntityBackedModelInstance(meta as any, {});

  const sym = Symbol('test');
  (instance as any)[sym] = 'symbol-value';
  expect((instance as any)[sym]).toBe('symbol-value');
});

test('createEntityBackedModelInstance deleteProperty with symbol delegates to Reflect', () => {
  class Target {}
  const meta = fakeMeta(Target, { type: Target });
  const instance = createEntityBackedModelInstance(meta as any, {});

  const sym = Symbol('del');
  (instance as any)[sym] = 'val';
  delete (instance as any)[sym];
  // Symbol-keyed delete goes through Reflect.deleteProperty
  expect((instance as any)[sym]).toBeUndefined();
});
