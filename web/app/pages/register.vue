<script setup lang="ts">
import { ref } from 'vue'
import { UserPlus, User, Shield, Download, ArrowRight, CheckCircle2 } from 'lucide-vue-next'
import nacl from 'tweetnacl'

const { login } = useAuth()
const { showSuccess, showError } = useToast()
const config = useRuntimeConfig()

const username = ref('')
const isLoading = ref(false)
const registeredIdentity = ref<{ cert: string, key: string } | null>(null)

const handleRegister = async () => {
  if (!username.value) {
    showError('Username is required')
    return
  }

  isLoading.value = true
  try {
    // 1. Generate local key pair using TweetNaCl
    const seed = window.crypto.getRandomValues(new Uint8Array(32))
    const keyPair = nacl.sign.keyPair.fromSeed(seed)
    const pubKey = keyPair.publicKey

    // Helper to wrap seed in PKCS#8 DER for Ed25519
    const getPrivateKeyPEM = (seedBytes: Uint8Array) => {
      const header = new Uint8Array([
        0x30, 0x2e, 0x02, 0x01, 0x00, 0x30, 0x05, 0x06, 0x03, 0x2b, 0x65, 0x70, 0x04, 0x22, 0x04, 0x20
      ])
      const pkcs8 = new Uint8Array(header.length + seedBytes.length)
      pkcs8.set(header)
      pkcs8.set(seedBytes, header.length)
      const b64 = btoa(String.fromCharCode(...pkcs8))
      return `-----BEGIN PRIVATE KEY-----\n${b64.match(/.{1,64}/g)?.join('\n')}\n-----END PRIVATE KEY-----`
    }
    
    const localKeyPEM = getPrivateKeyPEM(seed)
    
    // 2. Call Register API
    const response = await fetch(`${config.public.apiBase}/v1/register`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        username: username.value,
        public_key: btoa(String.fromCharCode(...pubKey))
      })
    })

    if (!response.ok) {
      const errorText = await response.text()
      throw new Error(errorText || 'Registration failed')
    }

    const data = await response.json()
    
    // 3. Prepare identity
    const decodeB64 = (str: string) => {
      if (!str) return ''
      try {
        return atob(str.replace(/\s/g, ''))
      } catch (e) {
        return str
      }
    }
    
    const certPEM = decodeB64(data.certificate)
    // Use server key if provided (CA-issued mode), otherwise use our local key
    const keyPEM = decodeB64(data.private_key) || localKeyPEM
    
    registeredIdentity.value = {
      cert: certPEM,
      key: keyPEM
    }

    showSuccess('Identity created successfully!')
  } catch (err: any) {
    showError(err.message)
  } finally {
    isLoading.value = false
  }
}

const downloadIdentity = () => {
  if (!registeredIdentity.value) return
  
  const zip = (filename: string, content: string) => {
    const blob = new Blob([content], { type: 'text/plain' })
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = filename
    a.click()
    URL.revokeObjectURL(url)
  }

  zip(`${username.value}.crt`, registeredIdentity.value.cert)
  zip(`${username.value}.key`, registeredIdentity.value.key)
}

const proceedToChat = async () => {
  if (!registeredIdentity.value) return
  
  const success = await login(registeredIdentity.value.cert, registeredIdentity.value.key)
  if (success) {
    navigateTo('/chat')
  }
}
</script>

<template>
  <div class="min-h-screen flex items-center justify-center p-6 bg-background relative overflow-hidden">
    <div class="absolute top-[-10%] right-[-10%] w-[40%] h-[40%] bg-primary/10 rounded-full blur-[120px]"></div>
    <div class="absolute bottom-[-10%] left-[-10%] w-[40%] h-[40%] bg-secondary/5 rounded-full blur-[120px]"></div>

    <div class="w-full max-w-md glass-card space-y-8 animate-in fade-in slide-in-from-bottom-4 duration-700">
      <div class="text-center space-y-2">
        <h1 class="text-3xl font-bold font-heading tracking-tight text-white flex items-center justify-center gap-3">
          <UserPlus class="w-8 h-8 text-primary" />
          Create Identity
        </h1>
        <p class="text-slate-400 text-sm">Register a new cryptographically signed persona</p>
      </div>

      <div v-if="!registeredIdentity" class="space-y-6">
        <div class="space-y-2">
          <label class="text-xs font-bold uppercase tracking-widest text-slate-500 px-1">Display Username</label>
          <div class="relative group">
            <User class="absolute left-4 top-1/2 -translate-y-1/2 w-5 h-5 text-slate-500 group-focus-within:text-primary transition-colors" />
            <input 
              v-model="username"
              type="text" 
              placeholder="e.g. Satoshi"
              class="w-full bg-white/5 border border-white/10 rounded-xl py-4 pl-12 pr-4 text-white focus:outline-none focus:border-primary/50 focus:bg-white/10 transition-all"
              @keyup.enter="handleRegister"
            />
          </div>
          <p class="text-[10px] text-slate-500 px-1">This will be embedded in your mTLS certificate.</p>
        </div>

        <div class="p-4 rounded-xl bg-primary/5 border border-primary/20 space-y-2">
          <div class="flex items-center gap-2 text-primary">
            <Shield class="w-4 h-4" />
            <span class="text-xs font-bold uppercase tracking-tight">Privacy Notice</span>
          </div>
          <p class="text-[11px] text-slate-400 leading-relaxed">
            Identity generation uses Ed25519 signatures. Your username is public, but your messages are secured by the resulting certificate.
          </p>
        </div>

        <button 
          @click="handleRegister"
          :disabled="isLoading || !username"
          class="w-full btn-primary !py-4 flex items-center justify-center gap-2 text-lg disabled:opacity-50 disabled:cursor-not-allowed group"
        >
          <span v-if="isLoading" class="w-5 h-5 border-2 border-white/20 border-t-white rounded-full animate-spin"></span>
          <template v-else>
            Generate Keys
            <ArrowRight class="w-5 h-5 group-hover:translate-x-1 transition-transform" />
          </template>
        </button>
      </div>

      <div v-else class="space-y-6 animate-in zoom-in-95 duration-500">
        <div class="flex flex-col items-center text-center space-y-4">
          <div class="w-20 h-20 bg-green-500/10 rounded-full flex items-center justify-center">
            <CheckCircle2 class="w-10 h-10 text-green-500" />
          </div>
          <div>
            <h2 class="text-xl font-bold text-white">Identity Ready!</h2>
            <p class="text-slate-400 text-sm">Your secure certificate and keys have been generated.</p>
          </div>
        </div>

        <div class="grid grid-cols-2 gap-4">
          <button 
            @click="downloadIdentity"
            class="flex flex-col items-center gap-2 p-4 rounded-xl bg-white/5 border border-white/10 hover:bg-white/10 transition-colors group"
          >
            <Download class="w-6 h-6 text-primary group-hover:scale-110 transition-transform" />
            <span class="text-xs font-bold text-slate-300">Backup Files</span>
          </button>
          
          <button 
            @click="proceedToChat"
            class="flex flex-col items-center gap-2 p-4 rounded-xl bg-primary/10 border border-primary/20 hover:bg-primary/20 transition-colors group"
          >
            <ArrowRight class="w-6 h-6 text-primary group-hover:translate-x-1 transition-transform" />
            <span class="text-xs font-bold text-primary">Start Chatting</span>
          </button>
        </div>

        <p class="text-[10px] text-center text-slate-500 italic">
          Keep your .key file safe. Without it, you cannot access your account.
        </p>
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
