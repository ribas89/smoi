import { useEffect, useState } from 'react'

type Theme = 'light' | 'dark'

function preferredTheme(): Theme {
  const saved = localStorage.getItem('smoi-theme')

  if (saved === 'light' || saved === 'dark') return saved

  return window.matchMedia('(prefers-color-scheme: dark)').matches
    ? 'dark'
    : 'light'
}

export function useTheme() {
  const [theme, setTheme] = useState<Theme>(preferredTheme)

  useEffect(() => {
    document.documentElement.dataset.bsTheme = theme
    localStorage.setItem('smoi-theme', theme)
  }, [theme])

  function toggleTheme() {
    setTheme(previous => previous === 'dark' ? 'light' : 'dark')
  }

  return { theme, toggleTheme }
}
