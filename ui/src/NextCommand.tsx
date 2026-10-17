import {useEffect, useState} from 'react'
import {Check, Copy, Terminal} from 'lucide-react'
import type {NextAction} from './types'

export function NextCommand({action}: {action: NextAction}) {
 const [copied, setCopied] = useState(false)
 const [error, setError] = useState('')
 useEffect(() => {
  if (!copied) return
  const timeout = setTimeout(() => setCopied(false), 2000)
  return () => clearTimeout(timeout)
 }, [copied])
 const copy = async () => {
  try {
   await navigator.clipboard.writeText(action.command)
   setCopied(true); setError('')
  } catch {setError('Sélectionnez la commande pour la copier.')}
 }
 return <aside className="next-command" aria-label="Prochaine commande suggérée">
  <div className="next-command-heading"><Terminal size={15}/><strong>{action.label}</strong><span>{action.target === 'agent' ? 'Dans votre agent' : 'Dans le terminal'}</span></div>
  <div className="next-command-copy"><code tabIndex={0}>{action.command}</code><button className="icon-button" type="button" aria-label={copied ? 'Commande copiée' : 'Copier la commande'} title={copied ? 'Copié' : 'Copier'} onClick={copy}>{copied ? <Check size={14}/> : <Copy size={14}/>}</button></div>
  <p>{action.reason}</p>
  <span className="sr-only" role="status">{copied ? 'Commande copiée' : ''}</span>
  {error && <p role="status">{error}</p>}
 </aside>
}
