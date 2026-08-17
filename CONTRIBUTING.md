# Contribuindo

Obrigado pelo interesse. Este documento diz o que é útil contribuir e como.

## O que ajuda mais

**Rejeições reais.** Se a SEFAZ rejeitou uma nota por algo que a validação local
devia ter pego, isso é ouro. Abra uma issue com o código de rejeição, o motivo
devolvido e o trecho do XML — sem dados reais de clientes e sem certificado.

**Endereços de serviço divergentes.** Os estados mudam endereços sem aviso. Se o
que está na tabela não bate com o Portal da NF-e, avise; a correção é de uma
linha.

**Cenários tributários.** A biblioteca foi exercitada em venda comum, Simples
Nacional e substituição tributária. Diferimento, partilha interestadual,
exportação e regimes específicos de UF têm menos rodagem.

**Os itens do roteiro.** Eventos, distribuição de DF-e, CT-e, MDF-e e DANFE.
Antes de começar algo grande, abra uma issue para combinarmos a abordagem — a
infraestrutura comum já existe e vale reaproveitar.

## Ambiente

Você precisa de Go 1.23 ou mais recente. Nada além disso: a biblioteca não usa
CGO nem ferramentas externas.

```bash
git clone https://github.com/mschunke/gonfe
cd gonfe
go test ./...
```

Para rodar tudo o que a CI roda:

```bash
gofmt -l .            # não pode listar nada
go vet ./...
go test -race ./...
go test -coverprofile=cobertura.out ./... && go tool cover -func=cobertura.out
```

## Convenções do código

**Português.** Nomes de tipo e de função, comentários e mensagens de erro são em
português. Os nomes dos campos dos documentos fiscais seguem o XSD exatamente —
`VProd`, `CMunFG`, `ICMSSN102` — para que a conferência contra o Manual de
Orientação do Contribuinte seja direta. Não traduza nem "melhore" esses nomes.

**Nada de `float64` em valor fiscal.** Use `tipos.Decimal` sempre. Se você
precisar de um construtor novo, ele não deve receber `float64`.

**Comentários explicam o porquê.** O código já diz o quê. Comentário bom é o que
registra a regra do leiaute, a razão de uma decisão ou a armadilha que a
próxima pessoa cairia. Comentário que repete a assinatura da função é ruído.

**Erros com contexto.** Embrulhe com `%w` e diga qual campo, qual arquivo, qual
endereço. `fmt.Errorf("sefaz: falha na comunicação com %s: %w", endereco, err)`
economiza meia hora de quem for depurar.

**Formatação.** `gofmt` resolve. A CI reprova o que não estiver formatado.

## Testes

Toda mudança de comportamento precisa de teste. Alguns padrões que o projeto
segue e que vale manter:

- **Vetores conferíveis.** Onde há algoritmo — dígito verificador, hash,
  canonicalização —, prefira casos que um leitor consiga verificar à mão, ou
  contraprova por uma implementação independente dentro do próprio teste. Um
  teste que só confirma o que o código faz não prova nada.
- **Casos oficiais.** A canonicalização é conferida contra os casos 3.2 e 3.3 da
  especificação Canonical XML 1.0 do W3C. Se você tocar nela, esses testes
  precisam continuar passando.
- **Sem rede.** Os testes não acessam a internet. Para exercitar o cliente
  SEFAZ, use `httptest.Server` e `sefaz.Config.HTTP`, como em
  `sefaz/cliente_test.go`.
- **Sem certificado real.** Use `internal/certtest` para gerar certificados
  sintéticos.
- **Mensagens em português**, dizendo o que se esperava e o que veio:
  `t.Errorf("vProd = %s, queria %s", got, want)`.

## Documentação

A documentação de referência é o próprio GoDoc — comentários nos tipos e funções
exportados. Os guias em `docs/` cobrem o que não cabe em GoDoc: o porquê das
decisões e os caminhos completos.

Para ver o site localmente:

```bash
pip install -r docs/requirements.txt
mkdocs serve
```

## Enviando a mudança

1. Abra uma issue antes de mudanças grandes, para combinarmos a abordagem.
2. Trabalhe em um branch a partir de `main`.
3. Deixe a CI verde: formatação, `vet`, testes com `-race`, `staticcheck` e
   `govulncheck`.
4. Escreva a mensagem de commit em português, no imperativo, explicando o
   porquê e não só o quê. Se o commit corrige uma rejeição da SEFAZ, cite o
   código.
5. Abra o pull request descrevendo o problema que ele resolve.

## Segurança

Se você encontrar uma vulnerabilidade — algo que permita forjar uma assinatura,
vazar a chave privada ou o CSC —, não abra issue pública. Veja
[SECURITY.md](SECURITY.md).

## Licença

Ao contribuir, você concorda em licenciar sua contribuição sob a
[licença MIT](LICENSE) do projeto.
