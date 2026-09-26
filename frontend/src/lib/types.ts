// Shared TypeScript shapes that mirror the PocketBase collection schemas
// defined in pb-wiki/migrations. Keep these in sync when you change a field.

import type { RecordModel } from 'pocketbase'

export type Role = 'admin' | 'editor' | 'viewer'

export interface UserRecord extends RecordModel {
  email: string
  verified: boolean
  emailVisibility: boolean
  role: Role
  groups: string[] // group record ids
  name?: string
  avatar?: string
  expand?: { groups?: GroupRecord[] }
}

export interface GroupRecord extends RecordModel {
  name: string
}

export interface WikiConfig {
  id: string
  title: string
  private_default: boolean
  require_login: boolean
  default_landing_path: string
}

export type AccessLevel = 'public' | 'private' | 'restricted'

export interface DocumentRecord extends RecordModel {
  path: string
  title: string
  body: string
  access: AccessLevel
  groups: string[] // group record ids; used when access is 'restricted'
  nav_order: number // sidebar sorts siblings by this, then by name
  updated_by: string
  created: string
  updated: string
  expand?: { groups?: GroupRecord[]; updated_by?: UserRecord }
}
