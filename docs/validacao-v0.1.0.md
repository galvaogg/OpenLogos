# Validação do OpenLogos v0.1.0

Validação em Windows 11 25H2 x64, build 26200, com Go 1.26.3 e Intel Core i5-11400H. Os testes usaram uma biblioteca descartável com conteúdo acadêmico.

## Desempenho

Mediana de cinco execuções de `BenchmarkGioFrame` com a mesma carga na base `45c1973` e na versão integrada. O teste mede CPU para layout e roteamento de eventos Gio a 1260 × 810. A carga inclui um quadro com 24 nós em movimento e uma biblioteca com 300 lições anotadas.

| Cenário | Base | v0.1.0 | Alocações por frame |
| --- | ---: | ---: | ---: |
| Mover o quadro | 0,865 ms | 0,483 ms | 334 → 271 |
| Minhas notas | 0,878 ms | 0,212 ms | 3.530 → 215 |
| Ler um card | 0,323 ms | 0,253 ms | 366 → 367 |

O quadro reutiliza o desenho da grade durante o deslocamento e recalcula a geometria quando muda o conteúdo. Minhas notas monta a lista uma vez ao entrar na página. A mediana caiu 44% no quadro e 76% em Minhas notas.

```powershell
go test -run '^$' -bench '^BenchmarkGioFrame$' -benchmem -benchtime 700ms -count 5
```

Use a mesma versão de `performance_windows_test.go` na base e na versão atual para repetir a comparação.

## Quadros e exportação

- O teste `TestManualBoardWorkflowThroughGio` cria um quadro em branco, edita título e formas, move um nó, conecta e edita a seta, desfaz alterações, reabre a biblioteca e exporta. O arquivo conserva as posições e usa elementos nativos do Excalidraw. [Quadro manual no Excalidraw](media/validacao-manual-excalidraw.webp)
- Três quadros do Codex foram usados como regressão de texto longo. A geometria exportada tinha 22 blocos fora da caixa na base e zero após a correção, com margem interna de pelo menos 10 unidades para os casos da amostra. [Testes de layout](../board_layout_test.go)
- O mapa de Filosofia foi inspecionado em janelas de 1260 × 810 e 750 × 580. Na janela menor, **Enquadrar** mostra a estrutura e **Focar nó** abre o texto completo em escala de leitura. [Antes](media/validacao-filosofia-antes.webp) · [Janela estreita](media/validacao-filosofia-compacto.webp) · [Nó em foco](media/validacao-filosofia-foco.webp)
- Os três quadros de comparação e o manual abriram no editor Excalidraw. Foram conferidas formas, texto, setas e rótulos. [Biologia](media/validacao-biologia-excalidraw.webp) · [Filosofia](media/validacao-filosofia-excalidraw.webp) · [Matemática](media/validacao-matematica-excalidraw.webp)
- O Codex voltou a gerar quadros para as três provas com 21, 16 e 16 nós, respectivamente. Um primeiro pedido de Filosofia produziu um nó inválido. A versão final faz uma segunda tentativa após falha de validação, com orientação sobre IDs, conexões e comprimento de texto. A repetição real passou. Uma revisão real do ciclo da água também preservou os IDs e acrescentou o nó solicitado.

## Suíte de estudo

Nove lições reais do Codex, 63 cards, 22 exercícios com gabarito explicado. As lições foram salvas e reabertas em três pastas no app. [Pastas da biblioteca de teste](media/validacao-pastas.webp)

| Pasta | Lições | Exercícios | Conteúdo conferido |
| --- | ---: | ---: | --- |
| Biologia | 3 | 8 | Características dos cordados, peixes, anfíbios, répteis, aves, mamíferos, circulação e sangue humano. |
| Filosofia | 4 | 9 | Educação e práxis em Freire, Gramsci e Piaget, liberdade e ética em Comte e Sartre, felicidade e ataraxia, silogismo e tipos de inferência em um caso de investigação. |
| Matemática | 2 | 5 | Probabilidade, união, interseção, partes, áreas e volumes de pirâmide, cone e esfera. |

No dado equilibrado da lição, os conjuntos A = {2, 4, 6} e B = {4, 5, 6} dão P(A ∪ B) = 4/6 e P(A ∩ B) = 2/6. A probabilidade condicional P(A | B) = 2/3 confirma a dependência. Os exemplos de geometria foram recalculados, incluindo o hemisfério de raio 4 cm, cuja área total é 48π cm². [Exercício no app](media/validacao-exercicio.webp) · [Probabilidade, OpenStax](https://openstax.org/books/introductory-statistics-2e/pages/3-introduction) · [Geometria, OpenStax](https://openstax.org/books/prealgebra-2e/pages/9-6-solve-geometry-applications-volume-and-surface-area)

O percurso do sangue e as características evolutivas foram conferidos com os capítulos de referência. As distinções de Piaget, estoicismo, lógica e positivismo foram verificadas com as obras e referências indicadas. [Cordados, OpenStax](https://openstax.org/books/biology-2e/pages/29-chapter-summary) · [Circulação, OpenStax](https://openstax.org/books/biology-2e/pages/40-1-overview-of-the-circulatory-system) · [Fundação Jean Piaget](https://www.fondationjeanpiaget.ch/fjp/site/presentation/index.php?DOCID=1267) · [Discours sur l’esprit positif](https://fr.wikisource.org/wiki/Discours_sur_l%E2%80%99esprit_positif) · [Stoicism](https://plato.stanford.edu/entries/stoicism/) · [Abduction](https://plato.stanford.edu/entries/abduction/)

## Busca Google e builds

- O teste da busca cobriu a rota `searchLite`, configuração ausente, erros HTTP e origem por card no leitor e no Markdown. Com o Codex real e uma resposta HTTP de teste, o agente formulou a consulta de probabilidade e usou o trecho como fonte de uma questão adaptada. O teste contra a API real depende de um Google Agent Search configurado, com projeto, app e chave. [Configuração e teste ao vivo](busca-google.md)
- O Windows x64 passou em `go test ./... -count=1`, `go test -race ./... -count=1`, `go vet ./...` e no teste de abertura do executável.
- Os builds de macOS e Linux ficaram pendentes. Os testes de compilação cruzada falharam porque o ponto de entrada e a interface foram escritos para Windows.
