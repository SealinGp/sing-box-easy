<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { userService } from '../services'
import { KeyIcon, UserIcon, EyeIcon, EyeSlashIcon } from '@heroicons/vue/24/outline'

const router = useRouter()

const username = ref('')
const password = ref('')
const error = ref('')
const loading = ref(false)
const showPassword = ref(false)

const handleLogin = async () => {
  console.log('[Login.vue] handleLogin clicked. Username:', username.value)
  if (!username.value || !password.value) {
    error.value = 'Please enter both username and password'
    return
  }

  error.value = ''
  loading.value = true

  try {
    console.log('[Login.vue] Calling userService.login...')
    const res = await userService.login(username.value, password.value)
    // debugger
    console.log('[Login.vue] userService.login successful. Result:', res)
    
    // Redirect to dashboard
    console.log('[Login.vue] Routing push to /dashboard...')
    await router.push('/dashboard')
    console.log('[Login.vue] Routing push completed.')
  } catch (err: any) {
    console.error('[Login.vue] Login sequence threw error:', err)
    error.value = err.message || 'Incorrect username or password'
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <div class="min-h-screen flex items-center justify-center bg-gradient-to-br from-slate-900 via-purple-950 to-indigo-950 px-4 py-12 relative overflow-hidden font-sans">
    <!-- Atmospheric Background Glows -->
    <div class="absolute top-1/4 left-1/4 w-96 h-96 bg-purple-500/10 rounded-pill blur-3xl animate-pulse"></div>
    <div class="absolute bottom-1/4 right-1/4 w-96 h-96 bg-indigo-500/10 rounded-pill blur-3xl animate-pulse" style="animation-delay: 1s"></div>

    <!-- Login Container -->
    <div class="w-full max-w-md bg-slate-900/80 backdrop-blur-xl border border-white/10 p-8 rounded-surface shadow-float relative z-10 animate-slide-up">
      <!-- Header -->
      <div class="text-center mb-8">
        <div class="inline-flex items-center justify-center p-3 bg-gradient-to-tr from-primary-600 to-indigo-600 rounded-surface shadow-float shadow-primary-500/20 mb-4 animate-bounce" style="animation-duration: 3s">
          <img src="/logo.jpg" alt="Logo" class="h-12 w-12 rounded-control" />
        </div>
        <h1 class="text-2xl font-bold text-white tracking-tight">Sing Box Easy</h1>
        <p class="text-slate-400 mt-2 text-sm">Sign in to control your proxy service</p>
      </div>

      <!-- Error Alert -->
      <div v-if="error" class="mb-6 p-4 rounded-surface bg-red-500/15 border border-red-500/30 text-red-200 text-sm flex items-center gap-2 animate-fade-in">
        <span class="w-1.5 h-1.5 rounded-pill bg-red-400 animate-ping"></span>
        <span>{{ error }}</span>
      </div>

      <!-- Form -->
      <form @submit.prevent="handleLogin" class="space-y-6">
        <!-- Username -->
        <div>
          <label class="block text-xs font-semibold uppercase tracking-wider text-slate-300 mb-2">Username</label>
          <div class="relative">
            <span class="absolute inset-y-0 left-0 flex items-center pl-4 text-slate-400">
              <UserIcon class="h-5 w-5" />
            </span>
            <input
              type="text"
              v-model="username"
              required
              class="w-full bg-slate-950/50 border border-white/10 rounded-control py-3 pl-11 pr-4 text-white placeholder-slate-500 focus:outline-none focus:border-primary-500 focus:ring-2 focus:ring-primary-500/20 transition-all duration-200"
              placeholder="Enter your username"
            />
          </div>
        </div>

        <!-- Password -->
        <div>
          <label class="block text-xs font-semibold uppercase tracking-wider text-slate-300 mb-2">Password</label>
          <div class="relative">
            <span class="absolute inset-y-0 left-0 flex items-center pl-4 text-slate-400">
              <KeyIcon class="h-5 w-5" />
            </span>
            <input
              :type="showPassword ? 'text' : 'password'"
              v-model="password"
              required
              class="w-full bg-slate-950/50 border border-white/10 rounded-control py-3 pl-11 pr-12 text-white placeholder-slate-500 focus:outline-none focus:border-primary-500 focus:ring-2 focus:ring-primary-500/20 transition-all duration-200"
              placeholder="Enter your password"
            />
            <button
              type="button"
              @click="showPassword = !showPassword"
              class="absolute inset-y-0 right-0 flex items-center pr-4 text-slate-400 hover:text-slate-200 transition-colors"
            >
              <EyeSlashIcon v-if="showPassword" class="h-5 w-5" />
              <EyeIcon v-else class="h-5 w-5" />
            </button>
          </div>
        </div>

        <!-- Submit Button -->
        <button
          type="submit"
          :disabled="loading"
          class="w-full bg-gradient-to-r from-primary-600 to-indigo-600 hover:from-primary-500 hover:to-indigo-500 disabled:from-primary-800 disabled:to-indigo-800 text-white font-semibold py-3 px-4 rounded-control shadow-float shadow-primary-500/20 hover:shadow-primary-500/30 transition-all duration-300 active:scale-[0.98] flex items-center justify-center gap-2 cursor-pointer disabled:cursor-not-allowed group relative overflow-hidden"
        >
          <!-- Shiny Hover Effect -->
          <div class="absolute inset-0 w-1/2 h-full bg-white/10 skew-x-[-30deg] -left-1/2 group-hover:animate-[shine_0.75s_ease-in-out]"></div>

          <span v-if="loading" class="w-5 h-5 border-2 border-white/20 border-t-white rounded-pill animate-spin"></span>
          <span v-else>Sign In</span>
        </button>
      </form>

      <!-- Footer Help -->
      <div class="text-center mt-8 pt-6 border-t border-white/5">
        <p class="text-xs text-slate-500">
          First run? Try logging in with the default seeded credentials:<br />
          <span class="font-mono text-slate-400 select-all">admin</span> / <span class="font-mono text-slate-400 select-all">admin</span>
        </p>
      </div>
    </div>
  </div>
</template>

<style scoped>
@keyframes shine {
  100% {
    left: 125%;
  }
}
</style>
