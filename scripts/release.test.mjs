import {test} from 'node:test'
import assert from 'node:assert/strict'
import {createHash} from 'node:crypto'
import {mkdtempSync, writeFileSync, rmSync} from 'node:fs'
import {tmpdir} from 'node:os'
import path from 'node:path'
import {archiveNames, validateTag, verifyArchives} from './release.mjs'

test('release tags must be stable versions and match HEAD', () => {
  assert.equal(validateTag('v1.2.3', () => 'same-commit'), '1.2.3')
  for (const tag of ['1.2.3', 'v1.2', 'v01.2.3', 'v1.2.3-rc.1']) assert.throws(() => validateTag(tag))
  assert.throws(() => validateTag('v1.2.3', (_, args) => args.includes('HEAD') ? 'other' : 'tagged'), /HEAD/)
})

test('all four archives are required and corruption is rejected', t => {
  const dir = mkdtempSync(path.join(tmpdir(), 'specview-checksums-'))
  t.after(() => rmSync(dir, {recursive: true, force: true}))
  const names = archiveNames('1.2.3')
  assert.equal(names.length, 4)
  const manifest = names.map(name => {
    writeFileSync(path.join(dir, name), name)
    return createHash('sha256').update(name).digest('hex') + '  ' + name
  }).join('\n')
  writeFileSync(path.join(dir, 'SHA256SUMS'), manifest)
  verifyArchives(dir, '1.2.3')
  writeFileSync(path.join(dir, names[0]), 'corrupted')
  assert.throws(() => verifyArchives(dir, '1.2.3'), /Checksum mismatch/)
  rmSync(path.join(dir, names[0]))
  assert.throws(() => verifyArchives(dir, '1.2.3'), /inventory/)
})
