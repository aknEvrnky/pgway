export type ToastKind = 'success' | 'error' | 'info'

export interface ToastItem {
  id: number
  kind: ToastKind
  message: string
}

const TOAST_TTL_MS = 4000

export function useToast() {
  const toasts = useState<ToastItem[]>('app-toasts', () => [])
  const seq = useState('app-toast-seq', () => 0)

  function dismiss(id: number) {
    toasts.value = toasts.value.filter(t => t.id !== id)
  }

  function push(kind: ToastKind, message: string) {
    seq.value += 1
    const id = seq.value
    toasts.value = [...toasts.value, { id, kind, message }]
    if (import.meta.client) {
      window.setTimeout(() => dismiss(id), TOAST_TTL_MS)
    }
  }

  return {
    toasts,
    dismiss,
    success: (message: string) => push('success', message),
    error: (message: string) => push('error', message),
    info: (message: string) => push('info', message),
  }
}
