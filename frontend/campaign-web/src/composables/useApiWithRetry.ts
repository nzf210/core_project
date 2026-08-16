import { ref } from 'vue'

export interface ApiState<T> {
  data: T | null
  loading: boolean
  error: string | null
}

export function useApiWithRetry<T>() {
  const state = ref<ApiState<T>>({
    data: null,
    loading: false,
    error: null
  })

  async function execute(
    apiCall: () => Promise<Response>,
    options: {
      onSuccess?: (data: T) => void
      onError?: (error: Error) => void
      silent?: boolean
    } = {}
  ): Promise<T | null> {
    state.value.loading = true
    state.value.error = null

    try {
      const response = await apiCall()
      const result = await response.json()

      if (!response.ok) {
        throw new Error(result.message || `HTTP ${response.status}`)
      }

      if (result.success && result.data) {
        state.value.data = result.data
        options.onSuccess?.(result.data)
        return result.data
      }

      throw new Error(result.message || 'Request failed')
    } catch (error) {
      const errorMessage = error instanceof Error ? error.message : 'Network error'

      if (!options.silent) {
        state.value.error = errorMessage
      }

      options.onError?.(error as Error)
      return null
    } finally {
      state.value.loading = false
    }
  }

  function reset() {
    state.value.data = null
    state.value.error = null
    state.value.loading = false
  }

  return {
    state,
    execute,
    reset
  }
}
