import { Fragment, Text, isVNode, type VNode } from 'vue'

/** Vue loops and conditional templates produce nested fragments. */
export function elements(nodes: unknown, tag: string): VNode[] {
  if (!Array.isArray(nodes)) return []
  return nodes.flatMap(node => {
    if (!isVNode(node)) return []
    if (node.type === tag) return [node]
    if (node.type === Fragment) return elements(node.children, tag)
    return []
  })
}

export function nodeText(nodes: unknown): string {
  if (typeof nodes === 'string' || typeof nodes === 'number') return String(nodes)
  if (Array.isArray(nodes)) return nodes.map(nodeText).join(' ').replace(/\s+/g, ' ').trim()
  if (!isVNode(nodes)) return ''
  if (nodes.type === Text || typeof nodes.type === 'string') return nodeText(nodes.children)
  return ''
}

export function containsCheckbox(nodes: unknown): boolean {
  if (Array.isArray(nodes)) return nodes.some(containsCheckbox)
  if (!isVNode(nodes)) return false
  return (nodes.type === 'input' && nodes.props?.type === 'checkbox') || containsCheckbox(nodes.children)
}
