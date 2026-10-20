// Variant C — "Top tabs, week board, quick look": no sidebar. Lists are tabs, Areas/Projects live in a popover.
// Today has a 7-day rail you can drop tasks onto; Upcoming is a board of days (drag a card to reschedule it).
// Space opens a quick-look overlay instead of expanding in place.
import { useRef, useState } from 'react'
import { Flag, X, FolderOpen, Plus } from 'lucide-react'
import { useStore, useNav, useDnd } from '../store'
import { LISTS, areas, projects, counts, viewTitle, children, fmtDate, iso, TODAY, projectName, type Task } from '../data'

export function VariantC() {
  const { tasks, view, setView, secs, toggle, flash, setDate, add } = useStore()
  const nav = useNav(); const dnd = useDnd(); const c: any = counts(tasks); const [menu, setMenu] = useState(false)
  const dragId = useRef<string>(); const [dropDay, setDropDay] = useState<string>(); const [draft, setDraft] = useState('')
  const days = Array.from({ length: 7 }, (_, i) => iso(i + 1))
  const onDay = (d: string) => tasks.filter(t => !t.parentId && !t.done && t.when.kind === 'date' && t.when.date === d)
  const q = tasks.find(t => t.id === nav.open)
  const dayDnd = (d: string) => ({
    onDragOver: (e: React.DragEvent) => { e.preventDefault(); setDropDay(d) }, onDragLeave: () => setDropDay(x => (x === d ? undefined : x)),
    onDrop: (e: React.DragEvent) => { e.preventDefault(); if (dragId.current) setDate(dragId.current, d); dragId.current = undefined; setDropDay(undefined) },
  })
  const dragProps = (id: string) => ({ draggable: true, onDragStart: () => { dragId.current = id }, onDragEnd: () => { dragId.current = undefined; setDropDay(undefined) } })

  const card = (t: Task, compact?: boolean) => {
    const selected = nav.sel === t.id; const subs = children(tasks, t.id); const od = t.deadline && t.deadline <= TODAY
    return (
      <div key={t.id} {...(compact ? dragProps(t.id) : { ...dnd.props(t.id), onDragStartCapture: () => { dragId.current = t.id } })} onClick={() => nav.setSel(t.id)} onDoubleClick={() => nav.setOpen(t.id)}
        className={`flex cursor-grab items-start gap-3 rounded-2xl border bg-white ${compact ? 'p-2.5 text-[13px]' : 'p-3.5 text-[15px]'} shadow-sm transition ${dnd.over === t.id && !compact ? 'border-emerald-500' : selected ? 'border-emerald-500 ring-2 ring-emerald-200' : 'border-stone-200 hover:border-stone-300'}`}>
        <button onClick={e => { e.stopPropagation(); toggle(t.id) }} aria-label="Complete" className={`mt-0.5 h-[18px] w-[18px] shrink-0 rounded-md border-2 ${t.done ? 'border-emerald-500 bg-emerald-500' : 'border-stone-300 hover:border-emerald-500'}`} />
        <div className="min-w-0 flex-1">
          <div className={`leading-snug ${t.done ? 'text-stone-400 line-through' : ''}`}>{t.title}</div>
          <div className="mt-1 flex flex-wrap items-center gap-x-2 gap-y-0.5 text-xs text-stone-400">
            {t.projectId && <span className="inline-flex items-center gap-1"><FolderOpen size={11} />{projectName(t.projectId)}</span>}
            {t.deadline && <span className={`inline-flex items-center gap-1 rounded-full px-1.5 ${od ? 'bg-rose-50 text-rose-600' : 'bg-stone-100'}`}><Flag size={10} />{fmtDate(t.deadline)}</span>}
            {subs.length > 0 && <span>☑ {subs.filter(s => s.done).length}/{subs.length}</span>}
            {t.tags.map(g => <span key={g}>#{g}</span>)}
          </div>
        </div>
      </div>)
  }

  const board = view === 'upcoming'
  return (
    <div className="flex h-screen flex-col bg-stone-50 text-stone-900">
      <nav className="relative z-30 flex items-center gap-1 border-b border-stone-200 bg-white px-5 py-2.5">
        <span className="mr-4 font-serif text-lg font-semibold tracking-tight">Zenith</span>
        {LISTS.map(l => (
          <button key={l.id} onClick={() => setView(l.id)} className={`rounded-full px-3.5 py-1.5 text-sm ${view === l.id ? 'bg-stone-900 text-white' : 'text-stone-600 hover:bg-stone-100'}`}>
            {l.name}{!!c[l.id] && <span className={`ml-1.5 text-xs ${view === l.id ? 'text-stone-300' : 'text-stone-400'}`}>{c[l.id]}</span>}</button>))}
        <div className="relative ml-2">
          <button onClick={() => setMenu(m => !m)} className={`flex items-center gap-1.5 rounded-full border px-3.5 py-1.5 text-sm ${view.startsWith('project') || view.startsWith('area') ? 'border-emerald-500 text-emerald-700' : 'border-stone-200 text-stone-600'}`}>
            <FolderOpen size={14} />{view.startsWith('project') || view.startsWith('area') ? viewTitle(view) : 'Areas & projects'}</button>
          {menu && (
            <div className="absolute left-0 top-11 w-72 rounded-2xl border border-stone-200 bg-white p-3 shadow-xl">
              {areas.map(a => (<div key={a.id} className="mb-2">
                <button onClick={() => { setView(`area:${a.id}`); setMenu(false) }} className="px-2 text-xs font-semibold uppercase tracking-wide text-stone-400 hover:text-stone-700">{a.name}</button>
                {projects.filter(p => p.areaId === a.id).map(p => <button key={p.id} onClick={() => { setView(`project:${p.id}`); setMenu(false) }} className="block w-full rounded-lg px-2 py-1.5 text-left text-sm hover:bg-stone-100">{p.name}</button>)}</div>))}
              <div className="px-2 text-xs font-semibold uppercase tracking-wide text-stone-400">Projects</div>
              {projects.filter(p => !p.areaId).map(p => <button key={p.id} onClick={() => { setView(`project:${p.id}`); setMenu(false) }} className="block w-full rounded-lg px-2 py-1.5 text-left text-sm hover:bg-stone-100">{p.name}</button>)}
            </div>)}
        </div>
        <span className="ml-auto text-xs text-stone-400">↑↓ select · Space quick look · x done · ⌘↑↓ reorder · 1–6 lists</span>
      </nav>

      <div className="flex min-h-0 flex-1">
        {board ? (
          <main className="flex-1 overflow-auto p-6">
            <h1 className="mb-4 font-serif text-3xl font-semibold">Upcoming <span className="font-sans text-sm font-normal text-stone-400">drag a card to another day to reschedule</span></h1>
            <div className="flex gap-3">
              {[...days, 'later'].map(d => {
                const list = d === 'later' ? tasks.filter(t => !t.parentId && !t.done && t.when.kind === 'date' && t.when.date > days[6]) : onDay(d)
                return (<div key={d} {...(d === 'later' ? {} : dayDnd(d))} className={`w-52 shrink-0 rounded-2xl p-2 ${dropDay === d ? 'bg-emerald-100/70 ring-2 ring-emerald-400' : 'bg-stone-100/70'}`}>
                  <div className="px-2 pb-2 pt-1"><div className="text-sm font-semibold">{d === 'later' ? 'Later' : fmtDate(d).split(',')[0]}</div><div className="text-xs text-stone-400">{d === 'later' ? `${list.length} tasks` : new Date(d + 'T12:00').toLocaleDateString(undefined, { day: 'numeric', month: 'short' })}</div></div>
                  <div className="space-y-2">{list.map(t => card(t, true))}{!list.length && <div className="rounded-xl border border-dashed border-stone-300 py-6 text-center text-xs text-stone-400">Free</div>}</div></div>)
              })}
            </div>
          </main>
        ) : (
          <>
            <main className="flex-1 overflow-auto px-8 pb-40 pt-8">
              <div className="mx-auto max-w-xl">
                <h1 className="mb-5 font-serif text-3xl font-semibold">{viewTitle(view)}</h1>
                {secs.map(s => (
                  <section key={s.key} className="mb-6 space-y-2">
                    {s.title && <h2 className="inline-block rounded-full bg-stone-200/70 px-3 py-0.5 text-xs font-semibold text-stone-600">{s.title}</h2>}
                    {s.tasks.map(t => card(t))}
                    {!s.tasks.length && <p className="text-sm text-stone-400">Nothing here.</p>}
                  </section>))}
                <div className="flex items-center gap-2 rounded-2xl border border-dashed border-stone-300 px-3.5 py-2.5 text-stone-400"><Plus size={16} />
                  <input value={draft} onChange={e => setDraft(e.target.value)} onKeyDown={e => { if (e.key === 'Enter' && draft.trim()) { add(draft.trim()); setDraft('') } }} placeholder="New task" className="flex-1 bg-transparent text-[15px] outline-none placeholder-stone-400" /></div>
              </div>
            </main>
            {view === 'today' && (
              <aside className="hidden w-72 shrink-0 overflow-auto border-l border-stone-200 bg-white p-4 xl:block">
                <div className="mb-3 text-xs font-semibold uppercase tracking-wide text-stone-400">Next 7 days · drop to reschedule</div>
                <div className="space-y-2">{days.map(d => (
                  <div key={d} {...dayDnd(d)} className={`rounded-xl border p-2.5 transition ${dropDay === d ? 'border-emerald-400 bg-emerald-50' : 'border-stone-100 bg-stone-50'}`}>
                    <div className="mb-1 flex justify-between text-xs"><b>{fmtDate(d)}</b><span className="text-stone-400">{onDay(d).length || ''}</span></div>
                    {onDay(d).map(t => <div key={t.id} className="truncate text-xs text-stone-600">• {t.title}</div>)}
                    {!onDay(d).length && <div className="text-xs text-stone-300">free</div>}</div>))}</div>
              </aside>)}
          </>)}
      </div>

      {q && (
        <div className="fixed inset-0 z-40 flex items-center justify-center bg-stone-900/30 backdrop-blur-[2px]" onClick={() => nav.setOpen(undefined)}>
          <div onClick={e => e.stopPropagation()} className="w-[34rem] max-w-[92vw] rounded-3xl bg-white p-6 shadow-2xl">
            <div className="flex items-start justify-between gap-4"><h2 className="font-serif text-2xl font-semibold leading-tight">{q.title}</h2><button onClick={() => nav.setOpen(undefined)} aria-label="Close"><X size={18} className="text-stone-400" /></button></div>
            <p className="mt-2 text-sm text-stone-500">{q.notes ?? 'No notes.'}</p>
            <div className="mt-4 space-y-1.5">{children(tasks, q.id).map(s => (
              <label key={s.id} className="flex items-center gap-2.5 text-sm"><input type="checkbox" checked={s.done} onChange={() => toggle(s.id)} className="h-4 w-4 accent-emerald-600" /><span className={s.done ? 'text-stone-400 line-through' : ''}>{s.title}</span></label>))}</div>
            <div className="mt-5 flex flex-wrap gap-2 text-xs">
              <span className="rounded-full bg-stone-100 px-3 py-1">When · {q.when.kind === 'date' ? fmtDate(q.when.date) : q.when.kind}</span>
              <span className={`rounded-full px-3 py-1 ${q.deadline && q.deadline <= TODAY ? 'bg-rose-50 text-rose-600' : 'bg-stone-100'}`}>Deadline · {q.deadline ? fmtDate(q.deadline) : '—'}</span>
              {q.repeater && <span className="rounded-full bg-stone-100 px-3 py-1">{q.repeater}</span>}
              {q.tags.map(g => <span key={g} className="rounded-full bg-stone-100 px-3 py-1">#{g}</span>)}</div>
          </div></div>)}
      {flash && <div className="fixed left-1/2 top-16 z-50 -translate-x-1/2 rounded-xl bg-stone-900 px-4 py-2 text-sm text-white shadow-lg">{flash}</div>}
    </div>
  )
}
