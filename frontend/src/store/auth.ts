import { create } from 'zustand'

interface AuthState {
  token: string | null
  setToken: (token: string) => void
  logout: () => void
}

export const useAuthStore = create<AuthState>((set) => ({
  token: localStorage.getItem('sgw_token'),
  setToken: (token) => {
    localStorage.setItem('sgw_token', token)
    set({ token })
  },
  logout: () => {
    localStorage.removeItem('sgw_token')
    set({ token: null })
  },
}))
