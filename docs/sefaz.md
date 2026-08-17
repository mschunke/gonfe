# Serviços da SEFAZ

O pacote [`sefaz`](https://pkg.go.dev/github.com/mschunke/gonfe/sefaz) conversa
com os ambientes autorizadores por SOAP 1.2 sobre TLS com autenticação mútua.
Nenhuma biblioteca de SOAP é usada — o envelope é montado e lido diretamente.

## Montando o cliente

```go
cliente, err := sefaz.NovoCliente(sefaz.Config{
    UF:          uf.RS,
    Ambiente:    nfe.Homologacao,
    Modelo:      nfe.ModeloNFe,
    Certificado: cert,
})
```

O cliente resolve sozinho qual ambiente autorizador atende a sua UF. Ele é
seguro para uso concorrente: não guarda estado entre chamadas, e reaproveita
conexões.

Todas as operações recebem um `context.Context`, então o cancelamento e o prazo
ficam sob o seu controle:

```go
ctx, cancelar := context.WithTimeout(context.Background(), 2*time.Minute)
defer cancelar()
```

## Status do serviço

É a chamada de diagnóstico. Se ela funciona, o certificado é válido, o
handshake TLS passou e o endereço está certo.

```go
status, err := cliente.StatusServico(ctx)
if err != nil {
    return err
}
if !status.EmOperacao() {
    return fmt.Errorf("%d %s", status.CStat, status.XMotivo)
}
```

## Autorização

```go
lote, err := nfe.MontarLote(idLote, sincrono, notasAssinadas...)
if err != nil {
    return err
}

envio, err := cliente.Autorizar(ctx, lote)
if err != nil {
    return err
}
```

**No envio síncrono** o protocolo vem na mesma resposta:

```go
if envio.ProtNFe != nil && envio.ProtNFe.Autorizada() {
    proc, _ := nfe.MontarNFeProc(assinada, envio.ProtNFe)
}
```

**No assíncrono** vem um recibo, que precisa ser consultado depois:

```go
resultado, err := cliente.EsperarProcessamento(ctx, envio.Recibo(), 3*time.Second, 20)
if err != nil {
    return err
}
prot := resultado.ProtocoloDa(chaveDaNota)
```

`EsperarProcessamento` consulta o recibo repetidamente até o lote sair do estado
"em processamento", respeitando o intervalo e o cancelamento do contexto. Para
controlar o laço você mesmo, use `ConsultarRecibo` diretamente.

!!! warning "Rejeição do lote e rejeição da nota são coisas diferentes"

    `Autorizar` devolve erro quando a SEFAZ recusa o **lote** — falha de
    esquema, certificado inválido, ambiente errado. Quando o lote é aceito mas
    uma **nota** é rejeitada, não há erro: o motivo está no `cStat` e no
    `xMotivo` do protocolo daquela nota. Sempre confira `prot.Autorizada()`.

## Consulta pela chave

```go
consulta, err := cliente.ConsultarNFe(ctx, chaveDeAcesso)
if err != nil {
    return err
}
if consulta.Autorizada() {
    fmt.Println(consulta.ProtNFe.Resumo())
}
for _, e := range consulta.ProcEventoNFe {
    fmt.Println(e.RetEvento.InfEvento.XEvento, e.RetEvento.InfEvento.DhRegEvento)
}
```

Serve para conferir a situação de uma nota, recuperar um protocolo perdido e
listar os eventos registrados — cancelamentos, cartas de correção,
manifestações.

## Consulta de cadastro

```go
cadastro, err := cliente.ConsultarCadastro(ctx, sefaz.ConsultaCadastro{
    CNPJ: "12345678000195",
})
for _, c := range cadastro.InfCons.InfCad {
    fmt.Println(c.XNome, c.IE, c.Habilitado())
}
```

Nem todas as UFs oferecem esse serviço; `URL` devolve erro quando o autorizador
não o expõe.

## Tratamento de erros

Rejeições da SEFAZ viram `*sefaz.ErroSefaz`, com o código e o motivo originais:

```go
var rejeicao *sefaz.ErroSefaz
if errors.As(err, &rejeicao) {
    log.Printf("rejeição %d: %s", rejeicao.CStat, rejeicao.XMotivo)
}
```

Erros de rede, TLS e HTTP vêm embrulhados com o endereço que falhou, o que
poupa tempo quando o problema é de infraestrutura.

## Endereços dos serviços

A biblioteca embute uma tabela com os endereços de todas as UFs, nos dois
ambientes e nos dois modelos. Para inspecioná-la:

```go
tabela := sefaz.TabelaDeEndpoints(nfe.ModeloNFe, nfe.Producao)
for unidade, servicos := range tabela {
    fmt.Println(unidade, servicos[sefaz.ServicoAutorizacao])
}
```

!!! warning "Confira antes de produção"

    Os estados mudam endereços sem aviso, e a tabela reflete o que estava
    publicado no Portal da NF-e quando esta versão foi escrita. Um endereço
    errado se manifesta como falha de conexão, não como rejeição.

    Rode `exemplos/status-servico` para a sua UF e, se divergir, sobreponha:

    ```go
    cliente, err := sefaz.NovoCliente(sefaz.Config{
        UF: uf.RS, Ambiente: nfe.Producao,
        Modelo: nfe.ModeloNFe, Certificado: cert,
        Endpoints: map[sefaz.Servico]string{
            sefaz.ServicoAutorizacao: "https://endereco-correto/NFeAutorizacao4",
        },
    })
    ```

### Ambientes autorizadores

A maior parte dos estados delega o processamento a um ambiente compartilhado:

| Autorizador | Atende |
| --- | --- |
| Próprio | AM, BA, GO, MG, MS, MT, PE, PR, SP |
| SVAN — Sefaz Virtual do Ambiente Nacional | MA |
| SVRS — Sefaz Virtual do Rio Grande do Sul | os demais estados, e o próprio RS |
| SVC-AN, SVC-RS | contingência |

Na NFC-e menos estados têm ambiente próprio; os que não têm usam a SVRS.

```go
autorizador, _ := sefaz.AutorizadorDe(uf.RJ, nfe.ModeloNFCe) // SVRS
```

## Cliente HTTP próprio

Para instrumentar, passar por proxy corporativo ou aplicar política de
repetição, forneça o seu `*http.Client`:

```go
transporte := &http.Transport{
    TLSClientConfig: &tls.Config{
        Certificates: []tls.Certificate{cert.TLS()},
        MinVersion:   tls.VersionTLS12,
    },
    Proxy: http.ProxyFromEnvironment,
}

cliente, err := sefaz.NovoCliente(sefaz.Config{
    UF: uf.RS, Ambiente: nfe.Producao, Modelo: nfe.ModeloNFe,
    HTTP: &http.Client{Transport: transporte, Timeout: 90 * time.Second},
})
```

Quando você fornece o cliente HTTP, a configuração TLS não é montada
automaticamente — é preciso incluir o certificado no transporte, como acima.
Para trocar só a configuração TLS mantendo o resto, use `Config.TLS`.

## Operações ainda não cobertas

`Chamar` envia uma mensagem já montada a qualquer serviço e devolve a resposta
crua, para quem precisa de algo que a biblioteca ainda não implementa:

```go
resposta, err := cliente.Chamar(ctx, sefaz.ServicoStatus, mensagem)
```

Eventos, inutilização de numeração e distribuição de DF-e estão no roteiro.
