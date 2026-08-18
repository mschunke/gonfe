# Perguntas frequentes

## A biblioteca emite notas com valor fiscal?

Sim, quando você troca `nfe.Homologacao` por `nfe.Producao`. Mas leia antes os
[três avisos](index.md#antes-de-producao): confira os endereços dos serviços,
homologue o seu cenário tributário e guarde os segredos fora do código.

## Por que minha nota foi rejeitada com "assinatura inválida"?

Quase sempre porque o XML foi alterado depois de assinado. As causas mais
comuns:

- Reserializar o documento — passar por um formatador, um parser XML que
  reordena atributos ou um sistema que reescreve o arquivo.
- Preencher o `infNFeSupl` da NFC-e **depois** de assinar. Ele tem de vir antes.
- Gravar o arquivo com BOM ou converter a codificação.

O GoNFE insere a assinatura sem reserializar o documento justamente para evitar
isso. Se você mexer nos bytes depois de `Assinar`, a assinatura quebra. Para
conferir se um documento continua íntegro:

```go
err := xmldsig.Verificar(documento)
```

## Posso usar certificado A3, em token ou cartão?

Sim, implementando a interface `xmldsig.Assinante`, que tem só dois métodos.
Veja [A3, HSM e assinatura remota](assinatura.md#a3-hsm-e-assinatura-remota).
O núcleo da biblioteca continua sem CGO; a dependência nativa fica isolada no
seu adaptador.

Lembre que a autenticação TLS é caminho separado: você também precisará de um
`tls.Certificate` apontando para o token, passado em `sefaz.Config.TLS`.

## Como cancelo uma nota?

```go
canc, _ := evento.NovoCancelamento(evento.DadosCancelamento{
    Chave: chave, CNPJ: cnpj, UF: uf.RS, Ambiente: nfe.Producao,
    Protocolo:     prot.InfProt.NProt,
    Justificativa: "Cancelamento por erro de digitacao no pedido",
})
assinado, _ := canc.AssinarCom(cert)
ret, _ := cliente.EnviarEvento(ctx, assinado)
```

O prazo costuma ser de 24 horas a partir da autorização. Veja
[Eventos](eventos.md), que cobre também carta de correção, manifestação do
destinatário e inutilização de numeração. O CT-e e o MDF-e têm eventos próprios,
descritos em [CT-e e MDF-e](transporte.md).

## E o DANFE em PDF?

```go
documento, err := danfe.Gerar(proc, danfe.Opcoes{})
```

O pacote `danfe` gera os cinco documentos auxiliares — DANFE, cupom da NFC-e,
DACTE, DACTE OS e DAMDFE — em Go puro, sem biblioteca gráfica. Veja
[Documentos auxiliares](danfe.md).

O QR Code da NFC-e é a única peça que continua vindo de fora: a biblioteca
produz o texto, e você passa a matriz codificada.

## Por que `tipos.Decimal` em vez de `float64`?

Porque `0.1 + 0.2` não dá `0.3` em ponto flutuante binário, e a SEFAZ rejeita
notas cujos totais não fecham. Veja [Valores decimais](decimais.md).

## Por que não existe construtor a partir de `float64`?

Justamente para que o erro de ponto flutuante não entre por acidente. Se o valor
chega como `float64` de um sistema que você não controla, formate-o
explicitamente com a precisão que você quer antes de converter:

```go
d, err := tipos.ParseDecimal(strconv.FormatFloat(v, 'f', 2, 64))
```

## Meus totais estão um centavo diferentes do ERP

Provavelmente por causa da ordem das operações. O leiaute manda arredondar o
tributo **por item** e somar depois; somar as bases e arredondar no fim dá
resultado diferente. Veja
[Arredondar por item, somar depois](decimais.md#arredondar-por-item-somar-depois).

Também vale conferir o critério: a biblioteca usa arredondamento comercial
(metade para longe do zero), que é o dos manuais da SEFAZ, e não o bancário
(metade para o par) que várias linguagens adotam por padrão.

## O endereço do serviço da minha UF está errado

Pode acontecer: os estados mudam endereços sem aviso, e a tabela embutida
reflete o que estava publicado quando a versão foi escrita. Confira no
[Portal da NF-e](https://www.nfe.fazenda.gov.br/portal/webServices.aspx) e
sobreponha:

```go
cliente, err := sefaz.NovoCliente(sefaz.Config{
    // …
    Endpoints: map[sefaz.Servico]string{
        sefaz.ServicoAutorizacao: "https://endereco-correto/NFeAutorizacao4",
    },
})
```

E, por favor,
[abra uma issue](https://github.com/mschunke/gonfe/issues/new) para que a tabela
seja corrigida.

## Como testo sem certificado real?

O pacote interno `certtest` gera certificados sintéticos com a estrutura de um
A1 da ICP-Brasil, e é o que a própria suíte usa. Ele não é exportado, mas o
[código](https://github.com/mschunke/gonfe/blob/main/internal/certtest/certtest.go)
serve de modelo — a peça pública que você precisa é `certificado.De`, que monta
um `*Certificado` a partir de qualquer par chave/certificado.

Para testar a comunicação, `sefaz.Config.HTTP` aceita um `*http.Client`
qualquer, o que permite apontar para um `httptest.Server` local. É assim que os
testes do pacote `sefaz` funcionam.

## Posso usar em produção hoje?

A biblioteca cobre o ciclo completo de NF-e, NFC-e, CT-e, CT-e OS e MDF-e —
emitir, transmitir, corrigir, cancelar e imprimir —, tem testes que exercitam da
canonicalização à resposta da SEFAZ, e a assinatura é conferida contra a
especificação do W3C. Dito isso, ela ainda não chegou à 1.0 e não tem anos de
rodagem em produção acumulados. Os pacotes mais novos, como o `cteos`, têm menos
rodagem que os demais.

O caminho responsável é o de sempre: homologue com o seu cenário tributário
real, confira os endereços da sua UF e mantenha um plano de contingência. Se
encontrar um problema, [abra uma issue](https://github.com/mschunke/gonfe/issues)
— é assim que a rodagem se acumula.

## A API vai mudar?

Antes da 1.0, pode mudar entre versões menores, sempre com registro no
[CHANGELOG](https://github.com/mschunke/gonfe/blob/main/CHANGELOG.md). Depois da
1.0, valem as regras usuais do Go.

## Encontrei um erro no leiaute ou uma rejeição que a validação não pegou

[Abra uma issue](https://github.com/mschunke/gonfe/issues/new) com o código de
rejeição, o motivo devolvido pela SEFAZ e, se possível, o trecho do XML — sem
dados reais de clientes nem o certificado. Rejeições reais são a melhor fonte de
regras de validação.
