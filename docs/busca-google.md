# Busca Google para questões de estudo

Use `/questoes frações equivalentes` ou escreva `Busque questões de frações do ENEM` no estúdio. Pedidos de treino como `/praticar frações` continuam criando exercícios sem busca.

## Configuração

1. Crie ou selecione um projeto no Google Cloud, habilite o faturamento e ative a API Discovery Engine. [Pré-requisitos do Agent Search](https://docs.cloud.google.com/generative-ai-app-builder/docs/before-you-begin)
2. No Agent Search, crie um app de busca personalizada com recursos Enterprise e conecte um repositório de conteúdo de sites públicos com **busca básica de sites**, em localização **Global**. Inclua padrões de URL dos acervos que deseja pesquisar, como os de provas e gabaritos do INEP, da Fuvest e da Comvest. Confira se a prévia do app devolve resultados. Anote o ID do app na lista **Apps**. [Configuração de busca em sites](https://docs.cloud.google.com/generative-ai-app-builder/docs/website-search-checklist)
3. No mesmo projeto, crie uma chave de API dedicada e restrinja o uso à API Discovery Engine. A busca básica usa o índice público do Google para até 50 padrões configurados e dispensa verificação de domínio. A utilização pode consumir cota e gerar cobrança no projeto. [Dados de sites públicos](https://docs.cloud.google.com/generative-ai-app-builder/docs/create-datastore-ingest) · [Limites de padrões](https://docs.cloud.google.com/generative-ai-app-builder/docs/prepare-data) · [Autenticação por chave](https://docs.cloud.google.com/generative-ai-app-builder/docs/migrate-from-cse#api-key-deploy)
4. No PowerShell, informe a configuração para a sessão e abra o executável a partir dela. A entrada da chave fica mascarada.

   ```powershell
   $env:OPENLOGOS_GOOGLE_SEARCH_PROJECT = "id-do-projeto"
   $env:OPENLOGOS_GOOGLE_SEARCH_ENGINE = "id-do-app"
   $env:OPENLOGOS_GOOGLE_SEARCH_KEY = [System.Net.NetworkCredential]::new("", (Read-Host "Chave da API Discovery Engine" -AsSecureString)).Password
   .\OpenLogos.exe
   Remove-Item Env:OPENLOGOS_GOOGLE_SEARCH_KEY
   ```

O app faz uma chamada HTTPS ao método `searchLite` do Agent Search e envia a chave no cabeçalho `X-Goog-Api-Key`. As sessões locais do Codex e do Claude Code continuam no modo restrito. A etapa de planejamento usa o assunto, e o Google recebe a consulta produzida pela IA. Anotações anexadas entram apenas na etapa de síntese do agente. [Contrato REST do searchLite](https://docs.cloud.google.com/generative-ai-app-builder/docs/reference/rest/v1/projects.locations.collections.engines.servingConfigs/searchLite) · [Chaves em chamadas REST](https://docs.cloud.google.com/docs/authentication/api-keys-use#using_an_api_key_with_rest)

Em novembro de 2026, o Agent Search aceita projetos novos. A Custom Search JSON API está fechada para novos clientes, com migração prevista até 1º de janeiro de 2027. O Google pede contato comercial para a alternativa de busca na web inteira. [Visão geral da Custom Search JSON API](https://developers.google.com/custom-search/v1/overview)

## Origem exibida

- **Questão original encontrada** exige enunciado e alternativas completos, na ordem, em um único trecho do Google. O gabarito é resolvido pelo agente. A fonte permite conferir a publicação e a banca.
- **Questão adaptada pela IA** indica que um trecho do resultado fundamenta o exercício.
- **Questão criada pela IA** indica exercício elaborado pelo agente, sem vínculo com resultado de busca.

A URL retornada pelo Google aparece no card, com ação para abrir no navegador, e no Markdown exportado. Trechos de busca podem vir truncados ou faltar. A lição informa quando a busca termina sem evidência suficiente. Configuração ausente, erro HTTP e falha de conexão aparecem no estúdio.

## Teste de integração

Com as três variáveis configuradas e uma sessão ativa do provedor, rode o teste no mesmo PowerShell. Selecione `Codex` ou `Claude Code` e inclua acervos com questões do ENEM no app de busca. O teste faz uma busca Google e duas chamadas do agente.

```powershell
$env:OPENLOGOS_TEST_AI = "Codex"
go test -run '^TestGoogleQuestionProviderLive$' -v
Remove-Item Env:OPENLOGOS_TEST_AI
```

`go test ./...` também cobre o contrato HTTP e a geração dos cards com respostas simuladas, sem consumir cota.
