import { ref, onUnmounted } from 'vue'
import type { ClientCommand, ServerEvent } from '../types/kanban'
import { useBoardStore } from '../stores/board'

export type SocketStatus = 'disconnected' | 'connecting' | 'connected'

export function useBoardSocket() {
    const store = useBoardStore()
    const status = ref<SocketStatus>('disconnected')

    let socket: WebSocket | null = null
    let reconnectTimer: ReturnType<typeof setTimeout> | null = null
    let reconnectAttempts = 0
    let currentBoardId: string | null = null
    let currentUserName = ''
    let intentionalClose = false

    function connect(boardId: string, userName: string) {
        disconnect()

        currentBoardId = boardId
        currentUserName = userName
        intentionalClose = false

        openSocket()
    }

    function openSocket() {
        if (!currentBoardId) return
        
        status.value = 'connecting'
        const url = `ws://localhost:8080/ws/boards/${currentBoardId}?name=${encodeURIComponent(currentUserName)}`
        socket = new WebSocket(url)

        socket.onopen = () => {
            status.value = 'connected'
            reconnectAttempts = 0
        }

        socket.onmessage = (event: MessageEvent) => {
            try{
                const parsed: ServerEvent = JSON.parse(event.data)
                store.handleServerEvent(parsed)
            } catch (e) {
                console.error('Failed to parse WebSocket message:', e)
            }
        } 

        socket.onclose = () => {
            status.value = 'disconnected'
            socket = null
            if (!intentionalClose) {
                scheduleReconnect()
            }
        }

        socket.onerror = () => {
            socket?.close()
        }
    }

    function scheduleReconnect() {
        // backoff capped at 10 sec.
        const delay = Math.min(1000 * Math.pow(2, reconnectAttempts), 10000)
        
        reconnectAttempts++
        reconnectTimer = setTimeout(() => {
            openSocket()
        }, delay)
    }

    function sendCommand(cmd: ClientCommand) {
        if (socket && socket.readyState == WebSocket.OPEN) {
            socket.send(JSON.stringify(cmd))
        } else {
            console.warn('Cannot send command while WebSocket is not open:', cmd)
        }
    }

    function disconnect() {
        intentionalClose = true

        if (reconnectTimer) { // if there are still any reconnections
            clearTimeout(reconnectTimer)
            reconnectTimer = null
        }
        if (socket) {
            socket.close()
            socket = null
        }

        status.value = 'disconnected'
    }

    onUnmounted(() => {
        disconnect()
    })

    return {
        status,
        connect,
        disconnect,
        sendCommand,
    }
}