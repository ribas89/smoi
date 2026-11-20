import { useLayoutEffect, useState } from 'react'

export const defaultTextSize = 16
export const minimumTextSize = 2
export const maximumTextSize = 32
export const textSizeStep = 2

function savedTextSize(): number {
  try {
    const value = Number(localStorage.getItem('smoi-text-size'))

    if (
      Number.isInteger(value) &&
      value >= minimumTextSize &&
      value <= maximumTextSize &&
      value % textSizeStep === 0
    ) {
      return value
    }
  } catch {
    return defaultTextSize
  }

  return defaultTextSize
}

export function useTextSize() {
  const [textSize, setTextSize] = useState(savedTextSize)

  useLayoutEffect(() => {
    document.documentElement.style.setProperty('--smoi-text-scale', String(textSize / defaultTextSize))

    try {
      localStorage.setItem('smoi-text-size', String(textSize))
    } catch {
      return
    }
  }, [textSize])

  function decreaseTextSize() {
    setTextSize(previous => Math.max(minimumTextSize, previous - textSizeStep))
  }

  function increaseTextSize() {
    setTextSize(previous => Math.min(maximumTextSize, previous + textSizeStep))
  }

  function resetTextSize() {
    setTextSize(defaultTextSize)
  }

  return {
    textSize,
    canDecrease: textSize > minimumTextSize,
    canIncrease: textSize < maximumTextSize,
    decreaseTextSize,
    increaseTextSize,
    resetTextSize
  }
}
