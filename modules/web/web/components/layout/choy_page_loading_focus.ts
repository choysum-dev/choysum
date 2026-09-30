// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * Minimal ancestor walk shape. Avoid typing root as ParentNode: Vue template
 * refs are HTMLElement with TipTap DOM augmentations that are not assignable
 * to lib.dom ParentNode under choysum typecheck.
 */
type WalkNode = {
  parentNode?: WalkNode | null;
};

type BlurFn = (
  root: object | null | undefined,
  doc?: Pick<Document, 'activeElement'>,
) => void;

/** True when node is root or nested under root (works without Node.contains). */
function isUnderRoot(root: object, node: WalkNode | null | undefined): boolean {
  let cur: WalkNode | null | undefined = node;
  while (cur) {
    if (cur === root) return true;
    cur = cur.parentNode ?? null;
  }
  return false;
}

/**
 * Blur the focused control when it is inside root so loading chrome can apply
 * inert without leaving assistive focus on a masked descendant.
 */
export function blurFocusedDescendantCore(
  root: object | null | undefined,
  doc: Pick<Document, 'activeElement'> = document,
): void {
  if (!root) return;
  const active = doc.activeElement as (WalkNode & { blur?: () => void }) | null;
  if (!active || typeof active.blur !== 'function') return;
  if (!isUnderRoot(root, active)) return;
  try {
    active.blur();
  } catch {
    // ignore hosts that throw from blur() (sync only; Element.blur returns void)
  }
}

let blurImpl: BlurFn = blurFocusedDescendantCore;

/** Production entry used by ChoyPage; dispatch allows FE tests to observe calls. */
export function blurFocusedDescendant(
  root: object | null | undefined,
  doc: Pick<Document, 'activeElement'> = document,
): void {
  blurImpl(root, doc);
}

/** Swap the blur implementation for mount tests; returns a restore function. */
export function setBlurFocusedDescendantImplForTest(next: BlurFn): () => void {
  const prev = blurImpl;
  blurImpl = next;
  return () => {
    blurImpl = prev;
  };
}
