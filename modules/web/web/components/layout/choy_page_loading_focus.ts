// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/** True when node is root or nested under root (works without Node.contains). */
function isUnderRoot(root: ParentNode, node: Node | null | undefined): boolean {
  let cur: Node | null | undefined = node;
  while (cur) {
    if (cur === root) return true;
    cur = cur.parentNode;
  }
  return false;
}

/**
 * Blur the focused control when it is inside root so loading chrome can apply
 * inert without leaving assistive focus on a masked descendant.
 */
export function blurFocusedDescendant(
  root: ParentNode | null | undefined,
  doc: Pick<Document, 'activeElement'> = document,
): void {
  if (!root) return;
  const active = doc.activeElement as { blur?: () => void } | null;
  if (!active || typeof active.blur !== 'function') return;
  if (!isUnderRoot(root, active as Node)) return;
  try {
    active.blur();
  } catch {
    // ignore hosts that reject blur
  }
}
