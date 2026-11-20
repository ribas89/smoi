export function conversationFromPath(): string {
  return window.location.pathname.match(/^\/c\/([^/]+)\/?$/)?.[1] || ''
}

export function conversationPath(id: string): string {
  return `/c/${encodeURIComponent(id)}`
}

export function navigate(path: string, replace = false): void {
  if (window.location.pathname === path) return
  window.history[replace ? 'replaceState' : 'pushState']({}, '', path)
}
