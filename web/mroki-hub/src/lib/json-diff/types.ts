export type TokenType = 'key' | 'string' | 'number' | 'boolean' | 'null' | 'bracket' | 'punctuation'

export interface Token {
  text: string
  type: TokenType
}

export type LineType =
  | 'normal'
  | 'added'
  | 'removed'
  | 'replaced-old'
  | 'replaced-new'
  | 'collapsed'

export interface DiffLine {
  tokens: Token[]
  type: LineType
  indent: number
  path: string
  /** Present only when type === 'collapsed'. The raw value to expand on demand. */
  collapsedValue?: unknown
  /** Present only when type === 'collapsed'. Number of lines that would be rendered if expanded. */
  collapsedLineCount?: number
  /** Present only when type === 'collapsed'. Whether a trailing comma is needed after expansion. */
  collapsedTrailingComma?: boolean
}

export interface SplitRow {
  left: DiffLine | null
  right: DiffLine | null
}
