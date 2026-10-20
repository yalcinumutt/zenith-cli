// Variant B — "Dense table + inspector": 28px rows, metadata in columns, nothing expands in the list.
// Selecting a row fills the right-hand inspector (Space toggles subtasks inline). Built for keyboard + big lists.
import { useStore, useNav, useDnd } from '../store'
import { LISTS, areas, projects, counts, viewTitle, children, fmtDate, TODAY, projectName, type Task } from '../data'

const GRID = 'grid grid-cols-[20px_minmax(0,1fr)_120px_150px_96px_96px] items-center gap-x-2'

export function VariantB() {
  const { tasks, view, setView, secs, toggle, flash, add } = useStore()
  const nav = useNav(); const dnd = useDnd(); const c: any = counts(tasks)
  const sel = tasks.find(t => t.id === nav.sel)
  const side = (id: any, label: string, n?: number) => (
    <button key={id} onClick={() => setView(id)} className={`flex w-full items-center justify-between rounded px-2 py-1 text-left ${view === id ? 'bg-neutral-900 text-white' : 'text-neutral-700 hover:bg-neutral-200'}`}>
      <span className="truncate">{label}</span>{!!n && <span className={`font-mono text-[10px] ${view === id ? 'text-neutral-300' : 'text-neutral-400'}`}>{n}</span>}</button>)

  return (
    <div className="flex h-screen bg-neutral-100 text-[13px] text-neutral-900">
      <aside className="w-48 shrink-0 space-y-px overflow-auto border-r border-neutral-300 bg-neutral-100 p-2 pt-3">
        {LISTS.map((l, i) => <div key={l.id} className="flex items-center"><kbd className="w-4 text-center font-mono text-[10px] text-neutral-400">{i + 1}</kbd><div className="flex-1">{side(l.id, l.name, c[l.id])}</div></div>)}
        {areas.map(a => (
          <div key={a.id} className="pt-3">
            <button onClick={() => setView(`area:${a.id}`)} className="px-2 pb-0.5 font-mono text-[10px] uppercase tracking-wider text-neutral-500">{a.name}</button>
            {projects.filter(p => p.areaId === a.id).map(p => side(`project:${p.id}`, p.name))}
          </div>))}
        <div className="pt-3"><div className="px-2 pb-0.5 font-mono text-[10px] uppercase tracking-wider text-neutral-500">Projects</div>
          {projects.filter(p => !p.areaId).map(p => side(`project:${p.id}`, p.name))}</div>
      </aside>

      <main className="flex min-w-0 flex-1 flex-col bg-white">
        <header className="flex items-baseline justify-between border-b border-neutral-200 px-4 py-2">
          <h1 className="text-sm font-semibold">{viewTitle(view)} <span className="ml-1 font-mono text-[11px] font-normal text-neutral-400">{secs.reduce((n, s) => n + s.tasks.length, 0)} tasks</span></h1>
          <span className="font-mono text-[10px] text-neutral-400">↑↓ select · Space subtasks · x done · ⌘↑↓ reorder · 1–6 lists</span>
        </header>
        <div className={`${GRID} border-b border-neutral-200 bg-neutral-50 px-4 py-1 font-mono text-[10px] uppercase tracking-wider text-neutral-400`}>
          <span /><span>Title</span><span>Tags</span><span>Project</span><span>When</span><span>Deadline</span></div>
        <div className="flex-1 overflow-auto">
          {secs.map(s => (
            <div key={s.key}>
              {s.title && <div className="sticky top-0 z-10 border-b border-neutral-200 bg-neutral-100 px-4 py-1 font-mono text-[10px] font-semibold uppercase tracking-wider text-neutral-500">{s.title} <span className="font-normal text-neutral-400">· {s.tasks.length}</span></div>}
              {s.tasks.map(row)}
            </div>))}
          <input onKeyDown={e => { const v = e.currentTarget; if (e.key === 'Enter' && v.value.trim()) { add(v.value.trim()); v.value = '' } }} placeholder="+ add task, Enter"
            className="w-full px-4 py-2 font-mono text-xs text-neutral-500 outline-none placeholder-neutral-300 focus:bg-neutral-50" />
        </div>
      </main>

      <aside className="w-80 shrink-0 overflow-auto border-l border-neutral-300 bg-neutral-50 p-4">
        {!sel ? <p className="text-xs text-neutral-400">Select a task (↑/↓) to inspect it.</p> : <Inspector t={sel} />}
      </aside>
      {flash && <div className="fixed left-1/2 top-3 -translate-x-1/2 rounded bg-neutral-900 px-3 py-1.5 text-xs text-white">{flash}</div>}
    </div>
  )

  function row(t: Task) {
    const subs = children(tasks, t.id); const selected = nav.sel === t.id; const open = nav.open === t.id; const od = t.deadline && t.deadline <= TODAY
    return (
      <div key={t.id}>
        <div {...dnd.props(t.id)} onClick={() => nav.setSel(t.id)}
          className={`${GRID} h-7 cursor-default border-b border-neutral-100 px-4 ${dnd.over === t.id ? 'border-t-2 border-t-indigo-500' : ''} ${selected ? 'bg-indigo-50 shadow-[inset_2px_0_0_#4f46e5]' : 'hover:bg-neutral-50'}`}>
          <input type="checkbox" checked={t.done} onChange={() => toggle(t.id)} onClick={e => e.stopPropagation()} className="h-3.5 w-3.5 accent-indigo-600" aria-label="Complete" />
          <span className={`flex min-w-0 items-center gap-1.5 truncate ${t.done ? 'text-neutral-400 line-through' : ''}`}>
            <span className="truncate">{t.title}</span>
            {subs.length > 0 && <button onClick={e => { e.stopPropagation(); nav.setOpen(open ? undefined : t.id) }} className="shrink-0 font-mono text-[10px] text-neutral-400">{open ? '▾' : '▸'} {subs.filter(s => s.done).length}/{subs.length}</button>}
          </span>
          <span className="truncate font-mono text-[11px] text-neutral-400">{t.tags.map(g => '#' + g).join(' ')}</span>
          <span className="truncate text-[11px] text-neutral-500">{projectName(t.projectId) ?? ''}</span>
          <span className="truncate text-[11px] text-neutral-600">{t.when.kind === 'date' ? fmtDate(t.when.date) : t.when.kind === 'anytime' ? '' : t.when.kind}{t.evening ? ' · eve' : ''}</span>
          <span className={`truncate text-[11px] ${od ? 'font-semibold text-rose-600' : 'text-neutral-500'}`}>{t.deadline ? fmtDate(t.deadline) : ''}</span>
        </div>
        {open && subs.map(s => (
          <div key={s.id} className="flex h-6 items-center border-b border-neutral-100 bg-neutral-50/60 pl-[2.1rem] pr-4">
            <label className="flex items-center gap-2 text-[12px] text-neutral-600"><input type="checkbox" checked={s.done} onChange={() => toggle(s.id)} className="h-3 w-3 accent-indigo-600" /><span className={s.done ? 'line-through text-neutral-400' : ''}>{s.title}</span></label></div>))}
      </div>)
  }

  function Inspector({ t }: { t: Task }) {
    const subs = children(tasks, t.id)
    const F = ({ k, v }: { k: string; v?: string }) => <div className="flex justify-between border-b border-neutral-200 py-1.5 text-xs"><span className="text-neutral-500">{k}</span><span className="font-medium">{v || '—'}</span></div>
    return (<div className="space-y-4">
      <div><div className="font-mono text-[10px] uppercase tracking-wider text-neutral-400">Task</div><h2 className="text-base font-semibold leading-snug">{t.title}</h2></div>
      <p className="text-xs leading-relaxed text-neutral-600">{t.notes ?? <i className="text-neutral-400">No notes</i>}</p>
      <div>{[F({ k: 'Status', v: t.done ? 'completed' : children(tasks, t.id).length && nav.open ? 'open' : 'open' }),
        F({ k: 'When', v: t.when.kind === 'date' ? fmtDate(t.when.date) : t.when.kind }), F({ k: 'Deadline', v: t.deadline ? fmtDate(t.deadline) : undefined }),
        F({ k: 'Project', v: projectName(t.projectId) }), F({ k: 'Tags', v: t.tags.join(', ') }), F({ k: 'Repeats', v: t.repeater })].map((x, i) => <div key={i}>{x}</div>)}</div>
      {subs.length > 0 && <div><div className="mb-1 font-mono text-[10px] uppercase tracking-wider text-neutral-400">Checklist {subs.filter(s => s.done).length}/{subs.length}</div>
        {subs.map(s => <label key={s.id} className="flex items-center gap-2 py-1 text-xs"><input type="checkbox" checked={s.done} onChange={() => toggle(s.id)} className="accent-indigo-600" /><span className={s.done ? 'text-neutral-400 line-through' : ''}>{s.title}</span></label>)}</div>}
    </div>)
  }
}
