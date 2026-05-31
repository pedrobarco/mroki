<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { getGate, updateGate, deleteGate } from '@/api'
import type { Gate } from '@/api'
import { useGateCache } from '@/composables/use-gate-cache'
import FieldListEditor from '@/components/gates/FieldListEditor.vue'
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Separator } from '@/components/ui/separator'
import { Switch } from '@/components/ui/switch'
import { Badge } from '@/components/ui/badge'
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
  AlertDialogTrigger,
} from '@/components/ui/alert-dialog'
import {
  ChevronLeft,
  Save,
  Lock,
  GitCompareArrows,
  TriangleAlert,
  Trash2,
  Check,
  Info,
} from 'lucide-vue-next'

const route = useRoute()
const router = useRouter()
const { setGate: cacheGate, getCachedGate } = useGateCache()

const gateId = route.params.id as string

const gate = ref<Gate | null>(null)
const loading = ref(true)
const error = ref<string | null>(null)
const saving = ref(false)
const saveError = ref<string | null>(null)
const saveSuccess = ref(false)
const deleting = ref(false)

// Form state
const name = ref('')
const redactedAdditionalFields = ref<string[]>([])
const diffIgnoredFields = ref<string[]>([])
const diffIncludedFields = ref<string[]>([])
const floatTolerance = ref('')
const sortArrays = ref(false)

// Default redacted fields (mirrors domain DefaultRedactedFields)
const defaultRedactedFields = [
  'headers.Authorization',
  'headers.Cookie',
  'headers.Set-Cookie',
  'headers.X-Api-Key',
]

function populateForm(g: Gate) {
  name.value = g.name
  redactedAdditionalFields.value = [...(g.redacted_fields ?? [])]
  diffIgnoredFields.value = [...(g.diff_config?.ignored_fields ?? [])]
  diffIncludedFields.value = [...(g.diff_config?.included_fields ?? [])]
  floatTolerance.value = g.diff_config?.float_tolerance
    ? g.diff_config.float_tolerance.toString()
    : ''
  sortArrays.value = g.diff_config?.sort_arrays ?? false
}

async function loadGate() {
  loading.value = true
  error.value = null

  // Settings only needs config data — use cache if available
  const cached = getCachedGate(gateId)
  if (cached) {
    gate.value = cached
    populateForm(cached)
    loading.value = false
    return
  }

  // Cache miss (e.g. direct link) — fetch from API
  try {
    const response = await getGate(gateId)
    gate.value = response.data
    cacheGate(response.data)
    populateForm(response.data)
  } catch (err) {
    error.value = err instanceof Error ? err.message : 'Failed to load gate'
  } finally {
    loading.value = false
  }
}

async function handleSave() {
  saving.value = true
  saveError.value = null
  saveSuccess.value = false

  try {
    const response = await updateGate(gateId, {
      name: name.value.trim(),
      diff_config: {
        ignored_fields: diffIgnoredFields.value,
        included_fields: diffIncludedFields.value,
        float_tolerance: floatTolerance.value ? parseFloat(floatTolerance.value) : 0,
        sort_arrays: sortArrays.value,
      },
      redacted_fields: redactedAdditionalFields.value,
    })

    gate.value = response.data
    cacheGate(response.data)
    saveSuccess.value = true
    setTimeout(() => (saveSuccess.value = false), 3000)
  } catch (err) {
    saveError.value = err instanceof Error ? err.message : 'Failed to update gate'
  } finally {
    saving.value = false
  }
}

async function handleDelete() {
  deleting.value = true
  try {
    await deleteGate(gateId)
    router.push('/gates')
  } catch (err) {
    error.value = err instanceof Error ? err.message : 'Failed to delete gate'
    deleting.value = false
  }
}

function goBack() {
  router.push(`/gates/${gateId}`)
}

onMounted(() => {
  loadGate()
})
</script>

<template>
  <div class="max-w-6xl mx-auto px-6 py-6">
    <!-- Back link -->
    <a
      class="inline-flex items-center gap-1.5 text-xs text-muted-foreground hover:text-foreground transition-colors mb-5 cursor-pointer"
      @click="goBack"
    >
      <ChevronLeft class="h-3.5 w-3.5" />
      Back to Gate
    </a>

    <!-- Loading -->
    <div v-if="loading" class="text-center py-12">
      <p class="text-muted-foreground">Loading gate settings...</p>
    </div>

    <!-- Error -->
    <Alert v-else-if="error && !gate" variant="destructive">
      <AlertTitle>Error</AlertTitle>
      <AlertDescription>{{ error }}</AlertDescription>
      <div class="mt-4">
        <Button variant="outline" size="sm" @click="loadGate">Retry</Button>
      </div>
    </Alert>

    <!-- Settings Content -->
    <div v-else-if="gate">
      <!-- Page Title -->
      <div class="flex items-center justify-between mb-6">
        <div>
          <h1 class="text-xl font-semibold tracking-tight mb-1">Gate Settings</h1>
          <div class="flex items-center gap-2 text-sm text-dim">
            <span class="font-medium text-muted-foreground">{{ gate.name }}</span>
            <span class="text-dim/50">&middot;</span>
            <code class="text-xs font-mono">{{ gate.id }}</code>
          </div>
        </div>
        <Button :disabled="saving" class="gap-2" @click="handleSave">
          <Save v-if="!saveSuccess" class="h-3.5 w-3.5" />
          <Check v-else class="h-3.5 w-3.5" />
          {{ saving ? 'Saving...' : saveSuccess ? 'Saved' : 'Save Changes' }}
        </Button>
      </div>

      <!-- Save feedback -->
      <Alert v-if="saveError" variant="destructive" class="mb-6">
        <AlertDescription>{{ saveError }}</AlertDescription>
      </Alert>

      <div class="space-y-6">
        <!-- Section: General -->
        <div class="bg-card border border-border rounded-xl overflow-hidden">
          <div class="px-5 py-4 border-b border-border/50">
            <h2 class="text-sm font-semibold tracking-tight">General</h2>
          </div>
          <div class="p-5">
            <div class="space-y-2 max-w-sm">
              <Label for="gate-name">Name</Label>
              <Input
                id="gate-name"
                v-model="name"
                type="text"
                placeholder="checkout-api"
                :disabled="saving"
              />
            </div>
          </div>
        </div>

        <!-- Section: Field Redaction -->
        <div class="bg-card border border-border rounded-xl overflow-hidden">
          <div class="px-5 py-4 border-b border-border/50">
            <div class="flex items-center gap-2 mb-1">
              <Lock class="h-4 w-4 text-warning" />
              <h2 class="text-sm font-semibold tracking-tight">Field Redaction</h2>
            </div>
            <p class="text-xs text-dim leading-relaxed">
              Sensitive header and body field values are replaced with
              <code class="text-xs font-mono bg-muted px-1.5 py-0.5 rounded text-warning"
                >[REDACTED]</code
              >
              before data is stored or compared. Use
              <code class="text-xs font-mono bg-muted px-1 py-0.5 rounded">headers.</code> or
              <code class="text-xs font-mono bg-muted px-1 py-0.5 rounded">body.</code> prefixes.
            </p>
          </div>

          <div class="p-5 space-y-5">
            <!-- Default Fields (read-only) -->
            <div>
              <div class="flex items-center gap-2 mb-2.5">
                <h3 class="text-xs font-medium text-accent-foreground">Default Fields</h3>
                <Badge variant="secondary" class="text-[10px] uppercase tracking-widest">
                  Always active
                </Badge>
              </div>
              <p class="text-xs text-dim mb-3">
                These fields are always redacted and cannot be removed.
              </p>
              <div class="flex flex-wrap gap-2">
                <span
                  v-for="field in defaultRedactedFields"
                  :key="field"
                  class="inline-flex items-center gap-1.5 text-xs font-mono bg-background/60 border border-border/50 text-muted-foreground px-3 py-1.5 rounded-lg"
                >
                  <Check class="h-2.5 w-2.5 text-success shrink-0" />
                  {{ field }}
                </span>
              </div>
            </div>

            <Separator />

            <!-- Additional Fields (editable) -->
            <div>
              <div class="flex items-center gap-2 mb-2.5">
                <h3 class="text-xs font-medium text-accent-foreground">Additional Fields</h3>
                <Badge variant="secondary" class="text-[10px] uppercase tracking-widest">
                  Per-gate
                </Badge>
              </div>
              <p class="text-xs text-dim mb-3">
                Add extra header or body fields to redact for this gate. Merged with defaults at
                runtime.
              </p>
              <FieldListEditor
                :fields="redactedAdditionalFields"
                placeholder="e.g. headers.X-Internal-Token, body.user.password"
                :disabled="saving"
                @add="redactedAdditionalFields.push($event)"
                @remove="redactedAdditionalFields.splice($event, 1)"
              />
            </div>
          </div>
        </div>
        <!-- Section: Diff Configuration -->
        <div class="bg-card border border-border rounded-xl overflow-hidden">
          <div class="px-5 py-4 border-b border-border/50">
            <div class="flex items-center gap-2 mb-1">
              <GitCompareArrows class="h-4 w-4 text-info" />
              <h2 class="text-sm font-semibold tracking-tight">Diff Configuration</h2>
            </div>
            <p class="text-xs text-dim leading-relaxed">
              Control how live vs. shadow JSON responses are compared. Fields use
              <span class="text-info">gjson path notation</span>.
            </p>
          </div>

          <div class="p-5 space-y-5">
            <!-- Ignored Fields -->
            <div>
              <h3 class="text-xs font-medium text-accent-foreground mb-2.5">Ignored Fields</h3>
              <p class="text-xs text-dim mb-3">
                Fields excluded from diff computation. Use for volatile values like timestamps.
              </p>
              <FieldListEditor
                :fields="diffIgnoredFields"
                placeholder="e.g. timestamp, request_id"
                :disabled="saving"
                @add="diffIgnoredFields.push($event)"
                @remove="diffIgnoredFields.splice($event, 1)"
              />
            </div>

            <Separator />

            <!-- Included Fields -->
            <div>
              <div class="flex items-center gap-2 mb-2.5">
                <h3 class="text-xs font-medium text-accent-foreground">Included Fields</h3>
                <Badge variant="secondary" class="text-[10px] uppercase tracking-widest">
                  Whitelist
                </Badge>
              </div>
              <p class="text-xs text-dim mb-3">
                When set, <em>only</em> these fields are compared (ignored fields still apply).
              </p>
              <FieldListEditor
                :fields="diffIncludedFields"
                placeholder="e.g. body.status, body.data"
                :disabled="saving"
                @add="diffIncludedFields.push($event)"
                @remove="diffIncludedFields.splice($event, 1)"
              />
              <div
                v-if="diffIncludedFields.length === 0"
                class="flex items-center gap-3 px-3 py-3 bg-background/60 border border-border/30 rounded-lg mt-3"
              >
                <Info class="h-3.5 w-3.5 text-dim shrink-0" />
                <span class="text-xs text-dim">
                  No included fields configured — all fields are compared.
                </span>
              </div>
            </div>

            <Separator />

            <!-- Float Tolerance -->
            <div>
              <h3 class="text-xs font-medium text-accent-foreground mb-2.5">Float Tolerance</h3>
              <p class="text-xs text-dim mb-3">
                Absolute tolerance for floating-point comparisons. Values within this tolerance are
                treated as equal.
              </p>
              <div class="flex items-center gap-3">
                <Input
                  v-model="floatTolerance"
                  type="number"
                  step="any"
                  min="0"
                  placeholder="0.001"
                  class="w-[180px] font-mono text-xs"
                  :disabled="saving"
                />
                <span class="text-xs text-dim">0 = exact comparison</span>
              </div>
            </div>

            <Separator />

            <!-- Array Order -->
            <div>
              <h3 class="text-xs font-medium text-accent-foreground mb-2.5">Array Order</h3>
              <p class="text-xs text-dim mb-3">
                Arrays are sorted before diffing so reordered elements aren't flagged as changes.
              </p>
              <div class="flex items-center gap-3">
                <Switch v-model="sortArrays" :disabled="saving" />
                <span class="text-xs text-muted-foreground">Ignore array element order</span>
              </div>
              <div
                class="flex items-start gap-3 px-3 py-3 bg-background/60 border border-border/30 rounded-lg mt-3"
              >
                <Info class="h-3.5 w-3.5 text-dim shrink-0 mt-0.5" />
                <span class="text-xs text-dim leading-relaxed">
                  When enabled, arrays are recursively sorted in both responses before comparison.
                  Reported diff indices reflect sorted order, not original order. Changes to this
                  setting do not reprocess past diffs.
                </span>
              </div>
            </div>
          </div>
        </div>

        <!-- Section: Danger Zone -->
        <div class="bg-card border border-destructive/20 rounded-xl overflow-hidden">
          <div class="px-5 py-4 border-b border-destructive/10">
            <div class="flex items-center gap-2">
              <TriangleAlert class="h-4 w-4 text-destructive" />
              <h2 class="text-sm font-semibold tracking-tight text-destructive">Danger Zone</h2>
            </div>
          </div>
          <div class="p-5 flex items-center justify-between">
            <div>
              <h3 class="text-xs font-medium text-foreground mb-0.5">Delete this gate</h3>
              <p class="text-xs text-dim">
                Permanently remove this gate and all its captured requests. This action cannot be
                undone.
              </p>
            </div>
            <AlertDialog>
              <AlertDialogTrigger as-child>
                <Button variant="outline" class="gap-1.5 text-destructive border-destructive/30">
                  <Trash2 class="h-3.5 w-3.5" />
                  Delete Gate
                </Button>
              </AlertDialogTrigger>
              <AlertDialogContent>
                <AlertDialogHeader>
                  <AlertDialogTitle>Delete gate</AlertDialogTitle>
                  <AlertDialogDescription>
                    This will permanently delete
                    <strong>{{ gate.name }}</strong>
                    and all its captured requests. This action cannot be undone.
                  </AlertDialogDescription>
                </AlertDialogHeader>
                <AlertDialogFooter>
                  <AlertDialogCancel>Cancel</AlertDialogCancel>
                  <AlertDialogAction
                    class="bg-destructive text-destructive-foreground hover:bg-destructive/90"
                    :disabled="deleting"
                    @click="handleDelete"
                  >
                    {{ deleting ? 'Deleting...' : 'Delete' }}
                  </AlertDialogAction>
                </AlertDialogFooter>
              </AlertDialogContent>
            </AlertDialog>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>
