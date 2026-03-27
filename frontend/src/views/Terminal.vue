<template>
  <div class="h-full flex flex-col">
    <div class="flex items-center justify-between mb-4">
      <div>
        <h1 class="text-2xl font-bold text-white">{{ t('terminal.title') }}</h1>
        <p class="text-gray-400 mt-1">{{ t('terminal.subtitle') }}</p>
      </div>
      <button class="btn-primary" @click="connectTerminal" :disabled="connected">
        {{ connected ? t('terminal.connected') : t('terminal.connect') }}
      </button>
    </div>
    
    <div class="flex-1 bg-black rounded-lg border border-panel-border p-2 overflow-hidden" ref="terminalContainer"></div>
  </div>
</template>

<script setup>
import { ref, onMounted, onBeforeUnmount } from 'vue'
import { useI18n } from 'vue-i18n'
import { Terminal } from '@xterm/xterm'
import { FitAddon } from '@xterm/addon-fit'
import '@xterm/xterm/css/xterm.css'
import { useAuthStore } from '../stores/auth'

const { t } = useI18n()
const authStore = useAuthStore()

const terminalContainer = ref(null)
const connected = ref(false)
let term = null
let fitAddon = null
let ws = null

function connectTerminal() {
  if (connected.value) return;

  if (!term) {
    term = new Terminal({
      theme: {
        background: '#000000',
        foreground: '#ffffff',
      },
      cursorBlink: true
    });
    fitAddon = new FitAddon();
    term.loadAddon(fitAddon);
    term.open(terminalContainer.value);
    fitAddon.fit();
  }

  const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
  const wsUrl = `${protocol}//${window.location.host}/api/v1/terminal/ws?token=${authStore.token}`;
  
  term.writeln(t('terminal.connecting'));
  
  ws = new WebSocket(wsUrl);

  ws.onopen = () => {
    connected.value = true;
    term.writeln('\r\n' + t('terminal.connected_msg') + '\r\n');
  };

  ws.onmessage = (event) => {
    term.write(event.data);
  };

  ws.onclose = () => {
    connected.value = false;
    term.writeln('\r\n' + t('terminal.disconnected') + '\r\n');
  };

  term.onData(data => {
    if (ws && ws.readyState === WebSocket.OPEN) {
      ws.send(data);
    }
  });

  window.addEventListener('resize', () => {
    if (fitAddon) fitAddon.fit();
  });
}

onMounted(() => {
  // optionally connect on mount
});

onBeforeUnmount(() => {
  if (ws) ws.close();
  if (term) term.dispose();
});
</script>
