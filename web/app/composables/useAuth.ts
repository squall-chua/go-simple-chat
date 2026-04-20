import { ref, computed, reactive, toRefs } from 'vue'

interface AuthState {
  userId: string | null
  username: string | null
  token: string | null
  isAuthenticated: boolean
}

const state = reactive<AuthState>({
  userId: null,
  username: null,
  token: null,
  isAuthenticated: false
})

export const useAuth = () => {
  const config = useRuntimeConfig()
  const { loadIdentity, saveIdentity, clearIdentity } = useIndexedDB()
  const { showError } = useToast()

  const login = async (cert: string, key: string) => {
    try {
      const response = await fetch(`${config.public.apiBase}/api/session`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ cert })
      })

      if (!response.ok) {
        const errorText = await response.text()
        throw new Error(errorText || 'Authentication failed')
      }

      const data = await response.json()
      
      state.token = data.token
      state.userId = data.userId
      state.username = data.username
      state.isAuthenticated = true

      // Set cookie for WebSocket proxy fallback
      document.cookie = `x-session-token=${data.token}; path=/; SameSite=Lax`

      // Save to IndexedDB for persistence
      await saveIdentity(cert, key)
      
      return true
    } catch (err: any) {
      showError(err.message)
      return false
    }
  }

  const logout = async () => {
    state.token = null
    state.userId = null
    state.username = null
    state.isAuthenticated = false
    await clearIdentity()
    navigateTo('/')
  }

  const restoreSession = async () => {
    const identity = await loadIdentity()
    if (identity) {
      return await login(identity.cert, identity.key)
    }
    return false
  }

  return {
    ...toRefs(state),
    login,
    logout,
    restoreSession
  }
}
