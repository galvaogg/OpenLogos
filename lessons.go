package main

func sampleLessons() []Lesson {
	return []Lesson{
		{
			ID:      "sample-navio-de-teseu",
			Title:   "O navio de Teseu",
			Topic:   "Metafísica · Identidade",
			Summary: "Investigue o que faz um objeto continuar sendo o mesmo quando suas partes mudam.",
			Cards: []Card{
				{
					Title: "Uma tábua por vez",
					Body:  "Em Atenas, as tábuas deterioradas do navio de Teseu eram substituídas por tábuas novas. Imagine que o reparo continue até que todas as peças tenham sido trocadas. É o mesmo navio que Teseu usou?",
					Hint:  "Pense num objeto seu que já foi consertado. Que troca de peça faria você começar a tratá-lo como outro objeto?",
				},
				{
					Title: "Ser o mesmo e ser igual",
					Body:  "Dois navios idênticos na aparência ainda são dois navios. Chamamos isso de semelhança qualitativa. Identidade numérica significa haver um único objeto, como o navio antes e depois de ganhar uma nova camada de tinta.",
					Hint:  "Compare duas cópias do mesmo livro com a sua cópia antes e depois de ganhar anotações. Em qual caso há um único objeto?",
				},
				{
					Title: "Agora temos dois navios",
					Body:  "Imagine que alguém guarde as tábuas antigas e as use para remontar o navio em outro porto. Um candidato tem a história contínua de uso e reparo. O outro tem a matéria original. Cada critério aponta para um candidato diferente.",
					Hint:  "Se você fosse curador de um museu, que fatos colocaria na placa de cada navio? Escreva a versão que sustenta melhor cada candidato.",
				},
				{
					Title:    "Escolha o critério",
					Question: "Se o critério adotado é a continuidade da história de uso e reparo, qual candidato ele favorece?",
					Hint:     "Siga a trajetória de uso ao longo do tempo. Em que momento cada embarcação começou a navegar?",
					Options: []string{
						"O navio remontado com as tábuas antigas",
						"O navio que recebeu as peças novas ao longo do tempo",
						"Os dois, desde que tenham a mesma aparência",
					},
					Correct:     1,
					Explanation: "O navio reparado continuou a ser usado enquanto as peças mudavam. O remontado disputa a identidade por outro critério, a preservação da matéria original.",
				},
				{
					Title:    "Teste a conclusão",
					Question: "Qual é o problema de dizer que cada um dos dois navios atuais é numericamente idêntico ao original?",
					Hint:     "Chame o original de O e os candidatos de A e B. Se A = O e B = O, que relação você pode concluir entre A e B?",
					Options: []string{
						"Os dois precisariam ter pelo menos uma tábua em comum",
						"Os dois precisariam ter exatamente a mesma aparência",
						"Os dois teriam de ser um único navio, mas estão em portos distintos ao mesmo tempo",
					},
					Correct:     2,
					Explanation: "A identidade numérica é transitiva. Se A é o original e B também é o original, A e B têm de ser o mesmo objeto. O caso descreve dois navios distintos.",
				},
			},
			Sources: []string{
				"Plutarco. Vida de Teseu, § 23. https://www.theoi.com/Text/PlutarchTheseus.html",
				"Stanford Encyclopedia of Philosophy. Identity Over Time, § 4. https://plato.stanford.edu/entries/identity-time/",
			},
		},
		{
			ID:      "sample-livre-arbitrio",
			Title:   "Livre-arbítrio e escolha",
			Topic:   "Metafísica · Ação",
			Summary: "Separe determinação, coerção e o poder de ter agido de outro modo.",
			Cards: []Card{
				{
					Title: "A decisão de devolver",
					Body:  "Lia encontra uma carteira e decide devolvê-la. Imagine que o estado do mundo antes da escolha, junto com as leis da natureza, determine exatamente essa decisão. Em que sentido ela ainda poderia ser livre?",
					Hint:  "Compare devolver a carteira por convicção e devolvê-la sob ameaça. O que mudou nas razões de Lia?",
				},
				{
					Title: "Duas exigências de liberdade",
					Body:  "Para o compatibilismo, uma ação pode ser livre mesmo num mundo determinista. Uma versão da posição destaca agir por razões próprias, sem coerção. Para o incompatibilismo, o determinismo exclui o livre-arbítrio.",
					Hint:  "Escreva uma situação que cada posição interpretaria de modo diferente. Que fato sobre a escolha cada uma considera decisivo?",
				},
				{
					Title: "Alguém estava pronto para intervir",
					Body:  "Imagine um observador que forçaria Lia a devolver a carteira se ela fosse escolher ficar com ela. Lia escolhe devolver por conta própria e o observador fica parado. Ela pode ser responsável pela ação, mesmo sem uma alternativa disponível?",
					Hint:  "Faça duas listas: o que realmente aconteceu e o que aconteceria se Lia cogitasse outra ação. Em qual delas entra a intervenção?",
				},
				{
					Title:    "Reconheça a posição",
					Question: "Uma filósofa afirma que Lia agiu livremente porque decidiu conforme suas próprias razões, mesmo supondo que sua ação estava determinada. Qual posição essa afirmação expressa?",
					Hint:     "Que relação existe entre as duas afirmações da filósofa? Ela aceita uma às custas da outra ou afirma que podem ser verdadeiras ao mesmo tempo?",
					Options: []string{
						"Compatibilismo",
						"Incompatibilismo",
						"Determinismo, por si só",
					},
					Correct:     0,
					Explanation: "A filósofa considera que liberdade e determinismo podem coexistir. O determinismo, sozinho, descreve como os acontecimentos decorrem do passado e das leis, deixando em aberto o que conta como liberdade.",
				},
				{
					Title:    "Encontre o alvo do exemplo",
					Question: "Que ideia o caso do observador põe diretamente em questão?",
					Hint:     "Lia age pelas próprias razões. Pense no que o observador retira da situação sem precisar interferir no que aconteceu.",
					Options: []string{
						"Toda ação voluntária tem causas",
						"Ser moralmente responsável exige poder ter agido de outro modo",
						"Agir por razões exige conhecer todas as alternativas",
					},
					Correct:     1,
					Explanation: "O caso convida a atribuir responsabilidade a Lia porque ela agiu por conta própria, embora o observador pudesse impedir a outra escolha. É um teste do papel das alternativas na responsabilidade.",
				},
			},
			Sources: []string{
				"Harry Frankfurt. Alternate Possibilities and Moral Responsibility (1969). https://doi.org/10.2307/2023833",
				"Stanford Encyclopedia of Philosophy. Free Will, § 2. https://plato.stanford.edu/entries/freewill/",
			},
		},
		{
			ID:      "sample-ontologia",
			Title:   "Os números existem?",
			Topic:   "Metafísica · Ontologia",
			Summary: "Use um exemplo matemático para reconhecer com quais entidades uma teoria se compromete.",
			Cards: []Card{
				{
					Title: "Apague o 7",
					Body:  "Escreva 7 numa folha e apague. O traço de lápis desapareceu, mas o que dizer do número que o símbolo representava? A ontologia pergunta que tipos de coisas existem. Neste caso, a questão é se há números além dos símbolos e práticas que usamos para contar.",
					Hint:  "Escreva a mesma quantidade com três representações diferentes. Na sua opinião, o que elas têm em comum?",
				},
				{
					Title: "Duas respostas possíveis",
					Body:  "O platonismo matemático considera os números objetos abstratos que existem independentemente das nossas inscrições e ideias. Um nominalista sobre números rejeita esses objetos e precisa explicar como a matemática funciona sem assumir que eles existem.",
					Hint:  "Imagine um universo sem seres que contem. Como cada posição explicaria o que ainda é verdade sobre números nesse universo?",
				},
				{
					Title: "O que uma teoria pressupõe?",
					Body:  "Uma teoria se compromete com certas entidades quando precisa delas para que suas afirmações sejam verdadeiras. Um platônico pode ler ‘existe um primo maior que 10’ como uma afirmação literal sobre números. Um nominalista pode tentar preservar o uso da frase dando a ela outra análise.",
					Hint:  "Escolha uma frase cotidiana começando com ‘existe’. Que tipos de coisa precisariam existir para a frase ser verdadeira numa leitura literal?",
				},
				{
					Title:    "Separe objeto e símbolo",
					Question: "Qual afirmação expressa o platonismo matemático?",
					Hint:     "Imagine que ninguém escreva ou pense em números. O que cada alternativa prevê que ainda existiria?",
					Options: []string{
						"O número 7 é o conjunto de todos os traços escritos com o símbolo 7",
						"O número 7 existe enquanto alguém consegue pensar nele",
						"O número 7 é um objeto abstrato, mesmo que ninguém o escreva ou pense nele",
					},
					Correct:     2,
					Explanation: "A posição atribui existência independente ao número. Uma inscrição física ou uma ideia pode representá-lo, mas para o platônico sua existência dispensa essas representações.",
				},
				{
					Title:    "Encontre o compromisso",
					Question: "Uma teoria afirma literalmente que existem números primos e só pode ser verdadeira se houver números abstratos. Qual é o compromisso ontológico da teoria?",
					Hint:     "Complete a frase ‘se essa teoria é verdadeira, então precisa existir…’ usando a condição que o enunciado dá.",
					Options: []string{
						"A existência de números abstratos",
						"A existência de um símbolo diferente para cada número",
						"A existência de um grupo físico para cada número primo",
					},
					Correct:     0,
					Explanation: "Nesse exemplo, a teoria exige que números abstratos existam para que sua afirmação seja verdadeira. Símbolos e grupos físicos são formas de representar ou exemplificar quantidades.",
				},
			},
			Sources: []string{
				"W. V. O. Quine. On What There Is (1948). https://www.jstor.org/stable/20123117",
				"Stanford Encyclopedia of Philosophy. Platonism in Metaphysics, § 4.1. https://plato.stanford.edu/entries/platonism/",
			},
		},
	}
}
