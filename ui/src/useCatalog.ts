import {useEffect, useState} from 'react'
import type {Catalog} from './types'

type Data = {catalog: Catalog | null; path: string; source: string; documentError: string; error: string}
export function useCatalog(path: string) {
 const [data, setData] = useState<Data>({catalog: null, path: '', source: '', documentError: '', error: ''})
 const [connected, setConnected] = useState(false)
 useEffect(() => {
  let disposed = false, generation = 0
  let controller: AbortController | undefined
  let retry: ReturnType<typeof setTimeout> | undefined
  const load = async () => {
   const request = ++generation
   controller?.abort(); controller = new AbortController()
   const signal = controller.signal
   try {
    for (let attempt = 0; attempt < 4; attempt++) {
     const response = await fetch('/api/catalog', {signal, cache: 'no-store'})
     if (!response.ok) throw new Error('Le catalogue est indisponible.')
     const catalog = await response.json() as Catalog
     let source = '', documentError = ''
     if (path) {
      const result = await fetch('/api/document?' + new URLSearchParams({path, revision: String(catalog.revision)}), {signal, cache: 'no-store'})
      if (result.status === 409) continue
      if (result.ok) source = (await result.json()).source
      else documentError = result.status === 404 ? 'Ce document a été supprimé ou déplacé.' : 'Ce document ne peut pas être lu : ' + await result.text()
     }
     if (!disposed && request === generation) setData({catalog, path, source, documentError, error: ''})
     return
    }
    throw new Error('Les fichiers changent rapidement. Nouvelle tentative en cours…')
   } catch (error) {
    if (disposed || signal.aborted || request !== generation) return
    setData(previous => ({...previous, error: String(error instanceof Error ? error.message : error)}))
    retry = setTimeout(load, 1500)
   }
  }
  void load()
  const events = new EventSource('/api/events')
  events.onopen = () => { if (!disposed) setConnected(true) }
  events.onmessage = () => { clearTimeout(retry); void load() }
  events.onerror = () => { if (!disposed) setConnected(false) }
  const focus = () => { void load() }
  window.addEventListener('focus', focus)
  return () => {disposed = true; ++generation; controller?.abort(); clearTimeout(retry); events.close(); window.removeEventListener('focus', focus)}
 }, [path])
 return {...data, connected, loading: !!path && data.path !== path}
}
