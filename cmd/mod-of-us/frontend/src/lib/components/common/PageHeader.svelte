<script lang="ts">
  let { title, description } = $props<{ title: string; description: string }>();
</script>

<header class="mb-6">
  <h1 class="text-2xl font-bold text-gray-800">{title}</h1>
  <p class="text-gray-500 mt-1">{description}</p>
</header>
