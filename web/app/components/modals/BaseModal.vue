<script setup lang="ts">
import { X } from 'lucide-vue-next'

defineProps<{
  show: boolean
  title: string
}>()

defineEmits(['close'])
</script>

<template>
  <Teleport to="body">
    <Transition
      enter-active-class="ease-out duration-300"
      enter-from-class="opacity-0"
      enter-to-class="opacity-100"
      leave-active-class="ease-in duration-200"
      leave-from-class="opacity-100"
      leave-to-class="opacity-0"
    >
      <div v-if="show" class="fixed inset-0 z-50 flex items-center justify-center p-4 sm:p-6">
        <!-- Backdrop -->
        <div class="absolute inset-0 bg-slate-950/80 backdrop-blur-sm" @click="$emit('close')"></div>
        
        <!-- Modal -->
        <div class="relative w-full max-w-lg glass-card !p-0 overflow-hidden shadow-[0_32px_120px_rgba(0,0,0,0.5)] border border-white/10 animate-in zoom-in-95 duration-300">
          <!-- Header -->
          <div class="px-6 py-4 border-b border-white/5 flex items-center justify-between bg-white/5">
            <h3 class="text-lg font-bold text-white tracking-tight">{{ title }}</h3>
            <button @click="$emit('close')" class="p-2 rounded-lg hover:bg-white/10 text-slate-500 hover:text-white transition-colors">
              <X class="w-5 h-5" />
            </button>
          </div>
          
          <!-- Content -->
          <div class="p-6">
            <slot />
          </div>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>
