<script lang="ts">
  import { getRecentActivities } from "$lib/mocks/activity";

  const activities = getRecentActivities();

  function getActivityColor(type: string) {
    switch (type) {
      case "detect":
        return "bg-green-500";
      case "install":
        return "bg-blue-500";
      case "update":
        return "bg-yellow-500";
      case "error":
        return "bg-red-500";
      default:
        return "bg-gray-400";
    }
  }
</script>

<div
  class="bg-white p-6 rounded-xl shadow-sm border border-gray-100 hover:shadow-md transition-shadow"
>
  <h2 class="text-lg font-semibold mb-2">最近のアクティビティ</h2>
  <div class="space-y-3">
    {#each activities as activity}
      <div
        class="flex items-center gap-3 text-sm text-gray-600 p-2 bg-gray-50 rounded"
      >
        <div
          class={`w-2 h-2 rounded-full ${getActivityColor(activity.type)}`}
        ></div>
        <span>{activity.message}</span>
      </div>
    {/each}
  </div>
</div>
