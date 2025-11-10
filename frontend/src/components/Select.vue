<script setup lang="ts">
import { Listbox, ListboxButton, ListboxOptions, ListboxOption } from '@headlessui/vue'
import { CheckIcon, ChevronUpDownIcon } from '@heroicons/vue/24/outline'

interface Option {
  label: string
  value: string | number
  disabled?: boolean
}

interface Props {
  modelValue?: string | number | null
  options: Option[]
  placeholder?: string
  label?: string
  error?: string
  required?: boolean
  disabled?: boolean
  fullWidth?: boolean
}

const props = withDefaults(defineProps<Props>(), {
  fullWidth: true,
  disabled: false,
  required: false,
  placeholder: 'Select an option',
})

const emit = defineEmits<{
  'update:modelValue': [value: string | number]
}>()

const selectedLabel = (value: string | number | null | undefined) => {
  const option = props.options.find((o) => o.value === value)
  return option?.label || props.placeholder
}
</script>

<template>
  <div :class="fullWidth ? 'w-full' : ''">
    <label v-if="label" class="block text-sm font-medium text-gray-700 mb-1">
      {{ label }}
      <span v-if="required" class="text-red-500 ml-1">*</span>
    </label>

    <Listbox
      :model-value="modelValue"
      :disabled="disabled"
      @update:model-value="(value) => emit('update:modelValue', value)"
    >
      <div class="relative">
        <ListboxButton
          :class="[
            'relative w-full cursor-default rounded-lg border bg-white py-2 pl-3 pr-10 text-left focus:outline-none focus:ring-2 focus:ring-offset-0 transition-colors',
            error
              ? 'border-red-300 focus:border-red-500 focus:ring-red-500'
              : 'border-gray-300 focus:border-blue-500 focus:ring-blue-500',
            disabled ? 'bg-gray-50 cursor-not-allowed' : '',
          ]"
        >
          <span class="block truncate" :class="modelValue == null ? 'text-gray-400' : 'text-gray-900'">
            {{ selectedLabel(modelValue) }}
          </span>
          <span class="pointer-events-none absolute inset-y-0 right-0 flex items-center pr-2">
            <ChevronUpDownIcon class="h-5 w-5 text-gray-400" aria-hidden="true" />
          </span>
        </ListboxButton>

        <transition
          leave-active-class="transition duration-100 ease-in"
          leave-from-class="opacity-100"
          leave-to-class="opacity-0"
        >
          <ListboxOptions
            class="absolute z-10 mt-1 max-h-60 w-full overflow-auto rounded-md bg-white py-1 text-base shadow-lg ring-1 ring-black ring-opacity-5 focus:outline-none"
          >
            <ListboxOption
              v-for="option in options"
              :key="option.value"
              v-slot="{ active, selected }"
              :value="option.value"
              :disabled="option.disabled"
              as="template"
            >
              <li
                :class="[
                  active ? 'bg-blue-100 text-blue-900' : 'text-gray-900',
                  option.disabled ? 'opacity-50 cursor-not-allowed' : 'cursor-pointer',
                  'relative select-none py-2 pl-10 pr-4',
                ]"
              >
                <span :class="[selected ? 'font-semibold' : 'font-normal', 'block truncate']">
                  {{ option.label }}
                </span>
                <span
                  v-if="selected"
                  class="absolute inset-y-0 left-0 flex items-center pl-3 text-blue-600"
                >
                  <CheckIcon class="h-5 w-5" aria-hidden="true" />
                </span>
              </li>
            </ListboxOption>
          </ListboxOptions>
        </transition>
      </div>
    </Listbox>

    <p v-if="error" class="mt-1 text-sm text-red-600">{{ error }}</p>
  </div>
</template>
