import {registerHooks} from 'node:module'
import {readFileSync, existsSync} from 'node:fs'
import {fileURLToPath} from 'node:url'
import ts from 'typescript'
registerHooks({
 resolve(specifier, context, next) {
  if (specifier.startsWith('.') && context.parentURL) {
   const url = new URL(specifier, context.parentURL)
   if (!/\.[a-z]+$/i.test(url.pathname)) {
    for (const extension of ['.ts', '.tsx']) {const candidate = new URL(url.href + extension); if (existsSync(fileURLToPath(candidate))) return next(candidate.href, context)}
   }
  }
  return next(specifier, context)
 },
 load(url, context, next) {
  if (/\.tsx?$/.test(url)) return {format: 'module', shortCircuit: true, source: ts.transpileModule(readFileSync(fileURLToPath(url), 'utf8'), {compilerOptions: {jsx: ts.JsxEmit.ReactJSX, module: ts.ModuleKind.ESNext, target: ts.ScriptTarget.ES2022}}).outputText}
  return next(url, context)
 },
})
