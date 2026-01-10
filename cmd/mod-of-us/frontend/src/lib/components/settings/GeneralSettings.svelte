<script lang="ts">
  import { getSettings } from "$lib/mocks/settings";
  import { onMount } from "svelte";

  // @ts-ignore
  let settings = $state(getSettings());
</script>

<div class="p-6">
  <h3 class="text-lg font-medium text-gray-900 mb-4">一般設定</h3>
  {#await settings}
    <p class="text-gray-500 text-sm">読み込み中...</p>
  {:then data}
    <div class="space-y-4">
      <div class="flex items-center justify-between">
        <div>
          <label
            class="block text-sm font-medium text-gray-700"
            for="language-select">言語</label
          >
          <p class="text-sm text-gray-500">表示言語を選択します</p>
        </div>
        <select
          id="language-select"
          class="px-3 py-2 border border-gray-200 rounded-lg bg-white"
          bind:value={data.general.language}
        >
          <option value="ja">日本語</option>
          <option value="en">English</option>
        </select>
      </div>

      <div class="flex items-center justify-between">
        <div>
          <label
            class="block text-sm font-medium text-gray-700"
            for="darkmode-toggle">ダークモード</label
          >
          <p class="text-sm text-gray-500">外観テーマを切り替えます</p>
        </div>
        <button
          id="darkmode-toggle"
          class={`w-11 h-6 rounded-full relative cursor-pointer transition-colors ${data.general.darkMode ? "bg-blue-600" : "bg-gray-200"}`}
          onclick={() => (data.general.darkMode = !data.general.darkMode)}
        >
          <div
            class={`w-4 h-4 bg-white rounded-full absolute top-1 shadow-sm transition-all ${data.general.darkMode ? "left-6" : "left-1"}`}
          ></div>
        </button>
      </div>
    </div>
  {/await}
</div>
