<script setup lang="ts">
const { activityEvents } = useDashboardData()
</script>

<template>
  <div class="bg-surface-container rounded-xl ghost-border p-6">
    <div class="flex justify-between items-center mb-6">
      <h3 class="text-sm font-bold text-white">Recent Activity</h3>
      <button class="text-[10px] font-bold text-primary uppercase hover:underline">View History</button>
    </div>
    <div class="space-y-0">
      <MoleculesActivityEvent
        v-for="(event, i) in activityEvents"
        :key="event.id"
        :event="event"
        :is-last="i === activityEvents.length - 1"
      />
    </div>
  </div>
</template>
