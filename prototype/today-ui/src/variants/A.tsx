// Variant A — "Calm sidebar, inline expansion": the closest to Things 3. Roomy rows, one accent, tasks expand in place.
import { useState } from 'react'
import { Inbox, Star, Calendar, Layers, Moon, CheckCheck, Flag, Repeat, FolderOpen, GripVertical } from 'lucide-react'
import { useStore, useNav, useDnd } from '../store'
import { LISTS, areas, projects, counts, viewTitle, children, fmtDate, TODAY, projectName, type Task, type ViewId } from '../data'

const ICON: Record<string, any> = { inbox: Inbox, today: Star, upcoming: Calendar, anytime: Layers, someday: Moon, logbook: CheckCheck }

export function VariantA() {
  const { tasks, view, setView, secs, toggle, flash, add } = useStore()
  const nav = useNav(); const dnd = useDnd(); const c = counts(tasks); const [draft, setDraft] = useState('')
  const Item = ({ id, label, icon: I, n, indent }: { id: ViewId; label: string; icon?: any; n?: number; indent?: boolean }) => (
    <button onClick={() => setView(id)} className={`flex w-full items-center gap-2.5 rounded-md px-2.5 py-1.5 text-left text-[13px] ${indent ? 'pl-8' : ''} ${view === id ? 'bg-neutral-200/70 font-medium' : 'hover:bg-neutral-200/40'}`}>
      {I && <I size={16} className={id === 'today' ? 'text-amber-500' : id === 'upcoming' ? 'text-rose-500' : 'text-sky-600'} />}<span className="flex-1 truncate">{label}</span>
      {!!n && <span className="text-xs text-neutral-400">{n}</span>}
    </button>)
  return (
    <div className="flex h-screen bg-white text-neutral-900">
      <aside className="flex w-60 shrink-0 flex-col gap-0.5 overflow-auto border-r border-neutral-200 bg-neutral-50 p-3 pt-6">
        {LISTS.map(l => <Item key={l.id} id={l.id} label={l.name} icon={ICON[l.id]} n={(c as any)[l.id]} />)}
        {areas.map(a => (
          <div key={a.id} className="mt-5">
            <button onClick={() => setView(`area:${a.id}`)} className="px-2.5 pb-1 text-[11px] font-semibold uppercase tracking-wide text-neutral-400">{a.name}</button>
            {projects.filter(p => p.areaId === a.id).map(p => <Item key={p.id} id={`project:${p.id}`} label={p.name} icon={FolderOpen} />)}
          </div>
        ))}
        <div className="mt-5"><div className="px-2.5 pb-1 text-[11px] font-semibold uppercase tracking-wide text-neutral-400">Projects</div>
          {projects.filter(p => !p.areaId).map(p => <Item key={p.id} id={`project:${p.id}`} label={p.name} icon={FolderOpen} />)}</div>
      </aside>

      <main className="flex-1 overflow-auto">
        <div className="mx-auto max-w-2xl px-10 pb-40 pt-12">
          <h1 className="mb-6 flex items-baseline gap-3 text-3xl font-semibold tracking-tight">{viewTitle(view)}{view === 'today' && <span className="text-sm font-normal text-neutral-400">{fmtDate(TODAY)} · {new Date().toLocaleDateString(undefined, { day: 'numeric', month: 'long' })}</span>}</h1>
          {secs.map(s => (
            <section key={s.key} className="mb-6">
              {s.title && <h2 className="mb-1 border-b border-neutral-200 pb-1.5 pt-2 text-[13px] font-semibold text-neutral-700">{s.key === 'eve' ? '🌙 ' : ''}{s.title}</h2>}
              {s.tasks.map(row)}
              {!s.tasks.length && <p className="py-3 text-sm text-neutral-400">Nothing here.</p>}
            </section>
          ))}
          <input value={draft} onChange={e => setDraft(e.target.value)} onKeyDown={e => { if (e.key === 'Enter' && draft.trim()) { add(draft.trim()); setDraft('') } }}
            placeholder="+ New to-do" className="mt-2 w-full rounded-lg px-3 py-2 text-[15px] text-neutral-500 placeholder-neutral-400 outline-none focus:bg-neutral-50" />
        </div>
      </main>
      {flash && <div className="fixed left-1/2 top-4 -translate-x-1/2 rounded-lg bg-neutral-900 px-4 py-2 text-sm text-white shadow-lg">{flash}</div>}
      <p className="fixed bottom-16 right-4 hidden rounded-md bg-neutral-100 px-3 py-1.5 text-[11px] text-neutral-500 lg:block">↑↓ select · Space expand · x done · ⌘↑↓ reorder · 1–6 lists</p>
    </div>
  )

  function row(t: Task) {
    const subs = children(tasks, t.id); const isOpen = nav.open === t.id; const selected = nav.sel === t.id
    const overdue = t.deadline && t.deadline <= TODAY
    return (
      <div key={t.id} {...dnd.props(t.id)} onClick={() => nav.setSel(t.id)} onDoubleClick={() => nav.setOpen(isOpen ? undefined : t.id)}
        className={`group relative rounded-xl transition-shadow ${dnd.over === t.id ? 'before:absolute before:-top-px before:left-3 before:right-3 before:h-0.5 before:bg-sky-500' : ''} ${isOpen ? 'my-2 bg-white shadow-[0_2px_14px_rgba(0,0,0,0.12)] ring-1 ring-black/5' : selected ? 'bg-sky-100/70' : 'hover:bg-neutral-50'}`}>
        <div className="flex items-start gap-3 px-3 py-2.5">
          <GripVertical size={14} className="absolute -left-4 top-3.5 hidden text-neutral-300 group-hover:block" />
          <Check done={t.done} onClick={() => toggle(t.id)} />
          <div className="min-w-0 flex-1">
            <div className="flex items-baseline gap-2">
              <span className={`text-[15px] leading-snug ${t.done ? 'text-neutral-400 line-through' : ''}`}>{t.title}</span>
              {t.repeater && <Repeat size={12} className="translate-y-0.5 text-neutral-400" />}
              {t.tags.map(g => <span key={g} className="text-xs text-neutral-400">#{g}</span>)}
              {!isOpen && subs.length > 0 && <span className="text-xs text-neutral-400">{subs.filter(s => s.done).length}/{subs.length}</span>}
            </div>
            {(view === 'today' || view === 'upcoming' || view === 'anytime') && t.projectId && !isOpen && <div className="text-xs text-neutral-400">{projectName(t.projectId)}</div>}
          </div>
          {t.deadline && <span className={`flex shrink-0 items-center gap-1 pt-0.5 text-xs ${overdue ? 'font-medium text-rose-600' : 'text-neutral-400'}`}><Flag size={11} />{overdue && t.deadline < TODAY ? `${Math.round((+new Date(TODAY) - +new Date(t.deadline)) / 864e5)}d overdue` : fmtDate(t.deadline)}</span>}
          {t.when.kind === 'date' && view !== 'upcoming' && <span className="shrink-0 pt-0.5 text-xs text-rose-500">{fmtDate(t.when.date)}</span>}
        </div>
        {isOpen && (
          <div className="space-y-3 px-3 pb-3 pl-[2.9rem]">
            <p className="text-sm text-neutral-500">{t.notes ?? 'Notes'}</p>
            {subs.map(s => (
              <div key={s.id} className="flex items-center gap-2.5"><Check small done={s.done} onClick={() => toggle(s.id)} />
                <span className={`text-sm ${s.done ? 'text-neutral-400 line-through' : ''}`}>{s.title}</span></div>))}
            <div className="flex items-center gap-4 border-t border-neutral-100 pt-2.5 text-xs text-neutral-500">
              <span>When: <b className="font-medium text-neutral-700">{t.when.kind === 'date' ? fmtDate(t.when.date) : t.when.kind}</b></span>
              <span>Deadline: <b className="font-medium text-neutral-700">{t.deadline ? fmtDate(t.deadline) : '—'}</b></span>
              {t.repeater && <span>{t.repeater}</span>}
            </div>
          </div>)}
      </div>)
  }
}

function Check({ done, onClick, small }: { done: boolean; onClick: () => void; small?: boolean }) {
  const s = small ? 'h-4 w-4' : 'h-5 w-5'
  return <button onClick={e => { e.stopPropagation(); onClick() }} aria-label={done ? 'Mark open' : 'Mark complete'}
    className={`mt-0.5 flex ${s} shrink-0 items-center justify-center rounded-full border-[1.5px] ${done ? 'border-sky-500 bg-sky-500 text-white' : 'border-neutral-300 hover:border-sky-500'}`}>
    {done && <svg viewBox="0 0 12 12" className="h-2.5 w-2.5"><path d="M2.5 6.2 5 8.6l4.5-5" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round" /></svg>}</button>
}
