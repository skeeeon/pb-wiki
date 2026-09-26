import { pb } from '@/lib/pb'
import type { GroupRecord } from '@/lib/types'

// Groups are records (so API rules can compare them), but people edit them
// as a comma-separated list of names. These two helpers convert between the
// two. Only admins may create groups; an editor typing an unknown name gets
// an error instead.

export function groupNames(groups: GroupRecord[] | undefined): string {
  return (groups ?? []).map((g) => g.name).join(', ')
}

export async function groupIds(csv: string, create = false): Promise<string[]> {
  const names = [...new Set(csv.split(',').map((s) => s.trim()).filter(Boolean))]
  const ids: string[] = []
  for (const name of names) {
    const found = await pb
      .collection('groups')
      .getList<GroupRecord>(1, 1, { filter: pb.filter('name = {:name}', { name }) })
    if (found.items.length > 0) {
      ids.push(found.items[0].id)
    } else if (create) {
      ids.push((await pb.collection('groups').create<GroupRecord>({ name })).id)
    } else {
      throw new Error(`Unknown group "${name}". An admin can create it on the Users page.`)
    }
  }
  return ids
}
