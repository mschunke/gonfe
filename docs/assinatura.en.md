# Digital signature

The [`xmldsig`](https://pkg.go.dev/github.com/mschunke/gonfe/xmldsig) package
implements the XML Signature profile adopted by the Receita Federal. The profile
is narrow and fixed — and that is precisely what makes it interoperable.

## The SEFAZ profile

| Item | Value |
| --- | --- |
| Type | Enveloped |
| Reference | Exactly one, pointing at the `Id` attribute of the signed block |
| Canonicalisation | Canonical XML 1.0 without comments |
| Digest | SHA-1 |
| Signature | RSA with PKCS#1 v1.5 |
| Key | The signer's certificate embedded in `X509Data` |
| Position | Last child of the element containing the signed block |

In the NF-e and NFC-e the signed block is `infNFe`, and `Signature` goes in as
the last child of `NFe` — after `infNFeSupl`, when that exists.

```xml
<NFe xmlns="http://www.portalfiscal.inf.br/nfe">
  <infNFe Id="NFe43260312345678000195550010000000571102030403" versao="4.00">…</infNFe>
  <infNFeSupl>…</infNFeSupl>
  <Signature xmlns="http://www.w3.org/2000/09/xmldsig#">…</Signature>
</NFe>
```

## Signing

The shortest path goes through the document itself:

```go
assinada, err := n.AssinarCom(cert)  // prepares, serialises and signs
```

When you need to control the steps — for instance, to fill in the NFC-e QR Code
between preparation and signing:

```go
if err := n.Preparar(); err != nil {
    return err
}
if err := nfce.PreencherSuplemento(n, nfce.Opcoes{CSC: csc}); err != nil {
    return err
}

documento, err := n.XML()
if err != nil {
    return err
}
assinada, err := xmldsig.Assinar(documento, "infNFe", cert)
```

To sign every invoice in a batch you have already assembled:

```go
assinadas, err := xmldsig.AssinarTodos(lote, "infNFe", cert)
```

And for other documents that follow the same mechanics, just change the tag
name:

```go
assinado, err := xmldsig.Assinar(evento, "infEvento", cert)
```

## Why the original bytes are preserved

This is the part that causes the most pain in NF-e libraries, and the mechanism
is worth understanding.

The digest is computed over the **canonical form** of `infNFe`. If the library
re-serialised the document after signing — reordering an attribute, turning
`<a/>` into `<a></a>`, changing how a character is escaped — the digest SEFAZ
recomputes on arrival would no longer match the recorded `DigestValue`, and the
invoice would be rejected with "invalid signature".

GoNFE avoids that by not re-serialising anything. The signature is **inserted**
into the original bytes, immediately before the parent element's closing tag:

```go
saida = append(original[:posicaoDoFechamento], assinatura, original[posicaoDoFechamento:])
```

The position comes from the parser itself, which records the offset of every
closing tag while reading. Everything before the signature stays byte for byte
identical, including accents, entities and formatting.

## Canonicalisation

C14N 1.0 canonicalisation is implemented in an internal package, over an XML
tree that preserves the original namespace prefixes — something the standard
library's `encoding/xml` discards.

The implementation is checked against cases 3.2 and 3.3 of the W3C
[Canonical XML 1.0 specification](https://www.w3.org/TR/2001/REC-xml-c14n-20010315),
which cover whitespace preservation, empty-tag normalisation, attribute ordering
and namespace declaration propagation.

Known limitations, all irrelevant to fiscal XML: comments are discarded (the
NF-e uses the *without comments* variant), DTDs and external entities are not
supported, and the input must be UTF-8.

## Verifying

`Verificar` redoes the whole calculation: it recomputes the digest of the
referenced element, re-canonicalises `SignedInfo` and checks the signature with
the public key of the embedded certificate.

```go
if err := xmldsig.Verificar(documento); err != nil {
    return fmt.Errorf("documento adulterado ou mal assinado: %w", err)
}
```

It is useful for checking XML received from partners before importing it, and
for testing your own signing stack.

!!! note "Cryptographic verification, not trust verification"

    `Verificar` proves the document was not altered after signing and that
    whoever signed held the private key of the embedded certificate. It does
    **not** validate the ICP-Brasil chain, nor consult revocation lists. For
    that, extract the certificate with `xmldsig.Certificado(documento)` and
    validate it with `x509.Certificate.Verify` against the ICP-Brasil roots.

## Errors

The errors are sentinels, comparable with `errors.Is`:

| Error | Meaning |
| --- | --- |
| `ErrElementoNaoEncontrado` | The given tag does not exist in the document |
| `ErrSemAtributoId` | The element to sign has no `Id` attribute |
| `ErrJaAssinado` | A `Signature` already exists at the target position |
| `ErrSemAssinatura` | The document being verified is not signed |
| `ErrAssinaturaInvalida` | The digest or the signature does not check out |
| `ErrAlgoritmoNaoSuportado` | The document uses an algorithm outside the SEFAZ profile |

## A3, HSM and remote signing

`xmldsig.Assinar` does not ask for a `*certificado.Certificado` — it asks for an
interface:

```go
type Assinante interface {
    Sign(aleatorio io.Reader, resumo []byte, opcoes crypto.SignerOpts) ([]byte, error)
    DER() []byte
}
```

`Sign` is the standard [`crypto.Signer`](https://pkg.go.dev/crypto#Signer)
signature, and `DER` returns the signer's X.509 certificate. Anything that can
produce a PKCS#1 v1.5 signature over a SHA-1 digest will do: a PKCS#11 driver
for an A3 token, a cloud KMS client, a network HSM, or an internal signing
service.

```go
type assinanteA3 struct {
    sessao *pkcs11.Session
    cert   *x509.Certificate
}

func (a *assinanteA3) Sign(_ io.Reader, resumo []byte, _ crypto.SignerOpts) ([]byte, error) {
    return a.sessao.SignPKCS1v15SHA1(resumo)
}

func (a *assinanteA3) DER() []byte { return a.cert.Raw }

// and then
assinada, err := xmldsig.Assinar(documento, "infNFe", &assinanteA3{...})
```

The library core stays free of CGO; the native dependency is confined to your
adapter.

!!! info "TLS authentication is still a separate matter"

    Signing the XML and the TLS handshake use the same key, but through
    different paths. With A3, besides the `Assinante`, you need a
    `tls.Certificate` whose `PrivateKey` implements `crypto.Signer` pointing at
    the token, and to pass it in `sefaz.Config.TLS`.
