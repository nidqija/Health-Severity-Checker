export interface AppState {
  inputText: string
  loading: boolean
  result: { score: number; advice: string } | null
  error: string | null
}
