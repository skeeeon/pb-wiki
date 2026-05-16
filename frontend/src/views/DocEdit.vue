<script setup lang="ts">
import { computed, ref, watch, toRef } from 'vue'
import { useRouter } from 'vue-router'
import { MdEditor } from 'md-editor-v3'
import 'md-editor-v3/lib/style.css'
import {
  AlertDialogRoot,
  AlertDialogTrigger,
  AlertDialogPortal,
  AlertDialogOverlay,
  AlertDialogContent,
  AlertDialogTitle,
  AlertDialogDescription,
  AlertDialogCancel,
  AlertDialogAction,
} from 'reka-ui'

import { pb } from '@/lib/pb'
import { useDoc } from '@/composables/useDoc'
import { useAuthStore } from '@/stores/auth'
import { useDocsStore } from '@/stores/docs'
import { useTheme } from '@/composables/useTheme'

const props = defineProps<{ path: string; mode: 'edit' | 'new' }>()
const path = toRef(props, 'path')
const router = useRouter()
const auth = useAuthStore()
const docsStore = useDocsStore()
const { theme } = useTheme()

// In 'new' mode we never fetch — passing a sentinel path that won't exist so
// the composable's 404 branch fires and we stay in a blank-form state.
const fetchPath = () => (props.mode === 'edit' ? path.value : '\x00__pb-wiki-new__')
const { doc, loading } = useDoc(fetchPath)

const title = ref('')
const body = ref('')
const newPath = ref(path.value)
const saving = ref(false)
const deleting = ref(false)
const errorMsg = ref('')

// Cancel target — edit mode goes back to the saved doc, new mode pops up
// to the parent path (or home if it was already top-level).
const cancelTo = computed(() => {
  if (props.mode === 'edit') return `/doc/${path.value}`
  const parent = path.value.replace(/\/?[^/]*$/, '')
  return parent ? `/doc/${parent}` : '/'
})

// Re-hydrate the form when the underlying doc resolves (edit mode) or the
// route path changes (e.g. clicking another doc's Edit link).
watch(
  [doc, path],
  ([d, p]) => {
    if (props.mode === 'edit') {
      title.value = d?.title ?? ''
      body.value = d?.body ?? ''
      newPath.value = d?.path ?? p
    } else {
      title.value = ''
      body.value = ''
      newPath.value = p
    }
  },
  { immediate: true },
)

// md-editor-v3 calls this with the dropped/pasted files and expects either a
// promise-resolved string[] or invocation of callBack(string[]) with absolute
// asset URLs. We POST each file to the `assets` collection and return PB's
// auto-generated download URL.
async function onUploadImg(files: File[], callBack: (urls: string[]) => void) {
  try {
    const urls: string[] = []
    for (const file of files) {
      const fd = new FormData()
      fd.append('file', file)
      if (auth.record?.id) fd.append('uploaded_by', auth.record.id)
      const rec = await pb.collection('assets').create(fd)
      urls.push(pb.files.getURL(rec, rec.file))
    }
    callBack(urls)
  } catch (err) {
    errorMsg.value = `Upload failed: ${err instanceof Error ? err.message : String(err)}`
  }
}

async function save() {
  if (saving.value) return
  saving.value = true
  errorMsg.value = ''
  try {
    const data: Record<string, unknown> = {
      path: newPath.value.trim().replace(/^\/+|\/+$/g, ''),
      title: title.value,
      body: body.value,
      updated_by: auth.record?.id,
    }
    if (props.mode === 'edit' && doc.value) {
      await pb.collection('documents').update(doc.value.id, data)
    } else {
      await pb.collection('documents').create(data)
    }
    await docsStore.reload()
    router.push(`/doc/${data.path}`)
  } catch (err) {
    errorMsg.value = err instanceof Error ? err.message : String(err)
  } finally {
    saving.value = false
  }
}

async function deleteDoc() {
  if (!doc.value || deleting.value) return
  deleting.value = true
  try {
    await pb.collection('documents').delete(doc.value.id)
    await docsStore.reload()
    router.push('/')
  } catch (err) {
    errorMsg.value = err instanceof Error ? err.message : String(err)
  } finally {
    deleting.value = false
  }
}
</script>

<template>
  <div class="max-w-6xl mx-auto space-y-4">
    <header class="flex items-baseline justify-between gap-4 flex-wrap">
      <h1 class="text-lg font-semibold">
        {{ mode === 'edit' ? 'Edit' : 'New' }} document
      </h1>
      <nav class="flex items-center gap-2 text-sm">
        <RouterLink
          :to="cancelTo"
          class="inline-flex items-center px-3 py-1.5 rounded-md border border-zinc-300 dark:border-zinc-700 hover:bg-zinc-100 dark:hover:bg-zinc-800"
        >
          Cancel
        </RouterLink>
        <button
          type="button"
          :disabled="saving || (mode === 'edit' && loading)"
          class="rounded-md bg-brand-red hover:bg-brand-red-hover text-white px-3 py-1.5 font-medium disabled:opacity-60"
          @click="save"
        >
          {{ saving ? 'Saving…' : 'Save' }}
        </button>
      </nav>
    </header>

    <div class="grid gap-3 sm:grid-cols-[1fr_2fr]">
      <label class="block text-sm">
        <span class="text-zinc-700 dark:text-zinc-300">Path</span>
        <input
          v-model="newPath"
          class="mt-1 block w-full rounded-md border border-zinc-300 dark:border-zinc-700 bg-white dark:bg-zinc-950 px-3 py-2 text-sm font-mono"
          placeholder="engineering/runbooks/deploy"
        />
      </label>
      <label class="block text-sm">
        <span class="text-zinc-700 dark:text-zinc-300">Title</span>
        <input
          v-model="title"
          class="mt-1 block w-full rounded-md border border-zinc-300 dark:border-zinc-700 bg-white dark:bg-zinc-950 px-3 py-2 text-sm"
        />
      </label>
    </div>

    <MdEditor
      v-model="body"
      language="en-US"
      :theme="theme"
      :on-upload-img="onUploadImg"
      :style="{ height: 'calc(100dvh - 16rem)' }"
    />

    <p v-if="errorMsg" class="text-sm text-red-600 dark:text-red-400">
      {{ errorMsg }}
    </p>

    <!-- Danger zone — only shown when editing an existing doc. Kept far from
         the primary Save/Cancel actions so it's never a mis-click. -->
    <section
      v-if="mode === 'edit' && doc"
      class="mt-8 rounded-md border border-red-200 dark:border-red-900/50 p-4"
    >
      <div class="flex items-start justify-between gap-4 flex-wrap">
        <div class="min-w-0">
          <h2 class="text-sm font-semibold text-red-700 dark:text-red-400">Danger zone</h2>
          <p class="text-xs text-zinc-600 dark:text-zinc-400 mt-1">
            Deleting <code class="font-mono">{{ doc.path || '/' }}</code> is permanent.
            Children at deeper paths are not removed automatically.
          </p>
        </div>
        <AlertDialogRoot>
          <AlertDialogTrigger
            as="button"
            type="button"
            class="shrink-0 inline-flex items-center gap-1.5 rounded-md border border-red-300 dark:border-red-900/70 text-red-700 dark:text-red-400 px-3 py-1.5 text-sm font-medium hover:bg-red-50 dark:hover:bg-red-950/30"
          >
            <svg class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <polyline points="3 6 5 6 21 6" />
              <path d="M19 6l-1 14a2 2 0 0 1-2 2H8a2 2 0 0 1-2-2L5 6" />
              <path d="M10 11v6M14 11v6" />
              <path d="M9 6V4a2 2 0 0 1 2-2h2a2 2 0 0 1 2 2v2" />
            </svg>
            Delete this page
          </AlertDialogTrigger>
          <AlertDialogPortal>
            <AlertDialogOverlay class="fixed inset-0 z-[80] bg-black/50 data-[state=open]:animate-in data-[state=open]:fade-in" />
            <AlertDialogContent
              class="fixed left-1/2 top-1/2 z-[81] w-[min(90vw,28rem)] -translate-x-1/2 -translate-y-1/2 rounded-md border border-zinc-200 dark:border-zinc-800 bg-white dark:bg-zinc-900 p-5 shadow-xl focus:outline-none"
            >
              <AlertDialogTitle class="text-base font-semibold">
                Delete this page?
              </AlertDialogTitle>
              <AlertDialogDescription class="mt-2 text-sm text-zinc-600 dark:text-zinc-400">
                <span class="font-medium text-zinc-900 dark:text-zinc-100">
                  {{ doc.title || doc.path || 'Untitled' }}
                </span>
                will be permanently removed. This cannot be undone.
              </AlertDialogDescription>
              <div class="mt-5 flex justify-end gap-2">
                <AlertDialogCancel
                  as="button"
                  type="button"
                  class="inline-flex items-center rounded-md border border-zinc-300 dark:border-zinc-700 px-3 py-1.5 text-sm hover:bg-zinc-100 dark:hover:bg-zinc-800"
                >
                  Cancel
                </AlertDialogCancel>
                <AlertDialogAction
                  as="button"
                  type="button"
                  :disabled="deleting"
                  class="inline-flex items-center rounded-md bg-red-600 hover:bg-red-700 text-white px-3 py-1.5 text-sm font-medium disabled:opacity-60"
                  @click="deleteDoc"
                >
                  {{ deleting ? 'Deleting…' : 'Delete' }}
                </AlertDialogAction>
              </div>
            </AlertDialogContent>
          </AlertDialogPortal>
        </AlertDialogRoot>
      </div>
    </section>
  </div>
</template>
