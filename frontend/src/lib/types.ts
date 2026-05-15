// Shared TypeScript shapes that mirror the PocketBase collection schemas
// defined in pb-wiki/migrations. Keep these in sync when you change a field.

import type { RecordModel } from 'pocketbase'

export type Role = 'admin' | 'editor' | 'viewer'

export interface UserRecord extends RecordModel {
  email: string
  verified: boolean
  emailVisibility: boolean
  role: Role
  groups: string[] | null
  name?: string
  avatar?: string
}

export interface WikiConfig {
  id: string
  title: string
  private_default: boolean
  require_login: boolean
  default_landing_path: string
}

export interface DocumentRecord {
  id: string
  path: string
  title: string
  body: string
  updated_by: string
  created: string
  updated: string
}

export type AccessLevel = 'public' | 'private' | 'restricted'

export interface AccessRuleRecord {
  id: string
  pattern: string
  access: AccessLevel
  groups: string[] | null
  priority: number
  description: string
  created: string
  updated: string
}
