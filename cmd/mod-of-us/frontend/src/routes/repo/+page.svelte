<script lang="ts">
  import PageHeader from "$lib/components/common/PageHeader.svelte";
  import RepoFilter from "$lib/components/repo/RepoFilter.svelte";
  import RepoList from "$lib/components/repo/RepoList.svelte";
</script>

<div class="space-y-6">
  <div class="flex justify-between items-center">
    <PageHeader
      title="リポジトリ"
      description="インストール済みおよび利用可能なModを管理します"
    />
    <button
      class="px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition-colors flex items-center gap-2"
    >
      <svg
        xmlns="http://www.w3.org/2000/svg"
        class="w-4 h-4"
        viewBox="0 0 24 24"
        fill="none"
        stroke="currentColor"
        stroke-width="2"
        stroke-linecap="round"
        stroke-linejoin="round"><path d="M12 5v14M5 12h14" /></svg
      >
      リポジトリ追加
    </button>
  </div>

  <div
    class="bg-white rounded-xl shadow-sm border border-gray-100 overflow-hidden"
  >
    <RepoFilter />
    <RepoList />
  </div>
</div>
