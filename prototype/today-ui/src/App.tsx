// PROTOTYPE — Zenith Today / Upcoming / sidebar. Question: what should list density, the task row, checklist
// expansion, headings, When/Deadline display, drag-reorder and keyboard nav feel like?
// Three structurally different variants on one page, switchable with ?variant=A|B|C or the ←/→ keys... except
// arrows drive list selection here, so use [ and ] to switch variants too.
import { useEffect, useState } from 'react'
import { ChevronLeft, ChevronRight } from 'lucide-react'
import { Provider } from './store'
import { VariantA } from './variants/A'
import { VariantB } from './variants/B'
import { VariantC } from './variants/C'

const VARIANTS = [
  { key: 'A', name: 'Calm sidebar, inline expansion', C: VariantA },
  { key: 'B', name: 'Dense table + inspector', C: VariantB },
  { key: 'C', name: 'Top tabs, week board, quick look', C: VariantC },
]
const read = () => new URLSearchParams(location.search).get('variant')?.toUpperCase() ?? 'A'

export function App() {
  const [key, setKey] = useState(read())
  const idx = Math.max(0, VARIANTS.findIndex(v => v.key === key))
  const go = (i: number) => {
    const k = VARIANTS[(i + VARIANTS.length) % VARIANTS.length].key; setKey(k)
    const p = new URLSearchParams(location.search); p.set('variant', k); history.replaceState(null, '', '?' + p)
  }
  useEffect(() => {
    const h = (e: KeyboardEvent) => {
      if (/^(INPUT|TEXTAREA)$/.test((e.target as HTMLElement).tagName)) return
      if (e.key === '[') go(idx - 1); if (e.key === ']') go(idx + 1)
    }
    window.addEventListener('keydown', h); return () => window.removeEventListener('keydown', h)
  })
  const V = VARIANTS[idx].C
  return (
    <Provider>
      <V />
      <div className="fixed bottom-4 left-1/2 z-50 flex -translate-x-1/2 items-center gap-3 rounded-full bg-neutral-900 px-3 py-2 text-sm text-white shadow-xl">
        <button onClick={() => go(idx - 1)} className="rounded-full p-1 hover:bg-white/15" aria-label="Previous variant"><ChevronLeft size={16} /></button>
        <span className="min-w-56 text-center"><b>{VARIANTS[idx].key}</b> · {VARIANTS[idx].name}</span>
        <button onClick={() => go(idx + 1)} className="rounded-full p-1 hover:bg-white/15" aria-label="Next variant"><ChevronRight size={16} /></button>
        <span className="hidden text-xs text-white/50 md:inline">[ ] switch</span>
      </div>
    </Provider>
  )
}
