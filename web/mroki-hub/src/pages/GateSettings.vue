<script setup lang="ts">
import { ref, computed, watch, onMounted, onUnmounted } from 'vue'
import { useRoute, useRouter, onBeforeRouteLeave } from 'vue-router'
import { useQuery } from '@tanstack/vue-query'
import { gateQuery, useUpdateGateMutation, useDeleteGateMutation } from '@/api'
import type { Gate } from '@/api'
import { useAppConfig } from '@/composables/use-app-config'
import { parseGoDuration, humanizeGoDuration, normalizeToGoDuration } from '@/lib/duration'
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
  Clock,
} from 'lucide-vue-next'

const route = useRoute()
const router = useRouter()

// Update and delete flow through mutation composables that own cache
// invalidation on success; this page keeps only form/validation/UI state.
const updateMutation = useUpdateGateMutation()
const deleteMutation = useDeleteGateMutation()
// The floor is best-effort guidance; useAppConfig fetches it once per session
// and a failed load simply degrades the copy without blocking editing.
const { config: appConfig } = useAppConfig()

// The global retention floor, shown so users can enter a valid override without
// guessing. Null until the server config has loaded.
const globalRetention = computed(() => appConfig.value?.retention ?? null)

// Human-friendly rendering of the floor for display copy (e.g. "30d (720h0m0s)"
// for round day counts). The raw value stays authoritative for validation.
const globalRetentionLabel = computed(() =>
  globalRetention.value ? humanizeGoDuration(globalRetention.value) : null
)

const gateId = computed(() => route.params.id as string)

// Gate reads flow through the shared query cache (keyed by id), so the detail is
// reused across GateDetail / RequestDetail without a bespoke in-memory cache.
const gateQueryResult = useQuery(computed(() => gateQuery(gateId.value)))
const gate = computed<Gate | null>(() => gateQueryResult.data.value ?? null)
const loading = computed(() => gateQueryResult.isPending.value)
const error = computed(() =>
  gateQueryResult.isError.value
    ? (gateQueryResult.error.value?.message ?? 'Failed to load gate')
    : null
)
function loadGate() {
  gateQueryResult.refetch()
}

const saving = computed(() => updateMutation.isPending.value)
const saveError = ref<string | null>(null)
const saveSuccess = ref(false)
const deleting = computed(() => deleteMutation.isPending.value)
const deleteError = ref<string | null>(null)
const deleteDialogOpen = ref(false)
const leaveDialogOpen = ref(false)

// Form state
const name = ref('')
const redactedAdditionalFields = ref<string[]>([])
const diffIgnoredFields = ref<string[]>([])
const diffIncludedFields = ref<string[]>([])
const floatTolerance = ref('')
const sortArrays = ref(false)
const retention = ref('')

// Pristine snapshot of the form used to detect unsaved edits. Seeded with the
// empty-form snapshot so a gate that never loaded is not treated as dirty.
let pendingRoute: string | null = null
let bypassGuard = false

function snapshot() {
  return JSON.stringify({
    name: name.value.trim(),
    redacted: redactedAdditionalFields.value,
    ignored: diffIgnoredFields.value,
    included: diffIncludedFields.value,
    tol: floatTolerance.value,
    sort: sortArrays.value,
    retention: retention.value.trim(),
  })
}

const pristine = ref(snapshot())
const isDirty = computed(() => snapshot() !== pristine.value)

// Client-side retention validation. Mirrors the server rules so obvious
// mistakes fail before the round-trip; the API stays the source of truth. An
// empty value is valid (resets to the global floor).
const retentionError = computed<string | null>(() => {
  const value = retention.value.trim()
  if (value === '') return null

  // Accept day/week shorthands (e.g. 30d, 2w) by expanding to Go units first.
  const normalized = normalizeToGoDuration(value)
  const ns = normalized === null ? null : parseGoDuration(normalized)
  if (ns === null) {
    return 'Enter a duration like 168h, 30d, 2w, or 1h30m.'
  }
  if (ns <= 0) {
    return 'Retention must be positive. Keep-forever (0) is not supported.'
  }

  const floorNs = globalRetention.value ? parseGoDuration(globalRetention.value) : null
  if (floorNs !== null && ns < floorNs) {
    return `Must be at least the global retention (${globalRetentionLabel.value}).`
  }
  return null
})

// Whether the field has been interacted with. Client errors stay silent until
// blur so a value being typed does not flash an error mid-entry; Save gating
// still uses the live validity below.
const retentionTouched = ref(false)

// A retention rejection from the server (e.g. the client floor check was
// skipped because /config failed to load). Co-located with the field rather
// than surfaced in the generic top alert. Cleared on any edit.
const retentionServerError = ref<string | null>(null)

// The message actually rendered under the field: the server rejection (always
// shown), else the client error once the field has been touched. Blur and paste
// both mark the field touched, so a pasted-in invalid value surfaces its reason
// immediately (rather than leaving Save silently disabled) while mid-typing
// keystrokes stay quiet.
const retentionFieldError = computed<string | null>(() => {
  if (retentionServerError.value) return retentionServerError.value
  return retentionTouched.value ? retentionError.value : null
})

function onRetentionInput() {
  retentionServerError.value = null
}

// A paste is a complete value, not mid-typing — surface any error right away.
function onRetentionPaste() {
  retentionTouched.value = true
}

function onRetentionBlur() {
  retentionTouched.value = true
}

// Save is blocked while a known-invalid retention is entered.
const canSave = computed(() => isDirty.value && !saving.value && retentionError.value === null)

// Programmatic description for the retention input: the persistent guidance
// prose, plus the inline error when one is shown.
const retentionDescribedBy = computed(() =>
  retentionFieldError.value ? 'gate-retention-desc gate-retention-error' : 'gate-retention-desc'
)

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
  retention.value = g.retention ?? ''
  pristine.value = snapshot()
}

// Seed the form from the fetched gate. A background refetch must not clobber
// in-progress edits, so re-populating is skipped while the form is dirty; the
// unsaved-changes guard still protects navigation.
watch(
  () => gateQueryResult.data.value,
  (g) => {
    if (g && !isDirty.value) populateForm(g)
  },
  { immediate: true }
)

async function handleSave() {
  // Guard against a programmatic save with known-invalid input; the disabled
  // button already prevents this in the UI.
  if (retentionError.value !== null) return

  // Expand any day/week shorthand into Go units the API understands. Empty
  // stays empty (reset to the global floor). A non-empty value that fails to
  // normalize can only happen if the validity gate above was bypassed — treat
  // it as a validation error rather than silently resetting to the global floor.
  const trimmedRetention = retention.value.trim()
  let retentionPayload = ''
  if (trimmedRetention !== '') {
    const normalized = normalizeToGoDuration(retention.value)
    if (normalized === null) {
      retentionServerError.value = 'Enter a duration like 168h, 30d, 2w, or 1h30m.'
      retentionTouched.value = true
      return
    }
    retentionPayload = normalized
  }

  saveError.value = null
  retentionServerError.value = null
  saveSuccess.value = false

  try {
    await updateMutation.mutateAsync({
      id: gateId.value,
      payload: {
        name: name.value.trim(),
        diff_config: {
          ignored_fields: diffIgnoredFields.value,
          included_fields: diffIncludedFields.value,
          float_tolerance: floatTolerance.value ? parseFloat(floatTolerance.value) : 0,
          sort_arrays: sortArrays.value,
        },
        redacted_fields: redactedAdditionalFields.value,
        retention: retentionPayload,
      },
    })

    // The mutation invalidates the gate detail and list caches on success, so
    // every consumer (this page, GateDetail, RequestDetail) refetches the
    // canonical gate rather than reading a hand-written copy.
    pristine.value = snapshot()
    saveSuccess.value = true
    setTimeout(() => (saveSuccess.value = false), 3000)
  } catch (err) {
    const message = err instanceof Error ? err.message : 'Failed to update gate'
    // Co-locate retention rejections (the one case client validation can miss,
    // e.g. when the floor was unknown) with the field; everything else stays in
    // the top save alert.
    if (retention.value.trim() !== '' && /retention|duration|floor|at least/i.test(message)) {
      retentionServerError.value = message
      retentionTouched.value = true
    } else {
      saveError.value = message
    }
  }
}

async function handleDelete() {
  deleteError.value = null
  try {
    await deleteMutation.mutateAsync(gateId.value)
    // The mutation drops the deleted gate's detail and refreshes the list and
    // stats reads on success, so the gone gate never lingers in a stale view.
    deleteDialogOpen.value = false
    bypassGuard = true
    router.push('/gates')
  } catch (err) {
    deleteError.value = err instanceof Error ? err.message : 'Failed to delete gate'
  }
}

// Returns true when navigation may proceed immediately; otherwise opens the
// discard-changes confirmation dialog and returns false.
function guardLeave(target: string): boolean {
  if (!isDirty.value || bypassGuard) return true
  pendingRoute = target
  leaveDialogOpen.value = true
  return false
}

function goBack() {
  const target = `/gates/${gateId.value}`
  if (guardLeave(target)) router.push(target)
}

function confirmLeave() {
  bypassGuard = true
  leaveDialogOpen.value = false
  if (pendingRoute) router.push(pendingRoute)
}

onBeforeRouteLeave((to) => {
  if (!isDirty.value || bypassGuard) return true
  pendingRoute = to.fullPath
  leaveDialogOpen.value = true
  return false
})

function onBeforeUnload(e: BeforeUnloadEvent) {
  if (isDirty.value) {
    e.preventDefault()
    e.returnValue = ''
  }
}

onMounted(() => {
  window.addEventListener('beforeunload', onBeforeUnload)
})

onUnmounted(() => {
  window.removeEventListener('beforeunload', onBeforeUnload)
})
</script>

<template>
  <div class="max-w-6xl mx-auto px-6 py-6">
    <!-- Back link -->
    <button
      type="button"
      class="inline-flex items-center gap-1.5 text-xs text-muted-foreground hover:text-foreground transition-colors mb-5 cursor-pointer rounded-sm focus-visible:outline-none focus-visible:ring-[3px] focus-visible:ring-ring focus-visible:ring-offset-2 focus-visible:ring-offset-background"
      @click="goBack"
    >
      <ChevronLeft class="h-3.5 w-3.5" aria-hidden="true" />
      Back to Gate
    </button>

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
      <div class="flex items-center justify-between mb-6 gap-4">
        <div class="min-w-0">
          <h1 class="text-xl font-semibold tracking-tight mb-1">Gate Settings</h1>
          <div class="flex items-center gap-2 text-sm text-dim min-w-0">
            <span class="font-medium text-muted-foreground truncate min-w-0">{{ gate.name }}</span>
            <span class="text-dim/50 shrink-0">&middot;</span>
            <code class="text-xs font-mono shrink-0">{{ gate.id }}</code>
          </div>
        </div>
        <Button :disabled="!canSave" class="gap-2 shrink-0" @click="handleSave">
          <Save v-if="!saveSuccess" class="h-3.5 w-3.5" aria-hidden="true" />
          <Check v-else class="h-3.5 w-3.5" aria-hidden="true" />
          {{ saving ? 'Saving...' : saveSuccess ? 'Saved' : 'Save Changes' }}
        </Button>
      </div>

      <!-- Save feedback. Retention is validated client-side before save, so
           this channel carries non-field failures (name conflicts, network). -->
      <Alert v-if="saveError" variant="destructive" role="alert" aria-live="assertive" class="mb-6">
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
              <Lock class="h-4 w-4 text-warning" aria-hidden="true" />
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
                  <Check class="h-2.5 w-2.5 text-success shrink-0" aria-hidden="true" />
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
        <!-- Section: Data Retention -->
        <div class="bg-card border border-border rounded-xl overflow-hidden">
          <div class="px-5 py-4 border-b border-border/50">
            <div class="flex items-center gap-2 mb-1">
              <Clock class="h-4 w-4 text-info" aria-hidden="true" />
              <h2 class="text-sm font-semibold tracking-tight">Data Retention</h2>
            </div>
            <p id="gate-retention-desc" class="text-xs text-dim leading-relaxed">
              How long captured requests for this gate are kept before the cleanup job removes them.
              Leave empty to use the global retention<template v-if="globalRetentionLabel">
                of
                <code class="text-xs font-mono bg-muted px-1 py-0.5 rounded">{{
                  globalRetentionLabel
                }}</code></template
              >. A custom value can be a Go duration (e.g.
              <code class="text-xs font-mono bg-muted px-1 py-0.5 rounded">168h</code>) or a
              day/week shorthand (e.g.
              <code class="text-xs font-mono bg-muted px-1 py-0.5 rounded">30d</code>,
              <code class="text-xs font-mono bg-muted px-1 py-0.5 rounded">2w</code>), and must be
              at least the global retention<template v-if="globalRetentionLabel">
                (<code class="text-xs font-mono bg-muted px-1 py-0.5 rounded">{{
                  globalRetentionLabel
                }}</code
                >)</template
              >. Keep-forever (<code class="text-xs font-mono bg-muted px-1 py-0.5 rounded">0</code
              >) is not supported.
            </p>
          </div>

          <div class="p-5">
            <div class="space-y-2 max-w-sm">
              <Label for="gate-retention">Retention period</Label>
              <Input
                id="gate-retention"
                v-model="retention"
                type="text"
                :placeholder="globalRetention ? `\u2265 ${globalRetention}` : 'Global default'"
                class="font-mono text-xs"
                :disabled="saving"
                :aria-invalid="retentionFieldError !== null"
                :aria-describedby="retentionDescribedBy"
                @input="onRetentionInput"
                @paste="onRetentionPaste"
                @blur="onRetentionBlur"
              />
              <p
                v-if="retentionFieldError"
                id="gate-retention-error"
                role="alert"
                class="flex items-center gap-1.5 text-xs text-destructive"
              >
                <TriangleAlert class="h-3.5 w-3.5 shrink-0" aria-hidden="true" />
                {{ retentionFieldError }}
              </p>
            </div>
            <div
              v-if="!retention.trim()"
              class="flex items-center gap-3 px-3 py-3 bg-background/60 border border-border/30 rounded-lg mt-3"
            >
              <Info class="h-3.5 w-3.5 text-dim shrink-0" aria-hidden="true" />
              <span class="text-xs text-dim">
                Using the global retention<template v-if="globalRetentionLabel">
                  of
                  <code class="text-xs font-mono bg-background px-1 py-0.5 rounded">{{
                    globalRetentionLabel
                  }}</code></template
                >. Requests are pruned on the global schedule.
              </span>
            </div>
          </div>
        </div>

        <!-- Section: Diff Configuration -->
        <div class="bg-card border border-border rounded-xl overflow-hidden">
          <div class="px-5 py-4 border-b border-border/50">
            <div class="flex items-center gap-2 mb-1">
              <GitCompareArrows class="h-4 w-4 text-info" aria-hidden="true" />
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
                <Info class="h-3.5 w-3.5 text-dim shrink-0" aria-hidden="true" />
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
                <Info class="h-3.5 w-3.5 text-dim shrink-0 mt-0.5" aria-hidden="true" />
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
              <TriangleAlert class="h-4 w-4 text-destructive" aria-hidden="true" />
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
            <AlertDialog v-model:open="deleteDialogOpen">
              <AlertDialogTrigger as-child>
                <Button variant="outline" class="gap-1.5 text-destructive border-destructive/30">
                  <Trash2 class="h-3.5 w-3.5" aria-hidden="true" />
                  Delete gate
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
                <Alert v-if="deleteError" variant="destructive" class="mt-2">
                  <AlertDescription>{{ deleteError }}</AlertDescription>
                </Alert>
                <AlertDialogFooter>
                  <AlertDialogCancel>Cancel</AlertDialogCancel>
                  <AlertDialogAction
                    class="bg-destructive text-destructive-foreground hover:bg-destructive/90"
                    :disabled="deleting"
                    @click="handleDelete"
                  >
                    {{ deleting ? 'Deleting gate...' : 'Delete gate' }}
                  </AlertDialogAction>
                </AlertDialogFooter>
              </AlertDialogContent>
            </AlertDialog>
          </div>
        </div>
      </div>

      <!-- Unsaved-changes leave guard -->
      <AlertDialog v-model:open="leaveDialogOpen">
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Discard unsaved changes?</AlertDialogTitle>
            <AlertDialogDescription>
              You have unsaved changes to this gate's settings. Leaving now will discard them.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Keep editing</AlertDialogCancel>
            <AlertDialogAction
              class="bg-destructive text-destructive-foreground hover:bg-destructive/90"
              @click="confirmLeave"
            >
              Discard changes
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  </div>
</template>
