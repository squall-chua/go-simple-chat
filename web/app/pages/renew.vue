<script setup lang="ts">
import { ref } from 'vue'
import { RefreshCw, User, Key, ArrowRight, CheckCircle2, ShieldAlert, Download } from 'lucide-vue-next'
import { signMessage } from '@/utils/crypto'

const { login } = useAuth()
const { showSuccess, showError } = useToast()
const config = useRuntimeConfig()

const username = ref('')
const keyFile = ref<File | null>(null)
const isLoading = ref(false)
const renewedIdentity = ref<{ cert: string, key: string } | null>(null)

const onFileChange = (e: Event) => {
  const target = e.target as HTMLInputElement
  if (target.files && target.files[0]) {
    keyFile.value = target.files[0]
  }
}

const readFile = (file: File): Promise<string> => {
  return new Promise((resolve) => {
    const reader = new FileReader()
    reader.onload = (e) => resolve(e.target?.result as string)
    reader.readAsText(file)
  })
}

const handleRenew = async () => {
  if (!username.value || !keyFile.value) {
    showError('Username and private key are required')
    return
  }

  isLoading.value = true
  try {
    const keyPEM = await readFile(keyFile.value)
    
    // 1. Get Challenge
    const challengeRes = await fetch(`${config.public.apiBase}/v1/auth/challenge`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ username: username.value })
    })

    if (!challengeRes.ok) {
      const txt = await challengeRes.text()
      throw new Error(txt || 'Failed to get challenge')
    }
    const challengeData = await challengeRes.json()
    const { user_id, nonce } = challengeData

    // 2. Sign Challenge
    const challengeStr = "RENEW_CERT:" + nonce
    const sigB64 = await signMessage(challengeStr, keyPEM)

    // 3. Submit Renewal
    const renewRes = await fetch(`${config.public.apiBase}/v1/auth/renew`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        user_id: user_id,
        signature: sigB64
      })
    })

    if (!renewRes.ok) {
      const txt = await renewRes.text()
      throw new Error(txt || 'Renewal failed')
    }

    const data = await renewRes.json()
    
    // 4. Prepare identity
    const decodeB64 = (str: string) => {
      if (!str) return ''
      try { return atob(str.replace(/\s/g, '')) } catch (e) { return str }
    }
    
    renewedIdentity.value = {
      cert: decodeB64(data.certificate),
      key: keyPEM // Keep the existing private key
    }

    showSuccess('Certificate renewed successfully!')
  } catch (err: any) {
    showError(err.message)
    console.error('Renewal error:', err)
  } finally {
    isLoading.value = false
  }
}

const downloadIdentity = () => {
  if (!renewedIdentity.value) return
  
  const blob = new Blob([renewedIdentity.value.cert], { type: 'text/plain' })
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = `${username.value}.crt`
  a.click()
  URL.revokeObjectURL(url)
}
</script>

<template>
  <div class="min-h-screen flex items-center justify-center p-6 bg-background relative overflow-hidden">
    <div class="absolute top-[-10%] left-[-10%] w-[40%] h-[40%] bg-cta/10 rounded-full blur-[120px]"></div>

    <div class="w-full max-w-md glass-card space-y-8 animate-in fade-in slide-in-from-bottom-4 duration-700">
      <div class="text-center space-y-2">
        <h1 class="text-3xl font-bold font-heading tracking-tight text-white flex items-center justify-center gap-3">
          <RefreshCw class="w-8 h-8 text-cta" />
          Renew Identity
        </h1>
        <p class="text-slate-400 text-sm">Expired certificate? Use your key to renew it.</p>
      </div>

      <div class="p-4 rounded-xl bg-cta/5 border border-cta/20 flex gap-3">
        <ShieldAlert class="w-5 h-5 text-cta shrink-0" />
        <p class="text-[11px] text-slate-400 leading-relaxed">
          Identity renewal requires your **original private key** (.key) and the **username** you registered with.
        </p>
      </div>

      <div class="space-y-6">
        <div class="space-y-2">
          <label class="text-xs font-bold uppercase tracking-widest text-slate-500 px-1">Registered Username</label>
          <div class="relative group">
            <User class="absolute left-4 top-1/2 -translate-y-1/2 w-5 h-5 text-slate-500 group-focus-within:text-cta transition-colors" />
            <input 
              v-model="username"
              type="text" 
              placeholder="e.g. Satoshi"
              class="w-full bg-white/5 border border-white/10 rounded-xl py-4 pl-12 pr-4 text-white focus:outline-none focus:border-cta/50 focus:bg-white/10 transition-all"
            />
          </div>
        </div>

        <div class="space-y-2">
          <label class="text-xs font-bold uppercase tracking-widest text-slate-500 px-1">Private Key (.key)</label>
          <div class="relative group">
            <input 
              type="file" 
              accept=".key,.pem"
              @change="onFileChange"
              class="absolute inset-0 w-full h-full opacity-0 cursor-pointer z-10"
            />
            <div :class="[
              'flex items-center gap-3 p-4 rounded-xl border-2 border-dashed transition-all',
              keyFile ? 'border-cta/50 bg-cta/5' : 'border-white/10 hover:border-white/20 bg-white/5'
            ]">
              <div :class="[
                'w-10 h-10 rounded-lg flex items-center justify-center transition-colors',
                keyFile ? 'bg-cta text-white' : 'bg-white/5 text-slate-500'
              ]">
                <Key class="w-5 h-5" />
              </div>
              <div class="flex-1 overflow-hidden">
                <p class="text-sm font-medium truncate" :class="keyFile ? 'text-white' : 'text-slate-400'">
                  {{ keyFile ? keyFile.name : 'Select private key file' }}
                </p>
              </div>
            </div>
          </div>
        </div>

        <button 
          @click="handleRenew"
          :disabled="isLoading || !username || !keyFile"
          class="w-full bg-cta hover:bg-cta/90 text-white font-bold rounded-xl py-4 flex items-center justify-center gap-2 text-lg disabled:opacity-50 disabled:cursor-not-allowed group transition-all"
        >
          <span v-if="isLoading" class="w-5 h-5 border-2 border-white/20 border-t-white rounded-full animate-spin"></span>
          <template v-else>
            Renew Certificate
            <ArrowRight class="w-5 h-5 group-hover:translate-x-1 transition-transform" />
          </template>
        </button>
      </div>

      <div v-if="renewedIdentity" class="space-y-6 pt-6 border-t border-white/10 animate-in fade-in zoom-in-95 duration-500">
        <div class="p-4 rounded-xl bg-green-500/10 border border-green-500/20 flex gap-4">
          <CheckCircle2 class="w-6 h-6 text-green-400 shrink-0" />
          <div class="space-y-1">
            <h3 class="text-white font-bold">Renewal Successful</h3>
            <p class="text-xs text-slate-400 leading-relaxed">
              Your identity has been refreshed. Download your new certificate to stay connected.
            </p>
          </div>
        </div>

        <button 
          @click="downloadIdentity"
          class="w-full bg-white text-black hover:bg-slate-200 font-bold rounded-xl py-4 flex items-center justify-center gap-2 text-lg transition-all"
        >
          <Download class="w-5 h-5" />
          Download .crt
        </button>

        <NuxtLink 
          to="/chat"
          class="w-full bg-cta hover:bg-cta/90 text-white font-medium rounded-xl py-4 flex items-center justify-center gap-2 transition-all"
        >
          Go to Chat
        </NuxtLink>
      </div>

      <div class="pt-4 border-t border-white/5">
        <NuxtLink to="/" class="text-sm text-slate-500 hover:text-white transition-colors flex items-center gap-2">
          <ArrowRight class="w-4 h-4 rotate-180" />
          Back to Launch
        </NuxtLink>
      </div>
    </div>
  </div>
</template>
