import { createContext, useContext, useState, useEffect, ReactNode } from 'react'
import { api, User } from '../lib/api'

interface AuthContextType {
  user: User | null
  isLoading: boolean
  login: (accessToken: string, refreshToken: string, user: User) => void
  logout: () => void
}

const AuthContext = createContext<AuthContextType | undefined>(undefined)

export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<User | null>(null)
  const [isLoading, setIsLoading] = useState(true)

  useEffect(() => {
    const token = localStorage.getItem('access_token')
    if (token) {
      loadUser()
    } else {
      setIsLoading(false)
    }
  }, [])

  async function loadUser() {
    try {
      const userData = await api.getProfile()
      setUser(userData)
    } catch (error) {
      // Token invalid, try refresh
      const refreshToken = localStorage.getItem('refresh_token')
      if (refreshToken) {
        try {
          const tokens = await api.refresh(refreshToken)
          localStorage.setItem('access_token', tokens.access_token)
          localStorage.setItem('refresh_token', tokens.refresh_token)
          const userData = await api.getProfile()
          setUser(userData)
        } catch {
          // Refresh failed, clear tokens
          localStorage.removeItem('access_token')
          localStorage.removeItem('refresh_token')
        }
      }
    } finally {
      setIsLoading(false)
    }
  }

  function login(accessToken: string, refreshToken: string, userData: User) {
    localStorage.setItem('access_token', accessToken)
    localStorage.setItem('refresh_token', refreshToken)
    setUser(userData)
  }

  async function logout() {
    const refreshToken = localStorage.getItem('refresh_token')
    if (refreshToken) {
      try {
        await api.logout(refreshToken)
      } catch {
        // Ignore logout errors
      }
    }
    localStorage.removeItem('access_token')
    localStorage.removeItem('refresh_token')
    setUser(null)
  }

  return (
    <AuthContext.Provider value={{ user, isLoading, login, logout }}>
      {children}
    </AuthContext.Provider>
  )
}

export function useAuth() {
  const context = useContext(AuthContext)
  if (context === undefined) {
    throw new Error('useAuth must be used within an AuthProvider')
  }
  return context
}
