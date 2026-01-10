<script lang="ts">
  import PageHeader from "$lib/components/common/PageHeader.svelte";
  import GeneralSettings from "$lib/components/settings/GeneralSettings.svelte";
  import GameSettings from "$lib/components/settings/GameSettings.svelte";
</script>

<div class="max-w-2xl space-y-6">
  <PageHeader title="設定" description="アプリケーションの設定を変更します" />

  <div
    class="bg-white rounded-xl shadow-sm border border-gray-100 divide-y divide-gray-100"
  >
    <GeneralSettings />
    <GameSettings />

    <div class="p-6">
      <button
        class="text-red-600 hover:text-red-700 text-sm font-medium flex items-center gap-2"
      >
        <svg
          xmlns="http://www.w3.org/2000/svg"
          class="w-4 h-4"
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          stroke-width="2"
          stroke-linecap="round"
          stroke-linejoin="round"
          ><path
            d="M3 6h18M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"
          /></svg
        >
        すべてのデータをリセット
      </button>
    </div>
  </div>
</div>
