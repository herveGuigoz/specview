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

## Restrictions de cette session

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

La CI fournie exécute les contrôles sur macOS, Linux et Windows. Ses résultats ne sont pas disponibles tant qu’elle n’a pas tourné. Sur chaque OS cible, vérifier également manuellement `specview <projet>` sans `--no-open` : le navigateur natif doit s’ouvrir après le démarrage ; Ctrl+C doit arrêter le serveur. Si l’ouvreur natif est absent, le serveur doit continuer et son URL rester utilisable.

## Build sans assets

Dans une copie de travail isolée, retirer `internal/web/dist` puis lancer `go build ./cmd/specview` doit échouer avec une erreur `go:embed` indiquant que les fichiers manquent. `make build` recrée les assets avant compilation.

## Acceptation et clôture

L’utilisateur a validé l’ensemble de l’application, après les ajustements d’interface et l’ajout des commandes suggérées et du lien de documentation, puis demandé la clôture des tâches. Les 25 tâches du changement `add-local-openspec-viewer` sont closes sur la base des vérifications disponibles et de cette acceptation. Aucun résultat de test navigateur, réseau réel ou CI multiplateforme supplémentaire n’a été fourni ou revendiqué lors de cette clôture. Les commandes ci-dessus restent disponibles pour compléter ces vérifications ultérieurement.
