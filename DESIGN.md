# Design do OpenLogos

O estúdio começa pelo assunto que a pessoa quer estudar. Biblioteca e arquivos ficam à esquerda. A área central alterna entre criação, lição e arquivos. No leitor, o roteiro à direita permite acessar qualquer card e o resumo.

## Visual

- Fundo carvão, superfícies grafite e ação principal em pervinca. Verde e coral marcam o feedback dos exercícios.
- Segoe UI em controles e texto. Georgia nos títulos de estudo e de cards.
- O mascote usa uma silhueta própria, com crista assimétrica, olhos ovais e uma grade de pontos. Ele olha para o campo quando recebe foco, pisca ao escrever e flutua durante a geração. O app respeita a configuração de animações do Windows.
- Os pontos reaparecem no ícone do app e inspiram os segmentos de navegação da lição. Os controles usam contorno e preenchimento para diferenciar foco, seleção e passagem do ponteiro.
- Em janelas estreitas, o painel direito sai de cena e a navegação por cards passa para a área central.

As cores, os controles e o mascote são definidos em `style_windows.go`. O ícone em `assets/openlogos.ico` está embutido no executável x64.
