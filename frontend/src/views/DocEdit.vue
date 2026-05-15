<script setup lang="ts">
import { ref, watch, toRef } from 'vue'
import { useRouter } from 'vue-router'
import { MdEditor } from 'md-editor-v3'
import 'md-editor-v3/lib/style.css'

import { pb } from '@/lib/pb'
import { useDoc } from '@/composables/useDoc'
import { useAuthStore } from '@/stores/auth'
import { useTheme } from '@/composables/useTheme'

const props = defineProps<{ path: string; mode: 'edit' | 'new' }>()
const path = toRef(props, 'path')
const router = useRouter()
const auth = useAuthStore()
const { theme } = useTheme()

// In 'new' mode we never fetch — passing a sentinel path that won't exist so
// the composable's 404 branch fires and we stay in a blank-form state.
const fetchPath = () => (props.mode === 'edit' ? path.value : '\x00__pb-wiki-new__')
const { doc, loading } = useDoc(fetchPath)

const title = ref('')
const body = ref('')
const newPath = ref(path.value)
const saving = ref(false)
const errorMsg = ref('')

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
    router.push(`/doc/${data.path}`)
  } catch (err) {
    errorMsg.value = err instanceof Error ? err.message : String(err)
  } finally {
    saving.value = false
  }
}

async function deleteDoc() {
  if (!doc.value) return
  if (!confirm(`Delete "${doc.value.title || doc.value.path || 'this page'}"?`)) return
  try {
    await pb.collection('documents').delete(doc.value.id)
    router.push('/')
  } catch (err) {
    errorMsg.value = err instanceof Error ? err.message : String(err)
  }
}
</script>

<template>
  <div class="max-w-6xl mx-auto space-y-4">
    <header class="flex items-baseline justify-between gap-4 flex-wrap">
      <h1 class="text-lg font-semibold">
        {{ mode === 'edit' ? 'Edit' : 'New' }} document
      </h1>
      <nav class="flex items-center gap-3 text-sm">
        <button
          v-if="mode === 'edit' && doc"
          type="button"
          class="text-red-600 dark:text-red-400 underline"
          @click="deleteDoc"
        >
          Delete
        </button>
        <button
          type="button"
          :disabled="saving || (mode === 'edit' && loading)"
          class="rounded-md bg-brand-red hover:bg-brand-red-hover text-white px-3 py-1.5 text-sm font-medium disabled:opacity-60"
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
  </div>
</template>
