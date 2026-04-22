import { ref, computed, reactive, toRefs } from 'vue'
import { signMessage } from '@/utils/crypto'

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

  const login = async (cert: string, key: string, silent = false) => {
    try {
      // 1. Get Challenge
      const challengeResponse = await fetch(`${config.public.apiBase}/api/session/challenge`, {
        credentials: 'include'
      })
      if (!challengeResponse.ok) {
        throw new Error('Failed to get login challenge')
      }
      const { nonce } = await challengeResponse.json()

      // 2. Sign Challenge (Proof of Possession)
      const signature = await signMessage(nonce, key)

      // 3. Submit
      const response = await fetch(`${config.public.apiBase}/api/session`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ cert, nonce, signature }),
        credentials: 'include'
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

      // Save to IndexedDB for persistence
      await saveIdentity(cert, key)

      return true
    } catch (err: any) {
      if (!silent) {
        showError(err.message)
      }
      return false
    }
  }

  const logout = async () => {
    // Call server to clear cookie
    try {
      await fetch(`${config.public.apiBase}/api/session`, { 
        method: 'DELETE',
        credentials: 'include'
      })
    } catch (e) {
      console.warn('Silent logout failure', e)
    }

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
      return await login(identity.cert, identity.key, true)
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
