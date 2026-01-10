<script lang="ts">
  import PageHeader from "$lib/components/common/PageHeader.svelte";
  import RecentActivity from "$lib/components/home/RecentActivity.svelte";
  import QuickLinks from "$lib/components/home/QuickLinks.svelte";
</script>

<div class="space-y-6">
  <PageHeader title="ホーム" description="Modインストーラーへようこそ" />

  <div class="grid grid-cols-1 md:grid-cols-2 gap-6">
    <RecentActivity />
    <QuickLinks />
  </div>
</div>
