import { translate } from '../i18n/LanguageProvider'
import { useEffect, useRef, useState } from 'react'
import { clearToken, exchangeLogin, getToken, logout, request } from '../services/api'
import type { User } from '../types'

export function useSession() {
  const [user, setUser] = useState<User | null>(null)
  const [isReady, setIsReady] = useState(false)
  const [error, setError] = useState('')
  const started = useRef(false)

  useEffect(() => {
    if (started.current) return
    started.current = true

    async function restoreSession() {
      try {
        const params = new URLSearchParams(window.location.search)

        if (params.get('login') === '1') {
          await exchangeLogin()
          const returnPath = sessionStorage.getItem('smoi-return-path')
          sessionStorage.removeItem('smoi-return-path')
          const destination = returnPath?.startsWith('/c/') || returnPath === '/admin'
            ? returnPath
            : window.location.pathname
          window.history.replaceState({}, '', destination)
          window.dispatchEvent(new PopStateEvent('popstate'))
        }

        if (getToken()) {
          const currentUser = await request<User>('/auth/me')
          setUser(currentUser)
        }
      } catch (cause) {
        clearToken()
        setError(errorMessage(cause, translate('unableToSignIn')))
      } finally {
        setIsReady(true)
      }
    }

    void restoreSession()
  }, [])

  async function signOut() {
    try {
      await logout()
    } catch {
      clearToken()
    } finally {
      setUser(null)
    }
  }

  return { user, isReady, error, setError, signOut }
}

export function errorMessage(cause: unknown, fallback: string): string {
  return cause instanceof Error ? cause.message : fallback
}
