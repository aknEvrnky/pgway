<template>
  <div class="space-y-8">
    <!-- Welcome Section -->
    <section>
      <p class="text-xs font-bold uppercase tracking-widest text-primary mb-1">System Overview</p>
      <h2 class="text-3xl font-black tracking-tighter text-on-surface">Kinetic Console</h2>
    </section>

    <!-- Metric Cards -->
    <OrganismsMetricCardsGrid />

    <!-- Charts Row -->
    <section class="grid grid-cols-1 lg:grid-cols-3 gap-6">
      <OrganismsThroughputChart class="lg:col-span-2" />
      <OrganismsHealthDoughnut />
    </section>

    <!-- Bottom Row -->
    <section class="grid grid-cols-1 lg:grid-cols-3 gap-6">
      <OrganismsActivityFeed class="lg:col-span-2" />
      <OrganismsQuickActions />
    </section>
  </div>
</template>
