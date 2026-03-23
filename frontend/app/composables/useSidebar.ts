const STORAGE_KEY = 'pgway-sidebar-collapsed'

export function useSidebar() {
  const isCollapsed = useState('sidebar-collapsed', () => false)

  onMounted(() => {
    isCollapsed.value = localStorage.getItem(STORAGE_KEY) === 'true'
  })

  function toggle() {
    isCollapsed.value = !isCollapsed.value
    if (import.meta.client) {
      localStorage.setItem(STORAGE_KEY, String(isCollapsed.value))
    }
  }

  const width = computed(() => isCollapsed.value ? '4rem' : '16rem')

  return { isCollapsed, toggle, width }
}
