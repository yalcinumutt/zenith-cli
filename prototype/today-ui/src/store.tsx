// PROTOTYPE — in-memory store, keyboard nav and drag reorder shared by all variants
import { createContext, useCallback, useContext, useEffect, useMemo, useRef, useState, type ReactNode } from 'react'
import { seed, sections, flat, children, LISTS, TODAY, type Task, type ViewId, type Section } from './data'

interface Store {
  tasks: Task[]; view: ViewId; setView: (v: ViewId) => void; secs: Section[]; ids: string[]
  toggle: (id: string) => void; reorder: (dragId: string, targetId: string) => void
  moveBy: (id: string, dir: -1 | 1) => void; setDate: (id: string, date: string) => void
  flash?: string; add: (title: string) => void
}
const Ctx = createContext<Store>(null as any)
export const useStore = () => useContext(Ctx)

const readView = (): ViewId => (new URLSearchParams(location.search).get('view') as ViewId) || 'today'

export function Provider({ children: kids }: { children: ReactNode }) {
  const [tasks, setTasks] = useState<Task[]>(seed)
  const [view, setViewState] = useState<ViewId>(readView())
  const [flash, setFlash] = useState<string>()
  const say = (m: string) => { setFlash(m); setTimeout(() => setFlash(undefined), 2200) }

  const setView = useCallback((v: ViewId) => {
    setViewState(v)
    const p = new URLSearchParams(location.search); p.set('view', v)
    history.replaceState(null, '', '?' + p.toString())
  }, [])

  const secs = useMemo(() => sections(tasks, view), [tasks, view])
  const ids = useMemo(() => flat(secs).map(t => t.id), [secs])

  const toggle = (id: string) => {
    const t = tasks.find(x => x.id === id); if (!t) return
    if (!t.done && children(tasks, id).some(c => !c.done)) return say('Finish the checklist first — a task with open subtasks cannot be completed directly')
    const done = !t.done
    setTasks(all => {
      let next = all.map(x => x.id === id ? { ...x, done, doneAt: done ? TODAY : undefined } : x)
      if (t.parentId) { // completing the last open subtask completes the parent; reopening one reopens it
        const sibs = next.filter(x => x.parentId === t.parentId)
        const allDone = sibs.every(x => x.done)
        next = next.map(x => x.id === t.parentId ? { ...x, done: allDone, doneAt: allDone ? TODAY : undefined } : x)
      }
      return next
    })
  }

  // fractional index (ADR 0007): a reorder is a single-row write
  const reorder = (dragId: string, targetId: string) => {
    if (dragId === targetId) return
    const list = ids.filter(i => i !== dragId); const ti = list.indexOf(targetId); if (ti < 0) return
    const target = tasks.find(t => t.id === targetId)!; const prev = tasks.find(t => t.id === list[ti - 1])
    const order = prev ? (prev.order + target.order) / 2 : target.order - 10
    setTasks(all => all.map(t => t.id === dragId ? { ...t, order, headingId: target.headingId, evening: target.evening } : t))
  }
  const moveBy = (id: string, dir: -1 | 1) => {
    const i = ids.indexOf(id), j = i + dir; if (i < 0 || j < 0 || j >= ids.length) return
    const other = tasks.find(t => t.id === ids[j])!; const beyond = tasks.find(t => t.id === ids[j + dir])
    const order = beyond ? (other.order + beyond.order) / 2 : other.order + dir * 10
    setTasks(all => all.map(t => t.id === id ? { ...t, order, headingId: other.headingId, evening: other.evening } : t))
  }
  const setDate = (id: string, date: string) => setTasks(all => all.map(t => t.id === id ? { ...t, when: { kind: 'date', date } } : t))
  const add = (title: string) => setTasks(all => [...all, { id: 'n' + Math.random(), title, when: view === 'today' ? { kind: 'today' } : { kind: 'anytime' }, done: false, tags: [], order: Math.max(...all.map(t => t.order)) + 10,
    projectId: view.startsWith('project:') ? view.slice(8) : undefined }])

  // list jump: plain digits 1–6 here (prototype only; the real app uses Mod+1–6)
  useEffect(() => {
    const h = (e: KeyboardEvent) => {
      if (isTyping(e) || e.metaKey || e.ctrlKey || e.altKey) return
      const i = Number(e.key) - 1; if (i >= 0 && i < LISTS.length) setView(LISTS[i].id)
    }
    window.addEventListener('keydown', h); return () => window.removeEventListener('keydown', h)
  }, [setView])

  return <Ctx.Provider value={{ tasks, view, setView, secs, ids, toggle, reorder, moveBy, setDate, flash, add }}>{kids}</Ctx.Provider>
}

const isTyping = (e: KeyboardEvent) => /^(INPUT|TEXTAREA)$/.test((e.target as HTMLElement)?.tagName) || (e.target as HTMLElement)?.isContentEditable

/** ↑/↓ select · Space quick-look/expand · →/← expand/collapse · x complete · ⌘/Ctrl+↑/↓ reorder · Esc close */
export function useNav() {
  const { ids, toggle, moveBy, view } = useStore()
  const [sel, setSel] = useState<string>(); const [open, setOpen] = useState<string>()
  const ref = useRef({ ids, sel, open }); ref.current = { ids, sel, open }
  useEffect(() => { setSel(undefined); setOpen(undefined) }, [view])
  useEffect(() => {
    const h = (e: KeyboardEvent) => {
      if (isTyping(e)) return
      const { ids, sel, open } = ref.current; const i = ids.indexOf(sel as string)
      const mod = e.metaKey || e.ctrlKey
      if (e.key === 'ArrowDown' || e.key === 'ArrowUp') {
        e.preventDefault(); const d = e.key === 'ArrowDown' ? 1 : -1
        if (mod && sel) return moveBy(sel, d as 1 | -1)
        const j = i < 0 ? 0 : Math.min(ids.length - 1, Math.max(0, i + d)); setSel(ids[j]); if (open) setOpen(ids[j])
      } else if (e.key === ' ' && sel) { e.preventDefault(); setOpen(open === sel ? undefined : sel) }
      else if (e.key === 'ArrowRight' && sel) setOpen(sel)
      else if ((e.key === 'ArrowLeft' || e.key === 'Escape') && open) setOpen(undefined)
      else if (e.key === 'x' && sel && !mod) toggle(sel)
    }
    window.addEventListener('keydown', h); return () => window.removeEventListener('keydown', h)
  }, [toggle, moveBy])
  return { sel, setSel, open, setOpen }
}

export function useDnd() {
  const { reorder } = useStore(); const [over, setOver] = useState<string>(); const drag = useRef<string>()
  return {
    over,
    props: (id: string) => ({
      draggable: true,
      onDragStart: (e: React.DragEvent) => { drag.current = id; e.dataTransfer.effectAllowed = 'move' },
      onDragOver: (e: React.DragEvent) => { e.preventDefault(); setOver(id) },
      onDragLeave: () => setOver(o => (o === id ? undefined : o)),
      onDrop: (e: React.DragEvent) => { e.preventDefault(); if (drag.current) reorder(drag.current, id); drag.current = undefined; setOver(undefined) },
      onDragEnd: () => { drag.current = undefined; setOver(undefined) },
    }),
  }
}
