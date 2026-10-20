// PROTOTYPE — fake data + derived lists (ADR 0004: lists are computed from When, Deadline, Status; never stored)
export type When = { kind: 'anytime' } | { kind: 'today' } | { kind: 'date'; date: string } | { kind: 'someday' }
export interface Task {
  id: string; title: string; notes?: string; when: When; evening?: boolean; deadline?: string
  done: boolean; areaId?: string; projectId?: string; headingId?: string; parentId?: string
  tags: string[]; order: number; repeater?: string; doneAt?: string
}
export interface Area { id: string; name: string }
export interface Project { id: string; name: string; areaId?: string }
export interface Heading { id: string; projectId: string; name: string }

export const iso = (off = 0) => { const d = new Date(); d.setDate(d.getDate() + off); return d.toISOString().slice(0, 10) }
export const TODAY = iso(0)
export const fmtDate = (s: string) => {
  if (s === TODAY) return 'Today'
  if (s === iso(1)) return 'Tomorrow'
  if (s === iso(-1)) return 'Yesterday'
  return new Date(s + 'T12:00').toLocaleDateString(undefined, { weekday: 'short', day: 'numeric', month: 'short' })
}

export const areas: Area[] = [{ id: 'work', name: 'Work' }, { id: 'health', name: 'Health' }, { id: 'home', name: 'Home' }]
export const projects: Project[] = [
  { id: 'launch', name: 'Launch Zenith desktop', areaId: 'work' },
  { id: 'q4', name: 'Q4 planning', areaId: 'work' },
  { id: 'marathon', name: 'Marathon training', areaId: 'health' },
  { id: 'move', name: 'Apartment move', areaId: 'home' },
  { id: 'rust', name: 'Learn Rust' },
]
export const headings: Heading[] = [
  { id: 'h-core', projectId: 'launch', name: 'Core & storage' },
  { id: 'h-ui', projectId: 'launch', name: 'Desktop UI' },
  { id: 'h-rel', projectId: 'launch', name: 'Release' },
  { id: 'h-prep', projectId: 'move', name: 'Before moving day' },
  { id: 'h-after', projectId: 'move', name: 'After' },
]

let n = 0
const T = (t: Partial<Task> & { title: string }): Task => ({ id: t.id ?? `t${++n}`, when: { kind: 'anytime' }, done: false, tags: [], order: ++n * 10, ...t })
export const seed: Task[] = [
  T({ id: 'a', title: 'Swap SQLite driver to modernc', projectId: 'launch', headingId: 'h-core', areaId: 'work', when: { kind: 'today' }, deadline: iso(0), tags: ['go'], notes: 'Ticket 0. Also enable Windows CLI target and fix the owner slug.' }),
  T({ id: 'a1', parentId: 'a', title: 'Add openDB with DSN pragmas' }),
  T({ id: 'a2', parentId: 'a', title: 'Run storage tests with CGO off', done: true, doneAt: TODAY }),
  T({ id: 'a3', parentId: 'a', title: 'Enable windows target in .goreleaser.yaml' }),
  T({ id: 'b', title: 'Write field_versions apply function', projectId: 'launch', headingId: 'h-core', areaId: 'work', when: { kind: 'today' }, tags: ['sync'] }),
  T({ id: 'c', title: 'Task row component', projectId: 'launch', headingId: 'h-ui', areaId: 'work', when: { kind: 'today' }, deadline: iso(-1), tags: ['ui'], notes: 'Round checkbox, quiet metadata, inline expansion.' }),
  T({ id: 'd', title: 'Command palette with cmdk', projectId: 'launch', headingId: 'h-ui', areaId: 'work', when: { kind: 'date', date: iso(2) }, tags: ['ui'] }),
  T({ id: 'e', title: 'Apple Developer account', projectId: 'launch', headingId: 'h-rel', areaId: 'work', deadline: iso(6), tags: ['admin'] }),
  T({ id: 'f', title: 'Reply to Sam about the roadmap', areaId: 'work', when: { kind: 'today' }, evening: false }),
  T({ id: 'g', title: 'Review pull request: reminders', areaId: 'work', when: { kind: 'today' } }),
  T({ id: 'h', title: 'Long run, 18 km', projectId: 'marathon', areaId: 'health', when: { kind: 'today' }, evening: true, tags: ['run'], repeater: 'Every Sunday' }),
  T({ id: 'i', title: 'Stretch and foam roll', projectId: 'marathon', areaId: 'health', when: { kind: 'today' }, evening: true, repeater: 'Daily' }),
  T({ id: 'j', title: 'Buy new running shoes', projectId: 'marathon', areaId: 'health', when: { kind: 'date', date: iso(1) } }),
  T({ id: 'k', title: 'Book the moving van', projectId: 'move', headingId: 'h-prep', areaId: 'home', when: { kind: 'date', date: iso(3) }, deadline: iso(9), tags: ['errand'] }),
  T({ id: 'l', title: 'Pack the kitchen', projectId: 'move', headingId: 'h-prep', areaId: 'home', when: { kind: 'date', date: iso(5) } }),
  T({ id: 'l1', parentId: 'l', title: 'Wrap glasses' }), T({ id: 'l2', parentId: 'l', title: 'Label boxes' }),
  T({ id: 'm', title: 'Change address with the bank', projectId: 'move', headingId: 'h-after', areaId: 'home' }),
  T({ id: 'n', title: 'Cancel old internet contract', projectId: 'move', headingId: 'h-after', areaId: 'home', deadline: iso(20) }),
  T({ id: 'o', title: 'Draft Q4 goals', projectId: 'q4', areaId: 'work', when: { kind: 'date', date: iso(7) } }),
  T({ id: 'p', title: 'Ask finance about the budget', projectId: 'q4', areaId: 'work' }),
  T({ id: 'q', title: 'Read chapter on ownership', projectId: 'rust', when: { kind: 'someday' } }),
  T({ id: 'r', title: 'Gift idea for mum', when: {kind:'anytime'} }),
  T({ id: 's', title: 'Look into standing desk options' }),
  T({ id: 'u', title: 'Renew passport', deadline: iso(14), tags: ['admin'] }),
  T({ id: 'v', title: 'Try the new ramen place', when: { kind: 'someday' } }),
  T({ id: 'w', title: 'Set up CI matrix', projectId: 'launch', headingId: 'h-rel', areaId: 'work', done: true, doneAt: iso(-1) }),
  T({ id: 'x', title: 'Pick a frontend stack', projectId: 'launch', headingId: 'h-ui', areaId: 'work', done: true, doneAt: TODAY }),
]

export type ViewId = 'inbox' | 'today' | 'upcoming' | 'anytime' | 'someday' | 'logbook' | `project:${string}` | `area:${string}`
export const LISTS: { id: ViewId; name: string }[] = [
  { id: 'inbox', name: 'Inbox' }, { id: 'today', name: 'Today' }, { id: 'upcoming', name: 'Upcoming' },
  { id: 'anytime', name: 'Anytime' }, { id: 'someday', name: 'Someday' }, { id: 'logbook', name: 'Logbook' },
]
export interface Section { key: string; title?: string; tasks: Task[]; date?: string }

const top = (t: Task) => !t.parentId
const byOrder = (a: Task, b: Task) => a.order - b.order
const inboxOf = (t: Task) => top(t) && !t.done && !t.projectId && !t.areaId && t.when.kind === 'anytime'
const todayOf = (t: Task) => top(t) && !t.done && (t.when.kind === 'today' || (t.when.kind === 'date' && t.when.date <= TODAY) || (t.deadline && t.deadline <= TODAY))
const upcomingOf = (t: Task) => top(t) && !t.done && t.when.kind === 'date' && t.when.date > TODAY

export const children = (all: Task[], id: string) => all.filter(t => t.parentId === id).sort(byOrder)
export const counts = (all: Task[]) => ({
  inbox: all.filter(inboxOf).length, today: all.filter(todayOf).length, upcoming: all.filter(upcomingOf).length,
})
export const viewTitle = (v: ViewId) =>
  LISTS.find(l => l.id === v)?.name ?? (v.startsWith('project:') ? projects.find(p => 'project:' + p.id === v)?.name : areas.find(a => 'area:' + a.id === v)?.name) ?? v

export function sections(all: Task[], v: ViewId): Section[] {
  const open = all.filter(top).filter(t => !t.done).sort(byOrder)
  if (v === 'inbox') return [{ key: 'i', tasks: all.filter(inboxOf).sort(byOrder) }]
  if (v === 'today') {
    const t = all.filter(todayOf).sort(byOrder)
    return [{ key: 'day', tasks: t.filter(x => !x.evening) }, { key: 'eve', title: 'This Evening', tasks: t.filter(x => x.evening) }].filter(s => s.tasks.length || s.key === 'day')
  }
  if (v === 'upcoming') {
    const groups = new Map<string, Task[]>()
    all.filter(upcomingOf).sort(byOrder).forEach(t => { const d = (t.when as any).date; groups.set(d, [...(groups.get(d) ?? []), t]) })
    return [...groups.entries()].sort(([a], [b]) => a.localeCompare(b)).map(([d, tasks]) => ({ key: d, title: fmtDate(d), date: d, tasks }))
  }
  if (v === 'anytime') return [{ key: 'any', tasks: open.filter(t => t.when.kind === 'anytime' && (t.projectId || t.areaId)) }]
  if (v === 'someday') return [{ key: 'some', tasks: open.filter(t => t.when.kind === 'someday') }]
  if (v === 'logbook') return [{ key: 'log', tasks: all.filter(t => top(t) && t.done).sort((a, b) => (b.doneAt ?? '').localeCompare(a.doneAt ?? '')) }]
  if (v.startsWith('project:')) {
    const pid = v.slice(8); const mine = open.filter(t => t.projectId === pid)
    const hs = headings.filter(h => h.projectId === pid)
    return [{ key: 'none', tasks: mine.filter(t => !t.headingId) }, ...hs.map(h => ({ key: h.id, title: h.name, tasks: mine.filter(t => t.headingId === h.id) }))].filter(s => s.tasks.length || s.title)
  }
  const aid = v.slice(5)
  return [{ key: 'area', tasks: open.filter(t => t.areaId === aid && !t.projectId) }]
}
export const flat = (secs: Section[]) => secs.flatMap(s => s.tasks)
export const projectName = (id?: string) => projects.find(p => p.id === id)?.name
