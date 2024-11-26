<script setup lang="ts">
import { onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import DonateButton from '../components/DonateButton.vue'

const currentCategory = ref('')
const player1Input = ref('')
const player2Input = ref('')
const revealed = ref(false)
const btnRevealed = ref(false)
const streak = ref(0)

const router = useRouter()
const route = useRoute()

const roomID = ref('')
const inRoom = ref(false)

let socket: WebSocket

const isConnected = ref(false)

// Update roomID when it changes and update URL
watch(roomID, (newVal) => {
  if (newVal) {
    router.replace({ query: { room: newVal } })
  }
  else {
    router.replace({ query: {} })
  }
})

watch(player1Input, (newVal) => {
  socket.send(JSON.stringify({
    type: 'playerInput',
    value: newVal,
  }))
})

// Check for roomID in URL on mount
onMounted(() => {
  const roomFromURL = route.query.room
  if (typeof roomFromURL === 'string') {
    roomID.value = roomFromURL
    joinRoom()
  }
})

function newCategory() {
  socket.send(JSON.stringify({
    type: 'newCategory',
  }))
}

function createRoom() {
  roomID.value = Math.random().toString(36).substring(2, 8)

  joinRoom()
}

function joinRoom() {
  const serverUrl = import.meta.env.VITE_SERVER_URL || 'spiele.keksi.dev'

  if (window.location.hostname === 'localhost')
    socket = new WebSocket(`ws://localhost:8080/ws?room=${roomID.value}`)
  else
    socket = new WebSocket(`wss://${serverUrl}/ws?room=${roomID.value}`)

  inRoom.value = true

  socket.onopen = () => {
    console.log('Connected to server') // eslint-disable-line no-console
    isConnected.value = true
    newCategory()
  }

  socket.onmessage = (event) => {
    const data = JSON.parse(event.data)
    switch (data.type) {
      case 'playerInput':
        player2Input.value = data.value
        break
      case 'reveal':
        revealed.value = true
        break
      case 'streak':
        streak.value = data.value
        break
      case 'resetStreak':
        streak.value = 0
        break
      case 'newCategory':
        currentCategory.value = data.value
        player1Input.value = ''
        player2Input.value = ''
        revealed.value = false
        btnRevealed.value = false
        break
      case 'allRevealed':
        revealed.value = true
        break
    }
  }

  socket.onclose = (event) => {
    console.log(`WebSocket is closed now. Code: ${event.code}`) // eslint-disable-line no-console
    isConnected.value = false
  }

  socket.onerror = (error) => {
    console.log(`WebSocket Error: ${error}`) // eslint-disable-line no-console
  }
}

// function reconnect() {
//   if (!isConnected.value) {
//     joinRoom()
//   }
// }

function handleSubmit() {
  streak.value = Math.min(streak.value + 1, 5)

  socket.send(JSON.stringify({
    type: 'streak',
    value: streak.value,
  }))

  newCategory()
}

function handleReveal() {
  btnRevealed.value = true
  socket.send(JSON.stringify({
    type: 'reveal',
  }))
}

function resetStreak() {
  streak.value = 0

  socket.send(JSON.stringify({
    type: 'resetStreak',
  }))

  newCategory()
}

function nextCategory() {
  newCategory()
}

function copyRoomID() {
  navigator.clipboard.writeText(roomID.value)
}
</script>

<template>
  <div class="min-h-screen flex items-center justify-center bg-[#0a0a2a] text-white font-sans">
    <DonateButton />
    
    <div v-if="inRoom" class="absolute left-0 top-0 p-4">
      <div class="relative flex items-center gap-2 border-2 border-[#3a3a6a] rounded-xl bg-[#1a1a4a] p-2">
        <span class="font-light font-mono">
          {{ roomID }}
        </span>
        <button class="i-carbon-copy" @click="copyRoomID" />
      </div>
    </div>

    <div class="max-w-2xl w-full p-6 space-y-6">
      <h1 class="text-center space-y-1">
        <div class="text-2xl font-light">
          Das große
        </div>
        <div class="from-blue-300 via-purple-300 to-pink-300 bg-gradient-to-r bg-clip-text text-5xl text-transparent font-bold">
          ASSOZIATIONS
        </div>
        <div class="text-3xl font-light">
          Spiel
        </div>
      </h1>

      <div v-if="inRoom" class="max-w-2xl w-full p-6 space-y-6">
        <div class="border border-white rounded-full p-3 text-center text-2xl font-bold shadow-[0_0_10px_rgba(255,255,255,0.5)] transition-shadow duration-1000">
          {{ currentCategory }}
        </div>
        <div class="text-center text-2xl font-bold">
          Streak: {{ streak }}/5
        </div>
        <input
          v-model="player1Input"
          :disabled="revealed"
          type="text"
          class="input"
          placeholder="Eingabe"
        >
        <input
          v-if="revealed"
          :value="player2Input"
          type="text"
          class="input"
          disabled
        >
        <div class="grid grid-cols-4 gap-4">
          <button
            class="btn"
            :disabled="btnRevealed"
            @click="handleReveal"
          >
            REVEAL
          </button>
          <button
            class="btn"
            :disabled="!revealed"
            @click="handleSubmit"
          >
            RICHTIG
          </button>
          <button
            class="btn"
            :disabled="!revealed"
            @click="resetStreak"
          >
            FALSCH
          </button>
          <button
            class="btn"
            :disabled="!revealed"
            @click="nextCategory"
          >
            WEITER
          </button>
        </div>
      </div>

      <div v-else class="max-w-2xl w-full p-6 space-y-6">
        <input
          v-model="roomID"
          class="input"
          type="text"
          placeholder="Room ID"
        >
        <div class="grid grid-cols-2 gap-4">
          <button
            class="btn"
            :disabled="!roomID"
            @click="joinRoom"
          >
            Beitreten
          </button>
          <button
            class="btn"
            @click="createRoom"
          >
            Erstellen
          </button>
        </div>
      </div>

      <!-- <div v-if="inRoom && !isConnected" class="text-center mt-4">
        <p class="text-red-500">Disconnected from server</p>
        <button @click="reconnect" class="btn mt-2">
          Reconnect
        </button>
      </div> -->
    </div>
  </div>
</template>
