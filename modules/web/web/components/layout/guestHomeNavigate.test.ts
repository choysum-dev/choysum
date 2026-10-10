// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { createGuestHomeNavigate } from './guestHomeNavigate'

describe('createGuestHomeNavigate', () => {
  test('pushes through the router when available', () => {
    const pushes: string[] = []
    const navigate = createGuestHomeNavigate(
      () => '/meta/modules',
      () => ({
        push: (to: string) => {
          pushes.push(to)
          return Promise.resolve()
        },
      }),
    )
    navigate()
    expect(pushes).toEqual(['/meta/modules'])
  })

  test('falls back to location.assign when router is missing', () => {
    const assigns: string[] = []
    const original = globalThis.location
    Object.defineProperty(globalThis, 'location', {
      configurable: true,
      value: {
        assign: (url: string) => {
          assigns.push(url)
        },
      },
    })
    try {
      const navigate = createGuestHomeNavigate(
        () => '/meta/modules',
        () => undefined,
      )
      navigate()
      expect(assigns).toEqual(['/meta/modules'])
    } finally {
      Object.defineProperty(globalThis, 'location', {
        configurable: true,
        value: original,
      })
    }
  })

  test('falls back to location.assign when getRouter throws', () => {
    const assigns: string[] = []
    const original = globalThis.location
    Object.defineProperty(globalThis, 'location', {
      configurable: true,
      value: {
        assign: (url: string) => {
          assigns.push(url)
        },
      },
    })
    try {
      const navigate = createGuestHomeNavigate(
        () => '/land',
        () => {
          throw new Error('no router')
        },
      )
      navigate()
      expect(assigns).toEqual(['/land'])
    } finally {
      Object.defineProperty(globalThis, 'location', {
        configurable: true,
        value: original,
      })
    }
  })

  test('no-ops when location.assign is unavailable', () => {
    const original = globalThis.location
    Object.defineProperty(globalThis, 'location', {
      configurable: true,
      value: {},
    })
    try {
      const navigate = createGuestHomeNavigate(
        () => '/x',
        () => null,
      )
      expect(() => navigate()).not.toThrow()
    } finally {
      Object.defineProperty(globalThis, 'location', {
        configurable: true,
        value: original,
      })
    }
  })
})
