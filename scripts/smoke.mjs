// Check the distributed executable without a browser or development tools at runtime.
import assert from 'node:assert/strict'
import {execFileSync, spawn} from 'node:child_process'
import {once} from 'node:events'
import {copyFile, mkdir, mkdtemp, rm, writeFile} from 'node:fs/promises'
import {tmpdir} from 'node:os'
import path from 'node:path'

const [source, version] = process.argv.slice(2)
assert.ok(source && version, 'Usage: node scripts/smoke.mjs <binary> <version>')
const temporary = await mkdtemp(path.join(tmpdir(), 'specview-smoke-'))
const binary = path.join(temporary, 'specview')
const runtime = {cwd: temporary, env: {...process.env, PATH: ''}}
let server
try {
  await copyFile(path.resolve(source), binary)
  assert.equal(execFileSync(binary, ['--version'], {...runtime, encoding: 'utf8', timeout: 5000}), `specview ${version}\n`)
  await mkdir(path.join(temporary, 'openspec/specs/example'), {recursive: true})
  await writeFile(path.join(temporary, 'openspec/specs/example/spec.md'), '# Embedded release test\n')
  server = spawn(binary, ['--no-open', temporary], {...runtime, stdio: ['ignore', 'pipe', 'pipe']})
  const base = await new Promise((resolve, reject) => {
    let output = ''
    const timer = setTimeout(() => reject(new Error(`Startup timed out: ${output}`)), 5000)
    const finish = (error, url) => {clearTimeout(timer); error ? reject(error) : resolve(url)}
    server.stdout.on('data', chunk => {
      output += chunk
      const match = output.match(/http:\/\/127\.0\.0\.1:\d+/)
      if (match) finish(null, match[0])
    })
    server.stderr.on('data', chunk => {output += chunk})
    server.once('error', error => finish(error))
    server.once('exit', code => finish(new Error(`Server exited ${code}: ${output}`)))
  })
  async function get(resource) {
    const url = new URL(resource, base)
    assert.equal(url.origin, base, 'Assets must be local')
    const response = await fetch(url, {signal: AbortSignal.timeout(5000), redirect: 'error'})
    assert.equal(response.status, 200, url.href)
    return response
  }
  const html = await (await get('/')).text()
  const assets = [...html.matchAll(/(?:src|href)="([^"]+\.(?:js|css))"/g)].map(match => match[1])
  assert.ok(assets.some(asset => asset.endsWith('.js')) && assets.some(asset => asset.endsWith('.css')), 'Missing frontend assets')
  for (const asset of assets) assert.ok((await (await get(asset)).text()).length > 0, `Empty asset: ${asset}`)
  const catalog = await (await get('/api/catalog')).json()
  assert.ok(catalog.entries.some(entry => entry.documents.some(document => document.path === 'specs/example/spec.md')), 'Fixture missing from catalog')
  const document = await (await get(`/api/document?path=specs/example/spec.md&revision=${catalog.revision}`)).json()
  assert.equal(document.source, '# Embedded release test\n')
  console.log(`PASS: specview ${version}, isolated executable, embedded JS/CSS, catalog and document`)
} finally {
  if (server && server.exitCode === null && server.signalCode === null) {
    const closed = once(server, 'close')
    server.kill('SIGINT')
    const timer = setTimeout(() => server.kill('SIGKILL'), 3000)
    try {await closed} finally {clearTimeout(timer)}
  }
  await rm(temporary, {recursive: true, force: true})
}
