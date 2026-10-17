# Vérifications

Vérifications effectuées le 17 octobre 2026 sur macOS arm64, Go 1.27.2, Node.js 26.12.0.

## Résultats disponibles

- Résolution des racines, options CLI et configuration YAML : tests Go réussis.
- Inventaire imbriqué, artefacts supplémentaires, tâches hors code, erreurs et récupération : tests Go réussis.
- Projet sans droits d’écriture, traversées de chemins, liens symboliques et remplacement concurrent : tests Go réussis.
- API HTTP en mémoire : contenu/compteurs par révision, conflits, suppression, refus de mutation, Host, chemins et types MIME des assets testés.
- Suggestions de préparation, apply/archive, schémas personnalisés, métadonnées invalides, skip_specs et actualisation des commandes : tests Go réussis.
- SSE en mémoire : notification initiale, nouvelles révisions, abonnement lent et fermeture testés.
- Actualisation du catalogue de 1 001 documents sur disque local : environ 1,2 seconde après modification, avec détecteur de courses actif. Cela mesure le backend, pas encore le délai visible dans un navigateur réel.
- `go test -race ./...` : les tests exécutables dans cette session passent. Deux tests ouvrant des sockets sont explicitement ignorés ici, pour la restriction décrite ci-dessous.
- `go vet ./...`, contrôle TypeScript, cinq tests de rendu Markdown/routes/commande suggérée et build Vite : réussis.
- Build avec dépendances verrouillées depuis une copie propre : réussi. Le binaire copié répond à `--version` sans Go, Node.js ou OpenSpec dans son PATH.
- Six cibles compilées avec `CGO_ENABLED=0` : darwin/arm64, darwin/amd64, linux/arm64, linux/amd64, windows/arm64 et windows/amd64. Cela confirme la compilation, pas l’exécution native sur chaque OS.

## Restrictions du contrôle initial

L’environnement d’exécution interdit `bind` sur `127.0.0.1` (`operation not permitted`). Il interdit aussi le lancement de Chromium (enregistrement MachPort : `Permission denied`). Le serveur réel, les parcours navigateur, les captures d’écran et la revue visuelle n’ont donc pas pu être exécutés ici. Les tâches OpenSpec ont été closes à la demande explicite de l’utilisateur après sa validation de l’ensemble. Cette acceptation ne prouve pas l’exécution des tests bloqués ni les essais natifs sur chaque OS.

Le test d’intégration fourni ne désactive pas silencieusement ces vérifications : il doit échouer lorsque ces permissions manquent. Il copie le vrai binaire dans un répertoire isolé, retire les outils du PATH du serveur, prépare plus de 1 000 documents et bloque les requêtes navigateur externes.

## Exécution dans un terminal normal

Depuis la racine du dépôt :

```sh
make build
cd ui
npx playwright install chromium
cd ..
make integration
```

Pour utiliser un Chromium déjà installé : définir `SPECVIEW_BROWSER_EXECUTABLE` sur son chemin avant `make integration`. Les tests unitaires de rendu utilisent les hooks de modules Node.js ; Node.js 22.15+ est requis pour `make test` (Node.js 24 conseillé pour le développement).

Les résultats et captures sont écrits dans `test-results/` : `integration.json`, `changes-dark.png`, `reader-light.png`, `reader-mobile.png`. Le scénario contrôle lecture/source, Markdown sûr, filtres, liens directs et ancres, capacités imbriquées, archives, thèmes persistés, largeur mobile, clavier, édition de tâches, sauvegarde atomique, nouveau sous-répertoire, suppression, archivage, déconnexion/reconnexion et absence d’écriture par le lecteur.

La CI cible désormais macOS et Linux uniquement. Ses résultats ne sont pas disponibles tant qu’elle n’a pas tourné. Sur chaque OS cible, vérifier également manuellement `specview <projet>` sans `--no-open` : le navigateur natif doit s’ouvrir après le démarrage ; Ctrl+C doit arrêter le serveur. Si l’ouvreur natif est absent, le serveur doit continuer et son URL rester utilisable.

## Vérification des archives de distribution

GoReleaser OSS 2.18.0 est utilisé en local et dans les workflows. Depuis un checkout propre, sans `internal/web/dist/`, exécuter :

```sh
goreleaser check
make release
version=$(node -p "JSON.parse(require('fs').readFileSync('dist/metadata.json', 'utf8')).version")
node scripts/release.mjs verify dist "$version"
```

Le hook GoReleaser installe les dépendances npm verrouillées et compile le frontend avant Go. Le contrôle du manifeste exige les quatre archives macOS/Linux attendues et vérifie leurs SHA-256. En complément, inspecter chaque tar.gz : elle doit contenir le README et `specview`, sans frontend séparé. `git ls-files internal/web/dist` ne doit rien afficher.

Pour macOS ARM64, extraire l’archive finale hors du dépôt puis lancer le banc contre son exécutable :

```sh
extracted=$(mktemp -d)
tar -xzf "dist/specview_${version}_darwin_arm64.tar.gz" -C "$extracted"
node scripts/smoke.mjs "$extracted/specview" "$version"
```

Adapter le couple OS/architecture sur une autre machine. En CI de release, le même contrôle utilise l’archive Linux amd64 téléchargée depuis les assets du brouillon GitHub Release. La promotion publique suit exclusivement le succès des checksums et du test court, sans reconstruire les assets.

Le contrôle `scripts/smoke.mjs` copie le binaire dans un répertoire temporaire indépendant et le lance avec un PATH vide, sans frontend externe. Il vérifie la version, le démarrage HTTP, les assets JS/CSS référencés par la page, le catalogue et la lecture d’un document. Node.js pilote le test sur le runner ; le programme testé n’en dépend pas. Le processus et le dossier temporaire sont supprimés après le contrôle, y compris en cas d’échec.

Le packaging et la publication n’installent pas de navigateur. Le parcours Playwright existant reste dans la CI générale et via `make integration` pour vérifier le rendu et les interactions. Le contrôle court ne prétend pas vérifier le rendu visuel des styles et icônes.

Les deux tests `node --test scripts/release.test.mjs` couvrent les tags stables avec correspondance HEAD/tag et les checksums des quatre archives (succès, corruption et fichier manquant). Les tests de publication avec API GitHub simulée ont été retirés.

### Résultats du changement de distribution

Le 17 octobre 2026 sur macOS ARM64, GoReleaser a construit les six archives depuis une copie propre, sans assets préexistants. Inventaire, contenu des archives et SHA-256 ont été vérifiés. Le binaire macOS ARM64 extrait a passé le parcours navigateur complet avec Chromium, PATH applicatif vide, aucune requête externe ni erreur navigateur. Les essais négatifs de port interdit, navigateur absent et binaire inexistant ont échoué explicitement. La configuration GoReleaser et les deux workflows ont passé leurs validateurs ; les tests de publication simulée passent.

La chaîne intégrée a ensuite été rejouée depuis une seconde copie propre : `make release`, `SPECVIEW_REQUIRE_NETWORK_TESTS=1 make test`, contrôle des six archives et parcours navigateur du binaire final natif extrait. Les tests Go avec détection de courses ont réussi, y compris ceux ouvrant des sockets, sans omission ; typecheck et les cinq tests unitaires frontend ont réussi. Les six tests de publication, incluant le retour du sous-processus navigateur, réussissent également. Les actualisations navigateur mesurées sur cette copie vont de 468 à 1 338 ms. Le banc conserve son contrôle de délai maximal de trois secondes.

Ces résultats ne constituent pas une exécution native sur les cinq autres couples. Aucune GitHub Release ni aucun tag distant n’a été créé pendant ces vérifications ; le téléchargement réel d’assets de brouillon et leur promotion seront exécutés par Actions lors d’une publication autorisée.

## Build sans assets

Dans une copie de travail isolée, retirer `internal/web/dist` puis lancer `go build ./cmd/specview` doit échouer avec une erreur `go:embed` indiquant que les fichiers manquent. `make build` recrée les assets avant compilation.

## Acceptation et clôture

L’utilisateur a validé l’ensemble de l’application, après les ajustements d’interface et l’ajout des commandes suggérées et du lien de documentation, puis demandé la clôture des tâches. Les 25 tâches du changement `add-local-openspec-viewer` sont closes sur la base des vérifications disponibles et de cette acceptation. Aucun résultat de test navigateur, réseau réel ou CI multiplateforme supplémentaire n’a été fourni ou revendiqué lors de cette clôture. Les commandes ci-dessus restent disponibles pour compléter ces vérifications ultérieurement.

### Périmètre de distribution réduit

La distribution et la CI ciblent désormais uniquement macOS et Linux, chacun en amd64 et arm64 : quatre archives tar.gz et leur manifeste SHA-256. Les vérifications historiques à six cibles ci-dessus restent des résultats de la configuration précédente ; elles ne décrivent pas l’inventaire actuel. La documentation d’installation et les deltas OpenSpec sont alignés sur ces quatre cibles.

Après réduction du périmètre, les six tests de publication passent avec un inventaire attendu explicitement limité aux quatre archives. La configuration YAML confirme les deux systèmes et le format tar.gz ; la validation OpenSpec stricte et le contrôle des différences passent.

### Simplification des tests de distribution

À la demande de l’utilisateur, le parcours navigateur du packaging/publication est remplacé par le contrôle HTTP court ci-dessus. Les deux tests ciblés passent, ainsi que le contrôle réel de `bin/specview` (`dev`) copié hors dépôt avec PATH vide. L’ouverture d’un port local nécessite une exécution hors sandbox ; le test échoue explicitement si cette permission manque. Les résultats navigateur historiques restent ceux du contrôle précédent.
