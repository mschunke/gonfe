# Assinatura digital

O pacote [`xmldsig`](https://pkg.go.dev/github.com/mschunke/gonfe/xmldsig)
implementa o perfil de XML Signature adotado pela Receita Federal. O perfil é
estreito e fixo — e é justamente isso que o torna interoperável.

## O perfil da SEFAZ

| Item | Valor |
| --- | --- |
| Tipo | Envelopada (*enveloped*) |
| Referência | Uma só, apontando para o atributo `Id` do bloco assinado |
| Canonicalização | Canonical XML 1.0 sem comentários |
| Resumo | SHA-1 |
| Assinatura | RSA com PKCS#1 v1.5 |
| Chave | Certificado do signatário embutido em `X509Data` |
| Posição | Último filho do elemento que contém o bloco assinado |

Na NF-e e na NFC-e o bloco assinado é o `infNFe`, e o `Signature` entra como
último filho do `NFe` — depois do `infNFeSupl`, quando ele existe.

```xml
<NFe xmlns="http://www.portalfiscal.inf.br/nfe">
  <infNFe Id="NFe43260312345678000195550010000000571102030403" versao="4.00">…</infNFe>
  <infNFeSupl>…</infNFeSupl>
  <Signature xmlns="http://www.w3.org/2000/09/xmldsig#">…</Signature>
</NFe>
```

## Assinando

O caminho mais curto passa pelo próprio documento:

```go
assinada, err := n.AssinarCom(cert)  // prepara, serializa e assina
```

Quando você precisa controlar as etapas — por exemplo, para preencher o QR Code
da NFC-e entre a preparação e a assinatura:

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

Para assinar todas as notas de um lote já montado:

```go
assinadas, err := xmldsig.AssinarTodos(lote, "infNFe", cert)
```

E para outros documentos que sigam a mesma mecânica, basta trocar o nome da tag:

```go
assinado, err := xmldsig.Assinar(evento, "infEvento", cert)
```

## Por que os bytes originais são preservados

Esta é a parte que mais causa dor de cabeça em bibliotecas de NF-e, e vale
entender o mecanismo.

O resumo criptográfico é calculado sobre a **forma canônica** do `infNFe`. Se a
biblioteca reserializasse o documento depois de assinar — reordenando um
atributo, trocando `<a/>` por `<a></a>`, mudando o escape de um caractere — o
resumo que a SEFAZ recalcula ao receber não bateria mais com o `DigestValue`
gravado, e a nota seria rejeitada com "assinatura inválida".

O GoNFE evita isso não reserializando nada. A assinatura é **inserida** nos
bytes originais, imediatamente antes da tag de fechamento do elemento pai:

```go
saida = append(original[:posicaoDoFechamento], assinatura, original[posicaoDoFechamento:])
```

A posição vem do próprio parser, que registra o deslocamento de cada tag de
fechamento durante a leitura. Tudo que estava antes da assinatura continua byte
a byte idêntico, incluindo acentuação, entidades e formatação.

## Canonicalização

A canonicalização C14N 1.0 é implementada em um pacote interno, sobre uma
árvore XML que preserva os prefixos de namespace originais — algo que o
`encoding/xml` da biblioteca padrão descarta.

A implementação é conferida contra os casos 3.2 e 3.3 da
[especificação Canonical XML 1.0](https://www.w3.org/TR/2001/REC-xml-c14n-20010315)
do W3C, que cobrem preservação de espaço em branco, normalização de tags
vazias, ordenação de atributos e propagação de declarações de namespace.

Limitações conhecidas, todas irrelevantes para XML fiscal: comentários são
descartados (é a variante *without comments* que a NF-e usa), DTDs e entidades
externas não são suportadas, e a entrada precisa ser UTF-8.

## Verificando

`Verificar` refaz toda a conta: recalcula o resumo do elemento referenciado,
recanonicaliza o `SignedInfo` e confere a assinatura com a chave pública do
certificado embutido.

```go
if err := xmldsig.Verificar(documento); err != nil {
    return fmt.Errorf("documento adulterado ou mal assinado: %w", err)
}
```

Serve para conferir XMLs recebidos de parceiros antes de importá-los, e para
testar a sua própria pilha de assinatura.

!!! note "Verificação criptográfica, não de confiança"

    `Verificar` prova que o documento não foi alterado depois de assinado e que
    quem assinou detinha a chave privada do certificado embutido. Ela **não**
    valida a cadeia da ICP-Brasil nem consulta listas de revogação. Para isso,
    extraia o certificado com `xmldsig.Certificado(documento)` e valide-o com
    `x509.Certificate.Verify` contra as raízes da ICP-Brasil.

## Erros

Os erros são sentinelas, comparáveis com `errors.Is`:

| Erro | Significado |
| --- | --- |
| `ErrElementoNaoEncontrado` | A tag informada não existe no documento |
| `ErrSemAtributoId` | O elemento a assinar não tem o atributo `Id` |
| `ErrJaAssinado` | Já existe um `Signature` no lugar de destino |
| `ErrSemAssinatura` | O documento verificado não está assinado |
| `ErrAssinaturaInvalida` | O resumo ou a assinatura não conferem |
| `ErrAlgoritmoNaoSuportado` | O documento usa algoritmo fora do perfil da SEFAZ |

## A3, HSM e assinatura remota

`xmldsig.Assinar` não pede um `*certificado.Certificado` — pede uma interface:

```go
type Assinante interface {
    Sign(aleatorio io.Reader, resumo []byte, opcoes crypto.SignerOpts) ([]byte, error)
    DER() []byte
}
```

`Sign` é a assinatura padrão de [`crypto.Signer`](https://pkg.go.dev/crypto#Signer),
e `DER` devolve o certificado X.509 do signatário. Qualquer coisa que consiga
produzir uma assinatura PKCS#1 v1.5 de um resumo SHA-1 serve: um driver PKCS#11
para token A3, um cliente de KMS na nuvem, um HSM de rede ou um serviço interno
de assinatura.

```go
type assinanteA3 struct {
    sessao *pkcs11.Session
    cert   *x509.Certificate
}

func (a *assinanteA3) Sign(_ io.Reader, resumo []byte, _ crypto.SignerOpts) ([]byte, error) {
    return a.sessao.SignPKCS1v15SHA1(resumo)
}

func (a *assinanteA3) DER() []byte { return a.cert.Raw }

// e então
assinada, err := xmldsig.Assinar(documento, "infNFe", &assinanteA3{...})
```

O núcleo da biblioteca continua sem CGO; a dependência nativa fica isolada no
seu adaptador.

!!! info "Autenticação TLS continua sendo caso à parte"

    A assinatura do XML e o handshake TLS usam a mesma chave, mas por caminhos
    diferentes. Com A3, além do `Assinante`, você precisa de um
    `tls.Certificate` cujo `PrivateKey` implemente `crypto.Signer` apontando
    para o token, e passá-lo em `sefaz.Config.TLS`.
