// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

type GuestHomeRouter = { push: (to: string) => unknown }

/**
 * Build the Guest Home click handler: SPA push when a router is available,
 * otherwise native `location.assign` (router-less mounts / early boot).
 */
export function createGuestHomeNavigate(
  getHomeHref: () => string,
  getRouter: () => GuestHomeRouter | null | undefined,
): () => void {
  let router: GuestHomeRouter | null | undefined
  try {
    router = getRouter()
  } catch {
    router = undefined
  }
  if (router && typeof router.push === 'function') {
    return () => {
      void router!.push(getHomeHref())
    }
  }
  return () => {
    const loc = globalThis.location
    if (loc && typeof loc.assign === 'function') {
      loc.assign(getHomeHref())
    }
  }
}
