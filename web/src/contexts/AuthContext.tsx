import { createContext, useContext, useState, useEffect, ReactNode, useCallback } from 'react'
import { api, User, onSessionExpired } from '../lib/api'

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

  // Handle session expiry from api.ts - this is called when refresh token is also invalid
  const handleSessionExpired = useCallback(() => {
    setUser(null)
  }, [])

  useEffect(() => {
    // Register session expiry callback
    onSessionExpired(handleSessionExpired)

    const token = localStorage.getItem('access_token')
    if (token) {
      loadUser()
    } else {
      setIsLoading(false)
    }
  }, [handleSessionExpired])

  async function loadUser() {
    try {
      // api.ts now handles 401 + automatic refresh internally
      const userData = await api.getProfile()
      setUser(userData)
    } catch {
      // If we get here, either there's no valid session or a non-auth error occurred
      // Clear tokens just in case
      localStorage.removeItem('access_token')
      localStorage.removeItem('refresh_token')
      setUser(null)
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
