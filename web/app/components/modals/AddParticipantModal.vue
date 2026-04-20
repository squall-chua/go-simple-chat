<script setup lang="ts">
import { ref } from 'vue'
import { UserPlus, ArrowRight, User } from 'lucide-vue-next'

const props = defineProps<{
  show: boolean
}>()

const emit = defineEmits(['close'])
const { activeChannelId, fetchChannels } = useChannels()
const { token } = useAuth()
const { showSuccess, showError } = useToast()
const config = useRuntimeConfig()

const username = ref('')
const isLoading = ref(false)

const handleAdd = async () => {
  if (!username.value.trim() || !activeChannelId.value) return
  
  isLoading.value = true
  try {
    const response = await fetch(`${config.public.apiBase}/v1/channels/${activeChannelId.value}/participants`, {
      method: 'POST',
      headers: { 
        'x-session-token': token.value || '',
        'Content-Type': 'application/json'
      },
      body: JSON.stringify({ usernames: [username.value.trim()] })
    })
    
    if (!response.ok) throw new Error('Failed to add participant')
    
    showSuccess(`${username.value} added to channel`)
    username.value = ''
    await fetchChannels() // Refresh participants list
    emit('close')
  } catch (e: any) {
    showError(e.message)
  } finally {
    isLoading.value = false
  }
}
</script>

<template>
  <ModalsBaseModal :show="show" title="Add Participant" @close="emit('close')">
    <div class="space-y-6">
      <div class="space-y-2">
        <label class="text-xs font-bold uppercase tracking-widest text-slate-500 px-1">Participant Username</label>
        <div class="relative group">
          <User class="absolute left-4 top-1/2 -translate-y-1/2 w-5 h-5 text-slate-500 group-focus-within:text-primary transition-colors" />
          <input 
            v-model="username"
            type="text" 
            placeholder="e.g. Alice"
            class="w-full bg-white/5 border border-white/10 rounded-xl py-4 pl-12 pr-4 text-white focus:outline-none focus:border-primary/50 focus:bg-white/10 transition-all font-medium"
            @keyup.enter="handleAdd"
            autofocus
          />
        </div>
      </div>

      <button 
        @click="handleAdd"
        :disabled="isLoading || !username.trim()"
        class="w-full btn-primary !py-4 flex items-center justify-center gap-2 text-lg disabled:opacity-50 disabled:cursor-not-allowed group"
      >
        <span v-if="isLoading" class="w-5 h-5 border-2 border-white/20 border-t-white rounded-full animate-spin"></span>
        <template v-else>
          Add to Channel
          <ArrowRight class="w-5 h-5 group-hover:translate-x-1 transition-transform" />
        </template>
      </button>
    </div>
  </ModalsBaseModal>
</template>
