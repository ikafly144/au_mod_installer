<script lang="ts">
  import { getSettings } from "$lib/mocks/settings";

  // @ts-ignore
  let settings = $state(getSettings());
</script>

<div class="p-6">
  <h3 class="text-lg font-medium text-gray-900 mb-4">ゲーム設定</h3>
  {#await settings}
    <p class="text-gray-500 text-sm">読み込み中...</p>
  {:then data}
    <div class="space-y-4">
      <div>
        <label
          class="block text-sm font-medium text-gray-700 mb-1"
          for="install-path">Among Us インストール先</label
        >
        <div class="flex gap-2">
          <input
            id="install-path"
            type="text"
            bind:value={data.game.installPath}
            readonly
            class="flex-1 px-3 py-2 border border-gray-200 rounded-lg bg-gray-50 text-gray-600 text-sm"
          />
          <button
            class="px-4 py-2 border border-gray-200 rounded-lg hover:bg-gray-50 text-sm font-medium"
            >変更</button
          >
        </div>
      </div>
    </div>
  {/await}
</div>
