import type { DocumentRecord } from '@/lib/types'

export interface TreeNode {
  segment: string // last path segment, e.g. "deploy"
  fullPath: string // full slug, e.g. "engineering/runbooks/deploy"
  title: string | null // null until a document with this exact path exists
  children: TreeNode[]
}

// buildTree groups documents by their slash-separated path into a nested
// structure suitable for recursive rendering. Intermediate segments that have
// no document record (e.g. "engineering/" exists implicitly because
// "engineering/runbooks" does) get a node with title=null; the sidebar still
// renders them so the user can navigate down to the leaf.
export function buildTree(docs: DocumentRecord[]): TreeNode[] {
  const root: TreeNode = { segment: '', fullPath: '', title: null, children: [] }

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
        child = { segment: seg, fullPath: acc, title: null, children: [] }
        node.children.push(child)
      }
      node = child
    }
    node.title = d.title || node.segment
  }

  const sort = (n: TreeNode) => {
    n.children.sort((a, b) => a.segment.localeCompare(b.segment))
    n.children.forEach(sort)
  }
  sort(root)
  return root.children
}
