// Release orchestration. GoReleaser owns compilation, archives and checksums.
import {execFileSync} from 'node:child_process'
import {createHash} from 'node:crypto'
import {mkdtempSync, readFileSync, readdirSync, mkdirSync, rmSync} from 'node:fs'
import {tmpdir} from 'node:os'
import path from 'node:path'
import {pathToFileURL} from 'node:url'

export const run = (command, args, options = {}) => (execFileSync(command, args, {encoding: 'utf8', stdio: ['ignore', 'pipe', 'inherit'], ...options}) ?? '').trim()

export function validateTag(tag, execute = run) {
  if (!/^v(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$/.test(tag)) throw new Error('Expected a stable tag vX.Y.Z')
  const tagged = execute('git', ['rev-parse', '--verify', `refs/tags/${tag}^{commit}`])
  if (execute('git', ['rev-parse', 'HEAD']) !== tagged) throw new Error('HEAD does not match the tagged commit')
  return tag.slice(1)
}

export function archiveNames(version) {
  return ['darwin', 'linux'].flatMap(os => ['amd64', 'arm64'].map(arch => `specview_${version}_${os}_${arch}.tar.gz`)).sort()
}

export function verifyArchives(directory, version) {
  const expected = archiveNames(version)
  const actual = readdirSync(directory).filter(name => /\.(zip|tar\.gz)$/.test(name)).sort()
  if (JSON.stringify(actual) !== JSON.stringify(expected)) throw new Error('Unexpected archive inventory')
  const lines = readFileSync(path.join(directory, 'SHA256SUMS'), 'utf8').trim().split(/\r?\n/)
  const entries = lines.map(line => {
    const match = /^([a-f0-9]{64}) [ *](specview_[^/\\\s]+)$/.exec(line)
    if (!match) throw new Error('Invalid checksum manifest')
    return {hash: match[1], name: match[2]}
  })
  if (JSON.stringify(entries.map(entry => entry.name).sort()) !== JSON.stringify(expected)) throw new Error('Checksum inventory mismatch')
  for (const entry of entries) {
    const hash = createHash('sha256').update(readFileSync(path.join(directory, entry.name))).digest('hex')
    if (hash !== entry.hash) throw new Error(`Checksum mismatch: ${entry.name}`)
  }
}

function findRelease(tag, execute) {
  // Listing includes drafts with authentication; API failures must never imply absence.
  const pages = JSON.parse(execute('gh', ['api', '--paginate', '--slurp', 'repos/{owner}/{repo}/releases']))
  return pages.flat().find(release => release.tag_name === tag)
}

export function prepareRelease(tag, execute = run) {
  validateTag(tag, execute)
  const release = findRelease(tag, execute)
  if (!release) return
  if (!release.draft) throw new Error('Refusing to overwrite a public release')
  execute('gh', ['release', 'delete', tag, '--yes']) // Keep the tag itself.
}

export function publishRelease(tag, execute = run) {
  const version = validateTag(tag, execute)
  const release = findRelease(tag, execute)
  if (!release?.draft) throw new Error('Expected an existing draft release')
  const expected = [...archiveNames(version), 'SHA256SUMS'].sort()
  const assets = release.assets.map(asset => asset.name).sort()
  if (JSON.stringify(assets) !== JSON.stringify(expected)) throw new Error('Unexpected release assets')
  const temporary = mkdtempSync(path.join(tmpdir(), 'specview-release-test-'))
  try {
    execute('gh', ['release', 'download', tag, '--dir', temporary])
    verifyArchives(temporary, version)
    const extracted = path.join(temporary, 'extracted')
    mkdirSync(extracted)
    execute('tar', ['-xzf', path.join(temporary, `specview_${version}_linux_amd64.tar.gz`), '-C', extracted])
    execute(process.execPath, ['scripts/smoke.mjs', path.join(extracted, 'specview'), version], {stdio: 'inherit'})
    // Promote the already tested assets. No rebuild or upload occurs here.
    execute('gh', ['release', 'edit', tag, '--draft=false', '--latest'])
  } finally {
    rmSync(temporary, {recursive: true, force: true})
  }
}

if (process.argv[1] && import.meta.url === pathToFileURL(path.resolve(process.argv[1])).href) {
  try {
    const [command, argument, version] = process.argv.slice(2)
    if (command === 'validate-tag') console.log(validateTag(argument))
    else if (command === 'prepare') prepareRelease(argument)
    else if (command === 'publish') publishRelease(argument)
    else if (command === 'verify') verifyArchives(argument, version)
    else throw new Error('Usage: node scripts/release.mjs validate-tag|prepare|publish <tag> OR verify <directory> <version>')
  } catch (error) {
    console.error(error.message)
    process.exitCode = 1
  }
}
