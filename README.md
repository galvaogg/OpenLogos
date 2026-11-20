# OpenLogos

Ambiente de estudos com IA para Windows 11. Escreva o que quer aprender e comece uma lição com cards, exercícios e bibliografia. O MVP inclui três exemplos de metafísica para testar offline.

## Rodar

Abra `dist/OpenLogos.exe` depois de compilar ou receber o executável. A versão entregue é para Windows 11 x64.

Instale um dos CLIs e entre com sua conta no terminal. No app, clique no nome do agente no canto superior direito para escolher e verificar a sessão.

| Agente | Login | Versão testada |
| --- | --- | --- |
| [Codex](https://developers.openai.com/codex/cli) | `codex login` com conta ChatGPT | 0.171.0 |
| [Claude Code](https://code.claude.com/docs/en/setup) | `claude auth login` com assinatura Claude | 2.1.327 |

O app usa a sessão local e a geração consome a cota da assinatura. As lições de exemplo, a leitura e a exportação funcionam offline.

## Chat de estudo

- Escreva um assunto ou digite `/` no **Chat de estudo** para abrir o menu. Cada pedido gera uma lição com leitura, prática e resumo.

| Comando | Objetivo | Exemplo |
| --- | --- | --- |
| `/licao` | Aprender do começo | `/licao fases da Lua` |
| `/aprofundar` | Explorar relações e casos-limite | `/aprofundar efeitos das marés` |
| `/praticar` | Aplicar o conteúdo em exercícios | `/praticar frações equivalentes` |
| `/revisar` | Retomar conceitos e erros comuns | `/revisar leis de Newton` |

## Pastas, notas e Markdown

- Em **Pastas de estudo**, crie e renomeie pastas da biblioteca. Abra uma pasta para ver as lições reunidas ou iniciar um estudo com esse destino. No leitor, use o controle de pasta para mover a lição e conservar seu progresso.
- Em cada card, abra **Ver dica** ou **Pensar além** antes de responder ou avançar. Use **Escrever nota** para registrar suas ideias. O app salva as notas neste dispositivo. **Minhas notas** reabre o card de origem.
- **Aprofundar com IA**, no card, prepara um novo pedido com o trecho de estudo e a nota daquele card, quando houver. O anexo aparece no chat e pode ser removido antes do envio.
- **Arquivos de estudo** lista os arquivos `.md` em `Documentos\OpenLogos\Estudos`. Abra a pasta pelo app para editar ou adicionar anotações no seu editor. Atualize a lista e selecione o arquivo para anexá-lo.
- **Exportar Markdown** salva a lição, as dicas, o gabarito e a bibliografia na pasta Estudos. Cada exportação gera um arquivo e preserva edições anteriores.
- **Criar lição** envia o pedido e os anexos visíveis ao agente. Biblioteca, pastas, notas, progresso e preferências ficam em `%APPDATA%\OpenLogos\library.json`. A conversa recente fica disponível enquanto o app está aberto.

## Compilar

Use Go 1.26.3 no Windows.

```powershell
go test ./...
go build -trimpath -ldflags '-H=windowsgui -s -w' -o dist/OpenLogos.exe .
```

Go compila o app em um executável. Gio renderiza a interface na janela do Windows. Os sketches interativos ficam para uma próxima versão.
