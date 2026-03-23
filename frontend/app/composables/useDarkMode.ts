export function useDarkMode() {
  const isDark = useState('dark-mode', () => true)

  function toggle() {
    isDark.value = !isDark.value
    if (import.meta.client) {
      document.documentElement.classList.toggle('dark', isDark.value)
    }
  }

  onMounted(() => {
    document.documentElement.classList.toggle('dark', isDark.value)
  })

  return { isDark, toggle }
}
