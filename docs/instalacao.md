# Instalação

## Requisitos

- **Go 1.23** ou mais recente.
- Um **certificado digital A1** da ICP-Brasil, em arquivo `.pfx` ou `.p12`, com
  a senha.
- Para NFC-e, o **CSC** — Código de Segurança do Contribuinte — emitido pela
  SEFAZ do seu estado.

Não é preciso instalar bibliotecas nativas, compilador C nem nada além do Go: a
biblioteca não usa CGO.

## Adicionando ao projeto

```bash
go get github.com/mschunke/gonfe
```

Isso traz também a única dependência externa,
`software.sslmate.com/src/go-pkcs12`, usada para ler arquivos PKCS#12 cifrados
com os algoritmos modernos que as autoridades certificadoras emitem hoje.

## Conferindo a instalação

O caminho mais rápido para saber se está tudo no lugar é consultar o status do
serviço da sua SEFAZ. Essa chamada exercita o carregamento do certificado, a
autenticação mútua TLS e o endereço do serviço, tudo de uma vez:

```bash
go run github.com/mschunke/gonfe/exemplos/status-servico@latest \
  -cert ./certificado.pfx -uf RS
```

Ou, se você já clonou o repositório:

```bash
go run ./exemplos/status-servico -cert ./certificado.pfx -uf RS
```

A saída esperada é assim:

```text
certificado: COMERCIO EXEMPLO LTDA (CNPJ 12345678000195), emitido por AC SOLUTI ...
autorizador: SVRS
endereço:    https://nfe-homologacao.sefazrs.rs.gov.br/ws/NfeStatusServico/NfeStatusServico4.asmx
resposta:    107 Servico em Operacao
aplicação:   RS20260304
tudo certo: o ambiente está em operação
```

!!! tip "A senha do certificado"

    Prefira passar a senha pela variável de ambiente `GONFE_SENHA` a digitá-la
    na linha de comando — argumentos de processo ficam visíveis para outros
    usuários da máquina e costumam parar no histórico do shell.

    ```bash
    GONFE_SENHA=... go run ./exemplos/status-servico -cert ./certificado.pfx
    ```

## Se algo der errado

| Sintoma | Causa provável |
| --- | --- |
| `senha incorreta ou arquivo PKCS#12 corrompido` | Senha errada, ou o arquivo não é um PKCS#12 |
| `x509: certificate signed by unknown authority` | A cadeia da ICP-Brasil não está no armazenamento de confiança do sistema |
| `remote error: tls: handshake failure` | O servidor recusou o certificado do cliente; confira a validade e se o certificado é do CNPJ emitente |
| `context deadline exceeded` | Endereço errado, firewall ou proxy no caminho |
| `dial tcp: no such host` | O endereço do serviço mudou; confira no Portal da NF-e e sobreponha em `sefaz.Config.Endpoints` |

## Compilando para contêiner

Sem CGO, a compilação estática é direta:

```dockerfile
FROM golang:1.24 AS build
WORKDIR /src
COPY . .
RUN CGO_ENABLED=0 go build -o /bin/emissor ./cmd/emissor

FROM gcr.io/distroless/static-debian12
COPY --from=build /bin/emissor /emissor
ENTRYPOINT ["/emissor"]
```

A imagem `distroless/static` já traz o pacote de certificados raiz, necessário
para validar o certificado do servidor da SEFAZ no handshake TLS. Se você usar
`scratch`, copie o `ca-certificates.crt` explicitamente.
