import { translate } from '../i18n/LanguageProvider'
import type { Attachment } from '../types'

const sessionKey = 'smoi-session'

export function getToken(): string {
  return localStorage.getItem(sessionKey) || ''
}

export function setToken(token: string): void {
  localStorage.setItem(sessionKey, token)
}

export function clearToken(): void {
  localStorage.removeItem(sessionKey)
}

export async function request<T>(path: string, options: RequestInit = {}): Promise<T> {
  const headers = new Headers(options.headers)
  const token = getToken()

  if (token) headers.set('Authorization', `Bearer ${token}`)
  if (options.body && !(options.body instanceof FormData)) {
    headers.set('Content-Type', 'application/json')
  }

  const response = await fetch(`/api${path}`, {
    ...options,
    headers,
    cache: 'no-store'
  })

  if (!response.ok) {
    const details = await response.json().catch(() => ({})) as { error?: string }
    if (response.status === 401) clearToken()
    throw new Error(details.error || translate('requestFailed', { status: response.status }))
  }

  return response.json() as Promise<T>
}

export async function exchangeLogin(): Promise<void> {
  const response = await fetch('/api/auth/exchange', {
    method: 'POST',
    credentials: 'same-origin',
    cache: 'no-store'
  })

  if (!response.ok) {
    throw new Error(translate('loginExpired'))
  }

  const result = await response.json() as { token: string }
  setToken(result.token)
}

export function login(): void {
  if (window.location.pathname.startsWith('/c/') || window.location.pathname === '/admin') {
    sessionStorage.setItem('smoi-return-path', window.location.pathname)
  }
  window.location.assign('/api/auth/login')
}

export async function logout(): Promise<void> {
  try {
    await request('/auth/logout', { method: 'POST' })
  } finally {
    clearToken()
  }
}

export async function upload(conversationId: string, file: File): Promise<Attachment> {
  const form = new FormData()
  form.append('conversation_id', conversationId)
  form.append('file', file, file.name)

  return request<Attachment>('/attachments', {
    method: 'POST',
    body: form
  })
}

export async function attachmentBlob(id: string): Promise<Blob> {
  const response = await fetch(`/api/attachments/${encodeURIComponent(id)}?inline=1`, {
    headers: { Authorization: `Bearer ${getToken()}` },
    cache: 'no-store'
  })

  if (!response.ok) {
    throw new Error(translate('attachmentUnavailable'))
  }

  return response.blob()
}

export async function downloadAttachment(id: string, filename: string): Promise<void> {
  const blob = await attachmentBlob(id)
  const url = URL.createObjectURL(blob)
  const anchor = document.createElement('a')

  anchor.href = url
  anchor.download = filename
  document.body.appendChild(anchor)
  anchor.click()
  anchor.remove()
  window.setTimeout(() => URL.revokeObjectURL(url), 1000)
}

