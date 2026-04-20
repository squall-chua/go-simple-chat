<script setup lang="ts">
import { AlertCircle, CheckCircle2, X } from 'lucide-vue-next'

const { toast, clearToast } = useToast()
</script>

<template>
  <Transition
    enter-active-class="transform ease-out duration-300 transition"
    enter-from-class="translate-y-2 opacity-0 sm:translate-y-0 sm:translate-x-2"
    enter-to-class="translate-y-0 opacity-100 sm:translate-x-0"
    leave-active-class="transition ease-in duration-100"
    leave-from-class="opacity-100"
    leave-to-class="opacity-0"
  >
    <div 
      v-if="toast.visible"
      class="fixed bottom-6 right-6 z-50 flex items-center gap-3 p-4 rounded-2xl shadow-[0_20px_50px_rgba(0,0,0,0.3)] backdrop-blur-xl border border-white/10 min-w-[320px] max-w-md pointer-events-auto"
      :class="toast.type === 'error' ? 'bg-red-500/10' : 'bg-primary/10'"
    >
      <div :class="[
        'w-10 h-10 rounded-xl flex items-center justify-center shrink-0',
        toast.type === 'error' ? 'bg-red-500/20 text-red-500' : 'bg-green-500/20 text-green-500'
      ]">
        <AlertCircle v-if="toast.type === 'error'" class="w-6 h-6" />
        <CheckCircle2 v-else class="w-6 h-6" />
      </div>
      
      <div class="flex-1 space-y-0.5">
        <p class="text-[10px] font-bold uppercase tracking-[0.2em]" :class="toast.type === 'error' ? 'text-red-500' : 'text-green-500'">
          {{ toast.type === 'error' ? 'Security Alert' : 'System Update' }}
        </p>
        <p class="text-sm font-medium text-white leading-snug">{{ toast.message }}</p>
      </div>

      <button 
        @click="clearToast"
        class="p-1 rounded-lg hover:bg-white/5 text-slate-500 hover:text-white transition-colors"
      >
        <X class="w-4 h-4" />
      </button>

      <!-- Progress bar -->
      <div class="absolute bottom-0 left-4 right-4 h-0.5 bg-white/5 rounded-full overflow-hidden">
        <div 
          class="h-full bg-current transition-all duration-5000 linear"
          :class="toast.type === 'error' ? 'text-red-500' : 'text-green-500'"
          :style="{ width: toast.visible ? '0%' : '100%' }"
        ></div>
      </div>
    </div>
  </Transition>
</template>
