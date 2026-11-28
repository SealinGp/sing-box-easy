import { defineComponent, h, cloneVNode, TransitionGroup, Teleport, computed, ref, watch, onBeforeUnmount, nextTick, type PropType, type VNode } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute } from 'vue-router'
import SelectionDock from './SelectionDock.vue'
import { writeTextToClipboard } from '../utils/clipboard'
import CardActions from './CardActions.vue'
import CollectionCell from './CollectionCell.vue'
import CollectionViewToggle from './CollectionViewToggle.vue'
import { elements, nodeText, containsCheckbox } from '../utils/tableVNodes'
import { useCollectionToolbarTarget } from '../composables/useCollectionToolbarTarget'
import { useTableColumns } from '../composables/useTableColumns'
import type { TableColumn } from './Table.vue'

export default defineComponent({
  name: 'TableRecords',
  props: {
    columns: { type: Array as () => TableColumn[], default: () => [] },
    selectedKeys: { type: Array as PropType<string[]>, default: undefined },
    selectionDisabled: Boolean,
    isRowSelectable: Function as PropType<(key: string) => boolean>,
    tableClass: String,
    storageKey: String,
    presentation: { type: Boolean, default: true },
    loading: Boolean,
    transition: Boolean,
    selectable: { type: Boolean, default: true },
  },
  emits: ['update:selectedKeys'],
  setup(props, { slots, emit }) {
    const { t } = useI18n()
    const route = useRoute()
    const root = ref<HTMLElement | null>(null)
    const toolbarTarget = useCollectionToolbarTarget(root)
    const mode = ref<'list' | 'cards'>('list')
    const cardDensity = ref<'compact' | 'comfortable'>('comfortable')
    const internalSelected = ref(new Set<string>())
    watch(() => props.selectedKeys, value => {
      if (value) internalSelected.value = new Set(value)
    }, { immediate: true })
    const selected = computed({
      get: () => internalSelected.value,
      set: (value: Set<string>) => { internalSelected.value = value; emit('update:selectedKeys', [...value]) },
    })
    const error = ref('')
    const copied = ref(false)
    let timer: ReturnType<typeof setTimeout> | undefined
    const key = computed(() => props.storageKey || `collection:${route.path}`)
    watch(key, (value, previous) => {
      if (previous !== undefined) selected.value = new Set()
      try { mode.value = localStorage.getItem(`${value}:view`) === 'cards' ? 'cards' : 'list' } catch {}
      try { cardDensity.value = localStorage.getItem(`${value}:card-density`) === 'compact' ? 'compact' : 'comfortable' } catch {}
    }, { immediate: true })
    function changeMode(value: 'list' | 'cards') {
      mode.value = value
      try { localStorage.setItem(`${key.value}:view`, value) } catch {}
    }
    function changeDensity(value: 'compact' | 'comfortable') {
      cardDensity.value = value
      try { localStorage.setItem(`${key.value}:card-density`, value) } catch {}
    }
    useTableColumns(root, () => mode.value === 'list' && !props.loading, () => `${key.value}:columns`)
    onBeforeUnmount(() => clearTimeout(timer))
    function checkbox(checked: boolean, label: string, change: () => void, partial = false, disabled = false) {
      return h('input', {
        type: 'checkbox', checked, disabled: disabled || props.selectionDisabled, indeterminate: partial, 'aria-label': label,
        class: 'collection-checkbox', onChange: change,
        onClick: (event: Event) => event.stopPropagation(),
        onMousedown: (event: Event) => event.stopPropagation(),
      })
    }
    return () => {
      const rows = elements(slots.default?.(), 'tr')
      const headers = slots.head ? elements(slots.head(), 'th') : props.columns.map(column =>
        h('th', { key: column.key, class: [column.class, column.align === 'right' ? 'col-actions' : ''], 'data-min-width': column.minWidth }, column.label))
      // Preserve pages that already own selection and their business bulk actions.
      const selectionHeader = headers.find(header => String(header.props?.class).includes('col-select') || containsCheckbox(header.children))
      const ownedSelection = !!selectionHeader
      const selectable = props.selectable && !ownedSelection && !props.loading
      const records = rows.filter(row => !elements(row.children, 'td').some(cell => Number(cell.props?.colspan) > 1))
      const identity = (row: VNode) => String(row.key ?? rows.indexOf(row))
      const visible = records.map(identity).filter(id => props.isRowSelectable?.(id) !== false)
      const checked = visible.filter(id => selected.value.has(id))
      const clear = () => {
        if (props.selectionDisabled) return
        error.value = ''
        copied.value = false
        selected.value = new Set()
        void nextTick(() => root.value?.querySelector<HTMLInputElement>('input.collection-checkbox:not(:disabled)')?.focus())
      }
      const toggle = (id: string) => {
        if (props.selectionDisabled || !visible.includes(id)) return
        const next = new Set(selected.value)
        next.has(id) ? next.delete(id) : next.add(id)
        selected.value = next
      }
      const selectAll = () => {
        if (props.selectionDisabled) return
        const next = new Set(selected.value)
        visible.forEach(id => checked.length === visible.length ? next.delete(id) : next.add(id))
        selected.value = next
      }
      const copy = async () => {
        error.value = ''
        const text = records.filter(row => selected.value.has(identity(row))).map(row =>
          elements(row.children, 'td').map(cell => nodeText(cell.children)).join('\t')).join('\n')
        try {
          // DOM text includes component-rendered status labels; never copy hidden data.
          const activeRows = [...(root.value?.querySelectorAll<HTMLElement>('[data-collection-selected="true"]') ?? [])]
          await writeTextToClipboard(activeRows.length ? activeRows.map(row => row.innerText).join('\n') : text)
          copied.value = true
          clearTimeout(timer)
          timer = setTimeout(() => { copied.value = false }, 1800)
        } catch { error.value = t('common.copyFailed') }
      }
      const toolbar = props.presentation || selectable ? h('div', { class: 'collection-toolbar' }, [
        selectable && mode.value === 'cards' ? checkbox(checked.length === visible.length && !!visible.length, t('common.collection.selectAll'), selectAll, !!checked.length && checked.length < visible.length) : null,
        ownedSelection && mode.value === 'cards' ? h('span', { class: 'collection-owned-selection' }, selectionHeader?.children as VNode[]) : null,
        props.presentation ? h(CollectionViewToggle, { modelValue: mode.value, 'onUpdate:modelValue': changeMode }) : null,
        props.presentation && mode.value === 'cards' ? h('div', { class: 'collection-density', role: 'group', 'aria-label': t('common.collection.density') },
          (['compact', 'comfortable'] as const).map(value => h('button', {
            type: 'button', 'aria-pressed': cardDensity.value === value, onClick: () => changeDensity(value),
          }, t(`common.collection.${value}`)))) : null,
      ]) : null
      const labels = headers.map(header => nodeText(header.children))
      const showCards = props.presentation && mode.value === 'cards' && !props.loading
      const tableRows = rows.map(row => {
          const cells = elements(row.children, 'td')
          const detail = cells.some(cell => Number(cell.props?.colspan) > 1)
          return h('tr', { ...row.props, key: row.key ?? undefined, 'data-collection-selected': selected.value.has(identity(row)) }, [
            selectable && !detail ? h('td', { class: 'col-select' }, [checkbox(selected.value.has(identity(row)), t('common.collection.select'), () => toggle(identity(row)), false, !visible.includes(identity(row)))]) : null,
            // Rich cells own their detail interactions; a second tooltip would
            // compete with their controls and mask the component's content.
            ...cells.map(cell => detail ? cloneVNode(cell, { colspan: Number(cell.props?.colspan) + (selectable ? 1 : 0) }) : h('td', cell.props ?? {}, cell.props?.['data-cell-tooltip'] === 'false' ? cell.children as VNode[] : [h(CollectionCell, {}, { default: () => cell.children })])),
          ])
        })
      const content = showCards && slots.cards ? slots.cards({ selected: selected.value, toggle }) : showCards ? h('div', { class: 'collection-cards', role: 'list' }, rows.map(row => {
        const cells = elements(row.children, 'td')
        return h('article', {
          ...row.props, key: row.key ?? undefined, role: 'listitem', class: ['collection-card', row.props?.class],
          'data-collection-selected': selected.value.has(identity(row)),
        }, [
          selectable && records.includes(row) ? checkbox(selected.value.has(identity(row)), t('common.collection.select'), () => toggle(identity(row)), false, !visible.includes(identity(row))) : null,
          ...cells.map((cell, index) => String(cell.props?.class).includes('col-actions') ? h(CardActions, {}, { default: () => cell.children }) : h('section', { class: ['collection-card-field', cell.props?.class] }, [
            labels[index] ? h('div', { class: 'collection-field-label' }, labels[index]) : null,
            h('div', { class: 'collection-field-content' }, cell.children as VNode[]),
          ])),
        ])
      })) : h('table', { class: ['data-table', props.tableClass, { 'collection-compact': props.presentation }] }, [
        h('thead', [h('tr', [
          selectable ? h('th', { class: 'col-select', 'data-min-width': 56 }, [checkbox(checked.length === visible.length && !!visible.length, t('common.collection.selectAll'), selectAll, !!checked.length && checked.length < visible.length)]) : null,
          ...headers,
        ])]),
        props.transition ? h(TransitionGroup, { tag: 'tbody', moveClass: 'ui-row-move' }, () => tableRows) : h('tbody', tableRows),
      ])
      return h('div', { ref: root, 'data-card-density': cardDensity.value, class: ['collection-records', { 'has-selection-dock': selectable && checked.length > 0 }] }, [
        selectable ? h(SelectionDock, { count: checked.length, busy: props.selectionDisabled, copied: copied.value, error: error.value, onCopy: copy, onClear: clear }, slots.selectionActions ? { default: () => slots.selectionActions?.({ selectedKeys: checked, clear }) } : undefined) : null,
        toolbar && toolbarTarget.value ? h(Teleport, { to: toolbarTarget.value }, [toolbar]) : toolbar,
        h('div', { key: showCards ? 'cards' : 'list', class: 'collection-view' }, [content])])
    }
  },
})
