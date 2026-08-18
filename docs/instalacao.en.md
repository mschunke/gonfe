# Installation

## Requirements

- **Go 1.23** or newer.
- An **A1 digital certificate** from ICP-Brasil, as a `.pfx` or `.p12` file,
  with its password.
- For NFC-e, the **CSC** — Código de Segurança do Contribuinte — issued by your
  state's SEFAZ.

You do not need native libraries, a C compiler, or anything beyond Go itself:
the library does not use CGO.

## Adding it to your project

```bash
go get github.com/mschunke/gonfe
```

That also pulls in the single external dependency,
`software.sslmate.com/src/go-pkcs12`, used to read PKCS#12 files encrypted with
the modern algorithms certificate authorities issue today.

## Checking the installation

The quickest way to know everything is in place is to query your SEFAZ service
status. That single call exercises certificate loading, mutual TLS
authentication and the service endpoint all at once:

```bash
go run github.com/mschunke/gonfe/exemplos/status-servico@latest \
  -cert ./certificado.pfx -uf RS
```

Or, if you have already cloned the repository:

```bash
go run ./exemplos/status-servico -cert ./certificado.pfx -uf RS
```

The expected output looks like this:

```text
certificado: COMERCIO EXEMPLO LTDA (CNPJ 12345678000195), emitido por AC SOLUTI ...
autorizador: SVRS
endereço:    https://nfe-homologacao.sefazrs.rs.gov.br/ws/NfeStatusServico/NfeStatusServico4.asmx
resposta:    107 Servico em Operacao
aplicação:   RS20260304
tudo certo: o ambiente está em operação
```

!!! tip "The certificate password"

    Prefer passing the password through the `GONFE_SENHA` environment variable
    rather than typing it on the command line — process arguments are visible to
    other users on the machine and usually end up in your shell history.

    ```bash
    GONFE_SENHA=... go run ./exemplos/status-servico -cert ./certificado.pfx
    ```

## If something goes wrong

| Symptom | Likely cause |
| --- | --- |
| `senha incorreta ou arquivo PKCS#12 corrompido` | Wrong password, or the file is not a PKCS#12 |
| `x509: certificate signed by unknown authority` | The ICP-Brasil chain is not in the system trust store |
| `remote error: tls: handshake failure` | The server rejected the client certificate; check its validity and that it belongs to the issuing CNPJ |
| `context deadline exceeded` | Wrong endpoint, or a firewall or proxy in the way |
| `dial tcp: no such host` | The service address changed; check the NF-e portal and override it in `sefaz.Config.Endpoints` |

## Building for a container

With no CGO, static compilation is straightforward:

```dockerfile
FROM golang:1.24 AS build
WORKDIR /src
COPY . .
RUN CGO_ENABLED=0 go build -o /bin/emissor ./cmd/emissor

FROM gcr.io/distroless/static-debian12
COPY --from=build /bin/emissor /emissor
ENTRYPOINT ["/emissor"]
```

The `distroless/static` image already ships the root certificate bundle, which
you need in order to validate the SEFAZ server certificate during the TLS
handshake. If you use `scratch`, copy `ca-certificates.crt` in explicitly.
