import type { DocumentRecord } from '@/lib/types'

export interface TreeNode {
  segment: string // last path segment, e.g. "deploy"
  fullPath: string // full slug, e.g. "engineering/runbooks/deploy"
  title: string | null // null until a document with this exact path exists
  navOrder: number // the document's nav_order; 0 for implicit segments
  children: TreeNode[]
}

// buildTree groups documents by their slash-separated path into a nested
// structure suitable for recursive rendering. Intermediate segments that have
// no document record (e.g. "engineering/" exists implicitly because
// "engineering/runbooks" does) get a node with title=null; the sidebar still
// renders them so the user can navigate down to the leaf.
//
// Siblings sort by nav_order, then by segment name, so pages without a
// nav_order (0) keep alphabetical order.
export function buildTree(docs: DocumentRecord[]): TreeNode[] {
  const root: TreeNode = { segment: '', fullPath: '', title: null, navOrder: 0, children: [] }

  for (const d of docs) {
    if (d.path === '') {
      root.title = d.title || 'Home'
      continue
    }
    const segments = d.path.split('/')
    let node = root
    let acc = ''
    for (const seg of segments) {
      acc = acc ? `${acc}/${seg}` : seg
      let child = node.children.find((c) => c.segment === seg)
      if (!child) {
        child = { segment: seg, fullPath: acc, title: null, navOrder: 0, children: [] }
        node.children.push(child)
      }
      node = child
    }
    node.title = d.title || node.segment
    node.navOrder = d.nav_order ?? 0
  }

  const sort = (n: TreeNode) => {
    n.children.sort((a, b) => a.navOrder - b.navOrder || a.segment.localeCompare(b.segment))
    n.children.forEach(sort)
  }
  sort(root)
  return root.children
}

// childrenOf returns the nodes directly under path, in sidebar order. Folders
// with no page of their own (title null) are included, as in the sidebar.
export function childrenOf(docs: DocumentRecord[], path: string): TreeNode[] {
  let nodes = buildTree(docs)
  if (path === '') return nodes
  for (const seg of path.split('/')) {
    const next = nodes.find((n) => n.segment === seg)
    if (!next) return []
    nodes = next.children
  }
  return nodes
}

// suggestionsFor finds pages a missing path may have meant, by its last
// segment: pages that end in that segment first (likely moved), then pages
// whose path or title contains it.
export function suggestionsFor(docs: DocumentRecord[], path: string, limit = 5): DocumentRecord[] {
  const seg = path.split('/').pop()?.toLowerCase() ?? ''
  if (seg.length < 2) return []
  const score = (d: DocumentRecord) => {
    const p = d.path.toLowerCase()
    if (p === seg || p.endsWith('/' + seg)) return 2
    if (p.includes(seg) || (d.title ?? '').toLowerCase().includes(seg)) return 1
    return 0
  }
  return docs
    .map((d) => ({ d, s: score(d) }))
    .filter((x) => x.s > 0 && x.d.path !== path && !x.d.path.startsWith(path + '/'))
    .sort((a, b) => b.s - a.s || a.d.path.localeCompare(b.d.path))
    .slice(0, limit)
    .map((x) => x.d)
}
