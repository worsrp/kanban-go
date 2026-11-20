import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import type { Board, Column, Card, BoardStateResponse, ServerEvent } from '../types/kanban'

const API_BASE = 'http://localhost:8080/api'

export const useBoardStore = defineStore('board', () => {
    const board = ref<Board | null>(null)
    const columns = ref<Column[]>([])
    const cards = ref<Card[]>([])
    const activeUsers = ref<string[]>([])
    const isLoading = ref(false)
    const error = ref<string | null>(null)

    // Group and sort cards by column_id compute
    const cardsByColumn = computed(() => {
        const map: Record<string, Card[]> = {}
        for (const col of columns.value) {
            map[col.id] = []
        }
        for (const card of cards.value) {
            if (!map[card.column_id]) {
                map[card.column_id] = []
            }
            map[card.column_id].push(card)
        }
        for (const colID in map) {
            map[colID].sort((a,b) => a.position - b.position)
        }
        return map
    })

    // Create a new board function
    async function createBoard(title: string): Promise<string | null> {
        isLoading.value = true
        error.value = null

        try{
            const res = await fetch(`${API_BASE}/boards`, {
                method: 'POST',
                headers: {'Content-Type': 'application/json'},
                body: JSON.stringify({title})
            })
        if (!res.ok) throw new Error('Failed to create a new board')
        
        const data: BoardStateResponse = await res.json() 
        board.value = data.board
        columns.value = data.columns ?? []
        cards.value = data.cards ?? []
        
        return data.board.id 
        } catch (err) {
            error.value = (err as Error).message
            return null
        } finally {
            isLoading.value = false
        }
    }
    // Fetch initial board state
    async function fetchBoard(boardId: string): Promise<boolean> {
        isLoading.value = true
        error.value = null
        
        try{
            const res = await fetch(`${API_BASE}/boards/${boardId}`)
            if (!res.ok) throw new Error('Board nott found')
            
            const data: BoardStateResponse = await res.json()
            board.value = data.board
            columns.value = data.columns
            cards.value = data.cards

            return true
        } catch (err) {
            error.value = (err as Error).message
            return false
        } finally {
            isLoading.value = false
        }
    }

    // Apply websoceket event
    function handleServerEvent(event: ServerEvent) {
        switch (event.type) {
            case 'CARD_CREATED': {
                if(!cards.value.some((c) => c.id === event.payload.id)) {
                    cards.value.push(event.payload)
                }
                break
            }

            case 'CARD_MOVED': 

            case 'CARD_UPDATED': {
                const idx = cards.value.findIndex((c) => c.id === event.payload.id)
                if (idx !== -1) { // when card is found
                    cards.value[idx] = event.payload
                }
                break
            }

            case 'CARD_DELETED': {
                cards.value = cards.value.filter((c) => c.id !== event.payload.card_id)
                break
            }

            case 'PRESENCE_UPDATE': {
                activeUsers.value = event.payload.active_users
                break
            }

            case 'ERROR': {
                error.value = event.payload.error
                break
            }
        }
    }

    // Calculate position when dropping a card
    function computeDropPosition(targetColumnId: string, targetIndex: number, draggedCardId: string): number {
        //choose every cards in the target column
        const colCards = (cardsByColumn.value[targetColumnId] ?? []).filter((c) => c.id !== draggedCardId)
        
        if (colCards.length === 0) return 1.0
        if (targetIndex <= 0) return colCards[0].position / 2.0
        if (targetIndex >= colCards.length) return colCards[colCards.length - 1].position + 1.0

        const prev = colCards[targetIndex - 1].position
        const next = colCards[targetIndex].position

        return (prev + next) / 2.0
    }

    return {
        board,
        columns,
        cards,
        activeUsers,
        isLoading,
        error,
        cardsByColumn,
        createBoard,
        fetchBoard,
        handleServerEvent,
        computeDropPosition,
    }
})