# Design do OpenLogos

O estúdio começa no chat de estudo. A barra lateral dá acesso a pastas, notas, lições recentes e arquivos Markdown. No leitor, o roteiro à direita leva aos cards e ao resumo. As notas acompanham o card e podem dar contexto a um novo pedido da IA.

## Visual

- Fundo carvão, superfícies grafite e ação principal em pervinca. Verde e coral marcam o feedback dos exercícios.
- Segoe UI em controles e texto. Georgia nos títulos de estudo e de cards.
- O mascote usa crista assimétrica, olhos ovais e uma grade de pontos. No chat em repouso, ele pisca com um pulso breve da grade. Durante a geração, flutua para indicar atividade.
- Os pontos reaparecem no ícone do app e inspiram os segmentos de navegação da lição. Os controles usam contorno e preenchimento para diferenciar foco, seleção e passagem do ponteiro.
- O compositor e o editor de notas usam gradiente e reflexo de borda renderizados pelo Gio. A navegação desliza o card no sentido da sequência. Dicas e feedback têm entradas curtas. A configuração de animações do Windows desativa esses movimentos.
- Em janelas estreitas, o painel direito sai de cena e a navegação por cards passa para a área central.

As cores, os controles e o mascote são definidos em `style_windows.go`. O ícone em `assets/openlogos.ico` está embutido no executável x64.
