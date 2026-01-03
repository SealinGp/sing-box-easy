<script setup lang="ts">
import { ref } from 'vue'
import Select from './Select.vue'

// Example data for client-side search
const countries = [
  { label: 'United States', value: 'US' },
  { label: 'Canada', value: 'CA' },
  { label: 'Mexico', value: 'MX' },
  { label: 'Brazil', value: 'BR' },
  { label: 'United Kingdom', value: 'UK' },
  { label: 'France', value: 'FR' },
  { label: 'Germany', value: 'DE' },
  { label: 'Italy', value: 'IT' },
  { label: 'Spain', value: 'ES' },
  { label: 'China', value: 'CN' },
  { label: 'Japan', value: 'JP' },
  { label: 'South Korea', value: 'KR' },
  { label: 'India', value: 'IN' },
  { label: 'Australia', value: 'AU' },
  { label: 'New Zealand', value: 'NZ' },
]

// Example data for server-side search simulation
const allUsers = [
  { label: 'John Doe', value: 1 },
  { label: 'Jane Smith', value: 2 },
  { label: 'Bob Johnson', value: 3 },
  { label: 'Alice Williams', value: 4 },
  { label: 'Charlie Brown', value: 5 },
  { label: 'Diana Davis', value: 6 },
  { label: 'Edward Miller', value: 7 },
  { label: 'Fiona Wilson', value: 8 },
  { label: 'George Moore', value: 9 },
  { label: 'Helen Taylor', value: 10 },
]

// Selected values
const selectedCountry = ref<string | null>(null)
const selectedUser = ref<number | null>(null)
const selectedBasic = ref<string | null>(null)

// Server-side search options (starts empty)
const serverSearchOptions = ref<typeof allUsers>([])
const serverLoading = ref(false)

// Handle server-side search (simulated)
const handleServerSearch = async (query: string, setLoading: (loading: boolean) => void) => {
  // Simulate API call with delay
  await new Promise(resolve => setTimeout(resolve, 500))

  // Filter users based on query (simulating server-side filtering)
  const filtered = allUsers.filter(user =>
    user.label.toLowerCase().includes(query.toLowerCase())
  )

  serverSearchOptions.value = filtered
  setLoading(false)
}

// Basic options without search
const basicOptions = [
  { label: 'Option 1', value: 'opt1' },
  { label: 'Option 2', value: 'opt2' },
  { label: 'Option 3', value: 'opt3' },
  { label: 'Disabled Option', value: 'opt4', disabled: true },
]
</script>

<template>
  <div class="space-y-8 p-6">
    <h2 class="text-2xl font-bold text-gray-900 dark:text-gray-100">Select Component Examples</h2>

    <!-- Basic Select without search -->
    <div class="space-y-2">
      <h3 class="text-lg font-semibold text-gray-800 dark:text-gray-200">Basic Select (No Search)</h3>
      <p class="text-sm text-gray-600 dark:text-gray-400">Standard select without search functionality</p>
      <Select
        v-model="selectedBasic"
        :options="basicOptions"
        placeholder="Choose an option"
        label="Basic Selection"
      />
      <p class="text-sm text-gray-600 dark:text-gray-400">
        Selected: {{ selectedBasic || 'None' }}
      </p>
    </div>

    <!-- Client-side search -->
    <div class="space-y-2">
      <h3 class="text-lg font-semibold text-gray-800 dark:text-gray-200">Client-Side Search</h3>
      <p class="text-sm text-gray-600 dark:text-gray-400">Search is performed locally in the browser</p>
      <Select
        v-model="selectedCountry"
        :options="countries"
        :searchable="true"
        placeholder="Select a country"
        search-placeholder="Type to search countries..."
        no-options-text="No countries found"
        label="Country"
      />
      <p class="text-sm text-gray-600 dark:text-gray-400">
        Selected: {{ selectedCountry || 'None' }}
      </p>
    </div>

    <!-- Server-side search -->
    <div class="space-y-2">
      <h3 class="text-lg font-semibold text-gray-800 dark:text-gray-200">Server-Side Search</h3>
      <p class="text-sm text-gray-600 dark:text-gray-400">Search queries are sent to server (simulated with delay)</p>
      <Select
        v-model="selectedUser"
        :options="serverSearchOptions"
        :searchable="true"
        :server-side-search="true"
        :loading="serverLoading"
        :debounce="300"
        placeholder="Search for a user"
        search-placeholder="Type to search users..."
        no-options-text="No users found"
        label="User"
        @search="handleServerSearch"
      />
      <p class="text-sm text-gray-600 dark:text-gray-400">
        Selected: {{ selectedUser || 'None' }}
      </p>
    </div>

    <!-- Usage Instructions -->
    <div class="border-t border-gray-200 dark:border-gray-700 pt-6">
      <h3 class="text-lg font-semibold text-gray-800 dark:text-gray-200 mb-4">Usage Instructions</h3>
      <div class="space-y-3 text-sm text-gray-600 dark:text-gray-400">
        <div>
          <strong>Basic Usage:</strong>
          <pre class="bg-gray-100 dark:bg-gray-900 p-2 rounded mt-1"><code>&lt;Select v-model="selected" :options="options" /&gt;</code></pre>
        </div>

        <div>
          <strong>Client-Side Search:</strong>
          <pre class="bg-gray-100 dark:bg-gray-900 p-2 rounded mt-1"><code>&lt;Select
  v-model="selected"
  :options="options"
  :searchable="true"
/&gt;</code></pre>
        </div>

        <div>
          <strong>Server-Side Search:</strong>
          <pre class="bg-gray-100 dark:bg-gray-900 p-2 rounded mt-1"><code>&lt;Select
  v-model="selected"
  :options="filteredOptions"
  :searchable="true"
  :server-side-search="true"
  @search="handleSearch"
/&gt;

// In script:
const handleSearch = (query, setLoading) => {
  // Perform API call
  fetchOptions(query).then(results => {
    filteredOptions.value = results
    setLoading(false)
  })
}</code></pre>
        </div>

        <div class="mt-4">
          <strong>Props:</strong>
          <ul class="list-disc list-inside mt-2 space-y-1">
            <li><code>searchable</code> - Enable search functionality (default: false)</li>
            <li><code>serverSideSearch</code> - Use server-side search instead of client-side (default: false)</li>
            <li><code>searchPlaceholder</code> - Placeholder text for search input</li>
            <li><code>noOptionsText</code> - Text shown when no options match search</li>
            <li><code>debounce</code> - Debounce delay for server-side search in ms (default: 250)</li>
            <li><code>loading</code> - Show loading state</li>
          </ul>
        </div>

        <div class="mt-4">
          <strong>Events:</strong>
          <ul class="list-disc list-inside mt-2 space-y-1">
            <li><code>@search</code> - Emitted when searching (server-side mode only). Params: (query, setLoading)</li>
            <li><code>@update:modelValue</code> - Emitted when selection changes</li>
          </ul>
        </div>
      </div>
    </div>
  </div>
</template>