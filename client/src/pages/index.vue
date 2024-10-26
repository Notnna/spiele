<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'

const router = useRouter()

const games = ref([
  {
    title: 'Assoziationsspiel',
    description: 'Das große Assoziations Spiel',
    route: '/assoziationsspiel',
  },
])

function navigateToGame(route: string) {
  router.push(route)
}
</script>

<template>
  <div class="min-h-screen flex items-center justify-center bg-[#0a0a2a] text-white font-sans">
    <div class="max-w-4xl w-full p-6 space-y-6">
      <h1 class="text-center space-y-1">
        <div class="text-2xl font-light">
          Willkommen zu
        </div>
        <div class="from-blue-300 via-purple-300 to-pink-300 bg-gradient-to-r bg-clip-text text-5xl text-transparent font-bold">
          SPIELE.KEKSI.DEV
        </div>
      </h1>

      <div class="grid grid-cols-1 md:grid-cols-2 gap-6">
        <div
          v-for="game in games"
          :key="game.title"
          class="border-2 border-[#3a3a6a] rounded-xl bg-[#1a1a4a] p-6 space-y-4 cursor-pointer hover:shadow-[0_0_10px_rgba(255,255,255,0.2)] transition-shadow duration-300"
          @click="navigateToGame(game.route)"
        >
          <h2 class="text-3xl font-bold">{{ game.title }}</h2>
          <p class="text-lg font-light">{{ game.description }}</p>
          <button class="btn w-full">
            Spielen
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

