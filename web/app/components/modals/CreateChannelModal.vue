<script setup lang="ts">
import { ref } from 'vue'
import { Hash, ArrowRight } from 'lucide-vue-next'

const props = defineProps<{
  show: boolean
}>()

const emit = defineEmits(['close'])
const { createChannel } = useChannels()
const { showSuccess, showError } = useToast()

const name = ref('')
const isLoading = ref(false)

const handleCreate = async () => {
  if (!name.value.trim()) return
  
  isLoading.value = true
  try {
    const channelId = await createChannel(name.value.trim(), 'TYPE_GROUP', [])
    if (channelId) {
      showSuccess(`Channel #${name.value} created`)
      name.value = ''
      emit('close')
    }
  } catch (e: any) {
    showError(e.message)
  } finally {
    isLoading.value = false
  }
}
</script>

<template>
  <ModalsBaseModal :show="show" title="Create New Channel" @close="emit('close')">
    <div class="space-y-6">
      <div class="space-y-2">
        <label class="text-xs font-bold uppercase tracking-widest text-slate-500 px-1">Channel Name</label>
        <div class="relative group">
          <Hash class="absolute left-4 top-1/2 -translate-y-1/2 w-5 h-5 text-slate-500 group-focus-within:text-primary transition-colors" />
          <input 
            v-model="name"
            type="text" 
            placeholder="e.g. general"
            class="w-full bg-white/5 border border-white/10 rounded-xl py-4 pl-12 pr-4 text-white focus:outline-none focus:border-primary/50 focus:bg-white/10 transition-all font-medium"
            @keyup.enter="handleCreate"
            autofocus
          />
        </div>
        <p class="text-[10px] text-slate-500 px-1">Use lowercase, numbers, and hyphens only.</p>
      </div>

      <div class="p-4 rounded-xl bg-primary/5 border border-primary/10 flex gap-3">
        <div class="w-8 h-8 rounded-lg bg-primary/20 flex items-center justify-center shrink-0">
          <Hash class="w-4 h-4 text-primary" />
        </div>
        <p class="text-[11px] text-slate-400 leading-relaxed">
          Group channels allow multiple participants to chat together. You can add participants after creation.
        </p>
      </div>

      <button 
        @click="handleCreate"
        :disabled="isLoading || !name.trim()"
        class="w-full btn-primary !py-4 flex items-center justify-center gap-2 text-lg disabled:opacity-50 disabled:cursor-not-allowed group"
      >
        <span v-if="isLoading" class="w-5 h-5 border-2 border-white/20 border-t-white rounded-full animate-spin"></span>
        <template v-else>
          Create Channel
          <ArrowRight class="w-5 h-5 group-hover:translate-x-1 transition-transform" />
        </template>
      </button>
    </div>
  </ModalsBaseModal>
</template>
