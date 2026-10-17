# Architecture

`cmd/specview` résout un projet, ouvre une racine filesystem confinée, démarre le catalogue et le serveur HTTP. `internal/project` choisit la racine locale, `internal/catalog` produit les snapshots en mémoire, `internal/server` expose la consultation et les notifications, `internal/web` embarque le build React/Vite.

## Contrat HTTP

Tous les endpoints acceptent GET et HEAD, refusent POST/PUT/PATCH/DELETE avec 405, n’activent pas CORS et utilisent `Cache-Control: no-store`. Le serveur écoute uniquement sur `127.0.0.1` et refuse les Host étrangers. Les assets proviennent exclusivement de `embed.FS`.

- `/api/catalog` : `{revision, project, entries, diagnostics, stale}`. Une entrée expose `{id, name, category, documents, progress, nextAction?}` ; catégorie : `specs`, `changes` ou `archive`. Un document expose `{path, name, kind, error?}`. Progression : `{done, total}` ; 0/0 signifie aucune progression mesurable.
- `/api/document?path=<chemin relatif encodé>&revision=<révision>` : `{revision, path, source}`. 400 sans révision valide, 403 pour chemin interdit, 404 si absent, 409 si la révision a changé, 422 si le document est illisible.
- `/api/events` : flux SSE. `data: <révision>` à la connexion et à chaque changement ; commentaire heartbeat toutes les dix secondes. HEAD retourne uniquement les en-têtes. Le client resynchronise entièrement après reconnexion.

## Actualisation

Un scan séquentiel chaque seconde lit les contenus et leurs empreintes, sans se fier aux seules dates ou tailles. Les nouveaux répertoires, sauvegardes atomiques et déplacements en archive sont ainsi détectés. Les snapshots publiés sont immuables ; leurs compteurs et contenus proviennent des mêmes lectures. Une seule révision est retenue, donc un client recevant 409 recharge le catalogue puis le document. Le frontend publie ensemble ces deux réponses et rejette les réponses obsolètes.

Une erreur locale produit un diagnostic sur le document sans bloquer les autres. Une erreur globale conserve le dernier snapshot marqué périmé. Aucun journal d’événements ni index ne persiste. Les files SSE sont bornées à la dernière révision et nettoyées à la déconnexion. L’arrêt du processus annule les requêtes et le scan.

## Limites

Le lecteur inventorie récursivement les Markdown de `specs/` et `changes/`, en séparant `changes/archive/`. Les chemins sont les identifiants, jamais le seul nom. Les métadonnées YAML restent informatives et les erreurs ne sont pas réparées. Les liens symboliques ne sont pas inventoriés ; les lectures utilisent `os.Root` pour empêcher les sorties de racine, y compris lors de remplacements concurrents. Aucun répertoire projet n’est exposé par un serveur de fichiers générique.

Un ensemble d’écritures externes n’est pas une transaction : le scan converge vers l’état stabilisé. L’objectif de rafraîchissement de trois secondes vise 1 000 documents Markdown de taille modérée sur disque local. Les stores externes et les workspaces multiples sont hors périmètre.

## Commande suggérée

`nextAction` contient `{label, command, target, reason}`, avec `target` valant `agent` ou `terminal`. Il est présent uniquement pour les changements actifs. La suggestion se calcule depuis le même snapshot que les tâches et tient compte du schéma de projet, de la surcharge `.openspec.yaml` et de `skip_specs`. Pour `spec-driven`, proposition et deltas précèdent les tâches ; le design est conditionnel et son absence seule ne bloque pas la suggestion d’implémentation. Un schéma inconnu, une métadonnée invalide, un artefact illisible ou des tâches non mesurables renvoient à `openspec status`. Les arguments de commandes terminal sont protégés par des guillemets shell. Les commandes d’agent utilisent les noms de skills OpenSpec de ce projet. L’interface ne propose que leur copie, sans nouvel endpoint ni exécution de commande.
