import {Fragment, type ReactNode} from 'react'
import {marked} from 'marked'
import type {Catalog, Route} from './types'
import {urlFor} from './types'

type Token = {type: string; text?: string; raw?: string; tokens?: Token[]; depth?: number; href?: string; lang?: string; ordered?: boolean; start?: number; items?: (Token & {task?: boolean; checked?: boolean})[]; header?: Token[]; rows?: Token[][]; align?: (string | null)[]}
function slug(text: string) { return text.toLowerCase().replace(/[^\p{L}\p{N}\s_-]/gu, '').trim().replace(/\s+/g, '-') }

export function Markdown({source, path, catalog, route}: {source: string; path: string; catalog: Catalog; route: Route}) {
 const ids = new Map<string, number>()
 const headingID = (text: string) => {const base = slug(text); const n = ids.get(base) ?? 0; ids.set(base, n + 1); return n ? `${base}-${n}` : base}
 const link = (href: string | undefined, children: ReactNode, key: number): ReactNode => {
  if (!href) return <span key={key}>{children}</span>
  if (/^https?:\/\//i.test(href)) return <a key={key} href={href} target="_blank" rel="noreferrer noopener">{children} ↗</a>
  if (/^[a-z][a-z\d+.-]*:/i.test(href) || href.startsWith('//')) return <span key={key} title="Lien non pris en charge">{children}</span>
  try {
   const resolved = new URL(href, `http://specview.local/${path}`)
   const target = decodeURIComponent(resolved.pathname.slice(1))
   const owner = catalog.entries.find(entry => entry.documents.some(d => d.path === target))
   if (owner) return <a key={key} href={urlFor(owner.category, owner.id, target, decodeURIComponent(resolved.hash.slice(1)))}>{children}</a>
  } catch { /* Invalid local paths are never navigated. */ }
  return <span key={key} className="unavailable-link" title="Document hors catalogue">{children} (hors catalogue)</span>
 }
 const inline = (tokens: Token[] = []): ReactNode => tokens.map((token, i) => {
  switch (token.type) {
   case 'strong': return <strong key={i}>{inline(token.tokens)}</strong>
   case 'em': return <em key={i}>{inline(token.tokens)}</em>
   case 'del': return <del key={i}>{inline(token.tokens)}</del>
   case 'codespan': return <code key={i}>{decode(token.text ?? '')}</code>
   case 'br': return <br key={i}/>
   case 'link': return link(token.href, inline(token.tokens), i)
   case 'image': return link(token.href, <span>Image : {decode(token.text ?? '')}</span>, i)
   case 'html': return <code key={i}>{token.text ?? token.raw}</code>
   default: return <Fragment key={i}>{token.tokens ? inline(token.tokens) : decode(token.text ?? '')}</Fragment>
  }
 })
 const render = (tokens: Token[] = []): ReactNode => tokens.map((token, i) => {
  switch (token.type) {
   case 'space':
   case 'checkbox': return null
   case 'heading': {
    const id = headingID(token.text ?? '')
    const content = inline(token.tokens)
    if (token.depth === 1) return <h1 key={i} id={id}>{content}</h1>
    return <details key={i} id={id} open className={`md-section level-${token.depth}`}><summary><span className="section-chevron">›</span><span role="heading" aria-level={token.depth}>{content}</span></summary></details>
   }
   case 'code': return <div className="code-block" key={i}>{token.lang && <div className="code-language">{token.lang}</div>}<pre><code>{token.text}</code></pre></div>
   case 'blockquote': return <blockquote key={i}>{render(token.tokens)}</blockquote>
   case 'list': {
    const children = token.items?.map((item, j) => <li key={j} className={item.task ? 'task-item' : undefined}>{item.task && <input type="checkbox" checked={!!item.checked} disabled aria-label={item.checked ? 'Tâche réalisée' : 'Tâche à faire'}/>}<div>{render(item.tokens)}</div></li>)
    return token.ordered ? <ol key={i} start={token.start}>{children}</ol> : <ul key={i}>{children}</ul>
   }
   case 'table': return <div className="table-wrap" key={i}><table><thead><tr>{token.header?.map((cell, j) => <th key={j}>{inline(cell.tokens)}</th>)}</tr></thead><tbody>{token.rows?.map((row, j) => <tr key={j}>{row.map((cell, k) => <td key={k}>{inline(cell.tokens)}</td>)}</tr>)}</tbody></table></div>
   case 'hr': return <hr key={i}/>
   case 'html': return <pre key={i} className="literal-html">{token.text ?? token.raw}</pre>
   default: return <p key={i}>{token.tokens ? inline(token.tokens) : decode(token.text ?? '')}</p>
  }
 })
 // Group sections on the Markdown AST, never by splitting raw fenced code.
 const blocks = marked.lexer(source, {gfm: true}) as unknown as Token[]
 // Build nested sections once. Non-heading blocks retain their original AST.
 const group = (tokens: Token[]): ReactNode => {
  const result: ReactNode[] = []
  for (let i = 0; i < tokens.length; i++) {
   const token = tokens[i]
   if (token.type === 'heading' && (token.depth ?? 1) >= 2) {
    const depth = token.depth ?? 2
    let end = i + 1
    while (end < tokens.length && !(tokens[end].type === 'heading' && (tokens[end].depth ?? 1) <= depth)) end++
    const id = headingID(token.text ?? '')
    result.push(<details key={i} id={id} open className={`md-section level-${depth}`}><summary><span className="section-chevron">›</span><span role="heading" aria-level={depth}>{inline(token.tokens)}</span></summary><div>{group(tokens.slice(i + 1, end))}</div></details>)
    i = end - 1
   } else result.push(<Fragment key={i}>{render([token])}</Fragment>)
  }
  return result
 }
 return <div className="prose" data-document={route.doc}>{group(blocks)}</div>
}
// Decode character references as text, never as markup or a URL.
function decode(value: string) {
 return value.replace(/&(#x[0-9a-f]+|#\d+|amp|lt|gt|quot|apos);/gi, (full, code: string) => {
  const named: Record<string, string> = {amp: '&', lt: '<', gt: '>', quot: '"', apos: "'"}
  if (named[code]) return named[code]
  const n = code[0] === '#' ? parseInt(code.slice(code[1]?.toLowerCase() === 'x' ? 2 : 1), code[1]?.toLowerCase() === 'x' ? 16 : 10) : NaN
  return Number.isFinite(n) && n > 0 && n <= 0x10ffff ? String.fromCodePoint(n) : full
 })
}
