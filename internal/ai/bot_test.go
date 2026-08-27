// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: Apache-2.0

package ai

import (
	"strings"
	"testing"
)

// TestCheckProtocolRejectsOtherVersions vérifie qu'un bot d'une autre version
// est écarté, et que le refus dit quoi faire.
//
// Le passage des six types de message à l'anglais a périmé les protocoles 1
// et 2 : un bot écrit contre eux enverrait « joue » là où ce binaire attend
// « play ». Sans ce contrôle il échouerait au troisième message, sur un type
// inconnu qui ne dit pas ce qui s'est passé — et son auteur n'a que ce message
// pour comprendre.
func TestCheckProtocolRejectsOtherVersions(t *testing.T) {
	if err := checkProtocol(BotProtocol); err != nil {
		t.Errorf("la version parlée est refusée : %v", err)
	}

	for _, annoncee := range []int{0, 1, 2} {
		err := checkProtocol(annoncee)
		if err == nil {
			t.Errorf("le protocole %d est accepté", annoncee)
			continue
		}
		if !strings.Contains(err.Error(), "mettre le bot a jour") {
			t.Errorf("protocole %d refusé par %q, sans dire de mettre le bot à jour",
				annoncee, err)
		}
	}

	err := checkProtocol(BotProtocol + 1)
	if err == nil {
		t.Fatalf("un protocole plus récent que %d est accepté", BotProtocol)
	}
	if !strings.Contains(err.Error(), "mettre le jeu a jour") {
		t.Errorf("refusé par %q, sans dire de mettre le jeu à jour", err)
	}
}
