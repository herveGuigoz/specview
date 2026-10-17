import {StrictMode, useEffect, useRef, useState} from 'react'
import {createRoot} from 'react-dom/client'
import {Archive, ArrowLeft, ArrowUpRight, BookOpen, Check, ChevronRight, Circle, CircleCheck, Code2, FileText, Folder, GitBranch, Layers, LockKeyhole, Menu, Moon, Search, Sun, WifiOff, X} from 'lucide-react'
import {Markdown} from './Markdown'
import {NextCommand} from './NextCommand'
import {useCatalog} from './useCatalog'
import {readRoute, urlFor, type Category, type Entry, type Route} from './types'
import './styles.css'

const categories = [
 {id: 'specs', label: 'Spécifications', icon: BookOpen, description: 'Les comportements de référence de votre projet.'},
 {id: 'changes', label: 'Changements', icon: GitBranch, description: 'Les propositions et travaux en cours, au même endroit.'},
 {id: 'archive', label: 'Archives', icon: Archive, description: 'Les changements archivés et leurs artefacts.'},
] as const
const labelOf = (kind: string) => ({proposal: 'Proposition', design: 'Design', tasks: 'Tâches', delta: 'Delta', spec: 'Spécification'}[kind] ?? 'Document')

function App() {
 const [route, setRoute] = useState<Route>(readRoute)
 const [query, setQuery] = useState('')
 const [theme, setTheme] = useState(() => {try {return localStorage.getItem('specview-theme') === 'light' ? 'light' : 'dark'} catch {return 'dark'}})
 const [menuOpen, setMenuOpen] = useState(false)
 const [sourceMode, setSourceMode] = useState(false)
 const searchRef = useRef<HTMLInputElement>(null)
 const data = useCatalog(route.doc)
 const catalog = data.catalog
 const entries = catalog?.entries ?? []
 const category = categories.find(c => c.id === route.view)!
 const current = entries.find(e => e.id === route.entry && e.category === route.view)
 const selected = current?.documents.find(d => d.path === route.doc)
 const filtered = entries.filter(e => e.category === route.view && `${e.name} ${e.id} ${e.documents.map(d => d.path).join(' ')}`.toLocaleLowerCase().includes(query.trim().toLocaleLowerCase()))
 const tasks = entries.filter(e => e.category === 'changes').reduce((p, e) => ({done: p.done + e.progress.done, total: p.total + e.progress.total}), {done: 0, total: 0})
 useEffect(() => {const change = () => {setRoute(readRoute()); setMenuOpen(false)}; window.addEventListener('hashchange', change); return () => window.removeEventListener('hashchange', change)}, [])
 useEffect(() => {document.documentElement.dataset.theme = theme; try {localStorage.setItem('specview-theme', theme)} catch {}}, [theme])
 useEffect(() => {setSourceMode(false)}, [route.doc])
 useEffect(() => {
  if (!route.anchor || data.loading) return
  requestAnimationFrame(() => {
   const el = document.getElementById(route.anchor)
   if (el) {let parent: HTMLElement | null = el; while (parent) {if (parent instanceof HTMLDetailsElement) parent.open = true; parent = parent.parentElement} el.scrollIntoView({block: 'start'})}
  })
 }, [route.anchor, route.doc, data.loading, data.source])
 useEffect(() => {
  const key = (e: KeyboardEvent) => {if (e.key === '/' && !(e.target instanceof HTMLInputElement) && !(e.target instanceof HTMLTextAreaElement)) {e.preventDefault();searchRef.current?.focus()} if (e.key === 'Escape') setMenuOpen(false)}
  window.addEventListener('keydown', key); return () => window.removeEventListener('keydown', key)
 }, [])
 useEffect(() => {document.title = `${current?.name ?? category.label} · specview`}, [current?.name, category.label])
 const openEntry = (e: Entry) => urlFor(e.category, e.id, e.documents[0]?.path)
 return <div className="app">
  {menuOpen && <button className="scrim" aria-label="Fermer la navigation" onClick={() => setMenuOpen(false)}/>}
  <aside className={`sidebar ${menuOpen ? 'is-open' : ''}`} aria-label="Navigation principale">
   <a className="brand" href={urlFor('changes')}><span className="brand-mark"><Layers size={21}/></span><span>specview<span className="brand-dot">.</span></span></a>
   <div className="project-card"><div className="project-avatar">{(catalog?.project.name ?? 'S').slice(0, 1).toUpperCase()}</div><div className="project-info"><strong>{catalog?.project.name ?? 'Projet OpenSpec'}</strong><span>OpenSpec workspace</span></div><ChevronRight size={13}/></div>
   <div className="nav-caption">EXPLORER</div>
   <nav>{categories.map(c => <a key={c.id} href={urlFor(c.id)} className={`nav-item ${route.view === c.id ? 'active' : ''}`} aria-current={route.view === c.id ? 'page' : undefined}><c.icon size={17}/><span>{c.label}</span><span className="count">{entries.filter(e => e.category === c.id).length}</span></a>)}</nav>
   <div className="nav-caption capability-label">CAPACITÉS <span>{entries.filter(e => e.category === 'specs').length}</span></div>
   <div className="capabilities"><CapabilityTree entries={entries.filter(e => e.category === 'specs')} current={route.entry}/>{!entries.some(e => e.category === 'specs') && <p className="side-empty">Les spécifications apparaîtront ici.</p>}</div>
   <a className="docs-link" href="https://openspec.dev/docs" target="_blank" rel="noopener noreferrer"><BookOpen size={14}/><span>Documentation OpenSpec</span><ArrowUpRight size={12}/><span className="sr-only"> (nouvel onglet)</span></a>
   <div className="sidebar-foot"><span className="connection" role="status" title={data.connected ? 'Suivi des changements actif' : 'Reconnexion au suivi des changements'}><span className={`live-dot ${data.connected ? '' : 'offline'}`} aria-hidden="true"/>{data.connected ? 'Connecté' : 'Reconnexion…'}</span><button className="icon-button" aria-label={theme === 'dark' ? 'Activer le thème clair' : 'Activer le thème sombre'} onClick={() => setTheme(t => t === 'dark' ? 'light' : 'dark')}>{theme === 'dark' ? <Sun size={16}/> : <Moon size={16}/>}</button></div>
  </aside>
  <main className="main">
   <header className="topbar"><button className="icon-button mobile-menu" aria-label="Ouvrir la navigation" aria-expanded={menuOpen} onClick={() => setMenuOpen(!menuOpen)}><Menu size={19}/></button><div className="breadcrumbs"><Folder size={14}/><span>{catalog?.project.name ?? 'Workspace'}</span><ChevronRight size={12}/><a href={urlFor(route.view)}>{category.label}</a>{current && <><ChevronRight size={12}/><span className="crumb-current">{current.name}</span></>}</div><span className="readonly"><LockKeyhole size={12}/>Lecture seule</span></header>
   {!data.connected && catalog && <div className="banner" role="status"><WifiOff size={14}/>Connexion interrompue. Les données affichées peuvent être anciennes.</div>}
   {data.error && <div className="banner error" role="alert">{data.error}</div>}
   {catalog?.stale && <div className="banner error" role="alert">Lecture du projet interrompue. Dernier état connu affiché.</div>}
   {!!catalog?.diagnostics.length && <details className="diagnostics"><summary>{catalog.diagnostics.length} diagnostic(s) de lecture</summary>{catalog.diagnostics.map((message, i) => <p key={i}>{message}</p>)}</details>}
   <div className="workspace">
    {!catalog ? <div className="empty-state"><Layers size={32}/><h1>{data.error ? 'Projet indisponible' : 'Ouverture du projet…'}</h1><p>{data.error || 'Lecture des artefacts OpenSpec.'}</p></div> : !route.entry ? <>
     <div className="page-heading"><div className="eyebrow"><span className="accent-line"/> VOTRE PROJET, EN CLAIR</div><div className="title-row"><div><h1>{category.label}<span className="title-count">{entries.filter(e => e.category === route.view).length}</span></h1><p>{category.description}</p></div></div></div>
     <div className="stats"><Stat icon={BookOpen} label="Spécifications" value={entries.filter(e => e.category === 'specs').length}/><Stat icon={GitBranch} label="Changements actifs" value={entries.filter(e => e.category === 'changes').length}/><Stat icon={CircleCheck} label="Tâches réalisées" value={tasks.total ? `${tasks.done} / ${tasks.total}` : '—'}/><Stat icon={Archive} label="Archives" value={entries.filter(e => e.category === 'archive').length}/></div>
     <section className="catalog-section" aria-label={category.label}><div className="catalog-toolbar"><div className="section-title"><category.icon size={15}/>{category.label}<span className="count">{filtered.length}</span></div><label className="search"><Search size={14}/><input ref={searchRef} aria-label="Rechercher" placeholder="Rechercher un nom, un chemin…" value={query} onChange={e => setQuery(e.target.value)}/>{query ? <button className="icon-button" aria-label="Effacer la recherche" onClick={() => setQuery('')}><X size={13}/></button> : <kbd>/</kbd>}</label></div>
      <div className="list-header"><span>NOM</span><span>ARTEFACTS</span><span>PROGRESSION</span><span/></div>
      <div className="entry-list">{filtered.map(e => <a key={e.id} href={openEntry(e)} className="entry-row"><div className="entry-primary"><span className={`entry-icon ${e.category}`}><category.icon size={17}/></span><div><strong>{e.name}</strong><span className="entry-path">{e.id}/</span></div></div><div className="artifact-badges">{e.documents.slice(0, 3).map(d => <span key={d.path} className="artifact-badge" title={d.path}><FileText size={11}/>{labelOf(d.kind)}</span>)}{e.documents.length > 3 && <span className="more-badge">+{e.documents.length - 3}</span>}{!e.documents.length && <span className="muted">Aucun artefact</span>}</div><Progress entry={e}/><ChevronRight size={15} className="row-arrow"/></a>)}</div>
      {!filtered.length && <div className="empty-state compact"><category.icon size={29}/><h2>{query ? 'Aucun résultat' : `Aucun élément pour le moment`}</h2><p>{query ? 'Essayez un autre nom ou chemin.' : 'Les documents apparaîtront automatiquement lorsqu’ils seront ajoutés au projet.'}</p>{query && <button className="subtle-button" onClick={() => setQuery('')}>Effacer le filtre</button>}</div>}
     </section>
     <div className="catalog-foot"><span><Circle size={10}/><code>openspec/</code></span></div>
    </> : <>
     <a className="back-link" href={urlFor(route.view)}><ArrowLeft size={14}/>Tous les éléments · {category.label.toLowerCase()}</a>
     <div className="detail-heading"><div className="eyebrow"><category.icon size={13}/>{category.label.toUpperCase()}</div><h1>{current?.name ?? route.entry.split('/').at(-1)}</h1><div className="detail-path"><Folder size={13}/><code>{route.entry}/</code><span className="separator">·</span><span>{current?.documents.length ?? 0} artefact(s)</span></div></div>
     {current && current.category !== 'specs' && <div className="detail-meta"><div className="artifact-presence">{['proposal', 'design', 'tasks'].map(kind => <span key={kind} className={current.documents.some(d => d.kind === kind) ? 'present' : ''}>{current.documents.some(d => d.kind === kind) ? <Check size={12}/> : <Circle size={11}/>} {labelOf(kind)}</span>)}</div><Progress entry={current}/></div>}
     {current?.nextAction && data.connected && !catalog.stale && !data.error && <NextCommand key={current.nextAction.command} action={current.nextAction}/>}
     <section className="reader-card" aria-label="Lecteur de document"><div className="artifact-tabs" role="navigation" aria-label="Artefacts">{current?.documents.map(d => <a key={d.path} className={route.doc === d.path ? 'selected' : ''} href={urlFor(route.view, current.id, d.path)} aria-current={route.doc === d.path ? 'page' : undefined}><FileText size={13}/>{d.name}</a>)}</div>
      <div className="reader-toolbar"><span className="reader-path"><FileText size={13}/>{route.doc || 'Aucun document sélectionné'}</span><div className="segmented"><button className={!sourceMode ? 'selected' : ''} aria-pressed={!sourceMode} onClick={() => setSourceMode(false)}><BookOpen size={13}/>Aperçu</button><button className={sourceMode ? 'selected' : ''} aria-pressed={sourceMode} onClick={() => setSourceMode(true)}><Code2 size={13}/>Source</button></div></div>
      {data.loading ? <div className="empty-state compact"><p>Lecture du document…</p></div> : data.documentError && data.path === route.doc ? <div className="empty-state compact"><FileText size={28}/><h2>Document indisponible</h2><p>{data.documentError}</p></div> : !current || !route.doc || !selected ? <div className="empty-state compact"><FileText size={28}/><h2>{!current ? 'Élément absent ou déplacé' : 'Sélectionnez un document'}</h2><p>Les artefacts disponibles sont accessibles depuis les onglets.</p></div> : sourceMode ? <pre className="source-view"><code>{data.source}</code></pre> : <article className="reader"><Markdown source={data.source} path={route.doc} catalog={catalog} route={route}/></article>}
     </section>
    </>}
   </div>
   <footer className="app-footer"><span><Layers size={12}/>specview<span className="footer-divider">/</span>OpenSpec reader</span></footer>
  </main>
 </div>
}
function Progress({entry}: {entry: Entry}) {
 const {done, total} = entry.progress
 return <div className="progress-cell">{total ? <><span className="progress-track" role="progressbar" aria-valuemin={0} aria-valuemax={total} aria-valuenow={done} aria-label={`Tâches de ${entry.name}`}><span style={{width: `${done / total * 100}%`}}/></span><span className="progress-label">{done}<span> / {total}</span></span></> : <span className="muted no-progress">—</span>}</div>
}
function Stat({icon: Icon, label, value}: {icon: typeof BookOpen; label: string; value: string | number}) {return <div className="stat"><div className="stat-label"><Icon size={14}/>{label}</div><strong>{value}</strong></div>}
function CapabilityTree({entries, current}: {entries: Entry[]; current: string}) {
 type Branch = {children: Map<string, Branch>; entry?: Entry}
 const root: Branch = {children: new Map()}
 for (const entry of entries) {let branch = root; for (const part of entry.name.split('/')) {if (!branch.children.has(part)) branch.children.set(part, {children: new Map()}); branch = branch.children.get(part)!} branch.entry = entry}
 const render = (branch: Branch) => [...branch.children].map(([name, child]) => <div key={name}>{child.entry ? <a className={`cap-item ${current === child.entry.id ? 'selected' : ''}`} href={urlFor('specs', child.entry.id, child.entry.documents[0]?.path)}><span className="cap-dot"/>{name}</a> : <div className="cap-folder"><Folder size={11}/>{name}</div>}{!!child.children.size && <div className="cap-children">{render(child)}</div>}</div>)
 return <>{render(root)}</>
}

createRoot(document.getElementById('root')!).render(<StrictMode><App/></StrictMode>)
