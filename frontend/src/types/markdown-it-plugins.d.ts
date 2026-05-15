// Type shims for markdown-it plugins that don't ship their own types.
declare module 'markdown-it-task-lists' {
  import type { PluginWithOptions } from 'markdown-it'
  interface TaskListOptions {
    enabled?: boolean
    label?: boolean
    labelAfter?: boolean
  }
  const plugin: PluginWithOptions<TaskListOptions>
  export default plugin
}

declare module 'markdown-it-container' {
  import type MarkdownIt from 'markdown-it'
  import type Token from 'markdown-it/lib/token.mjs'
  interface ContainerOptions {
    validate?: (params: string) => boolean
    render?: (tokens: Token[], idx: number, options: unknown, env: unknown, self: unknown) => string
    marker?: string
  }
  const plugin: (md: MarkdownIt, name: string, options?: ContainerOptions) => void
  export default plugin
}
