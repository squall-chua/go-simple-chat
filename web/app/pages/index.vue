<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { ShieldCheck, Upload, Key, FileText, ArrowRight } from 'lucide-vue-next'

const { login, isAuthenticated, restoreSession } = useAuth()
const { showSuccess, showError } = useToast()

const certFile = ref<File | null>(null)
const keyFile = ref<File | null>(null)
const isLoading = ref(false)
const isRestoring = ref(true)

const onFileChange = (e: Event, type: 'cert' | 'key') => {
  const target = e.target as HTMLInputElement
  if (target.files && target.files[0]) {
    if (type === 'cert') certFile.value = target.files[0]
    else keyFile.value = target.files[0]
  }
}

const readFile = (file: File): Promise<string> => {
  return new Promise((resolve) => {
    const reader = new FileReader()
    reader.onload = (e) => resolve(e.target?.result as string)
    reader.readAsText(file)
  })
}

const handleConnect = async () => {
  if (!certFile.value || !keyFile.value) {
    showError('Both certificate and key files are required')
    return
  }

  isLoading.value = true
  try {
    const cert = await readFile(certFile.value)
    const key = await readFile(keyFile.value)
    
    const success = await login(cert, key)
    if (success) {
      showSuccess('Successfully authenticated')
      navigateTo('/chat')
    }
  } catch (e: any) {
    showError(e.message)
  } finally {
    isLoading.value = false
  }
}

onMounted(async () => {
  const restored = await restoreSession()
  if (restored) {
    navigateTo('/chat')
  }
  isRestoring.value = false
})
</script>

<template>
  <div class="min-h-screen flex items-center justify-center p-6 relative overflow-hidden bg-background">
    <!-- Animated background elements -->
    <div class="absolute top-[-10%] left-[-10%] w-[40%] h-[40%] bg-primary/10 rounded-full blur-[120px] animate-pulse"></div>
    <div class="absolute bottom-[-10%] right-[-10%] w-[40%] h-[40%] bg-cta/5 rounded-full blur-[120px]"></div>

    <div v-if="isRestoring" class="flex flex-col items-center gap-4 animate-pulse">
      <div class="w-12 h-12 border-4 border-primary/20 border-t-primary rounded-full animate-spin"></div>
      <p class="text-slate-400 font-medium font-heading">Restoring identity...</p>
    </div>

    <div v-else class="w-full max-w-md glass-card space-y-8 animate-in fade-in slide-in-from-bottom-4 duration-700">
      <div class="text-center space-y-2">
        <h1 class="text-3xl font-bold font-heading tracking-tight text-white flex items-center justify-center gap-3">
          <ShieldCheck class="w-8 h-8 text-primary" />
          Secure messaging
        </h1>
        <p class="text-slate-400 text-sm">Secure mTLS-based messaging client</p>
      </div>

      <div class="space-y-6">
        <!-- Certificate Upload -->
        <div class="space-y-2">
          <label class="text-xs font-bold uppercase tracking-widest text-slate-500 px-1">Certificate (.crt)</label>
          <div class="relative group">
            <input 
              type="file" 
              accept=".crt,.pem"
              @change="onFileChange($event, 'cert')"
              class="absolute inset-0 w-full h-full opacity-0 cursor-pointer z-10"
            />
            <div :class="[
              'flex items-center gap-3 p-4 rounded-xl border-2 border-dashed transition-all',
              certFile ? 'border-primary/50 bg-primary/5' : 'border-white/10 hover:border-white/20 bg-white/5'
            ]">
              <div :class="[
                'w-10 h-10 rounded-lg flex items-center justify-center transition-colors',
                certFile ? 'bg-primary text-white' : 'bg-white/5 text-slate-500'
              ]">
                <FileText class="w-5 h-5" />
              </div>
              <div class="flex-1 overflow-hidden">
                <p class="text-sm font-medium truncate" :class="certFile ? 'text-white' : 'text-slate-400'">
                  {{ certFile ? certFile.name : 'Select certificate file' }}
                </p>
                <p class="text-[10px] text-slate-500 uppercase tracking-tighter">Identity proof</p>
              </div>
              <Upload v-if="!certFile" class="w-4 h-4 text-slate-600" />
            </div>
          </div>
        </div>

        <!-- Private Key Upload -->
        <div class="space-y-2">
          <label class="text-xs font-bold uppercase tracking-widest text-slate-500 px-1">Private Key (.key)</label>
          <div class="relative group">
            <input 
              type="file" 
              accept=".key,.pem"
              @change="onFileChange($event, 'key')"
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
                <p class="text-[10px] text-slate-500 uppercase tracking-tighter">Secret key</p>
              </div>
              <Upload v-if="!keyFile" class="w-4 h-4 text-slate-600" />
            </div>
          </div>
        </div>

        <button 
          @click="handleConnect"
          :disabled="isLoading || !certFile || !keyFile"
          class="w-full btn-primary !py-4 flex items-center justify-center gap-2 text-lg disabled:opacity-50 disabled:cursor-not-allowed group"
        >
          <span v-if="isLoading" class="w-5 h-5 border-2 border-white/20 border-t-white rounded-full animate-spin"></span>
          <template v-else>
            Connect to Chat
            <ArrowRight class="w-5 h-5 group-hover:translate-x-1 transition-transform" />
          </template>
        </button>
      </div>

      <div class="pt-6 border-t border-white/5 flex flex-col gap-3">
        <NuxtLink to="/register" class="text-sm text-slate-400 hover:text-white transition-colors flex items-center justify-between group">
          Don't have an identity? 
          <span class="text-primary font-bold flex items-center gap-1 group-hover:gap-2 transition-all">
            Register for free <ArrowRight class="w-4 h-4" />
          </span>
        </NuxtLink>
        <NuxtLink to="/renew" class="text-sm text-slate-400 hover:text-white transition-colors flex items-center justify-between group">
          Existing cert expired?
          <span class="text-cta font-bold flex items-center gap-1 group-hover:gap-2 transition-all">
            Renew certificate <ArrowRight class="w-4 h-4" />
          </span>
        </NuxtLink>
      </div>
    </div>
  </div>
</template>
