<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import DonateButton from '../components/DonateButton.vue'

const router = useRouter()

const games = ref([
  {
    title: 'Assoziations Spiel',
    description: 'In diesem Spiel geht es darum, möglichst viele Assoziationen zu finden.',
    playerCount: '2 Spieler',
    route: '/assoziationsspiel',
  },
])

function navigateToGame(route: string) {
  router.push(route)
}
</script>

<template>
  <div class="min-h-screen flex items-center justify-center bg-[#0a0a2a] text-white font-sans">
    <DonateButton />

    <div class="relative max-w-3xl w-full p-6 space-y-12">
      <h1 class="text-center space-y-3">
        <div class="text-2xl font-light tracking-wider text-blue-200/90">Willkommen zu</div>
        <div class="from-blue-300 via-purple-300 to-pink-300 bg-gradient-to-r bg-clip-text text-6xl text-transparent font-bold">
          SPIELE.KEKSI.DEV
        </div>
      </h1>

      <div class="flex justify-center">
        <div
          v-for="game in games"
          :key="game.title"
          class="w-full max-w-xl cursor-pointer rounded-3xl bg-[#1a1a4a]/80 backdrop-blur-sm transition-all duration-300 border border-white/[0.08] hover:shadow-[0_0_10px_rgba(255,255,255,0.2)]"
          @click="navigateToGame(game.route)"
        >
          <div class="p-8 border-b border-white/[0.08]">
            <div class="flex items-center justify-between gap-4">
              <h2 class="text-3xl font-bold">{{ game.title }}</h2>
              <span class="text-sm font-medium bg-white/[0.05] px-4 py-1.5 rounded-full border border-white/[0.08] whitespace-nowrap">
                {{ game.playerCount }}
              </span>
            </div>
          </div>
          
          <div class="p-8">
            <div class="flex items-center justify-between gap-8">
              <p class="text-base leading-relaxed text-white/80 text-left">{{ game.description }}</p>
              <button class="btn-sm shrink-0">Spielen</button>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>
