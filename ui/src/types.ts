export type Category = 'specs' | 'changes' | 'archive'
export type Artifact = {path: string; name: string; kind: string; error?: string}
export type NextAction = {label: string; command: string; target: 'agent' | 'terminal'; reason: string}
export type Entry = {id: string; name: string; category: Category; documents: Artifact[]; progress: {done: number; total: number}; nextAction?: NextAction}
export type Catalog = {revision: number; project: {name: string; path: string; openspec: string}; entries: Entry[]; diagnostics: string[]; stale: boolean}
export type Route = {view: Category; entry: string; doc: string; anchor: string}
export function readRoute(): Route {
 const q = new URLSearchParams(location.hash.slice(1))
 const raw = q.get('view')
 return {view: raw === 'specs' || raw === 'archive' ? raw : 'changes', entry: q.get('entry') ?? '', doc: q.get('doc') ?? '', anchor: q.get('anchor') ?? ''}
}
export function urlFor(view: Category, entry = '', doc = '', anchor = '') {
 const q = new URLSearchParams({view})
 if (entry) q.set('entry', entry)
 if (doc) q.set('doc', doc)
 if (anchor) q.set('anchor', anchor)
 return '#' + q.toString()
}
