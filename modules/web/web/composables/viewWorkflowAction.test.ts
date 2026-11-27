// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { ref } from 'vue';
import { ChoyMessage } from '@/web/web/composables/useChoyMessage';
import {
  isViewWorkflowPartialError,
  runViewWorkflowAction,
  useFormWorkflowAction,
  useListWorkflowAction,
  ViewWorkflowPartialError,
} from './viewWorkflowAction';

const origError = ChoyMessage.error;
const messageCalls: string[] = [];

beforeEach(() => {
  messageCalls.length = 0;
  ChoyMessage.error = ((msg: string) => {
    messageCalls.push(String(msg));
  }) as typeof ChoyMessage.error;
});

afterEach(() => {
  ChoyMessage.error = origError;
});

describe('runViewWorkflowAction', () => {
  test('runs call, afterSuccess, onSuccess in order', async () => {
    const busy = ref(false);
    const steps: string[] = [];
    await runViewWorkflowAction({
      busy,
      call: async () => {
        steps.push('call');
        expect(busy.value).toBe(true);
      },
      afterSuccess: async () => {
        steps.push('after');
      },
      onSuccess: async () => {
        steps.push('onSuccess');
      },
    });
    expect(steps).toEqual(['call', 'after', 'onSuccess']);
    expect(busy.value).toBe(false);
  });

  test('toasts and skips afterSuccess when call throws', async () => {
    const busy = ref(false);
    let after = 0;
    await runViewWorkflowAction({
      busy,
      call: async () => {
        throw new Error('boom');
      },
      afterSuccess: async () => {
        after += 1;
      },
      errorMessage: 'Custom fail',
    });
    expect(after).toBe(0);
    expect(busy.value).toBe(false);
    expect(messageCalls).toContain('Custom fail');
  });

  test('runs afterSuccess on partial workflow failure', async () => {
    const busy = ref(false);
    let after = 0;
    await runViewWorkflowAction({
      busy,
      call: async () => {
        throw new ViewWorkflowPartialError('partial');
      },
      afterSuccess: async () => {
        after += 1;
      },
      errorMessage: 'Partial fail',
    });
    expect(after).toBe(1);
    expect(messageCalls).toContain('Partial fail');
    expect(busy.value).toBe(false);
  });

  test('isViewWorkflowPartialError accepts class and duck-typed partialSuccess', () => {
    expect(isViewWorkflowPartialError(new ViewWorkflowPartialError('x'))).toBe(true);
    expect(isViewWorkflowPartialError({ partialSuccess: true, message: 'x' })).toBe(true);
    expect(isViewWorkflowPartialError(new Error('nope'))).toBe(false);
    expect(isViewWorkflowPartialError(null)).toBe(false);
  });

  test('runs afterSuccess for duck-typed partialSuccess errors', async () => {
    const busy = ref(false);
    let after = 0;
    await runViewWorkflowAction({
      busy,
      call: async () => {
        throw { partialSuccess: true, message: 'duck partial' };
      },
      afterSuccess: async () => {
        after += 1;
      },
      errorMessage: 'Duck fail',
    });
    expect(after).toBe(1);
    expect(messageCalls).toContain('Duck fail');
  });

  test('no-ops when already busy', async () => {
    const busy = ref(true);
    let calls = 0;
    await runViewWorkflowAction({
      busy,
      call: async () => {
        calls += 1;
      },
    });
    expect(calls).toBe(0);
  });

  test('resolveErrorMessage uses function, string, and fallbacks', async () => {
    const busy = ref(false);
    await runViewWorkflowAction({
      busy,
      call: async () => {
        throw new Error('rpc');
      },
      errorMessage: (err) => (err instanceof Error ? `fn:${err.message}` : 'fn:other'),
    });
    expect(messageCalls).toContain('fn:rpc');

    messageCalls.length = 0;
    await runViewWorkflowAction({
      busy,
      call: async () => {
        throw new Error('rpc2');
      },
      errorMessage: () => {
        throw new Error('formatter blew up');
      },
      fallbackErrorMessage: 'fallback',
    });
    expect(messageCalls).toContain('rpc2');

    messageCalls.length = 0;
    await runViewWorkflowAction({
      busy,
      call: async () => {
        throw { message: ' boxed ' };
      },
      errorMessage: '  string override  ',
    });
    expect(messageCalls).toContain('string override');

    messageCalls.length = 0;
    await runViewWorkflowAction({
      busy,
      call: async () => {
        throw 'plain';
      },
      fallbackErrorMessage: 'fallback',
    });
    expect(messageCalls).toContain('plain');
  });

  test('partial failure still runs afterSuccess when sync throws', async () => {
    const busy = ref(false);
    let after = 0;
    await runViewWorkflowAction({
      busy,
      call: async () => {
        throw new ViewWorkflowPartialError('partial');
      },
      afterSuccess: async () => {
        after += 1;
        throw new Error('sync blew up');
      },
    });
    expect(after).toBe(1);
    expect(busy.value).toBe(false);
  });
});

describe('useFormWorkflowAction', () => {
  test('display mode calls reloadDisplay after success', async () => {
    let reloads = 0;
    const formRef = ref({
      getViewMode: () => 'display' as const,
      reloadDisplay: async () => {
        reloads += 1;
      },
    });
    const { run, busy } = useFormWorkflowAction({ formRef });
    await run(async () => undefined);
    expect(reloads).toBe(1);
    expect(busy.value).toBe(false);
  });

  test('edit mode skips reloadDisplay', async () => {
    let reloads = 0;
    const formRef = ref({
      getViewMode: () => 'edit' as const,
      reloadDisplay: async () => {
        reloads += 1;
      },
    });
    const { run } = useFormWorkflowAction({ formRef, reload: 'display' });
    await run(async () => undefined);
    expect(reloads).toBe(0);
  });

  test('reload none never touches form', async () => {
    let reloads = 0;
    const formRef = ref({
      getViewMode: () => 'display' as const,
      reloadDisplay: async () => {
        reloads += 1;
      },
    });
    const { run } = useFormWorkflowAction({ formRef, reload: 'none' });
    await run(async () => undefined);
    expect(reloads).toBe(0);
  });

  test('create mode and refresh fallback paths', async () => {
    let refresh = 0;
    const formRef = ref({
      getViewMode: () => 'create' as const,
      reloadDisplay: async () => {
        refresh += 1;
      },
      refresh: async () => {
        refresh += 10;
      },
    });
    const { run } = useFormWorkflowAction({ formRef });
    await run(async () => undefined);
    expect(refresh).toBe(0);

    const formRef2 = ref({
      getViewMode: () => 'display' as const,
      refresh: async () => {
        refresh += 1;
      },
    });
    const { run: run2 } = useFormWorkflowAction({ formRef: formRef2 });
    await run2(async () => undefined);
    expect(refresh).toBe(1);
  });

  test('onSuccess runs after display reload', async () => {
    let onSuccess = 0;
    const formRef = ref({
      getViewMode: () => 'display' as const,
      reloadDisplay: async () => undefined,
    });
    const { run } = useFormWorkflowAction({
      formRef,
      onSuccess: async () => {
        onSuccess += 1;
      },
    });
    await run(async () => undefined);
    expect(onSuccess).toBe(1);
  });
});

describe('useListWorkflowAction', () => {
  test('apply calls softRefresh and optional clearSelection', async () => {
    let soft = 0;
    let cleared = 0;
    const listRef = ref({
      softRefresh: async () => {
        soft += 1;
      },
      clearSelection: () => {
        cleared += 1;
      },
    });
    const { run } = useListWorkflowAction({
      listRef,
      reload: 'apply',
      clearSelection: true,
    });
    await run(async () => undefined);
    expect(soft).toBe(1);
    expect(cleared).toBe(1);
  });

  test('reload none skips softRefresh', async () => {
    let soft = 0;
    const listRef = ref({
      softRefresh: async () => {
        soft += 1;
      },
    });
    const { run } = useListWorkflowAction({ listRef, reload: 'none' });
    await run(async () => undefined);
    expect(soft).toBe(0);
  });

  test('failed call does not softRefresh', async () => {
    let soft = 0;
    const listRef = ref({
      softRefresh: async () => {
        soft += 1;
      },
    });
    const { run } = useListWorkflowAction({ listRef });
    await run(async () => {
      throw new Error('nope');
    });
    expect(soft).toBe(0);
  });

  test('falls back to load then refresh when softRefresh is missing', async () => {
    let load = 0;
    let refresh = 0;
    const listRef = ref({
      load: async () => {
        load += 1;
      },
      refresh: async () => {
        refresh += 1;
      },
    });
    const { run } = useListWorkflowAction({ listRef });
    await run(async () => undefined);
    expect(load).toBe(1);
    expect(refresh).toBe(0);

    const listRef2 = ref({
      refresh: async () => {
        refresh += 1;
      },
    });
    const { run: run2 } = useListWorkflowAction({ listRef: listRef2 });
    await run2(async () => undefined);
    expect(refresh).toBe(1);
  });

  test('clearSelection only when enabled and softRefresh missing still loads', async () => {
    let cleared = 0;
    let load = 0;
    const listRef = ref({
      load: async () => {
        load += 1;
      },
      clearSelection: () => {
        cleared += 1;
      },
    });
    const { run } = useListWorkflowAction({ listRef, clearSelection: true });
    await run(async () => undefined);
    expect(load).toBe(1);
    expect(cleared).toBe(1);
  });
});
