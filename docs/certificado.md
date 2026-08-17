# Certificado digital

A emissão de documentos fiscais eletrônicos depende de um certificado digital
da ICP-Brasil. Ele cumpre dois papéis distintos, que vale separar na cabeça:

1. **Assina o XML.** O grupo `infNFe` é assinado com a chave privada, e a
   assinatura viaja dentro do documento.
2. **Identifica o cliente na conexão.** Os serviços da SEFAZ exigem
   autenticação mútua TLS: o mesmo certificado prova, no handshake, quem está
   se conectando.

## A1 e A3

| | A1 | A3 |
| --- | --- | --- |
| Onde fica a chave | Em arquivo `.pfx` ou `.p12` | Em token USB, cartão inteligente ou HSM |
| Validade típica | 1 ano | 1 a 3 anos |
| Automação | Direta | Depende de driver PKCS#11 e presença física |
| Suportado pelo GoNFE | Sim, nativamente | Pela interface `xmldsig.Assinante` |

O pacote [`certificado`](https://pkg.go.dev/github.com/mschunke/gonfe/certificado)
cobre A1 inteiramente, em Go puro e sem CGO. Certificados A3 podem ser usados
implementando `xmldsig.Assinante` — veja [Assinatura digital](assinatura.md#a3-hsm-e-assinatura-remota).

## Carregando

```go
cert, err := certificado.CarregarArquivo("certificado.pfx", senha)
if err != nil {
    return err
}
```

Há variantes para quem já tem os bytes em memória ou um `io.Reader`:

```go
cert, err := certificado.Carregar(dados, senha)      // []byte
cert, err := certificado.CarregarDe(reader, senha)   // io.Reader
```

O carregamento aceita tanto os algoritmos antigos — 3DES com SHA-1, ainda comuns
em certificados exportados por sistemas legados — quanto os modernos, AES-256
com PBES2, que as autoridades certificadoras emitem hoje.

## Inspecionando

```go
fmt.Println(cert.RazaoSocial())   // COMERCIO EXEMPLO LTDA
fmt.Println(cert.CNPJ())          // 12345678000195
fmt.Println(cert.Emissor())       // AC SOLUTI Multipla v5
fmt.Println(cert.ValidoAte())     // 2027-03-04 12:00:00 -0300
fmt.Println(cert.DiasParaVencer())
fmt.Println(cert.Descrever())     // resumo de uma linha para log
```

O CNPJ vem da extensão `subjectAltName`, onde a ICP-Brasil o registra sob o OID
`2.16.76.1.3.3`. Quando a extensão não está presente, a biblioteca recorre ao
sufixo do nome comum, que na ICP-Brasil tem o formato `RAZAO SOCIAL:CNPJ`. Em
certificados e-CPF, `CPF()` faz o equivalente com o OID `2.16.76.1.3.1`.

## Vigiando a validade

Certificado vencido derruba a emissão sem aviso prévio. Vale conferir na
inicialização do sistema e alertar com antecedência:

```go
if err := cert.ValidoEm(time.Now()); err != nil {
    return fmt.Errorf("certificado inutilizável: %w", err)
}
if dias := cert.DiasParaVencer(); dias < 30 {
    log.Printf("ATENÇÃO: o certificado vence em %d dias", dias)
}
```

## Guardando a senha

!!! danger "A senha do certificado é a chave do cofre"

    Quem tem o `.pfx` e a senha consegue emitir documentos fiscais em nome da
    empresa. Trate os dois com o mesmo cuidado que você trataria uma senha de
    banco.

Recomendações práticas:

- Nunca versione o `.pfx` nem a senha. O `.gitignore` do projeto já bloqueia
  `*.pfx`, `*.p12`, `*.pem` e `*.key`.
- Leia a senha de variável de ambiente, de um cofre de segredos ou do gerenciador
  de segredos da sua nuvem — não de arquivo de configuração em texto claro.
- Em contêiner, monte o certificado como segredo, não como camada de imagem.
- Restrinja a permissão do arquivo: `chmod 600 certificado.pfx`.

## Cadeia de certificação

Alguns serviços da SEFAZ exigem a cadeia completa no handshake TLS. Quando o
arquivo PKCS#12 traz os intermediários, eles são carregados em `cert.Cadeia` e
`cert.TLS()` os inclui automaticamente. Se o seu `.pfx` só tem o certificado do
titular e o handshake falhar, exporte-o de novo incluindo a cadeia.

Na direção contrária, para validar o certificado do servidor da SEFAZ, o Go usa
o armazenamento de raízes do sistema operacional. Em imagens `scratch`, copie o
`ca-certificates.crt` para dentro da imagem.

## Testes sem certificado real

O pacote interno `certtest` gera certificados sintéticos com a estrutura de um
A1 da ICP-Brasil — inclusive os OIDs do CNPJ no `subjectAltName` — e é o que a
própria suíte de testes usa. Ele não é exportado, mas o
[código](https://github.com/mschunke/gonfe/blob/main/internal/certtest/certtest.go)
serve de referência para montar o equivalente no seu projeto:

```go
// resumo do que certtest faz
chave, _ := rsa.GenerateKey(rand.Reader, 2048)
modelo := x509.Certificate{ /* CN "EMPRESA:CNPJ", SAN com OID 2.16.76.1.3.3 */ }
der, _ := x509.CreateCertificate(rand.Reader, &modelo, caCert, &chave.PublicKey, caChave)
folha, _ := x509.ParseCertificate(der)
cert, _ := certificado.De(chave, folha, caCert)
```

`certificado.De` é público justamente para permitir esse caminho: qualquer
`*rsa.PrivateKey` com o `*x509.Certificate` correspondente vira um
`*certificado.Certificado` utilizável.
