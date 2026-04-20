<script setup lang="ts">
import { ref } from 'vue'
import { Send, Plus, Paperclip, Smile, Command } from 'lucide-vue-next'

const { sendMessage } = useMessages()
const { activeChannel } = useChannels()
const { showError, showSuccess } = useToast()

const message = ref('')
const fileInput = ref<HTMLInputElement | null>(null)
const attachedFile = ref<File | null>(null)

const triggerFilePicker = () => {
  fileInput.value?.click()
}

const handleFileChange = (e: Event) => {
  const target = e.target as HTMLInputElement
  if (target.files && target.files[0]) {
    attachedFile.value = target.files[0]
  }
}

const handleSend = async () => {
  if ((!message.value.trim() && !attachedFile.value) || !activeChannel.value) return
  
  const content = message.value.trim()
  const fileToUpload = attachedFile.value
  
  message.value = ''
  attachedFile.value = null
  
  try {
    const medias = []
    if (fileToUpload) {
      // 1. Upload to S3 (via gRPC-Gateway)
      // File data must be base64 encoded for gRPC JSON (standard bytes mapping)
      const reader = new FileReader()
      const base64Data = await new Promise<string>((resolve) => {
        reader.onload = () => {
          const result = (reader.result as string) || ''
          resolve(result.split(',')[1] || '') // Remove data:mime;base64, prefix
        }
        reader.readAsDataURL(fileToUpload)
      })

      const uploadRes = await fetch(`${useRuntimeConfig().public.apiBase}/v1/upload`, {
        method: 'POST',
        headers: { 
          'x-session-token': useAuth().token.value || '',
          'Content-Type': 'application/json'
        },
        body: JSON.stringify({ 
          filename: fileToUpload.name,
          data: base64Data,
          content_type: fileToUpload.type
        })
      })
      
      if (!uploadRes.ok) throw new Error('Upload failed')
      const { url } = await uploadRes.json()
      
      medias.push({
        type: fileToUpload.type.startsWith('image/') ? 'image' : 'file',
        url: url,
        name: fileToUpload.name
      })
    }

    // 2. Send message
    await sendMessage(content, medias)
  } catch (e: any) {
    showError(e.message)
    message.value = content
    attachedFile.value = fileToUpload
  }
}

const showEmojiPicker = ref(false)
const commonEmojis = ['😊', '😂', '❤️', '🔥', '👍', '🙌', '🚀', '✨', '😢', '😍', '🤔', '😎']

const addEmoji = (emoji: string) => {
  message.value += emoji
  showEmojiPicker.value = false
}

const onKeydown = (e: KeyboardEvent) => {
  if (e.key === 'Enter' && !e.shiftKey) {
    e.preventDefault()
    handleSend()
  }
}
</script>

<template>
  <div class="h-[132px] p-4 bg-slate-900/40 backdrop-blur-xl border-t border-white/5 flex flex-col justify-center">
    <!-- Attachment Indicator -->
    <div v-if="attachedFile" class="flex items-center gap-2 mb-2 px-1 animate-in fade-in slide-in-from-bottom-1 duration-200">
      <div class="flex items-center gap-2 px-2 py-1 rounded-lg bg-primary/10 border border-primary/20 text-[10px] font-bold text-primary uppercase tracking-wider">
        <Paperclip class="w-3 h-3" />
        <span class="truncate max-w-[200px]">{{ attachedFile.name }}</span>
        <button @click="attachedFile = null" class="ml-1 p-0.5 hover:bg-primary/20 rounded transition-colors">
          <Command class="w-2.5 h-2.5 rotate-45" />
        </button>
      </div>
    </div>

    <!-- Input Area -->
    <div class="relative group">
      <input 
        ref="fileInput"
        type="file" 
        class="hidden" 
        @change="handleFileChange"
      />
      <div class="absolute -inset-0.5 bg-gradient-to-r from-primary/20 to-cta/20 rounded-2xl blur opacity-0 group-focus-within:opacity-100 transition duration-1000"></div>
      <div class="relative flex items-end gap-2 p-2 rounded-2xl bg-slate-950/50 border border-white/10 group-focus-within:border-primary/30 transition-all shadow-2xl">
        <textarea 
          v-model="message"
          rows="1"
          placeholder="Type your message..."
          class="flex-1 bg-transparent border-none focus:ring-0 text-slate-200 placeholder:text-slate-600 text-sm py-2 pl-3 min-h-[40px] max-h-48 resize-none custom-scrollbar"
          @keydown="onKeydown"
          :disabled="!activeChannel"
        ></textarea>
        
        <div class="flex items-center gap-1 mb-1 relative">
          <button 
            @click="triggerFilePicker"
            class="p-1.5 rounded-lg hover:bg-white/5 transition-colors" 
            :class="attachedFile ? 'text-primary' : 'text-slate-500 hover:text-slate-300'"
            title="Attach Files"
          >
            <Paperclip class="w-4 h-4" />
          </button>
          
          <div v-if="showEmojiPicker" class="absolute bottom-full right-0 mb-4 p-3 bg-slate-900 border border-white/10 rounded-2xl shadow-2xl z-50 flex flex-wrap gap-2 w-48 backdrop-blur-xl animate-in fade-in slide-in-from-bottom-2 duration-200">
            <button 
              v-for="emoji in commonEmojis" 
              :key="emoji"
              @click="addEmoji(emoji)"
              class="text-xl hover:scale-125 transition-transform"
            >
              {{ emoji }}
            </button>
          </div>

          <button 
            @click="showEmojiPicker = !showEmojiPicker"
            class="p-1.5 rounded-lg hover:bg-white/5 text-slate-500 hover:text-slate-300 transition-colors" 
            :class="{ 'text-primary bg-primary/10 border-primary/20': showEmojiPicker }"
            title="Emojis"
          >
            <Smile class="w-4 h-4" />
          </button>
        </div>

        <button 
          @click="handleSend"
          :disabled="(!message.trim() && !attachedFile) || !activeChannel"
          class="p-2.5 rounded-xl bg-primary text-white shadow-lg shadow-primary/20 hover:scale-105 active:scale-95 disabled:opacity-30 disabled:grayscale disabled:hover:scale-100 transition-all"
        >
          <Send class="w-5 h-5" />
        </button>
      </div>
    </div>
  </div>
</template>

<style scoped>
.custom-scrollbar::-webkit-scrollbar {
  width: 4px;
}
.custom-scrollbar::-webkit-scrollbar-thumb {
  background: rgba(255, 255, 255, 0.05);
  border-radius: 4px;
}
textarea {
  line-height: 1.5;
}
</style>
