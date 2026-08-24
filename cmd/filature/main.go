// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: Apache-2.0

// Commande filature : le jeu, et son mode serveur.
//
// Toute la configuration vient des drapeaux, lus et validés ici. Aucun flag ni
// os.Getenv dans internal/ — les dépendances sont injectées.
package main

import (
	"flag"
	"fmt"
	"os"
)

// version est injectée à la compilation par -ldflags.
var version = "dev"

func main() {
	heberger := flag.Bool("heberger", false, "héberger une partie en réseau")
	rejoindre := flag.String("rejoindre", "", "adresse d'une partie à rejoindre")
	greffons := flag.String("greffons", "greffons", "dossier des greffons")
	partie := flag.String("partie", "", "nom d'une partie enregistrée à reprendre")
	flag.Parse()

	if flag.Arg(0) == "version" {
		fmt.Println("filature", version)
		return
	}

	if err := executer(*heberger, *rejoindre, *greffons, *partie); err != nil {
		fmt.Fprintln(os.Stderr, "filature:", err)
		os.Exit(1)
	}
}

func executer(heberger bool, rejoindre, greffons, partie string) error {
	return fmt.Errorf("à implémenter : étape 5")
}
