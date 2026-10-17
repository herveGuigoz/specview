// Runs against the actual isolated binary. No development server is involved.
import {chromium} from 'playwright'
import assert from 'node:assert/strict'
import {mkdtemp, mkdir, writeFile, readFile, readdir, rename, rm, copyFile} from 'node:fs/promises'
import {tmpdir} from 'node:os'
import path from 'node:path'
import {fileURLToPath} from 'node:url'
import {spawn, execFileSync} from 'node:child_process'
import {createHash} from 'node:crypto'

const repo = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '../..')
const temp = await mkdtemp(path.join(tmpdir(), 'specview integration '))
const fixture = path.join(temp, 'my project')
const binary = path.join(temp, process.platform === 'win32' ? 'specview.exe' : 'specview')
const results = path.join(repo, 'test-results')
await mkdir(results, {recursive: true})
await copyFile(process.env.SPECVIEW_BINARY || path.join(repo, 'bin', path.basename(binary)), binary)
const binaryVersion = execFileSync(binary, ['--version'], {cwd: temp, env: {...process.env, PATH: ''}, encoding: 'utf8', timeout: 10000}).trim()
assert.match(binaryVersion, /^specview \S+$/)
if (process.env.SPECVIEW_EXPECT_VERSION) assert.equal(binaryVersion, 'specview ' + process.env.SPECVIEW_EXPECT_VERSION)
async function write(name, body) {const file = path.join(fixture, 'openspec', name); await mkdir(path.dirname(file), {recursive: true}); await writeFile(file, body)}
const initialTasks = '- [ ] First task\n- [x] Second task\n\n```md\n- [ ] Example only\n```\n'
const proposal = `# An example change

## Requirements

A reader for **OpenSpec**.

### Links

[Auth spec](specs/identity/user-auth/spec.md#requirements)

| Name | Value |
| --- | --- |
| Local | Yes |

<script>window.specviewPwned = true</script>

<img src="https://outside.invalid/raw-image">

[Bad link](javascript:alert(1))

![Remote image](https://outside.invalid/image)

\`\`\`go
${'a'.repeat(300)}
\`\`\`
`
await write('config.yaml', 'schema: spec-driven\n')
await write('specs/identity/user-auth/spec.md', '# Authentication\n\n## Requirements\n\nLocal spec.\n')
await write('changes/example/proposal.md', proposal)
await write('changes/example/tasks.md', initialTasks)
await write('changes/example/research/notes.md', '# Research\n')
await write('changes/example/specs/identity/user-auth/spec.md', '# Auth delta\n\n## Requirements\n\nThe delta.\n')
await write('changes/archive/example/proposal.md', '# Archived example\n')
for (let i = 0; i < 999; i++) await write(`specs/cap-${String(i).padStart(4, '0')}/spec.md`, '# Specification\n\n## Requirements\n\nA local document.\n')

async function fingerprint(dir) {
 const hash = createHash('sha256')
 async function visit(base, relative = '') {
  const entries = (await readdir(base, {withFileTypes: true})).sort((a, b) => a.name.localeCompare(b.name))
  for (const entry of entries) {const rel = path.join(relative, entry.name); hash.update(rel); if (entry.isDirectory()) await visit(path.join(base, entry.name), rel); else hash.update(await readFile(path.join(base, entry.name)))}
 }
 await visit(dir); return hash.digest('hex')
}
function launch(port) {
 // Runtime has no PATH tools available; only the copied executable and fixture.
 const child = spawn(binary, ['--no-open', ...(port ? ['--port', String(port)] : []), fixture], {cwd: temp, env: {...process.env, PATH: ''}, stdio: ['ignore', 'pipe', 'pipe']})
 let output = ''
 const ready = new Promise((resolve, reject) => {
  const timeout = setTimeout(() => reject(new Error('Server startup timed out: ' + output)), 10000)
  child.stdout.on('data', chunk => {output += chunk.toString(); const match = output.match(/http:\/\/127\.0\.0\.1:\d+/); if (match) {clearTimeout(timeout); resolve(match[0])}})
  child.stderr.on('data', chunk => {output += chunk.toString()})
  child.once('error', err => {clearTimeout(timeout); reject(err)})
  child.once('exit', code => {clearTimeout(timeout); reject(new Error(`Server exited ${code}: ${output}`))})
 })
 return {child, ready}
}
async function stop(server) {
 if (server.child.exitCode !== null || server.child.signalCode !== null) return
 await new Promise((resolve, reject) => {const timeout = setTimeout(() => {server.child.kill('SIGKILL'); reject(new Error('Server did not stop'))}, 6000); server.child.once('exit', () => {clearTimeout(timeout); resolve()}); server.child.kill(process.platform === 'win32' ? 'SIGTERM' : 'SIGINT')})
}
let server, browser
const measurements = []
try {
 server = launch()
 const base = await server.ready
 const port = new URL(base).port
 const before = await fingerprint(fixture)
 browser = await chromium.launch({headless: true, ...(process.env.SPECVIEW_BROWSER_EXECUTABLE ? {executablePath: process.env.SPECVIEW_BROWSER_EXECUTABLE} : {})})
 const context = await browser.newContext({viewport: {width: 1440, height: 1000}})
 const external = []
 await context.route('**/*', request => {if (new URL(request.request().url()).origin === base) return request.continue(); external.push(request.request().url()); return request.abort()})
 await context.grantPermissions(['clipboard-read', 'clipboard-write'])
 const page = await context.newPage()
 const pageErrors = []
 const assetResponses = []
 page.on('response', response => {if (/\.(js|css)$/.test(new URL(response.url()).pathname)) assetResponses.push({url: response.url(), status: response.status()})})
 page.on('pageerror', error => pageErrors.push(error.message))
 const navigate = (view, entry = '', doc = '', anchor = '') => page.goto(base + '/#' + new URLSearchParams({view, entry, doc, anchor}))
 await navigate('changes')
 await page.getByRole('link', {name: /example.*Proposition/}).first().waitFor()
 assert.ok(assetResponses.some(asset => asset.url.endsWith('.js') && asset.status === 200), 'embedded JavaScript missing')
 assert.ok(assetResponses.some(asset => asset.url.endsWith('.css') && asset.status === 200), 'embedded CSS missing')
 assert.ok(await page.locator('svg.lucide').count() > 0, 'embedded icons missing')
 assert.notEqual(await page.locator('body').evaluate(element => getComputedStyle(element).backgroundColor), 'rgba(0, 0, 0, 0)', 'styles not applied')
 await page.screenshot({path: path.join(results, 'changes-dark.png'), fullPage: true})
 await page.getByRole('textbox', {name: 'Rechercher'}).fill('does-not-exist')
 await page.getByRole('heading', {name: 'Aucun résultat'}).waitFor()
 await page.getByRole('button', {name: 'Effacer la recherche'}).click()
 await page.getByRole('link', {name: /example.*Proposition/}).first().click()
 await page.getByRole('heading', {name: 'An example change'}).waitFor()
 await page.getByLabel('Prochaine commande suggérée').getByText('/openspec-apply-change example', {exact: true}).waitFor()
 await page.getByRole('button', {name: 'Copier la commande'}).click()
 assert.equal(await page.evaluate(() => navigator.clipboard.readText()), '/openspec-apply-change example')
 assert.equal(await page.locator('.reader table').count(), 1)
 assert.equal(await page.locator('.reader img').count(), 0)
 assert.equal(await page.locator('.reader script').count(), 0)
 assert.equal(await page.evaluate(() => window.specviewPwned), undefined)
 await page.getByRole('button', {name: 'Source', exact: true}).click()
 assert.match(await page.locator('.source-view').textContent(), /<script>/)
 await page.getByRole('button', {name: 'Aperçu', exact: true}).click()
 await page.getByRole('link', {name: 'Auth spec'}).click()
 await page.getByRole('heading', {name: 'Auth delta'}).waitFor()
 await page.reload()
 await page.getByRole('heading', {name: 'Auth delta'}).waitFor()
 await page.getByRole('button', {name: 'Activer le thème clair'}).click()
 await page.reload()
 await page.waitForFunction(() => document.documentElement.dataset.theme === 'light')
 await page.screenshot({path: path.join(results, 'reader-light.png'), fullPage: true})
 await page.setViewportSize({width: 390, height: 844})
 await page.getByRole('button', {name: 'Ouvrir la navigation'}).click()
 await page.getByRole('button', {name: 'Activer le thème sombre'}).click()
 const closeNavigation = page.getByRole('button', {name: 'Fermer la navigation'})
 const closeBounds = await closeNavigation.boundingBox()
 const sidebarBounds = await page.getByRole('complementary', {name: 'Navigation principale'}).boundingBox()
 assert.ok(closeBounds && sidebarBounds && closeBounds.x >= sidebarBounds.x + sidebarBounds.width - 1, 'navigation backdrop overlaps the sidebar')
 await closeNavigation.click()
 await closeNavigation.waitFor({state: 'detached'})
 await page.waitForFunction(() => {
  const sidebar = document.querySelector('[aria-label="Navigation principale"]')
  return sidebar && sidebar.getBoundingClientRect().right <= 0
 })
 assert.ok(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth), 'page overflows at 390px')
 await page.keyboard.press('Tab')
 assert.ok(await page.evaluate(() => document.activeElement !== document.body), 'keyboard focus missing')
 await page.screenshot({path: path.join(results, 'reader-mobile.png'), fullPage: true})
 await page.setViewportSize({width: 1440, height: 1000})
 assert.equal(await fingerprint(fixture), before, 'viewing wrote to the fixture')
 await navigate('archive', 'changes/archive/example', 'changes/archive/example/proposal.md')
 await page.getByRole('heading', {name: 'Archived example'}).waitFor()
 assert.equal(await page.getByLabel('Prochaine commande suggérée').count(), 0)
 await navigate('changes', 'changes/example', 'changes/example/tasks.md')
 await page.locator('.reader input[type=checkbox]').first().waitFor()
 assert.equal(await page.locator('.reader input[type=checkbox]').count(), 2)
 assert.equal(await page.locator('.reader input[type=checkbox]:enabled').count(), 0)
 async function mutation(name, act, verify) {
  const start = performance.now(); await act(); await verify(); const elapsed = performance.now() - start
  measurements.push({name, milliseconds: Math.round(elapsed)}); assert.ok(elapsed < 3000, `${name} took ${elapsed}ms`)
 }
 await mutation('task edit', () => write('changes/example/tasks.md', initialTasks.replace('[ ] First', '[x] First')), async () => {
  await page.waitForFunction(() => document.querySelectorAll('.reader input:checked').length === 2, null, {timeout: 3000})
  assert.match(await page.locator('.detail-meta .progress-label').textContent(), /2\s*\/\s*2/)
  await page.getByLabel('Prochaine commande suggérée').getByText('/openspec-archive-change example', {exact: true}).waitFor({timeout: 3000})
 })
 await mutation('atomic save', async () => {await write('changes/example/temp.md', '- [x] Atomic task\n'); await rename(path.join(fixture, 'openspec/changes/example/temp.md'), path.join(fixture, 'openspec/changes/example/tasks.md'))}, () => page.getByText('Atomic task', {exact: true}).waitFor({timeout: 3000}))
 await mutation('new subdirectory', () => write('changes/example/research/new/notes.md', '# Fresh notes'), () => page.getByRole('link', {name: 'research/new/notes.md'}).waitFor({timeout: 3000}))
 await mutation('delete document', () => rm(path.join(fixture, 'openspec/changes/example/tasks.md')), () => page.getByRole('heading', {name: 'Document indisponible'}).waitFor({timeout: 3000}))
 await navigate('changes', 'changes/example', 'changes/example/proposal.md')
 await page.getByRole('heading', {name: 'An example change'}).waitFor()
 await mutation('archive change', () => rename(path.join(fixture, 'openspec/changes/example'), path.join(fixture, 'openspec/changes/archive/2026-example')), () => page.getByRole('heading', {name: 'Document indisponible'}).waitFor({timeout: 3000}))
 await navigate('archive', 'changes/archive/2026-example', 'changes/archive/2026-example/proposal.md')
 await page.getByRole('heading', {name: 'An example change'}).waitFor()
 await stop(server)
 await page.getByText('Connexion interrompue.', {exact: false}).waitFor({timeout: 15000})
 await write('changes/archive/2026-example/proposal.md', '# Reconnected content')
 server = launch(port); await server.ready
 await page.getByRole('heading', {name: 'Reconnected content'}).waitFor({timeout: 15000})
 const afterMutations = await fingerprint(fixture)
 await page.reload(); await page.getByRole('heading', {name: 'Reconnected content'}).waitFor()
 assert.equal(await fingerprint(fixture), afterMutations, 'reconnection wrote to fixture')
 assert.deepEqual(external, [], 'automatic external requests')
 assert.deepEqual(pageErrors, [], 'browser runtime errors')
 const report = {platform: process.platform, binaryVersion, documents: 1005, measurements, assetResponses, externalRequests: external, pageErrors}
 await writeFile(path.join(results, 'integration.json'), JSON.stringify(report, null, 2) + '\n')
 console.log(JSON.stringify(report, null, 2))
} finally {
 if (browser) await browser.close()
 if (server) await stop(server)
 await rm(temp, {recursive: true, force: true})
}
