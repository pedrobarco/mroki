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

// Format headers for display
const formatHeaders = (headers: Record<string, string[]>): string => {
  return JSON.stringify(headers, null, 2)
}

const liveHeaders = computed(() => formatHeaders(liveData.value.headers))
const shadowHeaders = computed(() => formatHeaders(shadowData.value.headers))

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
  <div class="space-y-6">
    <!-- Response Status Comparison -->
    <div class="grid grid-cols-2 gap-4">
      <div class="rounded-lg border p-4">
        <h3 class="text-sm font-semibold text-muted-foreground mb-2">Live Response</h3>
        <div class="space-y-2">
          <div class="flex items-center gap-2">
            <span class="text-sm font-medium">Status:</span>
            <span
              class="inline-flex items-center rounded-full px-2 py-1 text-xs font-medium"
              :class="{
                'bg-green-100 text-green-700':
                  liveResponse.status_code >= 200 && liveResponse.status_code < 300,
                'bg-yellow-100 text-yellow-700':
                  liveResponse.status_code >= 300 && liveResponse.status_code < 400,
                'bg-red-100 text-red-700': liveResponse.status_code >= 400,
              }"
            >
              {{ liveResponse.status_code }}
            </span>
          </div>
        </div>
      </div>

      <div class="rounded-lg border p-4">
        <h3 class="text-sm font-semibold text-muted-foreground mb-2">Shadow Response</h3>
        <div class="space-y-2">
          <div class="flex items-center gap-2">
            <span class="text-sm font-medium">Status:</span>
            <span
              class="inline-flex items-center rounded-full px-2 py-1 text-xs font-medium"
              :class="{
                'bg-green-100 text-green-700':
                  shadowResponse.status_code >= 200 && shadowResponse.status_code < 300,
                'bg-yellow-100 text-yellow-700':
                  shadowResponse.status_code >= 300 && shadowResponse.status_code < 400,
                'bg-red-100 text-red-700': shadowResponse.status_code >= 400,
              }"
            >
              {{ shadowResponse.status_code }}
            </span>
          </div>
        </div>
      </div>
    </div>

    <!-- Headers Comparison -->
    <div class="rounded-lg border">
      <div class="border-b bg-muted/50 px-4 py-2">
        <h3 class="text-sm font-semibold">Headers Comparison</h3>
      </div>
      <div class="p-4">
        <CodeDiff
          :old-string="liveHeaders"
          :new-string="shadowHeaders"
          language="json"
          output-format="side-by-side"
          :filename="'Live Headers'"
          :new-filename="'Shadow Headers'"
        />
      </div>
    </div>

    <!-- Side-by-Side Body Comparison -->
    <div class="rounded-lg border">
      <div class="border-b bg-muted/50 px-4 py-2">
        <h3 class="text-sm font-semibold">Response Body Comparison</h3>
        <p v-if="isBinary" class="text-xs text-muted-foreground mt-1">
          ⚠️ Binary content detected - cannot display diff
        </p>
        <p v-else class="text-xs text-muted-foreground mt-1">
          Format: {{ language.toUpperCase() }}
        </p>
      </div>
      <div class="p-4">
        <CodeDiff
          v-if="!isBinary"
          :old-string="liveBody"
          :new-string="shadowBody"
          :language="language"
          output-format="side-by-side"
          :filename="'Live Response'"
          :new-filename="'Shadow Response'"
        />
        <div v-else class="text-center py-8 text-muted-foreground">
          <p class="text-sm">Binary content cannot be displayed in diff viewer.</p>
          <p class="text-xs mt-2">Consider downloading the responses to inspect them.</p>
        </div>
      </div>
    </div>
  </div>
</template>
