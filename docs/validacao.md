# Validação

Cada ida à SEFAZ que termina em rejeição custa tempo e, em emissão interativa,
custa um cliente esperando no balcão. `NFe.Validar` existe para pegar antes o
que dá para pegar antes.

## O que a validação cobre

```go
if err := n.Validar(); err != nil {
    return err
}
```

- **Estrutura**: campos obrigatórios presentes, comprimentos dentro do leiaute,
  códigos dentro dos domínios permitidos.
- **Dígitos verificadores**: CNPJ e CPF do emitente, do destinatário e do
  responsável técnico.
- **Coerência entre grupos**: `vProd` do item bate com `qCom × vUnCom`; os
  totais batem com a soma dos itens; a soma dos pagamentos bate com o valor da
  nota mais o troco.
- **Exclusividade**: CNPJ ou CPF, nunca os dois; exatamente uma variação de
  ICMS, de PIS, de COFINS; ICMS ou ISSQN por item, nunca ambos.
- **Regras condicionais**: contingência exige `dhCont` e justificativa entre 15
  e 256 caracteres; homologação exige a razão social fixa no destinatário;
  destinatário contribuinte exige inscrição estadual.
- **Regras da NFC-e**: operação interna, saída, consumidor final, QR Code
  presente, sem grupo de cobrança, sem itens de serviço.

!!! warning "O que ela não cobre"

    `Validar` **não** substitui a validação de esquema XSD nem as centenas de
    regras de negócio que a SEFAZ aplica na autorização — regras de CFOP contra
    natureza da operação, benefícios fiscais por UF, sublimites do Simples
    Nacional, e por aí vai. Passar aqui reduz muito as rejeições; não as
    elimina. Homologue com notas reais do seu cenário tributário.

## Lendo os erros

`Validar` devolve `nfe.Erros`, uma fatia de `nfe.Erro`. Cada item aponta o
caminho do campo no leiaute e descreve o problema:

```go
if err := n.Validar(); err != nil {
    erros, ok := err.(nfe.Erros)
    if !ok {
        return err
    }
    for _, e := range erros {
        fmt.Printf("%s: %s\n", e.Campo, e.Mensagem)
    }
}
```

```text
ide.natOp: natureza da operação é obrigatória
emit.CNPJ: validacao: documento inválido: primeiro dígito verificador do CNPJ não confere
det[2].prod.NCM: NCM "1234"; informe 8 dígitos, ou "00" nos casos previstos
det[2].prod.vProd: vProd é 30.00 mas qCom × vUnCom dá 25.00
pag: a soma dos pagamentos é 50.00; esperado 99.70 (vNF mais o troco)
```

O caminho do campo é o mesmo do leiaute, com o índice do item entre colchetes
a partir de 1 — o que permite ligar o erro à linha certa na interface do seu
sistema.

Como `Erros` implementa `error`, imprimir direto também funciona:

```go
fmt.Println(n.Validar())
// nfe: 5 inconsistências:
//   - ide.natOp: natureza da operação é obrigatória
//   - ...
```

## Tolerância de arredondamento

Os somatórios são conferidos com um centavo de folga, que é o que o próprio
leiaute admite por causa dos rateios de frete e desconto. A tolerância é
ajustável globalmente:

```go
nfe.ToleranciaCentavos = tipos.D("0.02")
```

Mexer nisso é raro e vale pensar duas vezes: uma tolerância maior esconde erros
de cálculo que a SEFAZ vai apontar de qualquer forma.

## Validando documentos avulsos

O pacote [`validacao`](https://pkg.go.dev/github.com/mschunke/gonfe/validacao)
é independente do `nfe` e serve para validar entradas em qualquer lugar do seu
sistema — no cadastro de clientes, por exemplo:

```go
if err := validacao.ValidarCNPJ(entrada); err != nil {
    return err
}
if validacao.EhCPF(entrada) { /* … */ }

// Aceita qualquer um dos dois, decidindo pelo comprimento.
err := validacao.ValidarCPFouCNPJ(entrada)
```

Máscaras são aceitas e descartadas: `"11.222.333/0001-81"` e
`"11222333000181"` são equivalentes.

### CNPJ alfanumérico

`ValidarCNPJ` aceita o CNPJ alfanumérico introduzido pela Instrução Normativa
RFB 2.229/2024, em que as doze primeiras posições podem conter letras. O cálculo
dos dígitos verificadores usa o código ASCII de cada caractere menos 48, o que
faz o CNPJ numérico ser um caso particular do mesmo algoritmo.

!!! note "O leiaute 4.00 ainda é numérico"

    Um CNPJ alfanumérico válido para `validacao.ValidarCNPJ` pode ser rejeitado
    pela SEFAZ: o campo `CNPJ` do leiaute 4.00 ainda restringe a entrada a 14
    dígitos numéricos. A adaptação da NF-e ao formato alfanumérico virá por Nota
    Técnica.

### Inscrição estadual

`ValidarIE` confere o **formato** — comprimento e composição — de acordo com a
UF:

```go
err := validacao.ValidarIE("0961234567", uf.RS)
err := validacao.ValidarIE("ISENTO", uf.SP)   // aceito
```

A conferência dos dígitos verificadores **não** é feita, e isso é deliberado:
cada estado adota um algoritmo próprio, e uma implementação incompleta
rejeitaria inscrições legítimas, impedindo a emissão. Trate a função como filtro
de digitação e deixe a validação definitiva com a SEFAZ, que a faz na
autorização — ou, se precisar confirmar antes, use
`Cliente.ConsultarCadastro`.

### Formatação

```go
validacao.FormatarCPF("52998224725")     // 529.982.247-25
validacao.FormatarCNPJ("11222333000181") // 11.222.333/0001-81
validacao.FormatarCEP("90010000")        // 90010-000
```

Entradas com comprimento inesperado voltam intactas, para que a formatação
nunca corrompa um dado que você está exibindo.

## Chave de acesso

```go
if err := chave.Validar(s); err != nil {
    return err
}
c, err := chave.Parse(s)  // decompõe em cUF, ano, mês, CNPJ, modelo, série…
```

`Parse` aceita a chave com pontuação, com o prefixo `NFe` do atributo `Id` e
formatada em grupos de quatro, como impressa no DANFE. `chave.Formatar` faz o
caminho inverso.
