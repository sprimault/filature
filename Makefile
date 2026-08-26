# Cibles en anglais, commentaires en français : convention des autres projets.

BINAIRE   ?= filature
VERSION   ?= dev
LDFLAGS    = -s -w -X main.version=$(VERSION)
SORTIE    ?= .tmp

# makefile.local porte ce qui est propre au poste — GOTMPDIR, GOCACHE — et n'est
# pas versionné. Inclus ici et non en tête : une affectation immédiate qui y
# référencerait une variable définie plus haut trouverait une chaîne vide.
#
# Passer par le Makefile plutôt que d'appeler go à la main : une commande tapée
# directement perd ces variables, et l'échec est intermittent.
-include makefile.local

.PHONY: build run test race lint vulncheck sec cover binaries clean tools

build:
	go build -trimpath -ldflags "$(LDFLAGS)" -o $(SORTIE)/$(BINAIRE) ./cmd/filature

run:
	go run ./cmd/filature

# Aucun test ne doit ouvrir de fenêtre : les runners sont sans écran, et un test
# qui exigerait xvfb n'a rien à faire dans la suite par défaut.
#
# PKG et RUN restreignent la portée sans sortir du Makefile : une commande go
# tapée directement perd les réglages de makefile.local, et l'oubli ne se voit
# pas dans la sortie.
PKG ?= ./...
RUN ?=

test:
	go test $(if $(RUN),-run '$(RUN)') $(PKG)

race:
	CGO_ENABLED=1 go test -race ./...

lint:
	golangci-lint run

vulncheck:
	govulncheck ./...

# .tmp porte les programmes jetables, qui ne sont pas versionnés : gosec les
# analyse en local et pas en intégration continue, où le dossier n'existe pas.
# Sans exclusion, la cible échoue ici et passe là-bas, ce qui finit par la faire
# ignorer — et par masquer un vrai signalement.
sec:
	gosec -exclude-dir=.tmp ./...

cover:
	go test -coverprofile=$(SORTIE)/couverture.out ./...
	go tool cover -html=$(SORTIE)/couverture.out

# Une seule cible paramétrée, pas cinq recettes : la matrice complète vit dans
# le workflow, qui est le seul endroit où elle peut être exécutée. En local on
# ne produit que ce qui se croise sans compilateur C — Windows et wasm.
binary:
	CGO_ENABLED=$(CGO) GOOS=$(OS) GOARCH=$(ARCH) go build -trimpath \
	  -ldflags "$(LDFLAGS)" -o dist/$(BINAIRE)_$(OS)_$(ARCH)$(EXT) ./cmd/filature

binaries:
	$(MAKE) binary OS=windows ARCH=amd64 CGO=0 EXT=.exe
	$(MAKE) binary OS=js ARCH=wasm CGO=0
	@echo "Les cibles linux et darwin exigent une compilation native : voir release.yml"

clean:
	rm -rf $(SORTIE) dist

# La version de golangci-lint est épinglée et partagée avec le workflow, qui la
# reprend telle quelle. Avec @latest des deux côtés, le poste prend du retard
# dès qu'une version sort : le lint passe en local et échoue en intégration
# continue, sur du code que personne n'a touché. Les deux se changent ensemble.
GOLANGCI_VERSION ?= v2.13.1

tools:
	go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_VERSION)
	go install golang.org/x/vuln/cmd/govulncheck@latest
	go install github.com/securego/gosec/v2/cmd/gosec@latest
