# Digital certificate

Issuing electronic fiscal documents depends on an ICP-Brasil digital
certificate. It plays two distinct roles, worth keeping separate in your head:

1. **It signs the XML.** The `infNFe` group is signed with the private key, and
   the signature travels inside the document.
2. **It identifies the client on the connection.** SEFAZ services require mutual
   TLS authentication: the same certificate proves, during the handshake, who is
   connecting.

## A1 and A3

| | A1 | A3 |
| --- | --- | --- |
| Where the key lives | In a `.pfx` or `.p12` file | On a USB token, smart card or HSM |
| Typical validity | 1 year | 1 to 3 years |
| Automation | Straightforward | Needs a PKCS#11 driver and physical presence |
| Supported by GoNFE | Yes, natively | Through the `xmldsig.Assinante` interface |

The [`certificado`](https://pkg.go.dev/github.com/mschunke/gonfe/certificado)
package covers A1 completely, in pure Go and without CGO. A3 certificates can be
used by implementing `xmldsig.Assinante` — see
[Digital signature](assinatura.md#a3-hsm-e-assinatura-remota).

## Loading

```go
cert, err := certificado.CarregarArquivo("certificado.pfx", senha)
if err != nil {
    return err
}
```

There are variants for when you already have the bytes in memory or an
`io.Reader`:

```go
cert, err := certificado.Carregar(dados, senha)      // []byte
cert, err := certificado.CarregarDe(reader, senha)   // io.Reader
```

Loading accepts both the older algorithms — 3DES with SHA-1, still common in
certificates exported by legacy systems — and the modern ones, AES-256 with
PBES2, which certificate authorities issue today.

## Inspecting

```go
fmt.Println(cert.RazaoSocial())   // COMERCIO EXEMPLO LTDA
fmt.Println(cert.CNPJ())          // 12345678000195
fmt.Println(cert.Emissor())       // AC SOLUTI Multipla v5
fmt.Println(cert.ValidoAte())     // 2027-03-04 12:00:00 -0300
fmt.Println(cert.DiasParaVencer())
fmt.Println(cert.Descrever())     // one-line summary for logging
```

The CNPJ comes from the `subjectAltName` extension, where ICP-Brasil records it
under OID `2.16.76.1.3.3`. When that extension is absent, the library falls back
to the common name suffix, which in ICP-Brasil has the form
`RAZAO SOCIAL:CNPJ`. On e-CPF certificates, `CPF()` does the equivalent with OID
`2.16.76.1.3.1`.

## Watching the expiry date

An expired certificate takes issuing down without warning. It is worth checking
at startup and alerting well in advance:

```go
if err := cert.ValidoEm(time.Now()); err != nil {
    return fmt.Errorf("certificado inutilizável: %w", err)
}
if dias := cert.DiasParaVencer(); dias < 30 {
    log.Printf("ATENÇÃO: o certificado vence em %d dias", dias)
}
```

## Keeping the password

!!! danger "The certificate password is the key to the vault"

    Whoever holds the `.pfx` and its password can issue fiscal documents on
    behalf of the company. Treat both with the same care you would give a
    banking password.

Practical recommendations:

- Never commit the `.pfx` or the password. The project `.gitignore` already
  blocks `*.pfx`, `*.p12`, `*.pem` and `*.key`.
- Read the password from an environment variable, a secrets vault or your
  cloud provider's secret manager — not from a plaintext config file.
- In containers, mount the certificate as a secret, not as an image layer.
- Restrict file permissions: `chmod 600 certificado.pfx`.

## Certificate chain

Some SEFAZ services require the complete chain during the TLS handshake. When
the PKCS#12 file carries the intermediates, they are loaded into `cert.Cadeia`
and `cert.TLS()` includes them automatically. If your `.pfx` only holds the leaf
certificate and the handshake fails, export it again including the chain.

In the other direction, to validate the SEFAZ server certificate, Go uses the
operating system's root store. On `scratch` images, copy `ca-certificates.crt`
into the image.

## Testing without a real certificate

The internal `certtest` package generates synthetic certificates with the
structure of an ICP-Brasil A1 — including the CNPJ OIDs in `subjectAltName` —
and it is what the test suite itself uses. It is not exported, but the
[source](https://github.com/mschunke/gonfe/blob/main/internal/certtest/certtest.go)
serves as a reference for building the equivalent in your own project:

```go
// a summary of what certtest does
chave, _ := rsa.GenerateKey(rand.Reader, 2048)
modelo := x509.Certificate{ /* CN "EMPRESA:CNPJ", SAN with OID 2.16.76.1.3.3 */ }
der, _ := x509.CreateCertificate(rand.Reader, &modelo, caCert, &chave.PublicKey, caChave)
folha, _ := x509.ParseCertificate(der)
cert, _ := certificado.De(chave, folha, caCert)
```

`certificado.De` is public precisely to allow that path: any `*rsa.PrivateKey`
with its matching `*x509.Certificate` becomes a usable
`*certificado.Certificado`.
