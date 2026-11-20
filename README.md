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

## Estudos e Markdown

- **Novo estudo** cria uma lição sobre o assunto que você descreveu.
- **Arquivos de estudo** lista os arquivos `.md` em `Documentos\OpenLogos\Estudos`. Abra a pasta pelo app para editar ou adicionar anotações no seu editor. Atualize a lista e selecione o arquivo para anexá-lo.
- **Exportar Markdown**, dentro de uma lição, salva os cards, exercícios, gabarito e bibliografia na pasta Estudos. Edições anteriores são preservadas quando você exporta novamente.
- O assunto e o Markdown selecionado são enviados ao agente ao clicar em **Criar lição**. A biblioteca, o progresso e as preferências ficam em `%APPDATA%\OpenLogos\library.json`.

## Compilar

Use Go 1.26.3 no Windows.

```powershell
go test ./...
go build -trimpath -ldflags '-H=windowsgui -s -w' -o dist/OpenLogos.exe .
```

Go compila o app em um executável. Gio renderiza a interface na janela do Windows. Os sketches interativos ficam para uma próxima versão.
