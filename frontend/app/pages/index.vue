<template>
  <div class="space-y-8">
    <section>
      <p class="text-xs font-bold uppercase tracking-widest text-primary mb-1">System Overview</p>
      <h2 class="text-3xl font-black tracking-tighter text-on-surface">Kinetic Console</h2>
    </section>

    <OrganismsMetricCardsGrid />

    <section class="grid grid-cols-1 lg:grid-cols-3 gap-6">
      <OrganismsComingSoonPanel
        class="lg:col-span-2"
        title="Throughput Trends"
        subtitle="Historical request volume"
      />
      <OrganismsComingSoonPanel
        title="Proxy Health"
        subtitle="Pool distribution"
      />
    </section>

    <section class="grid grid-cols-1 lg:grid-cols-3 gap-6">
      <OrganismsAgentsPanel class="lg:col-span-2" />
      <OrganismsQuickActions />
    </section>
  </div>
</template>
