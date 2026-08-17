# Contribuindo

Obrigado pelo interesse. Esta página resume o que é útil contribuir e como; o
documento completo está em
[CONTRIBUTING.md](https://github.com/mschunke/gonfe/blob/main/CONTRIBUTING.md).

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
Antes de começar algo grande, abra uma issue para combinar a abordagem — a
infraestrutura comum já existe e vale reaproveitar.

## Rodando os testes

```bash
git clone https://github.com/mschunke/gonfe
cd gonfe
go test ./...
```

Tudo o que a CI verifica:

```bash
gofmt -l .            # não pode listar nada
go vet ./...
go test -race ./...
```

Os testes não acessam a internet e não precisam de certificado real: o pacote
interno `certtest` gera certificados sintéticos, e o cliente da SEFAZ é
exercitado contra um `httptest.Server`.

## Convenções

- **Português** em nomes, comentários e mensagens de erro. Os nomes dos campos
  dos documentos fiscais seguem o XSD exatamente — `VProd`, `CMunFG`,
  `ICMSSN102` — para facilitar a conferência contra o Manual de Orientação.
- **Nada de `float64`** em valor fiscal; use sempre `tipos.Decimal`.
- **Comentários explicam o porquê**, não o quê.
- **Erros com contexto**: embrulhe com `%w` e diga qual campo, arquivo ou
  endereço falhou.

## Testes que provam alguma coisa

Onde há algoritmo — dígito verificador, hash, canonicalização —, prefira casos
que um leitor consiga verificar à mão, ou contraprova por uma implementação
independente dentro do próprio teste. Um teste que só confirma o que o código já
faz não prova nada.

A canonicalização é conferida contra os casos 3.2 e 3.3 da especificação
Canonical XML 1.0 do W3C; se você tocar nela, esses testes precisam continuar
passando.

## Documentação

```bash
pip install -r docs/requirements.txt
mkdocs serve
```

## Segurança

Se você encontrar uma vulnerabilidade — algo que permita forjar uma assinatura,
vazar a chave privada ou o CSC —, não abra issue pública. Veja
[SECURITY.md](https://github.com/mschunke/gonfe/blob/main/SECURITY.md).
