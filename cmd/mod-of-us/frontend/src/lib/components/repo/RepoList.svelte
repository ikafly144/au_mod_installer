<script lang="ts">
  import { getMods } from "$lib/mocks/mods";

  const mods = getMods();
</script>

<div class="divide-y divide-gray-100">
  {#each mods as mod}
    <div
      class="p-4 hover:bg-gray-50 transition-colors flex items-center justify-between group"
    >
      <div class="flex items-center gap-4">
        <!-- アイコンプレースホルダー -->
        <div
          class="w-10 h-10 rounded bg-gray-200 flex items-center justify-center text-gray-400 font-bold text-xl uppercase"
        >
          {mod.name.substring(0, 1)}
        </div>
        <div>
          <h3 class="font-medium text-gray-900 flex items-center gap-2">
            {mod.name}
            {#if mod.installed}
              <span
                class="text-xs bg-green-100 text-green-700 px-2 py-0.5 rounded-full"
                >Installed</span
              >
            {/if}
            {#if mod.updateAvailable}
              <span
                class="text-xs bg-yellow-100 text-yellow-700 px-2 py-0.5 rounded-full"
                >Update</span
              >
            {/if}
          </h3>
          <p class="text-sm text-gray-500">
            {mod.description}
            <span class="text-gray-400 ml-2">v{mod.version}</span>
          </p>
        </div>
      </div>
      <div
        class="flex items-center gap-2 opacity-0 group-hover:opacity-100 transition-opacity"
      >
        <button
          class="px-3 py-1.5 text-sm bg-white border border-gray-200 text-gray-700 rounded-md hover:bg-gray-50"
          >詳細</button
        >
        {#if mod.installed}
          <button
            class="px-3 py-1.5 text-sm bg-blue-600 text-white rounded-md hover:bg-blue-700"
            >起動</button
          >
        {:else}
          <button
            class="px-3 py-1.5 text-sm bg-gray-800 text-white rounded-md hover:bg-gray-900"
            >インストール</button
          >
        {/if}
      </div>
    </div>
  {/each}
</div>
