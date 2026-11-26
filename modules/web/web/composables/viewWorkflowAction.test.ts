// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { ref } from 'vue';
import { ChoyMessage } from '@/web/web/composables/useChoyMessage';
import {
  runViewWorkflowAction,
  useFormWorkflowAction,
  useListWorkflowAction,
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
});
