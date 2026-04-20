<template>
  <div class="h-screen w-screen bg-background font-sans text-slate-200">
    <NuxtPage />
    <ToastBar />
  </div>
</template>

<style>
body {
  margin: 0;
  background: #0F172A;
  -webkit-font-smoothing: antialiased;
  -moz-osx-font-smoothing: grayscale;
}

*, *::before, *::after {
  box-sizing: border-box;
}

::-webkit-scrollbar {
  width: 6px;
}
::-webkit-scrollbar-track {
  background: transparent;
}
::-webkit-scrollbar-thumb {
  background: rgba(255, 255, 255, 0.05);
  border-radius: 10px;
}
::-webkit-scrollbar-thumb:hover {
  background: rgba(255, 255, 255, 0.1);
}
</style>
