export interface LabPalette {
  id: string
  name: string
  mode: 'light' | 'dark'
  description: string
  recommended?: boolean
  colors: readonly string[]
  canvas: string
  surface: string
  raised: string
  field: string
  border: string
  fieldBorder: string
  ink: string
  muted: string
  primary: string
  primaryHover: string
  onPrimary: string
  accent: string
  warm: string
  success: string
  warning: string
  danger: string
  nav: string
  navInk: string
  navMuted: string
  navActive: string
}

export const LOGO_COLORS = ['#17283E', '#8FBAFF'] as const
export const SELECTED_PALETTE_ID = 'sandstone'
export const labPalettes: readonly LabPalette[] = [
  {
    id: 'harbor', name: 'Harbor', mode: 'light',
    description: 'Blue mist, navy navigation and a restrained teal accent',
    colors: [...LOGO_COLORS, '#DFE8F1', '#2D7D83', '#C89B5E'],
    canvas: '#DFE8F1', surface: '#EDF3F8', raised: '#E1EBF4', field: '#F5F8FC',
    border: '#B7CADC', fieldBorder: '#768FA8', ink: '#17283E', muted: '#4B6076',
    primary: '#235EA4', primaryHover: '#194C89', onPrimary: '#F5F8FC',
    accent: '#28747B', warm: '#95682E', success: '#176858', warning: '#825813', danger: '#B13F50',
    nav: '#17283E', navInk: '#EDF3FC', navMuted: '#BACCE0', navActive: '#2A486A',
  },
  {
    id: 'sandstone', name: 'Sandstone', mode: 'light', recommended: true,
    description: 'Warm mineral surfaces, ink blue and muted sage',
    colors: [...LOGO_COLORS, '#E9E2D5', '#53766F', '#BD8C4A'],
    canvas: '#E9E2D5', surface: '#F5EFE3', raised: '#E7DFD0', field: '#FAF5EB',
    border: '#CABDA8', fieldBorder: '#928977', ink: '#243448', muted: '#5B625F',
    primary: '#315B83', primaryHover: '#244866', onPrimary: '#FAF5EB',
    accent: '#53766F', warm: '#91672E', success: '#356757', warning: '#7C5524', danger: '#A43D43',
    nav: '#243448', navInk: '#F5EFE3', navMuted: '#C8C8C1', navActive: '#3D5266',
  },
  {
    id: 'midnight', name: 'Midnight', mode: 'dark',
    description: 'Layered navy, the logo’s blue and sea-glass highlights',
    colors: [...LOGO_COLORS, '#0C1726', '#70CBB4', '#E2B679'],
    canvas: '#0C1726', surface: '#17283E', raised: '#223951', field: '#112236',
    border: '#405A75', fieldBorder: '#68829D', ink: '#E6EDF7', muted: '#ADC0D6',
    primary: '#8FBAFF', primaryHover: '#ACCDFE', onPrimary: '#14263D',
    accent: '#70CBB4', warm: '#E2B679', success: '#7DD2B4', warning: '#E8BD75', danger: '#F19AA9',
    nav: '#101E30', navInk: '#E6EDF7', navMuted: '#ADC0D6', navActive: '#284768',
  },
  {
    id: 'graphite', name: 'Graphite', mode: 'dark',
    description: 'Soft charcoal, glacier blue and a warm copper counterpoint',
    colors: [...LOGO_COLORS, '#242C36', '#83B9AB', '#C6A278'],
    canvas: '#181D24', surface: '#242C36', raised: '#303B48', field: '#1D242E',
    border: '#536171', fieldBorder: '#7A8798', ink: '#E8E9EC', muted: '#B4BFCD',
    primary: '#8FBAFF', primaryHover: '#ACCDFE', onPrimary: '#17283E',
    accent: '#83B9AB', warm: '#C6A278', success: '#89C8B5', warning: '#E6BE78', danger: '#F0A3A7',
    nav: '#121B28', navInk: '#E8E9EC', navMuted: '#B4BFCD', navActive: '#30435B',
  },
]

export function getLabPalette(id: unknown): LabPalette {
  return labPalettes.find(palette => palette.id === id) ?? labPalettes.find(palette => palette.id === SELECTED_PALETTE_ID)!
}
export function coolorsUrl(palette: LabPalette): string {
  return `https://coolors.co/${palette.colors.map(color => color.slice(1).toLowerCase()).join('-')}`
}
export function contrastRatio(foreground: string, background: string): number {
  const luminance = (hex: string) => {
    const rgb = [1, 3, 5].map(index => parseInt(hex.slice(index, index + 2), 16) / 255)
      .map(value => value <= .04045 ? value / 12.92 : ((value + .055) / 1.055) ** 2.4)
    return rgb[0]! * .2126 + rgb[1]! * .7152 + rgb[2]! * .0722
  }
  const a = luminance(foreground), b = luminance(background)
  return (Math.max(a, b) + .05) / (Math.min(a, b) + .05)
}
function mix(a: string, b: string, amount: number): string {
  return '#' + [1, 3, 5].map(index => Math.round(
    parseInt(a.slice(index, index + 2), 16) * (1 - amount) + parseInt(b.slice(index, index + 2), 16) * amount,
  ).toString(16).padStart(2, '0')).join('')
}

/** Applied only inside the isolated lab preview document, including its overlays. */
export function paletteTokens(palette: LabPalette): Record<string, string> {
  const dark = palette.mode === 'dark'
  const tokens: Record<string, string> = {
    '--color-primary': palette.primary, '--color-primary-hover': palette.primaryHover,
    '--color-on-primary': palette.onPrimary,
    '--color-primary-light': '#8FBAFF', '--color-primary-dark': palette.primary,
    '--color-bg-primary': palette.canvas, '--color-bg-secondary': palette.surface,
    '--color-bg-tertiary': palette.raised, '--color-text-primary': palette.ink,
    '--color-text-secondary': palette.muted, '--color-text-tertiary': palette.muted,
    '--color-text-inverse': palette.onPrimary, '--color-border': palette.border,
    '--color-border-light': palette.border, '--color-border-dark': palette.fieldBorder,
    '--color-success': palette.success, '--color-warning': palette.warning, '--color-danger': palette.danger,
    '--color-info': palette.primary,
    '--control-bg': palette.field, '--control-bg-hover': mix(palette.field, palette.primary, .06),
    '--control-bg-disabled': palette.raised, '--control-border': palette.fieldBorder,
    '--control-border-hover': palette.primary, '--control-border-focus': palette.primary,
    '--control-placeholder': palette.muted, '--control-text': palette.ink,
    '--glass-bg': palette.surface, '--glass-bg-strong': palette.surface,
    '--glass-border': palette.border, '--glass-border-muted': palette.border,
    '--glass-highlight': 'inset 0 0 0 transparent',
    '--shadow-surface': '0 1px 2px rgb(0 0 0 / 3%)',
    '--shadow-float': dark ? '0 8px 24px rgb(0 0 0 / 30%)' : '0 8px 24px rgb(23 40 62 / 14%)',
    '--table-head-bg': palette.raised, '--table-row-hover': mix(palette.surface, palette.primary, .07),
    '--table-row-selected': mix(palette.surface, palette.primary, .14),
    '--palette-on-primary': palette.onPrimary, '--palette-accent': palette.accent,
    '--palette-warm': palette.warm, '--palette-nav': palette.nav,
    '--palette-nav-ink': palette.navInk, '--palette-nav-muted': palette.navMuted,
    '--palette-nav-active': palette.navActive,
    '--color-gray-900': dark ? palette.canvas : palette.ink,
    '--color-gray-800': dark ? palette.surface : mix(palette.ink, palette.surface, .1),
    '--color-gray-700': dark ? palette.raised : palette.ink,
    '--color-gray-600': palette.muted, '--color-gray-500': palette.muted, '--color-gray-400': palette.muted,
    '--color-gray-300': dark ? palette.muted : palette.border,
    '--color-gray-200': dark ? palette.ink : palette.raised,
    '--color-gray-100': dark ? palette.ink : palette.raised,
    '--color-gray-50': dark ? palette.ink : palette.field,
  }
  const brand = '#235EA4'
  for (const [step, amount] of [[50, .94], [100, .86], [200, .72], [300, .52], [400, .3], [500, .12]] as const) {
    tokens[`--color-primary-${step}`] = mix(brand, '#F1F6FC', amount)
  }
  tokens['--color-primary-600'] = palette.primary
  for (const [step, amount] of [[700, .1], [800, .28], [900, .5], [950, .76]] as const) {
    tokens[`--color-primary-${step}`] = mix(brand, '#0C1726', amount)
  }
  return tokens
}
