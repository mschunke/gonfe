# SEFAZ web services

The [`sefaz`](https://pkg.go.dev/github.com/mschunke/gonfe/sefaz) package talks
to the authorising environments over SOAP 1.2 on TLS with mutual authentication.
No SOAP library is used — the envelope is built and read directly.

## Building the client

```go
cliente, err := sefaz.NovoCliente(sefaz.Config{
    UF:          uf.RS,
    Ambiente:    nfe.Homologacao,
    Modelo:      nfe.ModeloNFe,
    Certificado: cert,
})
```

The client works out on its own which authorising environment serves your state.
It is safe for concurrent use: it keeps no state between calls, and reuses
connections.

Every operation takes a `context.Context`, so cancellation and deadlines stay
under your control:

```go
ctx, cancelar := context.WithTimeout(context.Background(), 2*time.Minute)
defer cancelar()
```

## Service status

This is the diagnostic call. If it works, the certificate is valid, the TLS
handshake succeeded and the endpoint is right.

```go
status, err := cliente.StatusServico(ctx)
if err != nil {
    return err
}
if !status.EmOperacao() {
    return fmt.Errorf("%d %s", status.CStat, status.XMotivo)
}
```

## Authorisation

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

**In synchronous submission** the protocol comes back in the same response:

```go
if envio.ProtNFe != nil && envio.ProtNFe.Autorizada() {
    proc, _ := nfe.MontarNFeProc(assinada, envio.ProtNFe)
}
```

**In asynchronous submission** you get a receipt, to be queried later:

```go
resultado, err := cliente.EsperarProcessamento(ctx, envio.Recibo(), 3*time.Second, 20)
if err != nil {
    return err
}
prot := resultado.ProtocoloDa(chaveDaNota)
```

`EsperarProcessamento` polls the receipt until the batch leaves the "processing"
state, honouring the interval and the context's cancellation. To drive the loop
yourself, use `ConsultarRecibo` directly.

!!! warning "A rejected batch and a rejected invoice are different things"

    `Autorizar` returns an error when SEFAZ refuses the **batch** — schema
    failure, invalid certificate, wrong environment. When the batch is accepted
    but an **invoice** is rejected, there is no error: the reason is in that
    invoice's protocol `cStat` and `xMotivo`. Always check `prot.Autorizada()`.

## Lookup by access key

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

Useful to check an invoice's status, recover a lost protocol, and list the
events recorded against it — cancellations, letters of correction,
acknowledgements.

## Taxpayer registration lookup

```go
cadastro, err := cliente.ConsultarCadastro(ctx, sefaz.ConsultaCadastro{
    CNPJ: "12345678000195",
})
for _, c := range cadastro.InfCons.InfCad {
    fmt.Println(c.XNome, c.IE, c.Habilitado())
}
```

Not every state offers this service; `URL` returns an error when the authoriser
does not expose it.

## Error handling

SEFAZ rejections become `*sefaz.ErroSefaz`, carrying the original code and
reason:

```go
var rejeicao *sefaz.ErroSefaz
if errors.As(err, &rejeicao) {
    log.Printf("rejeição %d: %s", rejeicao.CStat, rejeicao.XMotivo)
}
```

Network, TLS and HTTP errors come wrapped with the address that failed, which
saves time when the problem is infrastructural.

## Service endpoints

The library embeds a table with the addresses for every state, in both
environments and for both models. To inspect it:

```go
tabela := sefaz.TabelaDeEndpoints(nfe.ModeloNFe, nfe.Producao)
for unidade, servicos := range tabela {
    fmt.Println(unidade, servicos[sefaz.ServicoAutorizacao])
}
```

!!! warning "Check before production"

    States change addresses without notice, and the table reflects what was
    published on the NF-e portal when this version was written. A wrong address
    shows up as a connection failure, not as a rejection.

    Run `exemplos/status-servico` for your state and, if it differs, override
    it:

    ```go
    cliente, err := sefaz.NovoCliente(sefaz.Config{
        UF: uf.RS, Ambiente: nfe.Producao,
        Modelo: nfe.ModeloNFe, Certificado: cert,
        Endpoints: map[sefaz.Servico]string{
            sefaz.ServicoAutorizacao: "https://endereco-correto/NFeAutorizacao4",
        },
    })
    ```

### Authorising environments

Most states delegate processing to a shared environment:

| Authoriser | Serves |
| --- | --- |
| Own environment | AM, BA, GO, MG, MS, MT, PE, PR, SP |
| SVAN — Sefaz Virtual do Ambiente Nacional | MA |
| SVRS — Sefaz Virtual do Rio Grande do Sul | every other state, including RS itself |
| SVC-AN, SVC-RS | contingency |

For the NFC-e fewer states run their own environment; those that do not use
SVRS.

```go
autorizador, _ := sefaz.AutorizadorDe(uf.RJ, nfe.ModeloNFCe) // SVRS
```

## Bringing your own HTTP client

To add instrumentation, go through a corporate proxy or apply a retry policy,
supply your own `*http.Client`:

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

When you supply the HTTP client, the TLS configuration is not built
automatically — you must include the certificate in the transport, as above. To
replace only the TLS configuration and keep the rest, use `Config.TLS`.

## The other documents

The CT-e and the MDF-e have their own endpoints and service names, so they have
their own clients:

```go
clienteCTe, _ := sefaz.NovoClienteCTe(sefaz.ConfigCTe{ /* … */ })
clienteMDFe, _ := sefaz.NovoClienteMDFe(sefaz.ConfigMDFe{ /* … */ })
```

The CT-e client serves both models 57 and 67, recognising which from the root
element of the signed document. The MDF-e is centralised: every state is served
by the Sefaz Virtual do Rio Grande do Sul. See
[CT-e and MDF-e](transporte.md).

Events and voiding of number ranges are in [Events](eventos.md); the queue of
documents issued against you is in [DF-e distribution](distribuicao.md).

## Operations not yet covered

`Chamar` sends an already-assembled message to any service and returns the raw
response, for when you need something the library does not implement yet:

```go
resposta, err := cliente.Chamar(ctx, sefaz.ServicoStatus, mensagem)
```
