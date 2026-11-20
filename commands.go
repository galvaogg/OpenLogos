package main

import (
	"errors"
	"strings"
)

type studyCommand struct {
	name, description, guidance, boardKind string
}

var studyCommands = []studyCommand{
	{"/licao", "Aprender um assunto do começo", "Organize a lição do básico ao aplicado. Explique os pré-requisitos essenciais.", ""},
	{"/aprofundar", "Explorar causas, limites e exemplos", "Vá além da introdução. Explore o porquê, compare ideias próximas, mostre um caso-limite e proponha exercícios de aplicação.", ""},
	{"/praticar", "Treinar com exercícios de aplicação", "Priorize problemas concretos e escolhas que exigem raciocínio. Use os cards de leitura como preparação curta com exemplos resolvidos e os exercícios para transferência.", ""},
	{"/revisar", "Recuperar os pontos centrais", "Faça uma revisão concisa dos conceitos centrais. Inclua erros comuns e perguntas de recuperação ativa, evitando simplesmente repetir as explicações.", ""},
	{name: "/mapa", description: "Relacionar conceitos em um mapa mental", boardKind: "mindmap"},
	{name: "/fluxo", description: "Mostrar etapas e decisões", boardKind: "flowchart"},
	{name: "/linha-do-tempo", description: "Organizar eventos em sequência", boardKind: "timeline"},
}

type studyRequest struct {
	command, topic, boardKind string
}

func parseStudyRequest(draft string) (studyRequest, error) {
	draft = strings.TrimSpace(draft)
	if !validText(draft, 240) {
		return studyRequest{}, errors.New("Digite o assunto para criar uma lição.")
	}
	if !strings.HasPrefix(draft, "/") {
		return studyRequest{command: studyCommands[0].name, topic: draft}, nil
	}
	fields := strings.Fields(draft)
	for _, command := range studyCommands {
		if strings.EqualFold(fields[0], command.name) {
			topic := strings.TrimSpace(draft[len(fields[0]):])
			if topic == "" {
				return studyRequest{}, errors.New("Digite um assunto depois de " + command.name + ".")
			}
			return studyRequest{command: command.name, topic: topic, boardKind: command.boardKind}, nil
		}
	}
	return studyRequest{}, errors.New("Comando desconhecido. Clique em / para ver os disponíveis.")
}

func (request studyRequest) guidance() string {
	for _, command := range studyCommands {
		if request.command == command.name && command.guidance != "" {
			return command.guidance
		}
	}
	return studyCommands[0].guidance
}
