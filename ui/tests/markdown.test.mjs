import './tsx-loader.mjs'
import {test} from 'node:test'
import assert from 'node:assert/strict'
import {createElement} from 'react'
import {renderToStaticMarkup} from 'react-dom/server'
import {Markdown} from '../src/Markdown.tsx'
import {NextCommand} from '../src/NextCommand.tsx'
import {readRoute, urlFor} from '../src/types.ts'

const doc = 'changes/example/proposal.md'
const neighbor = 'changes/example/specs/identity/user-auth/spec.md'
const entry = {id: 'changes/example', category: 'changes', name: 'example', documents: [{path: doc}, {path: neighbor}]}
const catalog = {entries: [entry]}
function render(source) {return renderToStaticMarkup(createElement(Markdown, {source, path: doc, catalog, route: {doc}}))}

test('renders GFM tasks, tables, nested headings and source code', () => {
 const html = render('# Proposal\n\n## Requirements\n\n### Nested\n\n- [x] Done\n- [ ] Todo\n\n| A | B |\n|---|---|\n| 1 | 2 |\n\n```go\n## code is not a heading\n```')
 assert.match(html, /<table>/);assert.match(html, /type="checkbox" disabled=""[^>]*checked=""/)
 assert.match(html, /id="requirements"/);assert.match(html, /id="nested"/)
 assert.equal((html.match(/<details/g) ?? []).length, 2)
 assert.match(html, /## code is not a heading/)
})
test('does not execute raw HTML or load images', () => {
 const html = render('<script>window.pwned=1</script>\n\n<img src="https://outside.test/image">\n\n[bad](javascript:alert(1))\n\n![image](https://outside.test/other)')
 assert.doesNotMatch(html, /<script|<img|href="javascript:/i)
 assert.match(html, /&lt;script&gt;/);assert.match(html, /Image/)
})
test('routes local links with full document paths and anchors', () => {
 const html = render('[Auth](specs/identity/user-auth/spec.md#requirements)\n\n[Outside](../../secret.md)')
 assert.match(html, /doc=changes%2Fexample%2Fspecs%2Fidentity%2Fuser-auth%2Fspec.md/)
 assert.match(html, /anchor=requirements/);assert.match(html, /hors catalogue/)
})
test('preserves full route identities and special characters', () => {
 const hash = urlFor('archive', 'changes/archive/2026-same', 'changes/archive/2026-same/a b#c.md', 'section')
 globalThis.location = {hash}
 assert.deepEqual(readRoute(), {view: 'archive', entry: 'changes/archive/2026-same', doc: 'changes/archive/2026-same/a b#c.md', anchor: 'section'})
})

test('next command shows where to paste it and a copy action', () => {
 const action = {label: 'Poursuivre l’implémentation', command: '/openspec-apply-change example', target: 'agent', reason: 'Des tâches restent à réaliser.'}
 const html = renderToStaticMarkup(createElement(NextCommand, {action}))
 assert.match(html, /Prochaine commande suggérée/)
 assert.match(html, /Dans votre agent/)
 assert.match(html, /\/openspec-apply-change example/)
 assert.match(html, /Copier la commande/)
 assert.doesNotMatch(html, /onclick=|<form/)
 const terminal = renderToStaticMarkup(createElement(NextCommand, {action: {...action, target: 'terminal'}}))
 assert.match(terminal, /Dans le terminal/)
})
