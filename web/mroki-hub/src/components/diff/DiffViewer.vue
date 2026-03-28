<script setup lang="ts">
import { computed } from 'vue'
import { CodeDiff } from 'v-code-diff'
import type { Response, PatchOp } from '@/api'

interface Props {
  liveResponse: Response
  shadowResponse: Response
  diffContent: PatchOp[]
}

const props = defineProps<Props>()

// Parse the diff content to extract the structured comparison
// The diff content is a go-cmp text format, but we'll construct the comparison
// from the actual response objects instead
interface ComparisonData {
  statusCode: number
  headers: Record<string, string[]>
  body: string
}

// Build comparison data from response
const buildComparisonData = (response: Response): ComparisonData => {
  return {
    statusCode: response.status_code,
    headers: response.headers,
    body: response.body,
  }
}

const liveData = computed(() => buildComparisonData(props.liveResponse))
const shadowData = computed(() => buildComparisonData(props.shadowResponse))

// Decode base64 body
const decodeBody = (body: string): string => {
  try {
    return atob(body)
  } catch (error) {
    console.error('Failed to decode base64 body:', error)
    return body // Return as-is if decode fails
  }
}

// Detect content type from decoded body
const detectContentType = (decoded: string): 'json' | 'xml' | 'html' | 'plaintext' | 'binary' => {
  // Try JSON first
  try {
    JSON.parse(decoded)
    return 'json'
  } catch {
    // Not JSON
  }

  // Check for XML
  if (decoded.trim().startsWith('<?xml') || decoded.trim().startsWith('<')) {
    // Simple heuristic: if it has XML/HTML tags
    if (decoded.includes('<!DOCTYPE html') || decoded.includes('<html')) {
      return 'html'
    }
    return 'xml'
  }

  // Check if it's printable text
  // If it contains mostly printable ASCII characters, treat as plaintext
  const printableRatio = (decoded.match(/[\x20-\x7E\n\r\t]/g) || []).length / decoded.length
  if (printableRatio > 0.95) {
    return 'plaintext'
  }

  // Otherwise, it's binary
  return 'binary'
}

// Format body based on content type
const formatBody = (body: string): { formatted: string; type: string; isBinary: boolean } => {
  const decoded = decodeBody(body)
  const contentType = detectContentType(decoded)

  if (contentType === 'binary') {
    return {
      formatted: '[Binary content - cannot display]',
      type: 'plaintext',
      isBinary: true,
    }
  }

  if (contentType === 'json') {
    try {
      const parsed = JSON.parse(decoded)
      return {
        formatted: JSON.stringify(parsed, null, 2),
        type: 'json',
        isBinary: false,
      }
    } catch {
      // Fallback to plaintext if JSON parsing fails
      return {
        formatted: decoded,
        type: 'plaintext',
        isBinary: false,
      }
    }
  }

  // For XML, HTML, and plaintext, return as-is
  return {
    formatted: decoded,
    type: contentType,
    isBinary: false,
  }
}

const liveFormatted = computed(() => formatBody(liveData.value.body))
const shadowFormatted = computed(() => formatBody(shadowData.value.body))

const liveBody = computed(() => liveFormatted.value.formatted)
const shadowBody = computed(() => shadowFormatted.value.formatted)

// Determine language for syntax highlighting
const language = computed(() => {
  // Use the detected type from live response
  return liveFormatted.value.type
})

// Check if content is binary
const isBinary = computed(() => {
  return liveFormatted.value.isBinary || shadowFormatted.value.isBinary
})
</script>

<template>
  <div>
    <!-- Response Body Comparison -->
    <div class="bg-card border border-border rounded-xl overflow-hidden">
      <!-- Card Header -->
      <div class="flex items-center justify-between px-5 py-3.5 border-b border-border">
        <div class="flex items-center gap-2">
          <h3 class="text-sm font-semibold">Response Comparison</h3>
          <span class="text-xs text-dim bg-accent px-2 py-0.5 rounded-md font-mono">
            {{ language }}
          </span>
        </div>
        <div class="flex items-center gap-3 text-xs">
          <div class="flex items-center gap-1.5 text-dim">
            <span class="w-2.5 h-2.5 rounded-sm bg-red-500/10 border border-red-500/30" />
            Removed
          </div>
          <div class="flex items-center gap-1.5 text-dim">
            <span class="w-2.5 h-2.5 rounded-sm bg-green-500/10 border border-green-500/30" />
            Added
          </div>
          <div class="flex items-center gap-1.5 text-dim">
            <span class="w-2.5 h-2.5 rounded-sm bg-amber-500/10 border border-amber-500/30" />
            Changed
          </div>
        </div>
      </div>

      <!-- Column Headers -->
      <div class="grid grid-cols-2 border-b border-border">
        <div class="flex items-center justify-between px-4 py-2.5 border-r border-border">
          <div class="flex items-center gap-1.5 text-xs uppercase tracking-widest text-dim">
            <span class="w-1.5 h-1.5 rounded-full bg-success" />
            Live Response
          </div>
          <span class="text-xs font-mono text-success">{{ liveResponse.status_code }} OK</span>
        </div>
        <div class="flex items-center justify-between px-4 py-2.5">
          <div class="flex items-center gap-1.5 text-xs uppercase tracking-widest text-dim">
            <span class="w-1.5 h-1.5 rounded-full bg-info" />
            Shadow Response
          </div>
          <span class="text-xs font-mono text-info">{{ shadowResponse.status_code }} OK</span>
        </div>
      </div>

      <!-- Diff Content -->
      <div v-if="!isBinary" class="p-4">
        <CodeDiff
          :old-string="liveBody"
          :new-string="shadowBody"
          :language="language"
          output-format="side-by-side"
          :filename="'Live Response'"
          :new-filename="'Shadow Response'"
        />
      </div>
      <div v-else class="text-center py-8 text-muted-foreground">
        <p class="text-sm">Binary content cannot be displayed in diff viewer.</p>
        <p class="text-xs mt-2">Consider downloading the responses to inspect them.</p>
      </div>
    </div>
  </div>
</template>
